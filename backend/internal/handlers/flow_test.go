package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestFlow(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "F", Name: "Flow"}, &p)
	a := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "B"})
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "C"})
	c := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "D"})
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"status": "in_progress"}, nil)
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"status": "done"}, nil)
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+b.Id.String(), map[string]any{"status": "done"}, nil)
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+c.Id.String(), map[string]any{"status": "cancelled"}, nil)
	newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "Other project"})

	var f api.ProjectFlow
	if s := doJSON(t, r, http.MethodGet, "/api/v1/projects/"+p.Id.String()+"/flow?days=7", nil, &f); s != http.StatusOK {
		t.Fatalf("flow = %d", s)
	}
	if len(f.Days) != 7 {
		t.Fatalf("days = %d, want 7", len(f.Days))
	}
	if n := len(f.Days[0].Counts); n != 0 {
		t.Errorf("a week ago = %v, want nothing (items made today)", f.Days[0].Counts)
	}
	today := f.Days[6].Counts
	if today["done"] != 2 || today["backlog"] != 1 || today["cancelled"] != 0 {
		t.Errorf("today = %v", today)
	}
	if len(f.Completed) != 2 {
		t.Fatalf("completed = %+v", f.Completed)
	}
	for _, it := range f.Completed {
		if it.LeadDays < 0 || it.LeadDays > 0.1 {
			t.Errorf("%s lead = %v, want ~0", it.Title, it.LeadDays)
		}
		if it.Title == "A" && it.CycleDays == nil {
			t.Error("A went through in progress, cycle time missing")
		}
		if it.Title == "B" && it.CycleDays != nil {
			t.Errorf("B skipped in progress, cycle = %v", *it.CycleDays)
		}
	}
	if s := doJSON(t, r, http.MethodGet, "/api/v1/projects/"+uuid.NewString()+"/flow", nil, nil); s != http.StatusNotFound {
		t.Errorf("missing project = %d", s)
	}
}
