package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Sprint is a timebox in a project (sprints table).
type Sprint struct {
	ID        uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID        `gorm:"type:uuid;not null;index"`
	Name      string           `gorm:"not null"`
	Goal      *string          `gorm:"type:text"`
	Status    api.SprintStatus `gorm:"type:sprint_status;not null;default:planned"`
	StartDate time.Time        `gorm:"type:date"`
	EndDate   time.Time        `gorm:"type:date"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Sprint) TableName() string { return "sprints" }

func (s Sprint) ToAPI(stats api.SprintStats) api.Sprint {
	return api.Sprint{
		Id: s.ID, ProjectId: s.ProjectID, Name: s.Name, Goal: s.Goal, Status: s.Status,
		StartDate: openapi_types.Date{Time: s.StartDate}, EndDate: openapi_types.Date{Time: s.EndDate},
		Stats: stats, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
