package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewVersionFromCreate_AlwaysStartsPlanned(t *testing.T) {
	desc := "First public release"
	target := openapi_types.Date{Time: time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)}
	pid := uuid.New()
	v := NewVersionFromCreate(api.VersionCreate{ProjectId: pid, Name: "v1.0", Description: &desc, TargetDate: &target})

	if v.Status != api.VersionStatusPlanned {
		t.Errorf("Status = %q, want planned", v.Status)
	}
	if v.ProjectID != pid || v.Name != "v1.0" || v.Description == nil || *v.Description != desc {
		t.Errorf("fields not copied: %+v", v)
	}
	if v.TargetDate == nil || !v.TargetDate.Equal(target.Time) {
		t.Errorf("TargetDate = %v, want %v", v.TargetDate, target.Time)
	}
	if v.ReleasedAt != nil {
		t.Errorf("ReleasedAt = %v, want nil", v.ReleasedAt)
	}
}

func TestVersion_ApplyUpdate_ReleasingStampsReleasedAt(t *testing.T) {
	v := Version{Status: api.VersionStatusInProgress}
	released := api.VersionStatusReleased
	v.ApplyUpdate(api.VersionUpdate{Status: &released})

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if v.Status != api.VersionStatusReleased {
		t.Errorf("Status = %q, want released", v.Status)
	}
	if v.ReleasedAt == nil || !v.ReleasedAt.Equal(today) {
		t.Errorf("ReleasedAt = %v, want today %v", v.ReleasedAt, today)
	}
}

func TestVersion_ApplyUpdate_ReleasingKeepsExistingReleasedAt(t *testing.T) {
	original := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	v := Version{Status: api.VersionStatusInProgress, ReleasedAt: &original}
	released := api.VersionStatusReleased
	v.ApplyUpdate(api.VersionUpdate{Status: &released})

	if !v.ReleasedAt.Equal(original) {
		t.Errorf("ReleasedAt = %v, want unchanged %v", v.ReleasedAt, original)
	}
}

func TestVersion_ApplyUpdate_ExplicitReleasedAtIsRespected(t *testing.T) {
	v := Version{Status: api.VersionStatusInProgress}
	released := api.VersionStatusReleased
	explicit := openapi_types.Date{Time: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)}
	v.ApplyUpdate(api.VersionUpdate{Status: &released, ReleasedAt: &explicit})

	if v.ReleasedAt == nil || !v.ReleasedAt.Equal(explicit.Time) {
		t.Errorf("ReleasedAt = %v, want %v", v.ReleasedAt, explicit.Time)
	}
}

func TestVersion_ApplyUpdate_LeavingReleasedDoesNotClearReleasedAt(t *testing.T) {
	original := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	v := Version{Status: api.VersionStatusReleased, ReleasedAt: &original}
	inProgress := api.VersionStatusInProgress
	v.ApplyUpdate(api.VersionUpdate{Status: &inProgress})

	if v.Status != api.VersionStatusInProgress {
		t.Errorf("Status = %q, want in_progress", v.Status)
	}
	if v.ReleasedAt == nil || !v.ReleasedAt.Equal(original) {
		t.Errorf("ReleasedAt = %v, want kept %v", v.ReleasedAt, original)
	}
}

func TestVersion_ApplyUpdate_NameDescriptionTargetDate(t *testing.T) {
	v := Version{Name: "v1", Status: api.VersionStatusPlanned}
	name, desc := "v1.0", "notes"
	target := openapi_types.Date{Time: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)}
	v.ApplyUpdate(api.VersionUpdate{Name: &name, Description: &desc, TargetDate: &target})

	if v.Name != "v1.0" || v.Description == nil || *v.Description != "notes" {
		t.Errorf("name/description not applied: %+v", v)
	}
	if v.TargetDate == nil || !v.TargetDate.Equal(target.Time) {
		t.Errorf("TargetDate = %v, want %v", v.TargetDate, target.Time)
	}
	if v.Status != api.VersionStatusPlanned || v.ReleasedAt != nil {
		t.Errorf("status/releasedAt changed unexpectedly: %+v", v)
	}
}

func TestVersion_ToAPI(t *testing.T) {
	target := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	rel := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	v := Version{ID: uuid.New(), ProjectID: uuid.New(), Name: "v1.0", Status: api.VersionStatusReleased, TargetDate: &target, ReleasedAt: &rel}
	out := v.ToAPI()

	if out.Id != v.ID || out.ProjectId != v.ProjectID || out.Name != "v1.0" || out.Status != api.VersionStatusReleased {
		t.Errorf("ToAPI basic fields wrong: %+v", out)
	}
	if out.TargetDate == nil || !out.TargetDate.Time.Equal(target) {
		t.Errorf("TargetDate = %v, want %v", out.TargetDate, target)
	}
	if out.ReleasedAt == nil || !out.ReleasedAt.Time.Equal(rel) {
		t.Errorf("ReleasedAt = %v, want %v", out.ReleasedAt, rel)
	}

	empty := Version{Name: "x", Status: api.VersionStatusPlanned}.ToAPI()
	if empty.TargetDate != nil || empty.ReleasedAt != nil || empty.Description != nil {
		t.Errorf("nil fields should stay nil: %+v", empty)
	}
}
