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

type deliveryPlan struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	Shared    bool      `gorm:"not null"`
	Config    string    `gorm:"type:jsonb;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (deliveryPlan) TableName() string { return "delivery_plans" }

func (h *Handler) planJSON(d deliveryPlan, me uuid.UUID) api.DeliveryPlan {
	var cfg api.DeliveryPlanConfig
	_ = json.Unmarshal([]byte(d.Config), &cfg)
	out := api.DeliveryPlan{
		Id: d.ID, Name: d.Name, Shared: d.Shared, Mine: d.OwnerID == me, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Lanes: cfg.Lanes, Types: cfg.Types, Months: api.DeliveryPlanMonths(cfg.Months), Markers: cfg.Markers,
	}
	if out.Lanes == nil {
		out.Lanes = []api.DeliveryPlanLane{}
	}
	if out.Types == nil {
		out.Types = []api.TaskType{}
	}
	if out.Markers == nil {
		out.Markers = []api.DeliveryPlanMarker{}
	}
	if out.Months == 0 {
		out.Months = 6
	}
	var u models.User
	if h.db.First(&u, "id = ?", d.OwnerID).Error == nil {
		pid := u.PersonID
		out.OwnerPersonId = &pid
	}
	return out
}

func (h *Handler) validatePlan(in api.DeliveryPlanInput) error {
	if len(in.Lanes) > 20 {
		return errors.New("a plan has up to 20 lanes")
	}
	for _, l := range in.Lanes {
		var n int64
		h.db.Model(&models.Project{}).Where("id = ?", l.ProjectId).Count(&n)
		if n == 0 {
			return errors.New("unknown project in a lane")
		}
		if l.TeamId != nil {
			h.db.Model(&projectTeam{}).Where("id = ? AND project_id = ?", *l.TeamId, l.ProjectId).Count(&n)
			if n == 0 {
				return errors.New("that team isn't in the lane's project")
			}
		}
	}
	for _, t := range in.Types {
		if !t.Valid() {
			return errors.New("unknown item type")
		}
	}
	if !in.Months.Valid() {
		return errors.New("a plan spans 3, 6 or 12 months")
	}
	if len(in.Markers) > 30 {
		return errors.New("up to 30 markers")
	}
	for _, m := range in.Markers {
		l := strings.TrimSpace(m.Label)
		if l == "" || utf8.RuneCountInString(l) > 60 {
			return errors.New("markers need a label of 1–60 characters")
		}
		if !m.Color.Valid() {
			return errors.New("unknown marker colour")
		}
	}
	return nil
}

func (h *Handler) readPlanInput(c *gin.Context, d *deliveryPlan) bool {
	var in api.DeliveryPlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return false
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a plan needs a name of 1–100 characters"})
		return false
	}
	if err := h.validatePlan(in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return false
	}
	for i := range in.Markers {
		in.Markers[i].Label = strings.TrimSpace(in.Markers[i].Label)
	}
	b, _ := json.Marshal(api.DeliveryPlanConfig{Lanes: in.Lanes, Types: in.Types, Months: api.DeliveryPlanConfigMonths(in.Months), Markers: in.Markers})
	d.Name, d.Config = name, string(b)
	if in.Shared != nil {
		d.Shared = *in.Shared
	}
	return true
}

// ListDeliveryPlans implements GET /api/v1/delivery-plans.
func (h *Handler) ListDeliveryPlans(c *gin.Context) {
	u, _ := currentUser(c)
	var rows []deliveryPlan
	if err := h.db.Where("owner_id = ? OR shared = ?", u.ID, true).Order("name").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list plans", Details: strptr(err.Error())})
		return
	}
	out := make([]api.DeliveryPlan, len(rows))
	for i, d := range rows {
		out[i] = h.planJSON(d, u.ID)
	}
	c.JSON(http.StatusOK, api.DeliveryPlanList{Data: out})
}

// CreateDeliveryPlan implements POST /api/v1/delivery-plans.
func (h *Handler) CreateDeliveryPlan(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	d := deliveryPlan{OwnerID: u.ID}
	if !h.readPlanInput(c, &d) {
		return
	}
	if err := h.db.Create(&d).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save plan", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, h.planJSON(d, u.ID))
}

func (h *Handler) loadPlan(c *gin.Context) (deliveryPlan, models.User, bool) {
	u, _ := currentUser(c)
	var d deliveryPlan
	if err := h.db.Where("id = ? AND (owner_id = ? OR shared = ?)", c.Param("id"), u.ID, true).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "plan not found"})
		return d, u, false
	}
	return d, u, true
}

// UpdateDeliveryPlan implements PATCH /api/v1/delivery-plans/{id}.
func (h *Handler) UpdateDeliveryPlan(c *gin.Context) {
	d, u, ok := h.loadPlan(c)
	if !ok {
		return
	}
	if d.OwnerID != u.ID {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner can change a plan"})
		return
	}
	if !h.readPlanInput(c, &d) {
		return
	}
	if err := h.db.Save(&d).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save plan", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, h.planJSON(d, u.ID))
}

// DeleteDeliveryPlan implements DELETE /api/v1/delivery-plans/{id}.
func (h *Handler) DeleteDeliveryPlan(c *gin.Context) {
	d, u, ok := h.loadPlan(c)
	if !ok {
		return
	}
	if d.OwnerID != u.ID && !h.hasRole(c, api.SystemRoleAdmin) {
		c.JSON(http.StatusForbidden, api.Error{Message: "only its owner or an admin can delete a plan"})
		return
	}
	if err := h.db.Delete(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete plan", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
