package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestWorkflow(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "W", Name: "Flow"}, &p)
	if len(p.Workflow) != 7 || p.Workflow[0].Key != "backlog" || p.Workflow[0].Category != api.TaskStatusBacklog {
		t.Fatalf("default workflow = %+v", p.Workflow)
	}
	// Tasks of a project with the default workflow: state follows status.
	var a api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "A"}, &a)
	if a.State != "backlog" || a.Status != api.TaskStatusBacklog {
		t.Errorf("new task = %s/%s", a.State, a.Status)
	}

	wf := []map[string]any{
		{"key": "new", "label": "New", "category": "backlog"},
		{"key": "ready", "label": "Ready", "category": "todo"},
		{"key": "doing", "label": "Doing", "category": "in_progress"},
		{"key": "code_review", "label": "Code review", "category": "in_review"},
		{"key": "qa", "label": "QA", "category": "in_review"},
		{"key": "shipped", "label": "Shipped", "category": "done"},
		{"key": "dropped", "label": "Dropped", "category": "cancelled"},
	}
	// The old states are still in use by A: they can't just disappear.
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"workflow": wf}, nil); s != http.StatusBadRequest {
		t.Errorf("dropping a used state = %d, want 400", s)
	}
	doJSON(t, r, http.MethodDelete, "/api/v1/tasks/"+a.Id.String(), nil, nil)
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"workflow": wf}, &p); s != http.StatusOK || len(p.Workflow) != 7 || p.Workflow[4].Label != "QA" {
		t.Fatalf("save workflow = %d %+v", s, p.Workflow)
	}
	for name, bad := range map[string][]map[string]any{
		"no done":       {{"key": "a", "label": "A", "category": "todo"}},
		"only done":     {{"key": "a", "label": "A", "category": "done"}},
		"duplicate key": {{"key": "a", "label": "A", "category": "todo"}, {"key": "a", "label": "B", "category": "done"}},
		"bad category":  {{"key": "a", "label": "A", "category": "later"}, {"key": "b", "label": "B", "category": "done"}},
		"bad key":       {{"key": "A b", "label": "A", "category": "todo"}, {"key": "b", "label": "B", "category": "done"}},
	} {
		if s := doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"workflow": bad}, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	var b api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "B"}, &b)
	if b.State != "new" || b.Status != api.TaskStatusBacklog {
		t.Errorf("new task in custom flow = %s/%s, want new/backlog", b.State, b.Status)
	}
	path := "/api/v1/tasks/" + b.Id.String()
	var got api.Task
	doJSON(t, r, http.MethodPatch, path, map[string]any{"state": "qa"}, &got)
	if got.State != "qa" || got.Status != api.TaskStatusInReview {
		t.Errorf("to qa = %s/%s", got.State, got.Status)
	}
	// Status alone (board of an old client, bulk edit) picks that category's first state.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"status": "in_review"}, &got)
	if got.State != "qa" {
		t.Errorf("same category keeps the state, got %s", got.State)
	}
	doJSON(t, r, http.MethodPatch, path, map[string]any{"status": "done"}, &got)
	if got.State != "shipped" || got.CompletedAt == nil {
		t.Errorf("status done = %s, completed %v", got.State, got.CompletedAt)
	}
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"state": "nope"}, nil); s != http.StatusBadRequest {
		t.Errorf("unknown state = %d, want 400", s)
	}
	var c api.Task
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "C", "state": "doing"}, &c); s != http.StatusCreated || c.Status != api.TaskStatusInProgress {
		t.Errorf("create in doing = %d %s", s, c.Status)
	}

	// WIP limits are per state now.
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"boardSettings": map[string]any{"wipLimits": map[string]int{"qa": 2}, "swimlanes": "none", "cardColor": "none"}}, nil); s != http.StatusOK {
		t.Errorf("WIP limit on a custom state = %d", s)
	}
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"boardSettings": map[string]any{"wipLimits": map[string]int{"todo": 2}, "swimlanes": "none", "cardColor": "none"}}, nil); s != http.StatusBadRequest {
		t.Errorf("WIP limit on a state the project doesn't have = %d, want 400", s)
	}
}
