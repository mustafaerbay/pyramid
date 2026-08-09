package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewProjectMemberFromCreate_Defaults(t *testing.T) {
	projectID, personID := uuid.New(), uuid.New()

	before := time.Now().UTC()
	m := NewProjectMemberFromCreate(api.ProjectMemberCreate{ProjectId: projectID, PersonId: personID})
	after := time.Now().UTC()

	if m.Role != api.ProjectRoleContributor {
		t.Errorf("Role = %q, want contributor (default)", m.Role)
	}
	if m.AllocationPct != 100 {
		t.Errorf("AllocationPct = %v, want 100 (default)", m.AllocationPct)
	}
	if m.JoinedAt.Before(before) || m.JoinedAt.After(after) {
		t.Errorf("JoinedAt = %v, want between %v and %v (defaults to now)", m.JoinedAt, before, after)
	}
	if m.LeftAt != nil {
		t.Errorf("LeftAt = %v, want nil for a new membership", m.LeftAt)
	}
}

func TestNewProjectMemberFromCreate_ExplicitRoleAllocationJoinedAt(t *testing.T) {
	lead := api.ProjectRoleLead
	pct := 50.0
	joined := openapi_types.Date{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}

	m := NewProjectMemberFromCreate(api.ProjectMemberCreate{
		ProjectId: uuid.New(), PersonId: uuid.New(),
		Role: &lead, AllocationPct: &pct, JoinedAt: &joined,
	})

	if m.Role != api.ProjectRoleLead {
		t.Errorf("Role = %q, want lead", m.Role)
	}
	if m.AllocationPct != 50 {
		t.Errorf("AllocationPct = %v, want 50", m.AllocationPct)
	}
	if !m.JoinedAt.Equal(joined.Time) {
		t.Errorf("JoinedAt = %v, want %v", m.JoinedAt, joined.Time)
	}
}

func TestProjectMember_ApplyUpdate_SettingLeftAtOffboards(t *testing.T) {
	m := ProjectMember{Role: api.ProjectRoleContributor, AllocationPct: 100}

	left := openapi_types.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)}
	m.ApplyUpdate(api.ProjectMemberUpdate{LeftAt: &left})

	if m.LeftAt == nil || !m.LeftAt.Equal(left.Time) {
		t.Errorf("LeftAt = %v, want %v", m.LeftAt, left.Time)
	}
}

func TestProjectMember_ApplyUpdate_ProjectAndPersonAreNotEditable(t *testing.T) {
	// ProjectMemberUpdate has no ProjectId/PersonId fields at all — this test
	// documents that reassignment is intentionally impossible via PATCH
	// (see the comment on ApplyUpdate) by simply confirming a full update
	// leaves the identity fields alone.
	projectID, personID := uuid.New(), uuid.New()
	m := ProjectMember{ProjectID: projectID, PersonID: personID, Role: api.ProjectRoleContributor}

	newRole := api.ProjectRoleReviewer
	m.ApplyUpdate(api.ProjectMemberUpdate{Role: &newRole})

	if m.ProjectID != projectID || m.PersonID != personID {
		t.Errorf("identity changed: ProjectID=%v PersonID=%v, want unchanged", m.ProjectID, m.PersonID)
	}
	if m.Role != api.ProjectRoleReviewer {
		t.Errorf("Role = %q, want reviewer", m.Role)
	}
}

func TestProjectMember_ApplyUpdate_AllocationAndJoinedAt(t *testing.T) {
	m := ProjectMember{AllocationPct: 100}

	pct := 40.0
	joined := openapi_types.Date{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	m.ApplyUpdate(api.ProjectMemberUpdate{AllocationPct: &pct, JoinedAt: &joined})

	if m.AllocationPct != 40 {
		t.Errorf("AllocationPct = %v, want 40", m.AllocationPct)
	}
	if !m.JoinedAt.Equal(joined.Time) {
		t.Errorf("JoinedAt = %v, want %v", m.JoinedAt, joined.Time)
	}
}

func TestProjectMember_ToAPI(t *testing.T) {
	id, projectID, personID := uuid.New(), uuid.New(), uuid.New()
	joined := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	left := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	m := ProjectMember{
		ID: id, ProjectID: projectID, PersonID: personID, Role: api.ProjectRoleMaintainer,
		AllocationPct: 75, JoinedAt: joined, LeftAt: &left,
	}

	out := m.ToAPI()

	if out.Id != id || out.ProjectId != projectID || out.PersonId != personID {
		t.Errorf("ids = %v/%v/%v, want %v/%v/%v", out.Id, out.ProjectId, out.PersonId, id, projectID, personID)
	}
	if !out.JoinedAt.Equal(joined) {
		t.Errorf("JoinedAt = %v, want %v", out.JoinedAt, joined)
	}
	if out.LeftAt == nil || !out.LeftAt.Equal(left) {
		t.Errorf("LeftAt = %v, want %v", out.LeftAt, left)
	}
	if out.AllocationPct != 75 {
		t.Errorf("AllocationPct = %v, want 75", out.AllocationPct)
	}
}
