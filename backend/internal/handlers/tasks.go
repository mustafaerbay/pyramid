package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListTasks implements GET /api/v1/tasks. Filter by projectId to see a
// single project's tasks — the usual case, driving the project dashboard's
// Tasks section (grouped by milestone client-side). milestoneId/assigneeId/
// status narrow it further; assigneeId matches any task that has that
// person among its (possibly several) assignees.
func (h *Handler) ListTasks(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Task{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}
	if m := c.Query("milestoneId"); m != "" {
		q = q.Where("milestone_id = ?", m)
	}
	if a := c.Query("assigneeId"); a != "" {
		q = q.Where("id IN (?)", h.db.Model(&models.TaskAssignee{}).Select("task_id").Where("person_id = ?", a))
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if ar := c.Query("areaId"); ar != "" {
		ids, err := h.areaAndBelow(ar)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "unknown areaId"})
			return
		}
		q = q.Where("area_id IN ?", ids)
	}
	if sp := c.Query("sprintId"); sp == "none" {
		q = q.Where("sprint_id IS NULL")
	} else if sp != "" {
		q = q.Where("sprint_id = ?", sp)
	}
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		q = q.Where("id IN (?)", h.db.Table("task_labels").
			Select("task_labels.task_id").
			Joins("JOIN labels ON labels.id = task_labels.label_id").
			Where("LOWER(labels.name) = LOWER(?)", tag))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count tasks", Details: strptr(err.Error())})
		return
	}

	var tasks []models.Task
	if err := q.Order("rank ASC, number ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list tasks", Details: strptr(err.Error())})
		return
	}

	if err := h.attachAssignees(tasks); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
		return
	}

	ptrs := make([]*models.Task, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}
	if err := h.attachTags(ptrs...); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load tags", Details: strptr(err.Error())})
		return
	}
	if err := h.attachBlockers(ptrs...); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load dependencies", Details: strptr(err.Error())})
		return
	}
	if err := h.attachRelated(ptrs...); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load links", Details: strptr(err.Error())})
		return
	}
	_ = h.attachCustom(ptrs...)
	if err := h.attachAvailability(ptrs...); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Task, len(tasks))
	for i := range tasks {
		data[i] = tasks[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.TaskList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// attachAssignees batch-loads task_assignees for the given tasks (one query
// for the whole page, not one per task) and populates each Task.AssigneeIDs
// in place.
func (h *Handler) attachAssignees(tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(tasks))
	for i := range tasks {
		ids[i] = tasks[i].ID
	}

	var links []models.TaskAssignee
	if err := h.db.Where("task_id IN ?", ids).Find(&links).Error; err != nil {
		return err
	}

	byTask := make(map[uuid.UUID][]uuid.UUID, len(tasks))
	manDays := make(map[uuid.UUID]map[uuid.UUID]float64, len(tasks))
	for _, l := range links {
		byTask[l.TaskID] = append(byTask[l.TaskID], l.PersonID)
		if manDays[l.TaskID] == nil {
			manDays[l.TaskID] = map[uuid.UUID]float64{}
		}
		manDays[l.TaskID][l.PersonID] = l.ManDays
	}
	for i := range tasks {
		tasks[i].AssigneeIDs = byTask[tasks[i].ID]
		tasks[i].ManDays = manDays[tasks[i].ID]
	}
	return nil
}

// attachAssigneesOne is attachAssignees for a single task. It exists because
// the obvious h.attachAssignees([]models.Task{t}) fills in a COPY of t (the
// slice literal copies the struct), silently leaving t.AssigneeIDs empty —
// which made GET/PATCH /tasks/{id} always return assigneeIds: [].
func (h *Handler) attachAssigneesOne(t *models.Task) error {
	ts := []models.Task{*t}
	if err := h.attachAssignees(ts); err != nil {
		return err
	}
	t.AssigneeIDs = ts[0].AssigneeIDs
	t.ManDays = ts[0].ManDays
	return nil
}

// replaceAssignees deletes every existing task_assignees row for taskID and
// inserts one row per (deduplicated) id in assigneeIDs, then leaves the
// task's in-memory AssigneeIDs set to match. A nil/empty assigneeIDs just
// clears the task's assignees.
func (h *Handler) replaceAssignees(t *models.Task, assigneeIDs []uuid.UUID) error {
	assignments := make([]api.TaskAssignment, len(assigneeIDs))
	for i, id := range assigneeIDs {
		assignments[i] = api.TaskAssignment{PersonId: id}
	}
	return h.replaceAssignments(t, assignments)
}

// replaceAssignments is replaceAssignees with each person's man-days. A
// person listed twice keeps the first entry.
func (h *Handler) replaceAssignments(t *models.Task, assignments []api.TaskAssignment) error {
	if err := h.db.Where("task_id = ?", t.ID).Delete(&models.TaskAssignee{}).Error; err != nil {
		return err
	}

	seen := make(map[uuid.UUID]bool, len(assignments))
	links := make([]models.TaskAssignee, 0, len(assignments))
	for _, a := range assignments {
		if seen[a.PersonId] {
			continue
		}
		seen[a.PersonId] = true
		links = append(links, models.TaskAssignee{TaskID: t.ID, PersonID: a.PersonId, ManDays: a.ManDays})
	}

	if len(links) > 0 {
		if err := h.db.Create(&links).Error; err != nil {
			return err
		}
	}

	t.AssigneeIDs = make([]uuid.UUID, len(links))
	t.ManDays = make(map[uuid.UUID]float64, len(links))
	for i, l := range links {
		t.AssigneeIDs[i] = l.PersonID
		t.ManDays[l.PersonID] = l.ManDays
	}
	return nil
}

// CreateTask implements POST /api/v1/tasks.
func (h *Handler) CreateTask(c *gin.Context) {
	var in api.TaskCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	t := models.NewTaskFromCreate(in)
	if err := t.ValidateDates(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if in.Assignments != nil {
		if err := models.ValidateAssignments(*in.Assignments); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	var tags []string
	if in.Tags != nil {
		var err error
		if tags, err = normalizeTags(*in.Tags); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}

	if err := h.validateParent(&t); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	var blockers []uuid.UUID
	if in.BlockedByIds != nil {
		blockers = dedupeIDs(*in.BlockedByIds)
		if err := h.validateBlockers(&t, blockers); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	var related []uuid.UUID
	if in.RelatedIds != nil {
		related = dedupeIDs(*in.RelatedIds)
	}
	if err := h.validateLinks(&t, related); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.validateSprint(&t); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.validateArea(&t); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	in0 := map[string]any{}
	if in.CustomValues != nil {
		in0 = *in.CustomValues
	}
	values, err := checkCustomValues(h.projectFields(t.ProjectID), in0)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	t.CustomValues = encodeValues(values)
	t.Values = values

	// Per-project sequential number (PYR-42 style), assigned here since it
	// needs a DB read the model constructor shouldn't own.
	var maxNumber int
	if err := h.db.Model(&models.Task{}).
		Where("project_id = ?", in.ProjectId).
		Select("COALESCE(MAX(number), 0)").
		Scan(&maxNumber).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to allocate task number", Details: strptr(err.Error())})
		return
	}
	t.Number = maxNumber + 1
	// New items go to the bottom of the project's order.
	var maxRank float64
	if err := h.db.Model(&models.Task{}).Where("project_id = ?", in.ProjectId).
		Select("COALESCE(MAX(rank), 0)").Scan(&maxRank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to place task", Details: strptr(err.Error())})
		return
	}
	t.Rank = maxRank + rankStep

	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create task", Details: strptr(err.Error())})
		return
	}

	if in.Assignments != nil {
		if err := h.replaceAssignments(&t, *in.Assignments); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to assign task", Details: strptr(err.Error())})
			return
		}
	} else if in.AssigneeIds != nil {
		if err := h.replaceAssignees(&t, *in.AssigneeIds); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to assign task", Details: strptr(err.Error())})
			return
		}
	}

	if err := h.replaceTags(&t, tags); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to tag task", Details: strptr(err.Error())})
		return
	}
	if err := h.replaceBlockers(&t, blockers); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save dependencies", Details: strptr(err.Error())})
		return
	}
	if err := h.replaceRelated(&t, related); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save links", Details: strptr(err.Error())})
		return
	}
	h.logTask(c, "created", t.ID, nil)
	h.notifyTaskChange(c, nil, nil, t)

	if err := h.attachAvailability(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, t.ToAPI())
}

