package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Milestone is the GORM persistence model for the `milestones` table
// defined in db/schema.sql.
type Milestone struct {
	ID          uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID   uuid.UUID           `gorm:"type:uuid;not null;index"`
	Name        string              `gorm:"not null"`
	Description *string             `gorm:"type:text"`
	Status      api.MilestoneStatus `gorm:"type:milestone_status;not null;default:pending"`
	StartDate   *time.Time          `gorm:"type:date"`
	DueDate     *time.Time          `gorm:"type:date"`
	CompletedAt *time.Time          `gorm:"type:date"`
	// VersionID is the release this milestone is planned into (optional).
	VersionID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the model to the existing pluralized table.
func (Milestone) TableName() string { return "milestones" }

// ToAPI converts the persistence model into the generated API DTO.
func (m Milestone) ToAPI() api.Milestone {
	out := api.Milestone{
		Id:          m.ID,
		ProjectId:   m.ProjectID,
		Name:        m.Name,
		Description: m.Description,
		Status:      m.Status,
		VersionId:   m.VersionID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.StartDate != nil {
		out.StartDate = &openapi_types.Date{Time: *m.StartDate}
	}
	if m.DueDate != nil {
		out.DueDate = &openapi_types.Date{Time: *m.DueDate}
	}
	if m.CompletedAt != nil {
		out.CompletedAt = &openapi_types.Date{Time: *m.CompletedAt}
	}
	return out
}

// NewMilestoneFromCreate builds a new Milestone from a create request. A
// milestone always starts life `pending` — there's no way to create one
// already in progress or completed, matching the Goal pattern (status is
// update-only).
func NewMilestoneFromCreate(in api.MilestoneCreate) Milestone {
	m := Milestone{
		ProjectID:   in.ProjectId,
		Name:        in.Name,
		Description: in.Description,
		Status:      api.MilestoneStatusPending,
		VersionID:   in.VersionId,
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		m.StartDate = &t
	}
	if in.DueDate != nil {
		t := in.DueDate.Time
		m.DueDate = &t
	}
	return m
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
// Note: projectId is deliberately not editable here — moving a milestone to
// a different project isn't supported, same rationale as ProjectMember not
// allowing personId/projectId changes after creation.
//
// clearVersion unassigns the milestone from its version; the handler sets
// it when the request body carries an explicit `"versionId": null`, which a
// *uuid.UUID field can't tell apart from an omitted key (same as Task).
func (m *Milestone) ApplyUpdate(in api.MilestoneUpdate, clearVersion bool) {
	if clearVersion {
		m.VersionID = nil
	} else if in.VersionId != nil {
		m.VersionID = in.VersionId
	}
	if in.Name != nil {
		m.Name = *in.Name
	}
	if in.Description != nil {
		m.Description = in.Description
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		m.StartDate = &t
	}
	if in.DueDate != nil {
		t := in.DueDate.Time
		m.DueDate = &t
	}
	if in.Status != nil {
		m.Status = *in.Status
		// Marking a milestone completed stamps today automatically, unless
		// the request already supplied an explicit completedAt (handled
		// below) or one was already set.
		if *in.Status == api.MilestoneStatusCompleted && m.CompletedAt == nil && in.CompletedAt == nil {
			today := time.Now().UTC().Truncate(24 * time.Hour)
			m.CompletedAt = &today
		}
	}
	if in.CompletedAt != nil {
		t := in.CompletedAt.Time
		m.CompletedAt = &t
	}
}
