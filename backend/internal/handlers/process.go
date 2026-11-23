package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Process templates: a project's workflow, types, fields, rules and board
// settings saved under a name, to give to other projects.

type processTemplate struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name        string     `gorm:"not null"`
	Description *string    `gorm:"type:text"`
	Process     string     `gorm:"type:jsonb;not null"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

func (processTemplate) TableName() string { return "process_templates" }

func processOf(p models.Project) api.Process {
	return api.Process{Workflow: p.States(), WorkItemTypes: p.ItemTypes(), CustomFields: p.Fields(), Rules: p.RulesOf(), BoardSettings: p.Board()}
}

func (t processTemplate) toAPI() api.ProcessTemplate {
	var pr api.Process
	_ = json.Unmarshal([]byte(t.Process), &pr)
	return api.ProcessTemplate{Id: t.ID, Name: t.Name, Description: t.Description, Process: pr, CreatedAt: t.CreatedAt}
}

// ListProcessTemplates implements GET /api/v1/process-templates.
func (h *Handler) ListProcessTemplates(c *gin.Context) {
	var rows []processTemplate
	h.db.Order("name").Find(&rows)
	out := make([]api.ProcessTemplate, len(rows))
	for i, t := range rows {
		out[i] = t.toAPI()
	}
	c.JSON(http.StatusOK, api.ProcessTemplateList{Data: out})
}

// CreateProcessTemplate implements POST /api/v1/process-templates.
func (h *Handler) CreateProcessTemplate(c *gin.Context) {
	var in api.ProcessTemplateCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a template needs a name and fromProjectId"})
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a template needs a name of 1–60 characters"})
		return
	}
	var n int64
	h.db.Model(&processTemplate{}).Where("LOWER(name) = LOWER(?)", name).Count(&n)
	if n > 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "there's already a template with that name"})
		return
	}
	var p models.Project
	if err := h.db.First(&p, "id = ?", in.FromProjectId).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "project not found"})
		return
	}
	b, _ := json.Marshal(processOf(p))
	t := processTemplate{ID: uuid.New(), Name: name, Description: in.Description, Process: string(b), CreatedAt: time.Now()}
	if u, ok := currentUser(c); ok {
		uid := u.ID
		t.CreatedBy = &uid
	}
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save template", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, t.toAPI())
}

// DeleteProcessTemplate implements DELETE /api/v1/process-templates/{id}.
func (h *Handler) DeleteProcessTemplate(c *gin.Context) {
	res := h.db.Delete(&processTemplate{}, "id = ?", c.Param("id"))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "template not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ApplyProcess implements POST /api/v1/projects/{id}/apply-process: the
// change goes through the same checks as editing the project.
func (h *Handler) ApplyProcess(c *gin.Context) {
	var p models.Project
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "project not found"})
		return
	}
	var in api.ApplyProcess
	_ = c.ShouldBindJSON(&in)
	var pr api.Process
	switch {
	case in.TemplateId != nil && in.FromProjectId == nil:
		var t processTemplate
		if err := h.db.First(&t, "id = ?", *in.TemplateId).Error; err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "template not found"})
			return
		}
		pr = t.toAPI().Process
	case in.FromProjectId != nil && in.TemplateId == nil:
		var src models.Project
		if err := h.db.First(&src, "id = ?", *in.FromProjectId).Error; err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "source project not found"})
			return
		}
		pr = processOf(src)
	default:
		c.JSON(http.StatusBadRequest, api.Error{Message: "give either templateId or fromProjectId"})
		return
	}
	h.updateProject(c, p, api.ProjectUpdate{
		Workflow: &pr.Workflow, WorkItemTypes: &pr.WorkItemTypes, CustomFields: &pr.CustomFields, Rules: &pr.Rules, BoardSettings: &pr.BoardSettings,
	})
}
