package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewMilestoneFromCreate_AlwaysStartsPending(t *testing.T) {
	m := NewMilestoneFromCreate(api.MilestoneCreate{ProjectId: uuid.New(), Name: "Beta launch"})

	if m.Status != api.MilestoneStatusPending {
		t.Errorf("Status = %q, want pending — a milestone can't be created already in progress or completed", m.Status)
	}
	if m.CompletedAt != nil {
		t.Errorf("CompletedAt = %v, want nil for a brand new milestone", m.CompletedAt)
	}
}

func TestMilestone_ApplyUpdate_CompletingStampsCompletedAtAutomatically(t *testing.T) {
	m := Milestone{Status: api.MilestoneStatusInProgress}

	completed := api.MilestoneStatusCompleted
	before := time.Now().UTC()
	m.ApplyUpdate(api.MilestoneUpdate{Status: &completed}, false)
	after := time.Now().UTC()

	if m.Status != api.MilestoneStatusCompleted {
		t.Errorf("Status = %q, want completed", m.Status)
	}
	if m.CompletedAt == nil {
		t.Fatal("CompletedAt = nil, want auto-stamped to today")
	}
	if m.CompletedAt.Before(before.Truncate(24*time.Hour)) || m.CompletedAt.After(after) {
		t.Errorf("CompletedAt = %v, want between %v and %v", m.CompletedAt, before, after)
	}
}

func TestMilestone_ApplyUpdate_CompletingDoesNotOverwriteExistingCompletedAt(t *testing.T) {
	original := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	m := Milestone{Status: api.MilestoneStatusInProgress, CompletedAt: &original}

	completed := api.MilestoneStatusCompleted
	m.ApplyUpdate(api.MilestoneUpdate{Status: &completed}, false)

	if !m.CompletedAt.Equal(original) {
		t.Errorf("CompletedAt = %v, want unchanged %v (already set, shouldn't be re-stamped)", m.CompletedAt, original)
	}
}

func TestMilestone_ApplyUpdate_ExplicitCompletedAtIsRespected(t *testing.T) {
	m := Milestone{Status: api.MilestoneStatusInProgress}

	completed := api.MilestoneStatusCompleted
	explicit := openapi_types.Date{Time: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)}
	m.ApplyUpdate(api.MilestoneUpdate{Status: &completed, CompletedAt: &explicit}, false)

	if m.CompletedAt == nil || !m.CompletedAt.Equal(explicit.Time) {
		t.Errorf("CompletedAt = %v, want the explicitly supplied %v, not auto-stamped today", m.CompletedAt, explicit.Time)
	}
}

func TestMilestone_ApplyUpdate_NonCompletingStatusChangeLeavesCompletedAtAlone(t *testing.T) {
	m := Milestone{Status: api.MilestoneStatusPending}

	inProgress := api.MilestoneStatusInProgress
	m.ApplyUpdate(api.MilestoneUpdate{Status: &inProgress}, false)

	if m.CompletedAt != nil {
		t.Errorf("CompletedAt = %v, want nil — moving to in_progress shouldn't stamp completion", m.CompletedAt)
	}
}

func TestMilestone_ApplyUpdate_NameDescriptionAndDatesAreEditable(t *testing.T) {
	m := Milestone{Name: "Old name"}

	newName := "Beta launch"
	newDesc := "Public beta goes live."
	start := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	due := openapi_types.Date{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	m.ApplyUpdate(api.MilestoneUpdate{
		Name: &newName, Description: &newDesc, StartDate: &start, DueDate: &due,
	}, false)

	if m.Name != newName {
		t.Errorf("Name = %q, want %q", m.Name, newName)
	}
	if m.Description == nil || *m.Description != newDesc {
		t.Errorf("Description = %v, want %q", m.Description, newDesc)
	}
	if m.StartDate == nil || !m.StartDate.Equal(start.Time) {
		t.Errorf("StartDate = %v, want %v", m.StartDate, start.Time)
	}
	if m.DueDate == nil || !m.DueDate.Equal(due.Time) {
		t.Errorf("DueDate = %v, want %v", m.DueDate, due.Time)
	}
}

func TestMilestone_ToAPI(t *testing.T) {
	id, projectID := uuid.New(), uuid.New()
	desc := "Locked the schema."
	due := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	m := Milestone{
		ID: id, ProjectID: projectID, Name: "Schema finalized",
		Description: &desc, Status: api.MilestoneStatusCompleted, DueDate: &due,
	}

	out := m.ToAPI()

	if out.Id != id || out.ProjectId != projectID {
		t.Errorf("Id/ProjectId = %v/%v, want %v/%v", out.Id, out.ProjectId, id, projectID)
	}
	if out.Description == nil || *out.Description != desc {
		t.Errorf("Description = %v, want %q", out.Description, desc)
	}
	if out.DueDate == nil || !out.DueDate.Equal(due) {
		t.Errorf("DueDate = %v, want %v", out.DueDate, due)
	}
	if out.StartDate != nil {
		t.Errorf("StartDate = %v, want nil (not set on this milestone)", out.StartDate)
	}
}

func TestMilestone_VersionId_CreateUpdateClear(t *testing.T) {
	v1, v2 := uuid.New(), uuid.New()
	m := NewMilestoneFromCreate(api.MilestoneCreate{ProjectId: uuid.New(), Name: "M1", VersionId: &v1})
	if m.VersionID == nil || *m.VersionID != v1 {
		t.Fatalf("create VersionID = %v, want %v", m.VersionID, v1)
	}
	if out := m.ToAPI(); out.VersionId == nil || *out.VersionId != v1 {
		t.Errorf("ToAPI VersionId = %v, want %v", out.VersionId, v1)
	}

	m.ApplyUpdate(api.MilestoneUpdate{}, false)
	if m.VersionID == nil || *m.VersionID != v1 {
		t.Errorf("absent versionId changed it to %v", m.VersionID)
	}

	m.ApplyUpdate(api.MilestoneUpdate{VersionId: &v2}, false)
	if m.VersionID == nil || *m.VersionID != v2 {
		t.Errorf("set VersionID = %v, want %v", m.VersionID, v2)
	}

	m.ApplyUpdate(api.MilestoneUpdate{}, true)
	if m.VersionID != nil {
		t.Errorf("clear left VersionID = %v, want nil", m.VersionID)
	}
}
