package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Goal is the GORM persistence model for the `goals` table defined in
// db/schema.sql. Team/department ownership columns exist on the table but
// aren't exposed yet — only person-owned goals are wired up in the API.
type Goal struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Title         string         `gorm:"not null"`
	Description   *string        `gorm:"type:text"`
	Scope         api.GoalScope  `gorm:"type:goal_scope;not null;default:individual"`
	Status        api.GoalStatus `gorm:"type:goal_status;not null;default:draft"`
	OwnerPersonID *uuid.UUID     `gorm:"type:uuid"`
	ProgressPct   float64        `gorm:"type:numeric(5,2);not null;default:0"`
	StartDate     *time.Time     `gorm:"type:date"`
	DueDate       *time.Time     `gorm:"type:date"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName pins the model to the existing pluralized table.
func (Goal) TableName() string { return "goals" }

// ToAPI converts the persistence model into the generated API DTO.
func (g Goal) ToAPI() api.Goal {
	out := api.Goal{
		Id:            g.ID,
		Title:         g.Title,
		Description:   g.Description,
		Scope:         g.Scope,
		Status:        g.Status,
		OwnerPersonId: g.OwnerPersonID,
		ProgressPct:   g.ProgressPct,
		CreatedAt:     g.CreatedAt,
		UpdatedAt:     g.UpdatedAt,
	}
	if g.StartDate != nil {
		out.StartDate = &openapi_types.Date{Time: *g.StartDate}
	}
	if g.DueDate != nil {
		out.DueDate = &openapi_types.Date{Time: *g.DueDate}
	}
	return out
}

// NewGoalFromCreate builds a new Goal from a create request, applying the
// same defaults the database would use.
func NewGoalFromCreate(in api.GoalCreate) Goal {
	g := Goal{
		Title:         in.Title,
		Description:   in.Description,
		OwnerPersonID: in.OwnerPersonId,
		Scope:         api.GoalScopeIndividual,
		Status:        api.GoalStatusDraft,
		ProgressPct:   0,
	}
	if in.Scope != nil {
		g.Scope = *in.Scope
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		g.StartDate = &t
	}
	if in.DueDate != nil {
		t := in.DueDate.Time
		g.DueDate = &t
	}
	return g
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
func (g *Goal) ApplyUpdate(in api.GoalUpdate) {
	if in.Title != nil {
		g.Title = *in.Title
	}
	if in.Description != nil {
		g.Description = in.Description
	}
	if in.Scope != nil {
		g.Scope = *in.Scope
	}
	if in.Status != nil {
		g.Status = *in.Status
	}
	if in.OwnerPersonId != nil {
		g.OwnerPersonID = in.OwnerPersonId
	}
	if in.ProgressPct != nil {
		g.ProgressPct = *in.ProgressPct
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		g.StartDate = &t
	}
	if in.DueDate != nil {
		t := in.DueDate.Time
		g.DueDate = &t
	}
}
