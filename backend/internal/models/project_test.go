package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewProjectFromCreate_Defaults(t *testing.T) {
	p := NewProjectFromCreate(api.ProjectCreate{Key: "PYR", Name: "Pyramid"})

	if p.Status != api.ProjectStatusPlanning {
		t.Errorf("Status = %q, want planning (a project always starts in planning)", p.Status)
	}
	if p.Priority != api.ProjectPriorityMedium {
		t.Errorf("Priority = %q, want medium (default)", p.Priority)
	}
	if p.Health != api.ProjectHealthOnTrack {
		t.Errorf("Health = %q, want on_track (default)", p.Health)
	}
}

func TestNewProjectFromCreate_ExplicitPriorityAndDates(t *testing.T) {
	high := api.ProjectPriorityHigh
	start := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	target := openapi_types.Date{Time: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}

	p := NewProjectFromCreate(api.ProjectCreate{
		Key: "PYR", Name: "Pyramid", Priority: &high, StartDate: &start, TargetDate: &target,
	})

	if p.Priority != api.ProjectPriorityHigh {
		t.Errorf("Priority = %q, want high", p.Priority)
	}
	if p.StartDate == nil || !p.StartDate.Equal(start.Time) {
		t.Errorf("StartDate = %v, want %v", p.StartDate, start.Time)
	}
	if p.TargetDate == nil || !p.TargetDate.Equal(target.Time) {
		t.Errorf("TargetDate = %v, want %v", p.TargetDate, target.Time)
	}
}

func TestProject_ApplyUpdate_StatusAndHealthAreEditableOnlyViaUpdate(t *testing.T) {
	// Unlike Create (which can't set status/health), Update can — this is
	// the whole point of ApplyUpdate existing as a separate path.
	p := Project{Key: "PYR", Name: "Pyramid", Status: api.ProjectStatusPlanning, Health: api.ProjectHealthOnTrack}

	active := api.ProjectStatusActive
	atRisk := api.ProjectHealthAtRisk
	p.ApplyUpdate(api.ProjectUpdate{Status: &active, Health: &atRisk})

	if p.Status != api.ProjectStatusActive {
		t.Errorf("Status = %q, want active", p.Status)
	}
	if p.Health != api.ProjectHealthAtRisk {
		t.Errorf("Health = %q, want at_risk", p.Health)
	}
	if p.Name != "Pyramid" {
		t.Errorf("Name = %q, want unchanged Pyramid", p.Name)
	}
}

func TestProject_ApplyUpdate_AllFields(t *testing.T) {
	p := Project{Key: "OLD", Name: "Old name"}

	key, name := "PYR", "Pyramid Platform"
	desc := "The core platform."
	status := api.ProjectStatusCompleted
	priority := api.ProjectPriorityCritical
	health := api.ProjectHealthOffTrack
	leadID := uuid.New()
	start := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	target := openapi_types.Date{Time: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}

	p.ApplyUpdate(api.ProjectUpdate{
		Key: &key, Name: &name, Description: &desc, Status: &status, Priority: &priority,
		Health: &health, LeadId: &leadID, StartDate: &start, TargetDate: &target,
	})

	if p.Key != key || p.Name != name {
		t.Errorf("key/name = %q/%q, want %q/%q", p.Key, p.Name, key, name)
	}
	if p.Description == nil || *p.Description != desc {
		t.Errorf("Description = %v, want %q", p.Description, desc)
	}
	if p.Status != status || p.Priority != priority || p.Health != health {
		t.Errorf("status/priority/health = %q/%q/%q, want %q/%q/%q", p.Status, p.Priority, p.Health, status, priority, health)
	}
	if p.LeadID == nil || *p.LeadID != leadID {
		t.Errorf("LeadID = %v, want %v", p.LeadID, leadID)
	}
	if p.StartDate == nil || !p.StartDate.Equal(start.Time) {
		t.Errorf("StartDate = %v, want %v", p.StartDate, start.Time)
	}
	if p.TargetDate == nil || !p.TargetDate.Equal(target.Time) {
		t.Errorf("TargetDate = %v, want %v", p.TargetDate, target.Time)
	}
}

func TestProject_ToAPI_OmitsNilDates(t *testing.T) {
	p := Project{ID: uuid.New(), Key: "PYR", Name: "Pyramid"}
	out := p.ToAPI()

	if out.StartDate != nil {
		t.Errorf("StartDate = %v, want nil", out.StartDate)
	}
	if out.TargetDate != nil {
		t.Errorf("TargetDate = %v, want nil", out.TargetDate)
	}
}
