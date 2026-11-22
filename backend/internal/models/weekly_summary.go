package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// WeeklySummary is the GORM persistence model for the
// `project_weekly_summaries` table defined in db/schema.sql. Added per
// user request for a per-project dated log of "summary of tasks progress"
// entries — one row per week, not a single overwritten box, so past weeks
// stay visible as history (mirrors the Milestone one-to-many shape).
type WeeklySummary struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index"`
	WeekOf    time.Time `gorm:"type:date;not null"`
	Summary   string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the model to the existing table name.
func (WeeklySummary) TableName() string { return "project_weekly_summaries" }

// ToAPI converts the persistence model into the generated API DTO.
func (w WeeklySummary) ToAPI() api.WeeklySummary {
	return api.WeeklySummary{
		Id:        w.ID,
		ProjectId: w.ProjectID,
		WeekOf:    openapi_types.Date{Time: w.WeekOf},
		Summary:   w.Summary,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

// NewWeeklySummaryFromCreate builds a new WeeklySummary from a create
// request.
func NewWeeklySummaryFromCreate(in api.WeeklySummaryCreate) WeeklySummary {
	return WeeklySummary{
		ProjectID: in.ProjectId,
		WeekOf:    in.WeekOf.Time,
		Summary:   in.Summary,
	}
}

// ApplyUpdate mutates the model in place with the non-nil fields of a
// patch. projectId is deliberately not editable — same convention as
// Milestone/ProjectMember not allowing projectId changes after creation.
func (w *WeeklySummary) ApplyUpdate(in api.WeeklySummaryUpdate) {
	if in.WeekOf != nil {
		w.WeekOf = in.WeekOf.Time
	}
	if in.Summary != nil {
		w.Summary = *in.Summary
	}
}
