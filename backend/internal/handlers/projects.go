package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListProjects implements GET /api/v1/projects.
func (h *Handler) ListProjects(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Project{})
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count projects", Details: strptr(err.Error())})
		return
	}

	var projects []models.Project
	if err := q.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list projects", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Project, len(projects))
	for i := range projects {
		data[i] = projects[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.ProjectList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateProject implements POST /api/v1/projects.
func (h *Handler) CreateProject(c *gin.Context) {
	var in api.ProjectCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	p := models.NewProjectFromCreate(in)
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create project", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, p.ToAPI())
}

// GetProject implements GET /api/v1/projects/{id}.
func (h *Handler) GetProject(c *gin.Context) {
	var p models.Project
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, p.ToAPI())
}

// UpdateProject implements PATCH /api/v1/projects/{id}.
func (h *Handler) UpdateProject(c *gin.Context) {
	var p models.Project
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project", Details: strptr(err.Error())})
		return
	}

	var in api.ProjectUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	if in.CustomFields != nil {
		if err := models.ValidateFields(*in.CustomFields); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	if in.WorkItemTypes != nil {
		if err := models.ValidateItemTypes(*in.WorkItemTypes); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
		keys := []string{""}
		for _, t := range *in.WorkItemTypes {
			keys = append(keys, t.Key)
		}
		var used []string
		h.db.Model(&models.Task{}).Distinct("custom_type").
			Where("project_id = ? AND custom_type IS NOT NULL AND custom_type NOT IN ?", p.ID, keys).Pluck("custom_type", &used)
		if len(used) > 0 {
			c.JSON(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("items still use %s; change their type first", strings.Join(used, ", "))})
			return
		}
	}
	states := p.States()
	if in.Workflow != nil {
		if err := models.ValidateWorkflow(*in.Workflow); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
		keys := make([]string, len(*in.Workflow))
		for i, s := range *in.Workflow {
			keys[i] = s.Key
		}
		var used []string
		h.db.Model(&models.Task{}).Distinct("COALESCE(state, CAST(status AS TEXT))").
			Where("project_id = ? AND COALESCE(state, CAST(status AS TEXT)) NOT IN ?", p.ID, keys).
			Pluck("COALESCE(state, CAST(status AS TEXT))", &used)
		if len(used) > 0 {
			c.JSON(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("items are still in %s; move them to another state first", strings.Join(used, ", "))})
			return
		}
		states = *in.Workflow
	}
	if in.BoardSettings != nil {
		if err := models.ValidateBoard(*in.BoardSettings, states); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
	}
	p.ApplyUpdate(in)
	if err := h.db.Save(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update project", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, p.ToAPI())
}

// DeleteProject implements DELETE /api/v1/projects/{id}.
func (h *Handler) DeleteProject(c *gin.Context) {
	res := h.db.Delete(&models.Project{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete project", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "project not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
