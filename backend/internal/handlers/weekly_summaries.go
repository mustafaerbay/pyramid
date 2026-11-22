package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListWeeklySummaries implements GET /api/v1/weekly-summaries. Filter by
// projectId to see a single project's weekly summary log — the usual case,
// driving the project dashboard's Weekly Summary section. Ordered
// most-recent-week first so the log reads like a history feed.
func (h *Handler) ListWeeklySummaries(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.WeeklySummary{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count weekly summaries", Details: strptr(err.Error())})
		return
	}

	var summaries []models.WeeklySummary
	if err := q.Order("week_of DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&summaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list weekly summaries", Details: strptr(err.Error())})
		return
	}

	data := make([]api.WeeklySummary, len(summaries))
	for i := range summaries {
		data[i] = summaries[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.WeeklySummaryList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateWeeklySummary implements POST /api/v1/weekly-summaries.
func (h *Handler) CreateWeeklySummary(c *gin.Context) {
	var in api.WeeklySummaryCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	w := models.NewWeeklySummaryFromCreate(in)
	if err := h.db.Create(&w).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create weekly summary", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, w.ToAPI())
}

// GetWeeklySummary implements GET /api/v1/weekly-summaries/{id}.
func (h *Handler) GetWeeklySummary(c *gin.Context) {
	var w models.WeeklySummary
	if err := h.db.First(&w, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "weekly summary not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load weekly summary", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, w.ToAPI())
}

// UpdateWeeklySummary implements PATCH /api/v1/weekly-summaries/{id}.
func (h *Handler) UpdateWeeklySummary(c *gin.Context) {
	var w models.WeeklySummary
	if err := h.db.First(&w, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "weekly summary not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load weekly summary", Details: strptr(err.Error())})
		return
	}

	var in api.WeeklySummaryUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	w.ApplyUpdate(in)
	if err := h.db.Save(&w).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update weekly summary", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, w.ToAPI())
}

// DeleteWeeklySummary implements DELETE /api/v1/weekly-summaries/{id}.
func (h *Handler) DeleteWeeklySummary(c *gin.Context) {
	res := h.db.Delete(&models.WeeklySummary{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete weekly summary", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "weekly summary not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
