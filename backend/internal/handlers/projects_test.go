package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestProjects_Create_AppliesDefaults(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Project
	status := doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{
		Key: "PYR", Name: "Pyramid",
	}, &created)

	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", status, http.StatusCreated)
	}
	// ProjectCreate has no status/health field at all — NewProjectFromCreate
	// is the only place these defaults come from, so this test exercises
	// that the HTTP layer actually calls it rather than leaving zero values.
	if created.Status != api.ProjectStatusPlanning {
		t.Errorf("Status = %q, want planning (default)", created.Status)
	}
	if created.Priority != api.ProjectPriorityMedium {
		t.Errorf("Priority = %q, want medium (default)", created.Priority)
	}
	if created.Health != api.ProjectHealthOnTrack {
		t.Errorf("Health = %q, want on_track (default)", created.Health)
	}
}

func TestProjects_Create_ExplicitPriority(t *testing.T) {
	r, _ := newTestRouter(t)

	high := api.ProjectPriorityHigh
	var created api.Project
	status := doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{
		Key: "PYR", Name: "Pyramid", Priority: &high,
	}, &created)

	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", status, http.StatusCreated)
	}
	if created.Priority != api.ProjectPriorityHigh {
		t.Errorf("Priority = %q, want high", created.Priority)
	}
}

func TestProjects_GetUpdateDelete(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "PYR", Name: "Pyramid"}, &created)

	var fetched api.Project
	status := doJSON(t, r, http.MethodGet, "/api/v1/projects/"+created.Id.String(), nil, &fetched)
	if status != http.StatusOK || fetched.Id != created.Id {
		t.Fatalf("get status=%d id=%v, want 200 and id=%v", status, fetched.Id, created.Id)
	}

	// Status/Health are only reachable via Update, not Create — this is the
	// whole reason ApplyUpdate exists as a distinct path from the model
	// constructor.
	active := api.ProjectStatusActive
	atRisk := api.ProjectHealthAtRisk
	var updated api.Project
	status = doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+created.Id.String(),
		api.ProjectUpdate{Status: &active, Health: &atRisk}, &updated)
	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if updated.Status != api.ProjectStatusActive {
		t.Errorf("Status = %q, want active", updated.Status)
	}
	if updated.Health != api.ProjectHealthAtRisk {
		t.Errorf("Health = %q, want at_risk", updated.Health)
	}
	if updated.Name != "Pyramid" {
		t.Errorf("Name = %q, want unchanged Pyramid", updated.Name)
	}

	status = doJSON(t, r, http.MethodDelete, "/api/v1/projects/"+created.Id.String(), nil, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", status, http.StatusNoContent)
	}
	status = doJSON(t, r, http.MethodGet, "/api/v1/projects/"+created.Id.String(), nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", status, http.StatusNotFound)
	}
}

func TestProjects_Get_NotFound(t *testing.T) {
	r, _ := newTestRouter(t)

	status := doJSON(t, r, http.MethodGet, "/api/v1/projects/"+uuid.New().String(), nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
}

func TestProjects_List_StatusFilter(t *testing.T) {
	r, _ := newTestRouter(t)

	// Two fresh projects: both default to status=planning.
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "PYR", Name: "Pyramid"}, nil)
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "ATL", Name: "Atlas"}, nil)

	var planning api.ProjectList
	status := doJSON(t, r, http.MethodGet, "/api/v1/projects?status=planning", nil, &planning)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want %d", status, http.StatusOK)
	}
	if len(planning.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(planning.Data))
	}

	var active api.ProjectList
	status = doJSON(t, r, http.MethodGet, "/api/v1/projects?status=active", nil, &active)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want %d", status, http.StatusOK)
	}
	if len(active.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0 for status=active filter", len(active.Data))
	}
}