// GetTask implements GET /api/v1/tasks/{id}.
func (h *Handler) GetTask(c *gin.Context) {
	var t models.Task
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task", Details: strptr(err.Error())})
		return
	}
	if err := h.attachAssigneesOne(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
		return
	}
	if err := h.attachTags(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load tags", Details: strptr(err.Error())})
		return
	}
	if err := h.attachBlockers(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load dependencies", Details: strptr(err.Error())})
		return
	}
	if err := h.attachRelated(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load links", Details: strptr(err.Error())})
		return
	}
	_ = h.attachCustom(&t)
	if err := h.attachAvailability(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, t.ToAPI())
}

// UpdateTask implements PATCH /api/v1/tasks/{id}.
func (h *Handler) UpdateTask(c *gin.Context) {
	var t models.Task
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task", Details: strptr(err.Error())})
		return
	}

	// What the task looked like before, for its history.
	if err := h.attachAssigneesOne(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
		return
	}
	if err := h.attachTags(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load tags", Details: strptr(err.Error())})
		return
	}
	if err := h.attachBlockers(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load dependencies", Details: strptr(err.Error())})
		return
	}
	if err := h.attachRelated(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load links", Details: strptr(err.Error())})
		return
	}
	_ = h.attachCustom(&t)
	before := taskSnapshot(t)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	var in api.TaskUpdate
	if err := json.Unmarshal(body, &in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	// A PATCH sends an explicit `null` — not just an omitted key — to move a
	// task out of its milestone. A plain `*uuid.UUID` field can't tell "key
	// absent" from "key present but null" after json.Unmarshal (both land as
	// nil), so re-parse into a raw map to check key presence ourselves
	// before deciding whether to clear.
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	clearMilestone := isExplicitNull(raw, "milestoneId")

	t.ApplyUpdate(in, clearMilestone)
	// These three can be cleared with an explicit null.
	if isExplicitNull(raw, "storyPoints") {
		t.StoryPoints = nil
	}
	if isExplicitNull(raw, "remainingManDays") {
		t.RemainingManDays = nil
	}
	if isExplicitNull(raw, "completedManDays") {
		t.CompletedManDays = nil
	}
	_, parentSent := raw["parentId"]
	if parentSent {
		t.ParentID = nil
		if !isExplicitNull(raw, "parentId") {
			t.ParentID = in.ParentId
		}
	}
	if parentSent || in.Type != nil {
		if err := h.validateParent(&t); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if err := t.ValidateDates(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if in.Assignments != nil {
		if err := models.ValidateAssignments(*in.Assignments); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	var tags []string
	if in.Tags != nil {
		var err error
		if tags, err = normalizeTags(*in.Tags); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	var blockers []uuid.UUID
	if in.BlockedByIds != nil {
		blockers = dedupeIDs(*in.BlockedByIds)
		if err := h.validateBlockers(&t, blockers); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if _, ok := raw["duplicateOfId"]; ok {
		t.DuplicateOfID = nil
		if !isExplicitNull(raw, "duplicateOfId") {
			t.DuplicateOfID = in.DuplicateOfId
		}
	}
	var related []uuid.UUID
	if in.RelatedIds != nil {
		related = dedupeIDs(*in.RelatedIds)
	}
	if err := h.validateLinks(&t, related); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if in.CustomValues != nil {
		values, err := checkCustomValues(h.projectFields(t.ProjectID), *in.CustomValues)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
		t.CustomValues = encodeValues(values)
		t.Values = values
	}
	if _, ok := raw["areaId"]; ok {
		t.AreaID = nil
		if !isExplicitNull(raw, "areaId") {
			t.AreaID = in.AreaId
		}
		if err := h.validateArea(&t); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if _, ok := raw["sprintId"]; ok {
		t.SprintID = nil
		if !isExplicitNull(raw, "sprintId") {
			t.SprintID = in.SprintId
		}
		if err := h.validateSprint(&t); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if err := h.db.Save(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update task", Details: strptr(err.Error())})
		return
	}

	// assigneeIds uses replace-the-whole-set semantics: present (even as an
	// empty array) means "these are the assignees now", absent means "leave
	// assignees untouched" — a *[]uuid.UUID pointer already tells those two
	// cases apart cleanly (an empty JSON array unmarshals to a non-nil,
	// zero-length slice), so this doesn't need the raw-body trick above.
	if in.Assignments != nil {
		if err := h.replaceAssignments(&t, *in.Assignments); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update task assignees", Details: strptr(err.Error())})
			return
		}
	} else if in.AssigneeIds != nil {
		if err := h.replaceAssignees(&t, *in.AssigneeIds); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update task assignees", Details: strptr(err.Error())})
			return
		}
	} else if err := h.attachAssigneesOne(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
		return
	}
	// tags: present (even empty) replaces the set, absent leaves it alone.
	if in.Tags != nil {
		if err := h.replaceTags(&t, tags); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update tags", Details: strptr(err.Error())})
			return
		}
	} else if err := h.attachTags(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load tags", Details: strptr(err.Error())})
		return
	}
	if in.BlockedByIds != nil {
		if err := h.replaceBlockers(&t, blockers); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save dependencies", Details: strptr(err.Error())})
			return
		}
	} else if err := h.attachBlockers(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load dependencies", Details: strptr(err.Error())})
		return
	}
	if in.RelatedIds != nil {
		if err := h.replaceRelated(&t, related); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save links", Details: strptr(err.Error())})
			return
		}
	}
	if err := h.attachAvailability(&t); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave", Details: strptr(err.Error())})
		return
	}
	after := taskSnapshot(t)
	if changes := diffSnapshots(before, after); len(changes) > 0 {
		h.logTask(c, "updated", t.ID, changes)
		h.notifyTaskChange(c, before, after, t)
	}

	c.JSON(http.StatusOK, t.ToAPI())
}

func isExplicitNull(raw map[string]json.RawMessage, key string) bool {
	v, ok := raw[key]
	if !ok {
		return false
	}
	return string(v) == "null"
}

// DeleteTask implements DELETE /api/v1/tasks/{id}. task_assignees rows are
// cleaned up by the DB via ON DELETE CASCADE — nothing to do here.
func (h *Handler) DeleteTask(c *gin.Context) {
	// Postgres cascades these; deleting explicitly keeps other databases
	// (the SQLite tests) from leaving links to a task that's gone.
	if err := h.db.Where("task_id = ? OR depends_on_id = ?", c.Param("id"), c.Param("id")).Delete(&models.TaskDependency{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete task", Details: strptr(err.Error())})
		return
	}
	if err := h.db.Where("task_id = ? OR other_id = ?", c.Param("id"), c.Param("id")).Delete(&relatedPair{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete task", Details: strptr(err.Error())})
		return
	}
	res := h.db.Delete(&models.Task{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete task", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
		return
	}
	h.deleteTaskFiles(c.Param("id"))
	c.Status(http.StatusNoContent)
}

// Work item levels, top first. A child may not be a higher level than its
// parent; the same level may nest (old task-under-task subtasks).
var typeLevel = map[api.TaskType]int{
	api.TaskTypeEpic: 4, api.TaskTypeFeature: 3, api.TaskTypeStory: 2, api.TaskTypeBug: 2, api.TaskTypeTask: 1,
}

const maxHierarchyDepth = 5

func level(t api.TaskType) int {
	if l, ok := typeLevel[t]; ok {
		return l
	}
	return 1
}

// validateParent checks t's place in the hierarchy (with t's new parent and
// type): parent in the same project, no loop, no child above its parent's
// level, at most 5 levels including t's own children.
func (h *Handler) validateParent(t *models.Task) error {
	if !t.Type.Valid() && t.Type != "" {
		return errors.New("unknown work item type")
	}
	var rows []struct {
		ID       uuid.UUID
		ParentID *uuid.UUID
		Type     api.TaskType
	}
	if err := h.db.Model(&models.Task{}).Select("id, parent_id, type").Where("project_id = ?", t.ProjectID).Scan(&rows).Error; err != nil {
		return err
	}
	parentOf := map[uuid.UUID]*uuid.UUID{}
	typeOf := map[uuid.UUID]api.TaskType{}
	children := map[uuid.UUID][]uuid.UUID{}
	for _, r := range rows {
		if r.ID == t.ID {
			continue
		}
		parentOf[r.ID] = r.ParentID
		typeOf[r.ID] = r.Type
		if r.ParentID != nil {
			children[*r.ParentID] = append(children[*r.ParentID], r.ID)
		}
	}

	if t.ID != uuid.Nil {
		for _, kid := range children[t.ID] {
			if level(typeOf[kid]) > level(t.Type) {
				return fmt.Errorf("a %s can't sit above a %s", t.Type, typeOf[kid])
			}
		}
	}
	if t.ParentID == nil {
		return nil
	}
	pid := *t.ParentID
	if pid == t.ID {
		return errors.New("a work item can't be its own parent")
	}
	ptype, ok := typeOf[pid]
	if !ok {
		var n int64
		h.db.Model(&models.Task{}).Where("id = ?", pid).Count(&n)
		if n == 0 {
			return errors.New("parent task not found")
		}
		return errors.New("parent task must be in the same project")
	}
	if level(t.Type) > level(ptype) {
		return fmt.Errorf("a %s can't go under a %s", t.Type, ptype)
	}
	// Walk up from the parent: meeting t is a loop; count the levels above.
	depth := 0
	for cur := &pid; cur != nil && depth <= maxHierarchyDepth+1; cur = parentOf[*cur] {
		if *cur == t.ID {
			return errors.New("that would put a work item under its own child")
		}
		depth++
	}
	// Levels t brings along: itself plus its deepest branch.
	var height func(id uuid.UUID, seen int) int
	height = func(id uuid.UUID, seen int) int {
		best := 0
		if seen > maxHierarchyDepth {
			return best
		}
		for _, kid := range children[id] {
			best = max(best, height(kid, seen+1))
		}
		return best + 1
	}
	h1 := 1
	if t.ID != uuid.Nil {
		h1 = height(t.ID, 0)
	}
	if depth+h1 > maxHierarchyDepth {
		return fmt.Errorf("work items can be nested at most %d levels deep", maxHierarchyDepth)
	}
	return nil
}
