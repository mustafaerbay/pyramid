package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

const maxQueryResults = 500

// enrichTasks loads everything ToAPI needs for a page of tasks.
func (h *Handler) enrichTasks(tasks []models.Task) error {
	if err := h.attachAssignees(tasks); err != nil {
		return err
	}
	ptrs := make([]*models.Task, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}
	for _, f := range []func(...*models.Task) error{h.attachTags, h.attachBlockers, h.attachRelated, h.attachAvailability} {
		if err := f(ptrs...); err != nil {
			return err
		}
	}
	return nil
}

func nonEmpty[T any](p *[]T) bool { return p != nil && len(*p) > 0 }

// filterTasks turns query filters into a WHERE on tasks.
func (h *Handler) filterTasks(f api.QueryFilters) (*gorm.DB, error) {
	q := h.db.Model(&models.Task{})
	if nonEmpty(f.ProjectIds) {
		q = q.Where("tasks.project_id IN ?", *f.ProjectIds)
	}
	if nonEmpty(f.Types) {
		q = q.Where("tasks.type IN ?", *f.Types)
	}
	if nonEmpty(f.Statuses) {
		q = q.Where("tasks.status IN ?", *f.Statuses)
	}
	if nonEmpty(f.Priorities) {
		q = q.Where("tasks.priority IN ?", *f.Priorities)
	}
	if nonEmpty(f.AssigneeIds) {
		q = q.Where("tasks.id IN (?)", h.db.Model(&models.TaskAssignee{}).Select("task_id").Where("person_id IN ?", *f.AssigneeIds))
	}
	if f.Unassigned != nil && *f.Unassigned {
		q = q.Where("tasks.id NOT IN (?)", h.db.Model(&models.TaskAssignee{}).Select("task_id"))
	}
	if nonEmpty(f.Tags) {
		lower := make([]string, len(*f.Tags))
		for i, t := range *f.Tags {
			lower[i] = strings.ToLower(strings.TrimSpace(t))
		}
		q = q.Where("tasks.id IN (?)", h.db.Table("task_labels").Select("task_labels.task_id").
			Joins("JOIN labels ON labels.id = task_labels.label_id").Where("LOWER(labels.name) IN ?", lower))
	}
	if nonEmpty(f.SprintIds) {
		q = q.Where("tasks.sprint_id IN ?", *f.SprintIds)
	}
	if nonEmpty(f.AreaIds) {
		var all []uuid.UUID
		for _, id := range *f.AreaIds {
			ids, err := h.areaAndBelow(id.String())
			if err != nil {
				return nil, errors.New("unknown area in areaIds")
			}
			all = append(all, ids...)
		}
		q = q.Where("tasks.area_id IN ?", all)
	}
	if f.Text != nil && strings.TrimSpace(*f.Text) != "" {
		esc := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(strings.TrimSpace(*f.Text)))
		q = q.Where(`LOWER(tasks.title) LIKE ? ESCAPE '\'`, "%"+esc+"%")
	}
	if f.DueAfter != nil {
		q = q.Where("tasks.due_date >= ?", f.DueAfter.Time)
	}
	if f.DueBefore != nil {
		q = q.Where("tasks.due_date <= ?", f.DueBefore.Time)
	}
	return q, nil
}

// RunQuery implements POST /api/v1/queries/run.
func (h *Handler) RunQuery(c *gin.Context) {
	var f api.QueryFilters
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid filters", Details: strptr(err.Error())})
		return
	}
	q, err := h.filterTasks(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to run query", Details: strptr(err.Error())})
		return
	}
	var tasks []models.Task
	if err := q.Joins("LEFT JOIN projects ON projects.id = tasks.project_id").
		Order("COALESCE(projects.name, ''), tasks.project_id, tasks.rank, tasks.number").
		Limit(maxQueryResults).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to run query", Details: strptr(err.Error())})
		return
	}
	if err := h.enrichTasks(tasks); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load results", Details: strptr(err.Error())})
		return
	}
	out := make([]api.Task, len(tasks))
	for i := range tasks {
		out[i] = tasks[i].ToAPI()
	}
	c.JSON(http.StatusOK, api.QueryResult{Data: out, Total: int(total), Truncated: total > maxQueryResults})
}

