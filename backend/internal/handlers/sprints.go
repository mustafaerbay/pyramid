package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"
)

func validSprint(s models.Sprint) error {
	if strings.TrimSpace(s.Name) == "" || utf8.RuneCountInString(s.Name) > 100 {
		return errors.New("a sprint needs a name of 1–100 characters")
	}
	if s.StartDate.IsZero() || s.EndDate.IsZero() {
		return errors.New("a sprint needs a start and an end date")
	}
	if s.EndDate.Before(s.StartDate) {
		return errors.New("the end date must be on or after the start date")
	}
	if !s.Status.Valid() {
		return errors.New("unknown sprint status")
	}
	return nil
}

// validateSprint checks that t's sprint (if any) belongs to t's project.
func (h *Handler) validateSprint(t *models.Task) error {
	if t.SprintID == nil {
		return nil
	}
	var n int64
	if err := h.db.Model(&models.Sprint{}).Where("id = ? AND project_id = ?", *t.SprintID, t.ProjectID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errors.New("the sprint must belong to the task's project")
	}
	return nil
}

// sprintTasks loads the non-cancelled items of the sprints with their
// assignees, and which of them carry work (no live children).
func (h *Handler) sprintTasks(ids []uuid.UUID) ([]models.Task, map[uuid.UUID]bool, error) {
	var tasks []models.Task
	if len(ids) == 0 {
		return nil, nil, nil
	}
	if err := h.db.Where("sprint_id IN ? AND status <> ?", ids, api.TaskStatusCancelled).Find(&tasks).Error; err != nil {
		return nil, nil, err
	}
	if err := h.attachAssignees(tasks); err != nil {
		return nil, nil, err
	}
	taskIDs := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}
	var parents []uuid.UUID
	if len(taskIDs) > 0 {
		if err := h.db.Model(&models.Task{}).Distinct("parent_id").
			Where("parent_id IN ? AND status <> ?", taskIDs, api.TaskStatusCancelled).Pluck("parent_id", &parents).Error; err != nil {
			return nil, nil, err
		}
	}
	leaf := map[uuid.UUID]bool{}
	for _, t := range tasks {
		leaf[t.ID] = true
	}
	for _, p := range parents {
		leaf[p] = false
	}
	return tasks, leaf, nil
}

func taskPoints(t models.Task) float64 {
	if t.StoryPoints == nil {
		return 0
	}
	return *t.StoryPoints
}

func (h *Handler) sprintStats(sprints []models.Sprint) (map[uuid.UUID]api.SprintStats, error) {
	ids := make([]uuid.UUID, len(sprints))
	for i, s := range sprints {
		ids[i] = s.ID
	}
	tasks, leaf, err := h.sprintTasks(ids)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]api.SprintStats{}
	for _, t := range tasks {
		st := out[*t.SprintID]
		done := t.Status == api.TaskStatusDone
		st.TaskCount++
		st.Points += taskPoints(t)
		md := 0.0
		if leaf[t.ID] {
			md = t.ToAPI().ManDays
		}
		st.ManDays += md
		if done {
			st.DoneCount++
			st.DonePoints += taskPoints(t)
			st.DoneManDays += md
		}
		out[*t.SprintID] = st
	}
	return out, nil
}

func (h *Handler) sprintJSON(c *gin.Context, status int, s models.Sprint) {
	stats, err := h.sprintStats([]models.Sprint{s})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load sprint", Details: strptr(err.Error())})
		return
	}
	c.JSON(status, s.ToAPI(stats[s.ID]))
}

func (h *Handler) loadSprint(c *gin.Context) (models.Sprint, bool) {
	var s models.Sprint
	if err := h.db.First(&s, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "sprint not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load sprint", Details: strptr(err.Error())})
		}
		return s, false
	}
	return s, true
}

