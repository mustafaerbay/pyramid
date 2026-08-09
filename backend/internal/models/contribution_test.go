package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestNewContributionFromCreate_DefaultsWeightToOne(t *testing.T) {
	before := time.Now().UTC()
	c := NewContributionFromCreate(api.ContributionCreate{
		PersonId: uuid.New(), Type: api.ContributionTypeCommit,
	})
	after := time.Now().UTC()

	if c.Weight != 1 {
		t.Errorf("Weight = %v, want 1 (default)", c.Weight)
	}
	if c.OccurredAt.Before(before) || c.OccurredAt.After(after) {
		t.Errorf("OccurredAt = %v, want between %v and %v (defaults to now)", c.OccurredAt, before, after)
	}
}

func TestNewContributionFromCreate_ExplicitWeightAndOccurredAt(t *testing.T) {
	weight := 2.5
	occurred := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)

	c := NewContributionFromCreate(api.ContributionCreate{
		PersonId: uuid.New(), Type: api.ContributionTypePullRequest,
		Weight: &weight, OccurredAt: &occurred,
	})

	if c.Weight != 2.5 {
		t.Errorf("Weight = %v, want 2.5", c.Weight)
	}
	if !c.OccurredAt.Equal(occurred) {
		t.Errorf("OccurredAt = %v, want %v", c.OccurredAt, occurred)
	}
}

func TestContribution_ApplyUpdate_ProjectIdCanBeClearedToNil(t *testing.T) {
	projectID := uuid.New()
	c := Contribution{ProjectID: &projectID, Type: api.ContributionTypeCommit}

	// ProjectId is a *uuid.UUID in ContributionUpdate — passing a non-nil
	// pointer to a nil-valued UUID would be the "clear" case in other
	// models, but this model's ApplyUpdate takes the pointer as-is (a
	// present *and* non-nil value reassigns; there's currently no way to
	// explicitly null it back out via this method, same class of limitation
	// as the milestoneId/isExplicitNull cases elsewhere in the app).
	newProjectID := uuid.New()
	c.ApplyUpdate(api.ContributionUpdate{ProjectId: &newProjectID})

	if c.ProjectID == nil || *c.ProjectID != newProjectID {
		t.Errorf("ProjectID = %v, want %v", c.ProjectID, newProjectID)
	}
}

func TestContribution_ApplyUpdate_AllFields(t *testing.T) {
	c := Contribution{Type: api.ContributionTypeCommit, Weight: 1}

	personID, projectID := uuid.New(), uuid.New()
	contribType := api.ContributionTypeDeployment
	title, url := "Deploy v2", "https://example.com/deploy/42"
	weight := 3.0
	occurred := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	c.ApplyUpdate(api.ContributionUpdate{
		PersonId: &personID, ProjectId: &projectID, Type: &contribType,
		Title: &title, Url: &url, Weight: &weight, OccurredAt: &occurred,
	})

	if c.PersonID != personID {
		t.Errorf("PersonID = %v, want %v", c.PersonID, personID)
	}
	if c.ProjectID == nil || *c.ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %v", c.ProjectID, projectID)
	}
	if c.Type != contribType {
		t.Errorf("Type = %q, want %q", c.Type, contribType)
	}
	if c.Title == nil || *c.Title != title {
		t.Errorf("Title = %v, want %q", c.Title, title)
	}
	if c.Url == nil || *c.Url != url {
		t.Errorf("Url = %v, want %q", c.Url, url)
	}
	if c.Weight != weight {
		t.Errorf("Weight = %v, want %v", c.Weight, weight)
	}
	if !c.OccurredAt.Equal(occurred) {
		t.Errorf("OccurredAt = %v, want %v", c.OccurredAt, occurred)
	}
}

func TestContribution_ToAPI_CarriesOptionalFields(t *testing.T) {
	title := "Fix outstanding bugs"
	c := Contribution{
		ID: uuid.New(), PersonID: uuid.New(), Type: api.ContributionTypeIssue,
		Title: &title, Weight: 1,
	}

	out := c.ToAPI()

	if out.Title == nil || *out.Title != title {
		t.Errorf("Title = %v, want %q", out.Title, title)
	}
	if out.Type != api.ContributionTypeIssue {
		t.Errorf("Type = %q, want issue", out.Type)
	}
}
