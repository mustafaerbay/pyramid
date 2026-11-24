package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListGoals implements GET /api/v1/goals.
func (h *Handler) ListGoals(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Goal{})
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if o := c.Query("ownerPersonId"); o != "" {
		q = q.Where("owner_person_id = ?", o)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count goals", Details: strptr(err.Error())})
		return
	}

	var goals []models.Goal
	if err := q.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&goals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list goals", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Goal, len(goals))
	for i := range goals {
		data[i] = goals[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.GoalList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateGoal implements POST /api/v1/goals.
func (h *Handler) CreateGoal(c *gin.Context) {
	var in api.GoalCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	g := models.NewGoalFromCreate(in)
	if err := h.db.Create(&g).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create goal", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, g.ToAPI())
}

// GetGoal implements GET /api/v1/goals/{id}.
func (h *Handler) GetGoal(c *gin.Context) {
	var g models.Goal
	if err := h.db.First(&g, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "goal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load goal", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, g.ToAPI())
}

// UpdateGoal implements PATCH /api/v1/goals/{id}.
func (h *Handler) UpdateGoal(c *gin.Context) {
	var g models.Goal
	if err := h.db.First(&g, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "goal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load goal", Details: strptr(err.Error())})
		return
	}

	var in api.GoalUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	g.ApplyUpdate(in)
	if err := h.db.Save(&g).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update goal", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, g.ToAPI())
}

// DeleteGoal implements DELETE /api/v1/goals/{id}.
func (h *Handler) DeleteGoal(c *gin.Context) {
	res := h.db.Delete(&models.Goal{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete goal", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "goal not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
