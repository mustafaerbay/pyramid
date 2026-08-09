package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ProjectMember is the GORM persistence model for the `project_members`
// table defined in db/schema.sql — the many-to-many join between people and
// projects, carrying each person's role and time allocation on that project.
// The (project_id, person_id) pair is unique: a person can be assigned to a
// project only once at a time (matching db/schema.sql's UNIQUE constraint).
type ProjectMember struct {
	ID            uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID     uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_project_members_unique"`
	PersonID      uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_project_members_unique"`
	Role          api.ProjectRole `gorm:"type:project_role;not null;default:contributor"`
	AllocationPct float64         `gorm:"type:numeric(5,2);not null;default:100"`
	JoinedAt      time.Time       `gorm:"type:date;not null;default:CURRENT_DATE"`
	LeftAt        *time.Time      `gorm:"type:date"`
}

// TableName pins the model to the existing pluralized table.
func (ProjectMember) TableName() string { return "project_members" }

// ToAPI converts the persistence model into the generated API DTO.
func (m ProjectMember) ToAPI() api.ProjectMember {
	out := api.ProjectMember{
		Id:            m.ID,
		ProjectId:     m.ProjectID,
		PersonId:      m.PersonID,
		Role:          m.Role,
		AllocationPct: m.AllocationPct,
		JoinedAt:      openapi_types.Date{Time: m.JoinedAt},
	}
	if m.LeftAt != nil {
		out.LeftAt = &openapi_types.Date{Time: *m.LeftAt}
	}
	return out
}

// NewProjectMemberFromCreate builds a new ProjectMember from a create
// request, applying the same defaults the database would use.
func NewProjectMemberFromCreate(in api.ProjectMemberCreate) ProjectMember {
	m := ProjectMember{
		ProjectID:     in.ProjectId,
		PersonID:      in.PersonId,
		Role:          api.ProjectRoleContributor,
		AllocationPct: 100,
		JoinedAt:      time.Now().UTC(),
	}
	if in.Role != nil {
		m.Role = *in.Role
	}
	if in.AllocationPct != nil {
		m.AllocationPct = *in.AllocationPct
	}
	if in.JoinedAt != nil {
		m.JoinedAt = in.JoinedAt.Time
	}
	return m
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
// Project and person are intentionally not editable here — reassigning means
// removing the membership and creating a new one, which keeps history clean.
func (m *ProjectMember) ApplyUpdate(in api.ProjectMemberUpdate) {
	if in.Role != nil {
		m.Role = *in.Role
	}
	if in.AllocationPct != nil {
		m.AllocationPct = *in.AllocationPct
	}
	if in.JoinedAt != nil {
		m.JoinedAt = in.JoinedAt.Time
	}
	if in.LeftAt != nil {
		t := in.LeftAt.Time
		m.LeftAt = &t
	}
}