// ListSprints implements GET /api/v1/sprints.
func (h *Handler) ListSprints(c *gin.Context) {
	projectID := c.Query("projectId")
	if _, err := uuid.Parse(projectID); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "projectId is required"})
		return
	}
	var sprints []models.Sprint
	if err := h.db.Where("project_id = ?", projectID).Order("start_date, name").Find(&sprints).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list sprints", Details: strptr(err.Error())})
		return
	}
	stats, err := h.sprintStats(sprints)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list sprints", Details: strptr(err.Error())})
		return
	}
	out := make([]api.Sprint, len(sprints))
	for i, s := range sprints {
		out[i] = s.ToAPI(stats[s.ID])
	}
	c.JSON(http.StatusOK, api.SprintList{Data: out})
}

// CreateSprint implements POST /api/v1/sprints.
func (h *Handler) CreateSprint(c *gin.Context) {
	var in api.SprintCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	s := models.Sprint{ProjectID: in.ProjectId, Name: strings.TrimSpace(in.Name), Goal: in.Goal,
		Status: api.SprintStatusPlanned, StartDate: in.StartDate.Time, EndDate: in.EndDate.Time}
	if in.Status != nil {
		s.Status = *in.Status
	}
	s.TeamID = in.TeamId
	if !h.canWriteProject(c, s.ProjectID) {
		return
	}
	if err := validSprint(s); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.validateSprintTeam(s); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.db.Create(&s).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create sprint", Details: strptr(err.Error())})
		return
	}
	h.sprintJSON(c, http.StatusCreated, s)
}

// GetSprint implements GET /api/v1/sprints/{id}.
func (h *Handler) GetSprint(c *gin.Context) {
	if s, ok := h.loadSprint(c); ok {
		h.sprintJSON(c, http.StatusOK, s)
	}
}

