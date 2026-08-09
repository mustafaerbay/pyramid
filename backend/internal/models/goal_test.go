package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewGoalFromCreate_Defaults(t *testing.T) {
	g := NewGoalFromCreate(api.GoalCreate{Title: "Ship v2"})

	if g.Scope != api.GoalScopeIndividual {
		t.Errorf("Scope = %q, want individual (default)", g.Scope)
	}
	if g.Status != api.GoalStatusDraft {
		t.Errorf("Status = %q, want draft (default, matches Milestone's update-only status pattern)", g.Status)
	}
	if g.ProgressPct != 0 {
		t.Errorf("ProgressPct = %v, want 0", g.ProgressPct)
	}
}

func TestNewGoalFromCreate_ExplicitScope(t *testing.T) {
	company := api.GoalScopeCompany
	g := NewGoalFromCreate(api.GoalCreate{Title: "Cut cloud spend", Scope: &company})

	if g.Scope != api.GoalScopeCompany {
		t.Errorf("Scope = %q, want company", g.Scope)
	}
}

func TestGoal_ApplyUpdate_ProgressAndStatusAreIndependentlyEditable(t *testing.T) {
	g := Goal{Title: "Ship v2", Status: api.GoalStatusDraft, ProgressPct: 0}

	active := api.GoalStatusActive
	pct := 65.0
	g.ApplyUpdate(api.GoalUpdate{Status: &active, ProgressPct: &pct})

	if g.Status != api.GoalStatusActive {
		t.Errorf("Status = %q, want active", g.Status)
	}
	if g.ProgressPct != 65 {
		t.Errorf("ProgressPct = %v, want 65", g.ProgressPct)
	}
	if g.Title != "Ship v2" {
		t.Errorf("Title = %q, want unchanged", g.Title)
	}
}

func TestGoal_ApplyUpdate_ProgressCanReachZeroExplicitly(t *testing.T) {
	// A pointer-to-float lets 0 be distinguished from "not supplied" — this
	// pins that a patch explicitly resetting progress to 0 actually works,
	// since `if in.ProgressPct != nil` (not `if *in.ProgressPct != 0`) is
	// the correct guard.
	g := Goal{Title: "Draft roadmap", ProgressPct: 40}

	zero := 0.0
	g.ApplyUpdate(api.GoalUpdate{ProgressPct: &zero})

	if g.ProgressPct != 0 {
		t.Errorf("ProgressPct = %v, want 0", g.ProgressPct)
	}
}

func TestGoal_ApplyUpdate_DescriptionOwnerAndDates(t *testing.T) {
	g := Goal{Title: "Ship v2"}

	desc := "Ship the rewritten platform."
	owner := uuid.New()
	start := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	due := openapi_types.Date{Time: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}

	g.ApplyUpdate(api.GoalUpdate{
		Description: &desc, OwnerPersonId: &owner, StartDate: &start, DueDate: &due,
	})

	if g.Description == nil || *g.Description != desc {
		t.Errorf("Description = %v, want %q", g.Description, desc)
	}
	if g.OwnerPersonID == nil || *g.OwnerPersonID != owner {
		t.Errorf("OwnerPersonID = %v, want %v", g.OwnerPersonID, owner)
	}
	if g.StartDate == nil || !g.StartDate.Equal(start.Time) {
		t.Errorf("StartDate = %v, want %v", g.StartDate, start.Time)
	}
	if g.DueDate == nil || !g.DueDate.Equal(due.Time) {
		t.Errorf("DueDate = %v, want %v", g.DueDate, due.Time)
	}
}

func TestGoal_ToAPI(t *testing.T) {
	id, owner := uuid.New(), uuid.New()
	desc := "Ship the rewritten platform."
	due := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	g := Goal{
		ID: id, Title: "Ship v2", Description: &desc, Scope: api.GoalScopeCompany,
		Status: api.GoalStatusActive, OwnerPersonID: &owner, ProgressPct: 65, DueDate: &due,
	}

	out := g.ToAPI()

	if out.Id != id {
		t.Errorf("Id = %v, want %v", out.Id, id)
	}
	if out.OwnerPersonId == nil || *out.OwnerPersonId != owner {
		t.Errorf("OwnerPersonId = %v, want %v", out.OwnerPersonId, owner)
	}
	if out.DueDate == nil || !out.DueDate.Equal(due) {
		t.Errorf("DueDate = %v, want %v", out.DueDate, due)
	}
	if out.ProgressPct != 65 {
		t.Errorf("ProgressPct = %v, want 65", out.ProgressPct)
	}
}
