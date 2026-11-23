package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestWorkItemTypes(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "T", Name: "Types"}, &p)
	if p.WorkItemTypes == nil || len(p.WorkItemTypes) != 0 {
		t.Errorf("default types = %#v, want []", p.WorkItemTypes)
	}
	path := "/api/v1/projects/" + p.Id.String()
	types := []map[string]any{
		{"key": "spike", "label": "Spike", "base": "task", "color": "pink"},
		{"key": "initiative", "label": "Initiative", "base": "epic", "color": "slate"},
	}
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"workItemTypes": types}, &p); s != http.StatusOK || len(p.WorkItemTypes) != 2 {
		t.Fatalf("save types = %d %+v", s, p.WorkItemTypes)
	}
	for name, bad := range map[string][]map[string]any{
		"built-in name": {{"key": "bug", "label": "Bug 2", "base": "bug", "color": "red"}},
		"bad base":      {{"key": "x", "label": "X", "base": "saga", "color": "red"}},
		"bad colour":    {{"key": "x", "label": "X", "base": "task", "color": "plaid"}},
		"duplicate":     {{"key": "x", "label": "X", "base": "task", "color": "red"}, {"key": "x", "label": "Y", "base": "task", "color": "red"}},
	} {
		if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"workItemTypes": bad}, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	var ini, spike api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "Big", "customType": "initiative"}, &ini)
	if ini.Type != api.TaskTypeEpic || ini.CustomType == nil || *ini.CustomType != "initiative" || ini.CustomTypeLabel == nil || *ini.CustomTypeLabel != "Initiative" {
		t.Fatalf("initiative = %+v", ini)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "S", "customType": "spike", "parentId": ini.Id}, &spike); s != http.StatusCreated || spike.Type != api.TaskTypeTask || *spike.CustomTypeColor != api.WorkItemTypeColorPink {
		t.Errorf("spike = %d %+v", s, spike)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "X", "customType": "nope"}, nil); s != http.StatusBadRequest {
		t.Errorf("unknown type = %d", s)
	}
	// Setting a built-in type drops the custom one; null clears it too.
	var got api.Task
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+spike.Id.String(), map[string]any{"type": "story"}, &got)
	if got.CustomType != nil || got.Type != api.TaskTypeStory {
		t.Errorf("after type story = %v %s", got.CustomType, got.Type)
	}
	// A type in use can't be removed.
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"workItemTypes": types[:1]}, nil); s != http.StatusBadRequest {
		t.Errorf("removing a used type = %d, want 400", s)
	}
}
