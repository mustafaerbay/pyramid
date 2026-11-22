package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

	plan, err := h.monthPlan(models.MonthStart(time.Now().UTC()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to compute workload", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Person, len(people))
	for i := range people {
		data[i] = withWorkload(people[i], plan)
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

	if err := models.ValidateCapacity(in.MonthlyCapacityManDays); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
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
	plan, err := h.monthPlan(models.MonthStart(time.Now().UTC()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to compute workload", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, withWorkload(p, plan))
}

// withWorkload converts p and fills this month's man-days and workload %
// (planned ÷ capacity; 0 when capacity is 0).
func withWorkload(p models.Person, plan monthPlan) api.Person {
	out := p.ToAPI()
	md := plan.byPerson[p.ID]
	pct := 0.0
	if cap := p.Capacity(); cap > 0 {
		pct = md / cap * 100
	}
	out.MonthManDays = &md
	out.WorkloadPct = &pct
	return out
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

	if err := models.ValidateCapacity(in.MonthlyCapacityManDays); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
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
