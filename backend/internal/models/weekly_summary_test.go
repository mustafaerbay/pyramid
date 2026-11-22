package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewWeeklySummaryFromCreate(t *testing.T) {
	projectID := uuid.New()
	weekOf := openapi_types.Date{Time: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)}

	w := NewWeeklySummaryFromCreate(api.WeeklySummaryCreate{
		ProjectId: projectID,
		WeekOf:    weekOf,
		Summary:   "Shipped the milestones timeline rework.",
	})

	if w.ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %v", w.ProjectID, projectID)
	}
	if !w.WeekOf.Equal(weekOf.Time) {
		t.Errorf("WeekOf = %v, want %v", w.WeekOf, weekOf.Time)
	}
	if w.Summary != "Shipped the milestones timeline rework." {
		t.Errorf("Summary = %q, want the supplied text", w.Summary)
	}
}

func TestWeeklySummary_ApplyUpdate_PartialUpdateOnlyTouchesSuppliedFields(t *testing.T) {
	original := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	w := WeeklySummary{WeekOf: original, Summary: "Original text."}

	newSummary := "Revised text with more detail."
	w.ApplyUpdate(api.WeeklySummaryUpdate{Summary: &newSummary})

	if w.Summary != newSummary {
		t.Errorf("Summary = %q, want %q", w.Summary, newSummary)
	}
	if !w.WeekOf.Equal(original) {
		t.Errorf("WeekOf = %v, want unchanged %v — update didn't supply weekOf", w.WeekOf, original)
	}
}

func TestWeeklySummary_ApplyUpdate_WeekOfIsEditable(t *testing.T) {
	w := WeeklySummary{WeekOf: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), Summary: "text"}

	newWeek := openapi_types.Date{Time: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)}
	w.ApplyUpdate(api.WeeklySummaryUpdate{WeekOf: &newWeek})

	if !w.WeekOf.Equal(newWeek.Time) {
		t.Errorf("WeekOf = %v, want %v", w.WeekOf, newWeek.Time)
	}
}

func TestWeeklySummary_ApplyUpdate_EmptyPatchChangesNothing(t *testing.T) {
	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	w := WeeklySummary{WeekOf: week, Summary: "unchanged"}

	w.ApplyUpdate(api.WeeklySummaryUpdate{})

	if w.Summary != "unchanged" || !w.WeekOf.Equal(week) {
		t.Errorf("fields changed on empty patch: Summary=%q WeekOf=%v", w.Summary, w.WeekOf)
	}
}

func TestWeeklySummary_ToAPI(t *testing.T) {
	id, projectID := uuid.New(), uuid.New()
	week := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	w := WeeklySummary{
		ID: id, ProjectID: projectID, WeekOf: week,
		Summary: "4 tasks done, 2 in progress.", CreatedAt: created, UpdatedAt: created,
	}

	out := w.ToAPI()

	if out.Id != id || out.ProjectId != projectID {
		t.Errorf("Id/ProjectId = %v/%v, want %v/%v", out.Id, out.ProjectId, id, projectID)
	}
	if !out.WeekOf.Equal(week) {
		t.Errorf("WeekOf = %v, want %v", out.WeekOf, week)
	}
	if out.Summary != "4 tasks done, 2 in progress." {
		t.Errorf("Summary = %q, want the original text", out.Summary)
	}
	if !out.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", out.CreatedAt, created)
	}
}
