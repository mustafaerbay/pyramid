package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Version is the GORM persistence model for the `project_versions` table
// defined in db/schema.sql: a planned release of a project. Milestones are
// attached to a version (milestones.version_id), which is what the project
// page's release plan is built from.
type Version struct {
	ID          uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID   uuid.UUID         `gorm:"type:uuid;not null;index"`
	Name        string            `gorm:"not null"`
	Description *string           `gorm:"type:text"`
	Status      api.VersionStatus `gorm:"type:version_status;not null;default:planned"`
	TargetDate  *time.Time        `gorm:"type:date"`
	ReleasedAt  *time.Time        `gorm:"type:date"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName pins the model to the schema.sql table.
func (Version) TableName() string { return "project_versions" }

// ToAPI converts the persistence model into the generated API DTO.
func (v Version) ToAPI() api.Version {
	out := api.Version{
		Id:          v.ID,
		ProjectId:   v.ProjectID,
		Name:        v.Name,
		Description: v.Description,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
	if v.TargetDate != nil {
		out.TargetDate = &openapi_types.Date{Time: *v.TargetDate}
	}
	if v.ReleasedAt != nil {
		out.ReleasedAt = &openapi_types.Date{Time: *v.ReleasedAt}
	}
	return out
}

// NewVersionFromCreate builds a new Version from a create request. A version
// always starts `planned` (status is update-only, like milestones).
func NewVersionFromCreate(in api.VersionCreate) Version {
	v := Version{
		ProjectID:   in.ProjectId,
		Name:        in.Name,
		Description: in.Description,
		Status:      api.VersionStatusPlanned,
	}
	if in.TargetDate != nil {
		t := in.TargetDate.Time
		v.TargetDate = &t
	}
	return v
}

// ApplyUpdate mutates the model in place with the non-nil fields of a
// patch. projectId is not editable. Releasing stamps releasedAt with today
// unless one is already set or given; it is never cleared automatically
// (same rule as Milestone.completedAt).
func (v *Version) ApplyUpdate(in api.VersionUpdate) {
	if in.Name != nil {
		v.Name = *in.Name
	}
	if in.Description != nil {
		v.Description = in.Description
	}
	if in.TargetDate != nil {
		t := in.TargetDate.Time
		v.TargetDate = &t
	}
	if in.Status != nil {
		v.Status = *in.Status
		if *in.Status == api.VersionStatusReleased && v.ReleasedAt == nil && in.ReleasedAt == nil {
			today := time.Now().UTC().Truncate(24 * time.Hour)
			v.ReleasedAt = &today
		}
	}
	if in.ReleasedAt != nil {
		t := in.ReleasedAt.Time
		v.ReleasedAt = &t
	}
}
