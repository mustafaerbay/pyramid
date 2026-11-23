package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

type dashboard struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	Shared    bool      `gorm:"not null"`
	Widgets   string    `gorm:"type:jsonb;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (dashboard) TableName() string { return "dashboards" }

func (h *Handler) dashboardJSON(d dashboard, me uuid.UUID) api.Dashboard {
	out := api.Dashboard{Id: d.ID, Name: d.Name, Shared: d.Shared, Mine: d.OwnerID == me, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, Widgets: []api.DashboardWidget{}}
	_ = json.Unmarshal([]byte(d.Widgets), &out.Widgets)
	var u models.User
	if h.db.First(&u, "id = ?", d.OwnerID).Error == nil {
		pid := u.PersonID
		out.OwnerPersonId = &pid
	}
	return out
}

// validateWidgets checks each widget's kind and what it needs; query
// widgets must use a saved query the owner can see.
func (h *Handler) validateWidgets(ws []api.DashboardWidget, owner uuid.UUID) error {
	if len(ws) > 30 {
		return errors.New("a dashboard holds up to 30 widgets")
	}
	seen := map[string]bool{}
	for _, w := range ws {
		if w.Id == "" || len(w.Id) > 40 || seen[w.Id] {
			return errors.New("every widget needs its own id")
		}
		seen[w.Id] = true
		if !w.Kind.Valid() {
			return fmt.Errorf("unknown widget kind %q", w.Kind)
		}
		if !w.Size.Valid() {
			return errors.New("widget size is small or wide")
		}
		if w.Title != nil && utf8.RuneCountInString(*w.Title) > 100 {
			return errors.New("widget titles are up to 100 characters")
		}
		if w.Chart != nil && !w.Chart.Valid() {
			return errors.New("unknown chart")
		}
		if (w.GroupBy != nil && !w.GroupBy.Valid()) || (w.StackBy != nil && !w.StackBy.Valid()) {
			return errors.New("unknown groupBy")
		}
		switch w.Kind {
		case api.DashboardWidgetKindQuery:
			if w.QueryId == nil {
				return errors.New("a query widget needs a saved query")
			}
			var n int64
			h.db.Model(&savedQuery{}).Where("id = ? AND (owner_id = ? OR shared = ?)", *w.QueryId, owner, true).Count(&n)
			if n == 0 {
				return errors.New("that saved query isn't yours or shared")
			}
		case api.DashboardWidgetKindBurndown, api.DashboardWidgetKindVelocity, api.DashboardWidgetKindCfd:
			if w.ProjectId == nil {
				return fmt.Errorf("a %s widget needs a project", w.Kind)
			}
			var n int64
			h.db.Model(&models.Project{}).Where("id = ?", *w.ProjectId).Count(&n)
			if n == 0 {
				return errors.New("unknown project in widget")
			}
		case api.DashboardWidgetKindNotes:
			if w.Text != nil && utf8.RuneCountInString(*w.Text) > 4000 {
				return errors.New("notes are up to 4000 characters")
			}
		}
	}
	return nil
}

func (h *Handler) readDashboardInput(c *gin.Context, d *dashboard) bool {
	var in api.DashboardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return false
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a dashboard needs a name of 1–100 characters"})
		return false
	}
	d.Name = name
	if in.Shared != nil {
		d.Shared = *in.Shared
	}
	if in.Widgets != nil {
		if err := h.validateWidgets(*in.Widgets, d.OwnerID); err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return false
		}
		b, _ := json.Marshal(*in.Widgets)
		d.Widgets = string(b)
	}
	if d.Widgets == "" {
		d.Widgets = "[]"
	}
	return true
}

// ListDashboards implements GET /api/v1/dashboards.
func (h *Handler) ListDashboards(c *gin.Context) {
	u, _ := currentUser(c)
	var rows []dashboard
	if err := h.db.Where("owner_id = ? OR shared = ?", u.ID, true).Order("name").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list dashboards", Details: strptr(err.Error())})
		return
	}
	out := make([]api.Dashboard, len(rows))
	for i, d := range rows {
		out[i] = h.dashboardJSON(d, u.ID)
	}
	c.JSON(http.StatusOK, api.DashboardList{Data: out})
}

// CreateDashboard implements POST /api/v1/dashboards.
func (h *Handler) CreateDashboard(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	d := dashboard{OwnerID: u.ID}
	if !h.readDashboardInput(c, &d) {
		return
	}
	if err := h.db.Create(&d).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save dashboard", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, h.dashboardJSON(d, u.ID))
}

func (h *Handler) loadDashboard(c *gin.Context) (dashboard, models.User, bool) {
	u, _ := currentUser(c)
	var d dashboard
	if err := h.db.Where("id = ? AND (owner_id = ? OR shared = ?)", c.Param("id"), u.ID, true).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "dashboard not found"})
		return d, u, false
	}
	return d, u, true
}

// GetDashboard implements GET /api/v1/dashboards/{id}.
func (h *Handler) GetDashboard(c *gin.Context) {
	if d, u, ok := h.loadDashboard(c); ok {
		c.JSON(http.StatusOK, h.dashboardJSON(d, u.ID))
	}
}

// UpdateDashboard implements PATCH /api/v1/dashboards/{id}.
func (h *Handler) UpdateDashboard(c *gin.Context) {
	d, u, ok := h.loadDashboard(c)
	if !ok {
		return
	}
	if d.OwnerID != u.ID {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner can change a dashboard"})
		return
	}
	if !h.readDashboardInput(c, &d) {
		return
	}
	if err := h.db.Save(&d).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save dashboard", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, h.dashboardJSON(d, u.ID))
}

// DeleteDashboard implements DELETE /api/v1/dashboards/{id}.
func (h *Handler) DeleteDashboard(c *gin.Context) {
	d, u, ok := h.loadDashboard(c)
	if !ok {
		return
	}
	if d.OwnerID != u.ID && !h.hasRole(c, api.SystemRoleAdmin) {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner or an admin can delete a dashboard"})
		return
	}
	if err := h.db.Delete(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete dashboard", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
