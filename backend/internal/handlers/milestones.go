package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListMilestones implements GET /api/v1/milestones. Filter by projectId to
// see a single project's milestones — that's the usual case, driving the
// project dashboard's monthly timeline.
func (h *Handler) ListMilestones(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Milestone{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count milestones", Details: strptr(err.Error())})
		return
	}

	// ASC on a nullable date puts NULLs last in Postgres by default, so
	// undated milestones naturally sort to the end rather than the front.
	var milestones []models.Milestone
	if err := q.Order("due_date ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&milestones).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list milestones", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Milestone, len(milestones))
	for i := range milestones {
		data[i] = milestones[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.MilestoneList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateMilestone implements POST /api/v1/milestones.
func (h *Handler) CreateMilestone(c *gin.Context) {
	var in api.MilestoneCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	if in.VersionId != nil && !h.versionInProject(c, *in.VersionId, in.ProjectId) {
		return
	}

	m := models.NewMilestoneFromCreate(in)
	if err := h.db.Create(&m).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create milestone", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, m.ToAPI())
}

// GetMilestone implements GET /api/v1/milestones/{id}.
func (h *Handler) GetMilestone(c *gin.Context) {
	var m models.Milestone
	if err := h.db.First(&m, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "milestone not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load milestone", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, m.ToAPI())
}

// UpdateMilestone implements PATCH /api/v1/milestones/{id}.
func (h *Handler) UpdateMilestone(c *gin.Context) {
	var m models.Milestone
	if err := h.db.First(&m, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "milestone not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load milestone", Details: strptr(err.Error())})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var in api.MilestoneUpdate
	if err := json.Unmarshal(body, &in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	// Explicit `"versionId": null` unassigns; an omitted key leaves it be
	// (see UpdateTask for why this needs the raw body).
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	clearVersion := isExplicitNull(raw, "versionId")
	if !clearVersion && in.VersionId != nil && !h.versionInProject(c, *in.VersionId, m.ProjectID) {
		return
	}

	m.ApplyUpdate(in, clearVersion)
	if err := h.db.Save(&m).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update milestone", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, m.ToAPI())
}

// versionInProject checks that versionID is a version of projectID. When it
// isn't (or doesn't exist) it writes a 400 and returns false.
func (h *Handler) versionInProject(c *gin.Context, versionID, projectID uuid.UUID) bool {
	var n int64
	if err := h.db.Model(&models.Version{}).Where("id = ? AND project_id = ?", versionID, projectID).Count(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to check version", Details: strptr(err.Error())})
		return false
	}
	if n == 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "version does not belong to this project"})
		return false
	}
	return true
}

// DeleteMilestone implements DELETE /api/v1/milestones/{id}.
func (h *Handler) DeleteMilestone(c *gin.Context) {
	res := h.db.Delete(&models.Milestone{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete milestone", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "milestone not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
