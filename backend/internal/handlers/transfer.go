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

// TransferTask implements POST /api/v1/tasks/{id}/transfer.
func (h *Handler) TransferTask(c *gin.Context) {
	var in struct {
		ProjectID uuid.UUID `json:"projectId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var root models.Task
	if err := h.db.First(&root, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
		return
	}
	var target models.Project
	if err := h.db.First(&target, "id = ?", in.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "unknown project"})
		return
	}
	if target.ID == root.ProjectID {
		c.JSON(http.StatusBadRequest, api.Error{Message: "the item is already in that project"})
		return
	}
	if !h.canWriteProject(c, root.ProjectID) || !h.canWriteProject(c, target.ID) {
		return
	}

	// The item and everything under it, parents before children.
	items := []models.Task{root}
	for i := 0; i < len(items) && len(items) < 2000; i++ {
		var kids []models.Task
		h.db.Where("parent_id = ? AND project_id = ?", items[i].ID, root.ProjectID).Order("rank, number").Find(&kids)
		items = append(items, kids...)
	}
	types := map[string]bool{}
	for _, d := range target.ItemTypes() {
		types[d.Key] = true
	}
	fields := map[string]bool{}
	for _, f := range target.Fields() {
		fields[f.Key] = true
	}
	from := root.ProjectID
	// States first (they read the target workflow), outside the transaction.
	for i := range items {
		t := &items[i]
		t.ProjectID = target.ID
		keep := t.State
		if err := h.resolveState(t, &keep, false); err != nil {
			t.State = ""
			if err := h.resolveState(t, nil, true); err != nil {
				c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
				return
			}
		}
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var maxNumber int
		var maxRank float64
		tx.Model(&models.Task{}).Where("project_id = ?", target.ID).Select("COALESCE(MAX(number), 0)").Scan(&maxNumber)
		tx.Model(&models.Task{}).Where("project_id = ?", target.ID).Select("COALESCE(MAX(rank), 0)").Scan(&maxRank)
		for i := range items {
			t := &items[i]
			maxNumber++
			maxRank += rankStep
			t.ProjectID, t.Number, t.Rank = target.ID, maxNumber, maxRank
			t.SprintID, t.AreaID, t.MilestoneID, t.BoardDone = nil, nil, nil, false
			if i == 0 {
				t.ParentID = nil
			}
			if t.CustomType != nil && !types[*t.CustomType] {
				t.CustomType = nil
			}
			if t.CustomValues != nil {
				var vals map[string]any
				_ = json.Unmarshal([]byte(*t.CustomValues), &vals)
				for k := range vals {
					if !fields[k] {
						delete(vals, k)
					}
				}
				t.CustomValues = encodeValues(vals)
			}
			if err := tx.Save(t).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to move the item", Details: strptr(err.Error())})
		return
	}
	for _, t := range items {
		h.logTask(c, "updated", t.ID, map[string]api.TaskFieldChange{"projectId": {From: from.String(), To: target.ID.String()}})
	}
	if err := h.enrichTasks(items[:1]); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load the item", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, items[0].ToAPI())
}

// --- item templates ---

type itemTemplate struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	Fields    string    `gorm:"type:jsonb;not null"`
	CreatedAt time.Time
}

func (itemTemplate) TableName() string { return "item_templates" }

func (t itemTemplate) toAPI() api.ItemTemplate {
	out := api.ItemTemplate{Id: t.ID, ProjectId: t.ProjectID, Name: t.Name, CreatedAt: t.CreatedAt}
	_ = json.Unmarshal([]byte(t.Fields), &out.Fields)
	return out
}

func (h *Handler) readTemplateInput(c *gin.Context, t *itemTemplate) bool {
	var in api.ItemTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return false
	}
	fail := func(err error) bool {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return false
	}
	var n int64
	h.db.Model(&models.Project{}).Where("id = ?", in.ProjectId).Count(&n)
	if n == 0 {
		return fail(errors.New("unknown project"))
	}
	if !h.canWriteProject(c, in.ProjectId) {
		return false
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return fail(errors.New("a template needs a name of 1–100 characters"))
	}
	f := in.Fields
	if f.Type != nil && !f.Type.Valid() {
		return fail(errors.New("unknown item type"))
	}
	if f.Priority != nil && !f.Priority.Valid() {
		return fail(errors.New("unknown priority"))
	}
	if f.Tags != nil {
		tags, err := normalizeTags(*f.Tags)
		if err != nil {
			return fail(err)
		}
		f.Tags = &tags
	}
	if f.Description != nil && utf8.RuneCountInString(*f.Description) > 20000 {
		return fail(errors.New("the description is too long"))
	}
	b, _ := json.Marshal(f)
	t.ProjectID, t.Name, t.Fields = in.ProjectId, name, string(b)
	return true
}

// ListItemTemplates implements GET /api/v1/item-templates.
func (h *Handler) ListItemTemplates(c *gin.Context) {
	var rows []itemTemplate
	h.db.Where("project_id = ?", c.Query("projectId")).Order("name").Find(&rows)
	out := make([]api.ItemTemplate, len(rows))
	for i, t := range rows {
		out[i] = t.toAPI()
	}
	c.JSON(http.StatusOK, api.ItemTemplateList{Data: out})
}

// CreateItemTemplate implements POST /api/v1/item-templates.
func (h *Handler) CreateItemTemplate(c *gin.Context) {
	var t itemTemplate
	if !h.readTemplateInput(c, &t) {
		return
	}
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save template", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, t.toAPI())
}

// UpdateItemTemplate implements PATCH /api/v1/item-templates/{id}.
func (h *Handler) UpdateItemTemplate(c *gin.Context) {
	var t itemTemplate
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "template not found"})
		return
	}
	if !h.readTemplateInput(c, &t) {
		return
	}
	h.db.Save(&t)
	c.JSON(http.StatusOK, t.toAPI())
}

// DeleteItemTemplate implements DELETE /api/v1/item-templates/{id}.
func (h *Handler) DeleteItemTemplate(c *gin.Context) {
	var t itemTemplate
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "template not found"})
		return
	}
	if !h.canWriteProject(c, t.ProjectID) {
		return
	}
	h.db.Delete(&t)
	c.Status(http.StatusNoContent)
}