// --- saved queries ---

type savedQuery struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	Filters   string    `gorm:"type:jsonb;not null"`
	GroupBy   string    `gorm:"not null"`
	Shared    bool      `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (savedQuery) TableName() string { return "saved_queries" }

func (h *Handler) savedJSON(q savedQuery, me uuid.UUID) api.SavedQuery {
	var f api.QueryFilters
	_ = json.Unmarshal([]byte(q.Filters), &f)
	out := api.SavedQuery{
		Id: q.ID, Name: q.Name, Filters: f, GroupBy: api.QueryGroupBy(q.GroupBy), Shared: q.Shared,
		Mine: q.OwnerID == me, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
	var u models.User
	if h.db.First(&u, "id = ?", q.OwnerID).Error == nil {
		pid := u.PersonID
		out.OwnerPersonId = &pid
	}
	return out
}

func readSavedInput(c *gin.Context, q *savedQuery) bool {
	var in api.SavedQueryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return false
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a saved query needs a name of 1–100 characters"})
		return false
	}
	b, _ := json.Marshal(in.Filters)
	q.Name, q.Filters = name, string(b)
	if in.GroupBy != nil {
		if !in.GroupBy.Valid() {
			c.JSON(http.StatusBadRequest, api.Error{Message: "unknown groupBy"})
			return false
		}
		q.GroupBy = string(*in.GroupBy)
	}
	if q.GroupBy == "" {
		q.GroupBy = string(api.QueryGroupByStatus)
	}
	if in.Shared != nil {
		q.Shared = *in.Shared
	}
	return true
}

// ListSavedQueries implements GET /api/v1/queries.
func (h *Handler) ListSavedQueries(c *gin.Context) {
	u, _ := currentUser(c)
	var rows []savedQuery
	if err := h.db.Where("owner_id = ? OR shared = ?", u.ID, true).Order("name").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list queries", Details: strptr(err.Error())})
		return
	}
	out := make([]api.SavedQuery, len(rows))
	for i, q := range rows {
		out[i] = h.savedJSON(q, u.ID)
	}
	c.JSON(http.StatusOK, api.SavedQueryList{Data: out})
}

// CreateSavedQuery implements POST /api/v1/queries.
func (h *Handler) CreateSavedQuery(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	q := savedQuery{OwnerID: u.ID}
	if !readSavedInput(c, &q) {
		return
	}
	if err := h.db.Create(&q).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save query", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, h.savedJSON(q, u.ID))
}

// loadSavedQuery finds a query the caller can see (own or shared).
func (h *Handler) loadSavedQuery(c *gin.Context) (savedQuery, models.User, bool) {
	u, _ := currentUser(c)
	var q savedQuery
	err := h.db.Where("id = ? AND (owner_id = ? OR shared = ?)", c.Param("id"), u.ID, true).First(&q).Error
	if err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "saved query not found"})
		return q, u, false
	}
	return q, u, true
}

// UpdateSavedQuery implements PATCH /api/v1/queries/{id}.
func (h *Handler) UpdateSavedQuery(c *gin.Context) {
	q, u, ok := h.loadSavedQuery(c)
	if !ok {
		return
	}
	if q.OwnerID != u.ID {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner can change a saved query"})
		return
	}
	if !readSavedInput(c, &q) {
		return
	}
	if err := h.db.Save(&q).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save query", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, h.savedJSON(q, u.ID))
}

// DeleteSavedQuery implements DELETE /api/v1/queries/{id}.
func (h *Handler) DeleteSavedQuery(c *gin.Context) {
	q, u, ok := h.loadSavedQuery(c)
	if !ok {
		return
	}
	if q.OwnerID != u.ID && !h.hasRole(c, api.SystemRoleAdmin) {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner or an admin can delete a saved query"})
		return
	}
	if err := h.db.Delete(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete query", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
