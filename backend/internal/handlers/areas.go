package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// Area paths: rows of a project's "Web/Checkout"-style areas (areas table).

type area struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null"`
	Path      string    `gorm:"not null"`
}

func (area) TableName() string { return "areas" }

func (a area) toAPI() api.Area { return api.Area{Id: a.ID, ProjectId: a.ProjectID, Path: a.Path} }

const maxAreaDepth = 4

// cleanPath trims each level and joins them with "/".
func cleanPath(raw string) (string, error) {
	parts := strings.Split(raw, "/")
	if len(parts) > maxAreaDepth {
		return "", errors.New("area paths can be at most 4 levels deep")
	}
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return "", errors.New("every level of an area path needs a name")
		}
		if utf8.RuneCountInString(p) > 40 {
			return "", errors.New("each level of an area path can be at most 40 characters")
		}
		parts[i] = p
	}
	return strings.Join(parts, "/"), nil
}

func (h *Handler) areaTaken(projectID uuid.UUID, path string, except uuid.UUID) bool {
	var n int64
	h.db.Model(&area{}).Where("project_id = ? AND LOWER(path) = LOWER(?) AND id <> ?", projectID, path, except).Count(&n)
	return n > 0
}

// validateArea checks that t's area (if any) belongs to t's project.
func (h *Handler) validateArea(t *models.Task) error {
	if t.AreaID == nil {
		return nil
	}
	var n int64
	if err := h.db.Model(&area{}).Where("id = ? AND project_id = ?", *t.AreaID, t.ProjectID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errors.New("the area must belong to the task's project")
	}
	return nil
}

// areaAndBelow is the ids of an area and all its sub-areas.
func (h *Handler) areaAndBelow(id string) ([]uuid.UUID, error) {
	var a area
	if err := h.db.First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	err := h.db.Model(&area{}).Where("project_id = ? AND (LOWER(path) = LOWER(?) OR LOWER(path) LIKE LOWER(?))", a.ProjectID, a.Path, a.Path+"/%").
		Pluck("id", &ids).Error
	return ids, err
}

// ListAreas implements GET /api/v1/areas.
func (h *Handler) ListAreas(c *gin.Context) {
	pid, err := uuid.Parse(c.Query("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "projectId is required"})
		return
	}
	var rows []area
	if err := h.db.Where("project_id = ?", pid).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list areas", Details: strptr(err.Error())})
		return
	}
	sort.Slice(rows, func(i, j int) bool { return strings.ToLower(rows[i].Path) < strings.ToLower(rows[j].Path) })
	out := make([]api.Area, len(rows))
	for i, a := range rows {
		out[i] = a.toAPI()
	}
	c.JSON(http.StatusOK, api.AreaList{Data: out})
}

// CreateArea implements POST /api/v1/areas.
func (h *Handler) CreateArea(c *gin.Context) {
	var in api.AreaCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	path, err := cleanPath(in.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if h.areaTaken(in.ProjectId, path, uuid.Nil) {
		c.JSON(http.StatusBadRequest, api.Error{Message: "this project already has that area"})
		return
	}
	a := area{ProjectID: in.ProjectId, Path: path}
	if err := h.db.Create(&a).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create area", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, a.toAPI())
}

func (h *Handler) loadArea(c *gin.Context) (area, bool) {
	var a area
	if err := h.db.First(&a, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "area not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load area", Details: strptr(err.Error())})
		}
		return a, false
	}
	return a, true
}

// UpdateArea implements PATCH /api/v1/areas/{id}: a rename that its
// sub-areas follow.
func (h *Handler) UpdateArea(c *gin.Context) {
	a, ok := h.loadArea(c)
	if !ok {
		return
	}
	var in api.AreaUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	path, err := cleanPath(in.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if h.areaTaken(a.ProjectID, path, a.ID) {
		c.JSON(http.StatusBadRequest, api.Error{Message: "this project already has that area"})
		return
	}
	var subs []area
	h.db.Where("project_id = ? AND LOWER(path) LIKE LOWER(?)", a.ProjectID, a.Path+"/%").Find(&subs)
	for _, s := range subs {
		np := path + s.Path[len(a.Path):]
		if strings.Count(np, "/")+1 > maxAreaDepth {
			c.JSON(http.StatusBadRequest, api.Error{Message: "that would make a sub-area deeper than 4 levels"})
			return
		}
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		for _, s := range subs {
			if err := tx.Model(&area{}).Where("id = ?", s.ID).Update("path", path+s.Path[len(a.Path):]).Error; err != nil {
				return err
			}
		}
		return tx.Model(&area{}).Where("id = ?", a.ID).Update("path", path).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to rename area", Details: strptr(err.Error())})
		return
	}
	a.Path = path
	c.JSON(http.StatusOK, a.toAPI())
}

// DeleteArea implements DELETE /api/v1/areas/{id}.
func (h *Handler) DeleteArea(c *gin.Context) {
	a, ok := h.loadArea(c)
	if !ok {
		return
	}
	var subs int64
	h.db.Model(&area{}).Where("project_id = ? AND LOWER(path) LIKE LOWER(?)", a.ProjectID, a.Path+"/%").Count(&subs)
	if subs > 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "delete or move its sub-areas first"})
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Task{}).Where("area_id = ?", a.ID).Update("area_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&a).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete area", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