// UpdateSprint implements PATCH /api/v1/sprints/{id}.
func (h *Handler) UpdateSprint(c *gin.Context) {
	s, ok := h.loadSprint(c)
	if !ok {
		return
	}
	if !h.canWriteProject(c, s.ProjectID) {
		return
	}
	body, _ := io.ReadAll(c.Request.Body)
	var in api.SprintUpdate
	if err := json.Unmarshal(body, &in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	if _, ok := raw["teamId"]; ok {
		s.TeamID = in.TeamId
		if isExplicitNull(raw, "teamId") {
			s.TeamID = nil
		}
		if err := h.validateSprintTeam(s); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if in.Name != nil {
		s.Name = strings.TrimSpace(*in.Name)
	}
	if in.Goal != nil {
		s.Goal = in.Goal
	}
	if in.Status != nil {
		s.Status = *in.Status
	}
	if in.StartDate != nil {
		s.StartDate = in.StartDate.Time
	}
	if in.EndDate != nil {
		s.EndDate = in.EndDate.Time
	}
	if err := validSprint(s); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.db.Save(&s).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update sprint", Details: strptr(err.Error())})
		return
	}
	h.sprintJSON(c, http.StatusOK, s)
}

// DeleteSprint implements DELETE /api/v1/sprints/{id}.
func (h *Handler) DeleteSprint(c *gin.Context) {
	s, ok := h.loadSprint(c)
	if !ok {
		return
	}
	if !h.canWriteProject(c, s.ProjectID) {
		return
	}
	if err := h.db.Model(&models.Task{}).Where("sprint_id = ?", s.ID).Update("sprint_id", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete sprint", Details: strptr(err.Error())})
		return
	}
	if err := h.db.Delete(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete sprint", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}

type statusChange struct {
	at     time.Time
	status string
}

// statusTimelines reads each task's status changes from its history.
func (h *Handler) statusTimelines(ids []uuid.UUID) (map[uuid.UUID][]statusChange, error) {
	out := map[uuid.UUID][]statusChange{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.ActivityLog
	if err := h.db.Where("entity_type = ? AND entity_id IN ? AND action = ?", entityTask, ids, "updated").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Changes == nil || r.EntityID == nil {
			continue
		}
		var ch map[string]api.TaskFieldChange
		if json.Unmarshal([]byte(*r.Changes), &ch) != nil {
			continue
		}
		st, ok := ch["status"]
		if !ok {
			continue
		}
		from, _ := st.From.(string)
		to, _ := st.To.(string)
		out[*r.EntityID] = append(out[*r.EntityID], statusChange{at: r.CreatedAt, status: from}, statusChange{at: r.CreatedAt, status: to})
	}
	for id := range out {
		sort.SliceStable(out[id], func(i, j int) bool { return out[id][i].at.Before(out[id][j].at) })
	}
	return out, nil
}

// statusAt is a task's status at time `at`, from its change pairs (from,
// to). Before the first change it had the first change's "from".
func statusAt(current string, changes []statusChange, at time.Time) string {
	if len(changes) == 0 {
		return current
	}
	status := changes[0].status
	for i := 0; i+1 < len(changes); i += 2 {
		if changes[i].at.After(at) {
			break
		}
		status = changes[i+1].status
	}
	return status
}

// GetSprintReport implements GET /api/v1/sprints/{id}/report.
func (h *Handler) GetSprintReport(c *gin.Context) {
	s, ok := h.loadSprint(c)
	if !ok {
		return
	}
	fail := func(err error) {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to build sprint report", Details: strptr(err.Error())})
	}
	tasks, leaf, err := h.sprintTasks([]uuid.UUID{s.ID})
	if err != nil {
		fail(err)
		return
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	timelines, err := h.statusTimelines(ids)
	if err != nil {
		fail(err)
		return
	}

	// Days: start .. min(end, today).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	last := s.EndDate
	if today.Before(last) {
		last = today
	}
	days := []api.SprintDay{}
	for d := s.StartDate; !d.After(last); d = d.AddDate(0, 0, 1) {
		endOfDay := d.AddDate(0, 0, 1)
		day := api.SprintDay{Date: openapi_types.Date{Time: d}}
		for _, t := range tasks {
			if !t.CreatedAt.Before(endOfDay) {
				continue
			}
			st := statusAt(string(t.Status), timelines[t.ID], endOfDay)
			if st == string(api.TaskStatusCancelled) {
				continue
			}
			md := 0.0
			if leaf[t.ID] {
				md = t.ToAPI().ManDays
			}
			day.ScopeCount++
			day.ScopePoints += taskPoints(t)
			day.ScopeManDays += md
			if st == string(api.TaskStatusDone) {
				day.DoneCount++
				day.DonePoints += taskPoints(t)
				day.DoneManDays += md
			}
		}
		days = append(days, day)
	}

	// Capacity: project members plus anyone with work in the sprint.
	planned := map[uuid.UUID]float64{}
	for _, t := range tasks {
		if !leaf[t.ID] {
			continue
		}
		for pid, md := range t.ManDays {
			planned[pid] += md
		}
	}
	var memberIDs []uuid.UUID
	if err := h.db.Model(&models.ProjectMember{}).Where("project_id = ?", s.ProjectID).Pluck("person_id", &memberIDs).Error; err != nil {
		fail(err)
		return
	}
	seen := map[uuid.UUID]bool{}
	var personIDs []uuid.UUID
	for _, id := range memberIDs {
		if !seen[id] {
			seen[id] = true
			personIDs = append(personIDs, id)
		}
	}
	for id := range planned {
		if !seen[id] {
			seen[id] = true
			personIDs = append(personIDs, id)
		}
	}
	capacity := []api.SprintCapacity{}
	if len(personIDs) > 0 {
		var people []models.Person
		if err := h.db.Where("id IN ?", personIDs).Order("first_name, last_name").Find(&people).Error; err != nil {
			fail(err)
			return
		}
		leave, err := h.approvedLeave(personIDs)
		if err != nil {
			fail(err)
			return
		}
		for _, p := range people {
			avail, leaveDays := models.SprintCapacity(p.Capacity(), s.StartDate, s.EndDate, leave[p.ID])
			capacity = append(capacity, api.SprintCapacity{
				PersonId: p.ID, AvailableManDays: avail, PlannedManDays: planned[p.ID], LeaveDays: leaveDays,
			})
		}
	}

	stats, err := h.sprintStats([]models.Sprint{s})
	if err != nil {
		fail(err)
		return
	}
	c.JSON(http.StatusOK, api.SprintReport{Sprint: s.ToAPI(stats[s.ID]), Days: days, Capacity: capacity})
}
