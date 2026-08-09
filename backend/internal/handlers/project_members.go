package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListProjectMembers implements GET /api/v1/project-members.
// Filter by projectId to see a project's team, or by personId to see every
// project a person is assigned to.
func (h *Handler) ListProjectMembers(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.ProjectMember{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}
	if p := c.Query("personId"); p != "" {
		q = q.Where("person_id = ?", p)
	}
	if r := c.Query("role"); r != "" {
		q = q.Where("role = ?", r)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count project members", Details: strptr(err.Error())})
		return
	}

	var members []models.ProjectMember
	if err := q.Order("joined_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list project members", Details: strptr(err.Error())})
		return
	}

	data := make([]api.ProjectMember, len(members))
	for i := range members {
		data[i] = members[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.ProjectMemberList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateProjectMember implements POST /api/v1/project-members.
func (h *Handler) CreateProjectMember(c *gin.Context) {
	var in api.ProjectMemberCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	m := models.NewProjectMemberFromCreate(in)
	if err := h.db.Create(&m).Error; err != nil {
		msg := "failed to assign person to project"
		if isUniqueViolation(err) {
			msg = "this person is already assigned to this project"
		}
		c.JSON(http.StatusBadRequest, api.Error{Message: msg, Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, m.ToAPI())
}

// GetProjectMember implements GET /api/v1/project-members/{id}.
func (h *Handler) GetProjectMember(c *gin.Context) {
	var m models.ProjectMember
	if err := h.db.First(&m, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project member", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, m.ToAPI())
}

// UpdateProjectMember implements PATCH /api/v1/project-members/{id}.
func (h *Handler) UpdateProjectMember(c *gin.Context) {
	var m models.ProjectMember
	if err := h.db.First(&m, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project member", Details: strptr(err.Error())})
		return
	}

	var in api.ProjectMemberUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	m.ApplyUpdate(in)
	if err := h.db.Save(&m).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update project member", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, m.ToAPI())
}

// DeleteProjectMember implements DELETE /api/v1/project-members/{id}.
func (h *Handler) DeleteProjectMember(c *gin.Context) {
	res := h.db.Delete(&models.ProjectMember{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to remove project member", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "project member not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// isUniqueViolation is a best-effort check for a Postgres unique constraint
// error, used to turn a raw DB error into a friendlier message when someone
// tries to assign a person to a project they're already on.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
