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

// ListProjectKickoffs implements GET /api/v1/project-kickoffs. Filter by
// projectId — that's the usual case, the project dashboard's Kickoff
// section uses this (pageSize=1) to check whether a kickoff already exists
// for a project, since project_id is unique (at most one kickoff each).
func (h *Handler) ListProjectKickoffs(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.ProjectKickoff{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count project kickoffs", Details: strptr(err.Error())})
		return
	}

	var kickoffs []models.ProjectKickoff
	if err := q.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&kickoffs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list project kickoffs", Details: strptr(err.Error())})
		return
	}

	if err := h.attachActionItems(kickoffs); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load kickoff action items", Details: strptr(err.Error())})
		return
	}

	data := make([]api.ProjectKickoff, len(kickoffs))
	for i := range kickoffs {
		data[i] = kickoffs[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.ProjectKickoffList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// attachActionItems batch-loads kickoff_action_items for the given kickoffs
// (one query for the whole page, not one per kickoff) and populates each
// ProjectKickoff.ActionItems in place. Mirrors tasks.go's attachAssignees.
func (h *Handler) attachActionItems(kickoffs []models.ProjectKickoff) error {
	if len(kickoffs) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(kickoffs))
	for i := range kickoffs {
		ids[i] = kickoffs[i].ID
	}

	var items []models.KickoffActionItem
	if err := h.db.Where("kickoff_id IN ?", ids).Order("created_at ASC").Find(&items).Error; err != nil {
		return err
	}

	byKickoff := make(map[uuid.UUID][]models.KickoffActionItem, len(kickoffs))
	for _, item := range items {
		byKickoff[item.KickoffID] = append(byKickoff[item.KickoffID], item)
	}
	for i := range kickoffs {
		// Explicitly non-nil so ToAPI emits `[]` rather than `null` even
		// when a kickoff has zero action items.
		kickoffs[i].ActionItems = byKickoff[kickoffs[i].ID]
		if kickoffs[i].ActionItems == nil {
			kickoffs[i].ActionItems = []models.KickoffActionItem{}
		}
	}
	return nil
}

// replaceActionItems deletes every existing kickoff_action_items row for
// kickoffID and inserts one row per entry in items, then leaves the
// kickoff's in-memory ActionItems to match. Mirrors tasks.go's
// replaceAssignees — full-set replace, no per-item diffing. Existing
// taskId links (from an earlier "convert to task" call) survive a save only
// if the caller echoes them back in the input, same as any other field.
func (h *Handler) replaceActionItems(k *models.ProjectKickoff, in []api.KickoffActionItemInput) error {
	if err := h.db.Where("kickoff_id = ?", k.ID).Delete(&models.KickoffActionItem{}).Error; err != nil {
		return err
	}

	rows := make([]models.KickoffActionItem, len(in))
	for i, item := range in {
		rows[i] = models.NewKickoffActionItemFromInput(k.ID, item)
	}

	if len(rows) > 0 {
		if err := h.db.Create(&rows).Error; err != nil {
			return err
		}
	}
	k.ActionItems = rows
	return nil
}

// CreateProjectKickoff implements POST /api/v1/project-kickoffs. The row is
// created first (to get an id), then action items are inserted against it —
// same two-step flow CreateTask uses for assignees. A second kickoff for the
// same project fails with a 400 (project_id is UNIQUE at the DB level).
func (h *Handler) CreateProjectKickoff(c *gin.Context) {
	var in api.ProjectKickoffCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	k := models.NewProjectKickoffFromCreate(in)
	if err := h.db.Create(&k).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create project kickoff (a project can only have one)", Details: strptr(err.Error())})
		return
	}

	if in.ActionItems != nil {
		if err := h.replaceActionItems(&k, *in.ActionItems); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save kickoff action items", Details: strptr(err.Error())})
			return
		}
	}

	c.JSON(http.StatusCreated, k.ToAPI())
}

// GetProjectKickoff implements GET /api/v1/project-kickoffs/{id}.
func (h *Handler) GetProjectKickoff(c *gin.Context) {
	var k models.ProjectKickoff
	if err := h.db.First(&k, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project kickoff not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project kickoff", Details: strptr(err.Error())})
		return
	}
	if err := h.attachActionItems([]models.ProjectKickoff{k}); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load kickoff action items", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, k.ToAPI())
}

// UpdateProjectKickoff implements PATCH /api/v1/project-kickoffs/{id}.
func (h *Handler) UpdateProjectKickoff(c *gin.Context) {
	var k models.ProjectKickoff
	if err := h.db.First(&k, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project kickoff not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project kickoff", Details: strptr(err.Error())})
		return
	}

	var in api.ProjectKickoffUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	k.ApplyUpdate(in)
	if err := h.db.Save(&k).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update project kickoff", Details: strptr(err.Error())})
		return
	}

	// actionItems uses replace-the-whole-set semantics, same as
	// TaskUpdate.assigneeIds: present (even as an empty array) means "these
	// are the action items now", absent means "leave them untouched".
	if in.ActionItems != nil {
		if err := h.replaceActionItems(&k, *in.ActionItems); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update kickoff action items", Details: strptr(err.Error())})
			return
		}
	} else if err := h.attachActionItems([]models.ProjectKickoff{k}); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load kickoff action items", Details: strptr(err.Error())})
		return
	}

	c.JSON(http.StatusOK, k.ToAPI())
}

// DeleteProjectKickoff implements DELETE /api/v1/project-kickoffs/{id}.
// kickoff_action_items rows are cleaned up by the DB via ON DELETE CASCADE.
func (h *Handler) DeleteProjectKickoff(c *gin.Context) {
	res := h.db.Delete(&models.ProjectKickoff{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete project kickoff", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "project kickoff not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
