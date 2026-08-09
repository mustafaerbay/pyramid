package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

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

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count tasks", Details: strptr(err.Error())})
		return
	}

	var tasks []models.Task
	if err := q.Order("number ASC").
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
	for _, l := range links {
		byTask[l.TaskID] = append(byTask[l.TaskID], l.PersonID)
	}
	for i := range tasks {
		tasks[i].AssigneeIDs = byTask[tasks[i].ID]
	}
	return nil
}

// replaceAssignees deletes every existing task_assignees row for taskID and
// inserts one row per (deduplicated) id in assigneeIDs, then leaves the
// task's in-memory AssigneeIDs set to match. A nil/empty assigneeIDs just
// clears the task's assignees.
func (h *Handler) replaceAssignees(t *models.Task, assigneeIDs []uuid.UUID) error {
	if err := h.db.Where("task_id = ?", t.ID).Delete(&models.TaskAssignee{}).Error; err != nil {
		return err
	}

	seen := make(map[uuid.UUID]bool, len(assigneeIDs))
	links := make([]models.TaskAssignee, 0, len(assigneeIDs))
	for _, id := range assigneeIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		links = append(links, models.TaskAssignee{TaskID: t.ID, PersonID: id})
	}

	if len(links) > 0 {
		if err := h.db.Create(&links).Error; err != nil {
			return err
		}
	}

	t.AssigneeIDs = make([]uuid.UUID, len(links))
	for i, l := range links {
		t.AssigneeIDs[i] = l.PersonID
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

	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create task", Details: strptr(err.Error())})
		return
	}

	if in.AssigneeIds != nil {
		if err := h.replaceAssignees(&t, *in.AssigneeIds); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to assign task", Details: strptr(err.Error())})
			return
		}
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
	if err := h.attachAssignees([]models.Task{t}); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
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
	if err := h.db.Save(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update task", Details: strptr(err.Error())})
		return
	}

	// assigneeIds uses replace-the-whole-set semantics: present (even as an
	// empty array) means "these are the assignees now", absent means "leave
	// assignees untouched" — a *[]uuid.UUID pointer already tells those two
	// cases apart cleanly (an empty JSON array unmarshals to a non-nil,
	// zero-length slice), so this doesn't need the raw-body trick above.
	if in.AssigneeIds != nil {
		if err := h.replaceAssignees(&t, *in.AssigneeIds); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update task assignees", Details: strptr(err.Error())})
			return
		}
	} else if err := h.attachAssignees([]models.Task{t}); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task assignees", Details: strptr(err.Error())})
		return
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
	res := h.db.Delete(&models.Task{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete task", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
