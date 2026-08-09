package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// Contribution is the GORM persistence model for the unified `contributions`
// feed table defined in db/schema.sql. task_id/repository_id/metadata exist
// on the table but aren't exposed yet.
type Contribution struct {
	ID         uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PersonID   uuid.UUID            `gorm:"type:uuid;not null"`
	ProjectID  *uuid.UUID           `gorm:"type:uuid"`
	Type       api.ContributionType `gorm:"type:contribution_type;not null"`
	Title      *string              `gorm:"type:text"`
	Url        *string              `gorm:"type:text"`
	Weight     float64              `gorm:"type:numeric(6,2);not null;default:1"`
	OccurredAt time.Time            `gorm:"not null;default:now()"`
	CreatedAt  time.Time
}

// TableName pins the model to the existing pluralized table.
func (Contribution) TableName() string { return "contributions" }

// ToAPI converts the persistence model into the generated API DTO.
func (c Contribution) ToAPI() api.Contribution {
	return api.Contribution{
		Id:         c.ID,
		PersonId:   c.PersonID,
		ProjectId:  c.ProjectID,
		Type:       c.Type,
		Title:      c.Title,
		Url:        c.Url,
		Weight:     c.Weight,
		OccurredAt: c.OccurredAt,
		CreatedAt:  c.CreatedAt,
	}
}

// NewContributionFromCreate builds a new Contribution from a create request,
// applying the same defaults the database would use.
func NewContributionFromCreate(in api.ContributionCreate) Contribution {
	c := Contribution{
		PersonID:   in.PersonId,
		ProjectID:  in.ProjectId,
		Type:       in.Type,
		Title:      in.Title,
		Url:        in.Url,
		Weight:     1,
		OccurredAt: time.Now().UTC(),
	}
	if in.Weight != nil {
		c.Weight = *in.Weight
	}
	if in.OccurredAt != nil {
		c.OccurredAt = *in.OccurredAt
	}
	return c
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
func (c *Contribution) ApplyUpdate(in api.ContributionUpdate) {
	if in.PersonId != nil {
		c.PersonID = *in.PersonId
	}
	if in.ProjectId != nil {
		c.ProjectID = in.ProjectId
	}
	if in.Type != nil {
		c.Type = *in.Type
	}
	if in.Title != nil {
		c.Title = in.Title
	}
	if in.Url != nil {
		c.Url = in.Url
	}
	if in.Weight != nil {
		c.Weight = *in.Weight
	}
	if in.OccurredAt != nil {
		c.OccurredAt = *in.OccurredAt
	}
}
