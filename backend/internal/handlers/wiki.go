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
)

type wikiPage struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID  `gorm:"type:uuid;not null"`
	ParentID  *uuid.UUID `gorm:"type:uuid"`
	Title     string     `gorm:"not null"`
	Body      string     `gorm:"not null"`
	Position  float64    `gorm:"not null"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (wikiPage) TableName() string { return "wiki_pages" }

type wikiRevision struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PageID    uuid.UUID  `gorm:"type:uuid;not null"`
	Title     string     `gorm:"not null"`
	Body      string     `gorm:"not null"`
	AuthorID  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
}

func (wikiRevision) TableName() string { return "wiki_revisions" }

func (p wikiPage) toAPI(withBody bool) api.WikiPage {
	out := api.WikiPage{Id: p.ID, ProjectId: p.ProjectID, ParentId: p.ParentID, Title: p.Title, Position: float32(p.Position), UpdatedBy: p.UpdatedBy, UpdatedAt: p.UpdatedAt, CreatedAt: p.CreatedAt}
	if withBody {
		b := p.Body
		out.Body = &b
	}
	return out
}

func (h *Handler) checkWikiParent(projectID uuid.UUID, self *uuid.UUID, parent *uuid.UUID) error {
	if parent == nil {
		return nil
	}
	seen := 0
	for cur := parent; cur != nil; seen++ {
		if self != nil && *cur == *self {
			return errors.New("a page can't go under itself")
		}
		var p wikiPage
		if err := h.db.First(&p, "id = ? AND project_id = ?", *cur, projectID).Error; err != nil {
			return errors.New("the parent page isn't in this project")
		}
		if seen > 50 {
			return errors.New("pages nest too deep")
		}
		cur = p.ParentID
	}
	return nil
}

func validWikiText(title string, body *string) error {
	if strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 200 {
		return errors.New("a page needs a title of 1–200 characters")
	}
	if body != nil && len(*body) > 200000 {
		return errors.New("a page holds up to 200,000 characters")
	}
	return nil
}

// ListWikiPages implements GET /api/v1/wiki-pages.
func (h *Handler) ListWikiPages(c *gin.Context) {
	var rows []wikiPage
	h.db.Select("id, project_id, parent_id, title, position, updated_by, created_at, updated_at").
		Where("project_id = ?", c.Query("projectId")).Order("position, title").Find(&rows)
	out := api.WikiPageList{Data: make([]api.WikiPage, len(rows))}
	for i, p := range rows {
		out.Data[i] = p.toAPI(false)
	}
	c.JSON(http.StatusOK, out)
}

// CreateWikiPage implements POST /api/v1/wiki-pages.
func (h *Handler) CreateWikiPage(c *gin.Context) {
	var in api.WikiPageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var n int64
	h.db.Model(&models.Project{}).Where("id = ?", in.ProjectId).Count(&n)
	if n == 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "unknown project"})
		return
	}
	if !h.canWriteProject(c, in.ProjectId) {
		return
	}
	if err := validWikiText(in.Title, in.Body); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.checkWikiParent(in.ProjectId, nil, in.ParentId); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	var maxPos float64
	h.db.Model(&wikiPage{}).Where("project_id = ?", in.ProjectId).Select("COALESCE(MAX(position), 0)").Scan(&maxPos)
	me, _ := h.actor(c)
	p := wikiPage{ProjectID: in.ProjectId, ParentID: in.ParentId, Title: strings.TrimSpace(in.Title), Position: maxPos + 1000}
	if in.Body != nil {
		p.Body = *in.Body
	}
	if me != uuid.Nil {
		p.UpdatedBy = &me
	}
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save the page", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, p.toAPI(true))
}

func (h *Handler) loadWikiPage(c *gin.Context) (wikiPage, bool) {
	var p wikiPage
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "page not found"})
		return p, false
	}
	return p, true
}

// GetWikiPage implements GET /api/v1/wiki-pages/{id}.
func (h *Handler) GetWikiPage(c *gin.Context) {
	if p, ok := h.loadWikiPage(c); ok {
		c.JSON(http.StatusOK, p.toAPI(true))
	}
}

// UpdateWikiPage implements PATCH /api/v1/wiki-pages/{id}.
func (h *Handler) UpdateWikiPage(c *gin.Context) {
	p, ok := h.loadWikiPage(c)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	body, _ := c.GetRawData()
	var in api.WikiPageUpdate
	if err := json.Unmarshal(body, &in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	if in.BaseUpdatedAt != nil && p.UpdatedAt.Sub(*in.BaseUpdatedAt) > time.Millisecond {
		c.JSON(http.StatusConflict, api.Error{Message: "someone saved this page after you opened it; copy your text, reload and try again"})
		return
	}
	title := p.Title
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
	}
	if err := validWikiText(title, in.Body); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if _, sent := raw["parentId"]; sent {
		if err := h.checkWikiParent(p.ProjectID, &p.ID, in.ParentId); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
		p.ParentID = in.ParentId
	}
	textChanged := title != p.Title || (in.Body != nil && *in.Body != p.Body)
	me, _ := h.actor(c)
	if textChanged {
		rev := wikiRevision{PageID: p.ID, Title: p.Title, Body: p.Body, AuthorID: p.UpdatedBy}
		h.db.Create(&rev)
		p.Title = title
		if in.Body != nil {
			p.Body = *in.Body
		}
		if me != uuid.Nil {
			p.UpdatedBy = &me
		}
	}
	if in.Position != nil {
		p.Position = float64(*in.Position)
	}
	if err := h.db.Save(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save the page", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, p.toAPI(true))
}

// DeleteWikiPage implements DELETE /api/v1/wiki-pages/{id}.
func (h *Handler) DeleteWikiPage(c *gin.Context) {
	p, ok := h.loadWikiPage(c)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	h.db.Model(&wikiPage{}).Where("parent_id = ?", p.ID).Update("parent_id", p.ParentID)
	h.db.Where("page_id = ?", p.ID).Delete(&wikiRevision{})
	h.db.Delete(&p)
	c.Status(http.StatusNoContent)
}

// ListWikiRevisions implements GET /api/v1/wiki-pages/{id}/revisions.
func (h *Handler) ListWikiRevisions(c *gin.Context) {
	p, ok := h.loadWikiPage(c)
	if !ok {
		return
	}
	var rows []wikiRevision
	h.db.Where("page_id = ?", p.ID).Order("created_at DESC").Limit(100).Find(&rows)
	out := api.WikiRevisionList{Data: make([]api.WikiRevision, len(rows))}
	for i, r := range rows {
		out.Data[i] = api.WikiRevision{Id: r.ID, Title: r.Title, Body: r.Body, AuthorId: r.AuthorID, CreatedAt: r.CreatedAt}
	}
	c.JSON(http.StatusOK, out)
}
