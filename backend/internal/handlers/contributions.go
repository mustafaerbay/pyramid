package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListContributions implements GET /api/v1/contributions.
func (h *Handler) ListContributions(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Contribution{})
	// Repeatable ?type=commit&type=issue — a contribution matches if its
	// type is any of the given values. QueryArray returns an empty slice
	// (not an error) when the key is absent, so this is a no-op filter by
	// default, same as the old single-value check.
	if types := c.QueryArray("type"); len(types) > 0 {
		q = q.Where("type IN ?", types)
	}
	if p := c.Query("personId"); p != "" {
		q = q.Where("person_id = ?", p)
	}
	if pr := c.Query("projectId"); pr != "" {
		q = q.Where("project_id = ?", pr)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count contributions", Details: strptr(err.Error())})
		return
	}

	var contributions []models.Contribution
	if err := q.Order("occurred_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&contributions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list contributions", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Contribution, len(contributions))
	for i := range contributions {
		data[i] = contributions[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.ContributionList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateContribution implements POST /api/v1/contributions.
func (h *Handler) CreateContribution(c *gin.Context) {
	var in api.ContributionCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	contribution := models.NewContributionFromCreate(in)
	if err := h.db.Create(&contribution).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create contribution", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, contribution.ToAPI())
}

// GetContribution implements GET /api/v1/contributions/{id}.
func (h *Handler) GetContribution(c *gin.Context) {
	var contribution models.Contribution
	if err := h.db.First(&contribution, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "contribution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load contribution", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, contribution.ToAPI())
}

// UpdateContribution implements PATCH /api/v1/contributions/{id}.
func (h *Handler) UpdateContribution(c *gin.Context) {
	var contribution models.Contribution
	if err := h.db.First(&contribution, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "contribution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load contribution", Details: strptr(err.Error())})
		return
	}

	var in api.ContributionUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	contribution.ApplyUpdate(in)
	if err := h.db.Save(&contribution).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update contribution", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, contribution.ToAPI())
}

// DeleteContribution implements DELETE /api/v1/contributions/{id}.
func (h *Handler) DeleteContribution(c *gin.Context) {
	res := h.db.Delete(&models.Contribution{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete contribution", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "contribution not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
