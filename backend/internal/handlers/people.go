package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListPeople implements GET /api/v1/people.
func (h *Handler) ListPeople(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Person{})
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if d := c.Query("departmentId"); d != "" {
		q = q.Where("department_id = ?", d)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count people", Details: strptr(err.Error())})
		return
	}

	var people []models.Person
	if err := q.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&people).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list people", Details: strptr(err.Error())})
		return
	}

	ids := make([]uuid.UUID, len(people))
	for i := range people {
		ids[i] = people[i].ID
	}
	workloads, err := h.workloadByPerson(ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to compute workload", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Person, len(people))
	for i := range people {
		data[i] = people[i].ToAPI()
		w := workloads[people[i].ID]
		data[i].WorkloadPct = &w
	}

	c.JSON(http.StatusOK, api.PersonList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreatePerson implements POST /api/v1/people.
func (h *Handler) CreatePerson(c *gin.Context) {
	var in api.PersonCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	p := models.NewPersonFromCreate(in)
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create person", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, p.ToAPI())
}

// GetPerson implements GET /api/v1/people/{id}.
func (h *Handler) GetPerson(c *gin.Context) {
	var p models.Person
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "person not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load person", Details: strptr(err.Error())})
		return
	}
	workloads, err := h.workloadByPerson([]uuid.UUID{p.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to compute workload", Details: strptr(err.Error())})
		return
	}
	out := p.ToAPI()
	w := workloads[p.ID]
	out.WorkloadPct = &w
	c.JSON(http.StatusOK, out)
}

// workloadByPerson sums allocation_pct over each person's current project
// memberships (joined on or before today, not yet left) on projects that are
// still open (not completed or cancelled). People with no such membership are
// absent from the map, i.e. read as 0. One query for the whole page.
func (h *Handler) workloadByPerson(ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := make(map[uuid.UUID]float64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		PersonID uuid.UUID
		Total    float64
	}
	err := h.db.Table("project_members AS pm").
		Select("pm.person_id, COALESCE(SUM(pm.allocation_pct), 0) AS total").
		Joins("JOIN projects p ON p.id = pm.project_id").
		Where("pm.person_id IN ?", ids).
		Where("pm.joined_at <= CURRENT_DATE").
		Where("pm.left_at IS NULL OR pm.left_at >= CURRENT_DATE").
		Where("p.status NOT IN ?", []string{"completed", "cancelled"}).
		Group("pm.person_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.PersonID] = r.Total
	}
	return out, nil
}

// UpdatePerson implements PATCH /api/v1/people/{id}.
func (h *Handler) UpdatePerson(c *gin.Context) {
	var p models.Person
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "person not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load person", Details: strptr(err.Error())})
		return
	}

	var in api.PersonUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	p.ApplyUpdate(in)
	if err := h.db.Save(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update person", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, p.ToAPI())
}

// DeletePerson implements DELETE /api/v1/people/{id}.
func (h *Handler) DeletePerson(c *gin.Context) {
	res := h.db.Delete(&models.Person{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete person", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "person not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
