package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestProcessTemplates(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")

	var src, dst api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "S", Name: "Source"}, admin, &src)
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "D", Name: "Target"}, admin, &dst)
	wf := []map[string]any{
		{"key": "new", "label": "New", "category": "backlog"}, {"key": "doing", "label": "Doing", "category": "in_progress"},
		{"key": "qa", "label": "QA", "category": "in_review"}, {"key": "shipped", "label": "Shipped", "category": "done"},
	}
	call(t, r, http.MethodPatch, "/api/v1/projects/"+src.Id.String(), map[string]any{
		"workflow":      wf,
		"workItemTypes": []map[string]any{{"key": "spike", "label": "Spike", "base": "task", "color": "pink"}},
		"customFields":  []map[string]any{{"key": "customer", "label": "Customer", "type": "text", "required": false}},
		"rules":         map[string]any{"requiredByState": map[string][]string{"qa": {"customer"}}, "defaults": map[string]any{}, "readOnly": []string{}},
	}, admin, nil)

	if s, _ := call(t, r, http.MethodPost, "/api/v1/process-templates", map[string]any{"name": "Kanban+QA", "fromProjectId": src.Id}, amy, nil); s != http.StatusForbidden {
		t.Errorf("member saves template = %d, want 403", s)
	}
	var tpl api.ProcessTemplate
	if s, _ := call(t, r, http.MethodPost, "/api/v1/process-templates", map[string]any{"name": "Kanban+QA", "fromProjectId": src.Id}, admin, &tpl); s != http.StatusCreated || len(tpl.Process.Workflow) != 4 {
		t.Fatalf("save template = %d %+v", s, tpl.Process.Workflow)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/process-templates", map[string]any{"name": "kanban+qa", "fromProjectId": src.Id}, admin, nil); s != http.StatusBadRequest {
		t.Errorf("duplicate name = %d", s)
	}

	// The target has an item in "todo", which the template doesn't have.
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": dst.Id, "title": "T", "status": "todo"}, amy, &task)
	ap := "/api/v1/projects/" + dst.Id.String() + "/apply-process"
	if s, _ := call(t, r, http.MethodPost, ap, map[string]any{"templateId": tpl.Id}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("dropping a used state = %d, want 400", s)
	}
	call(t, r, http.MethodDelete, "/api/v1/tasks/"+task.Id.String(), nil, amy, nil)
	var got api.Project
	if s, _ := call(t, r, http.MethodPost, ap, map[string]any{"templateId": tpl.Id}, amy, &got); s != http.StatusOK {
		t.Fatalf("apply = %d", s)
	}
	if len(got.Workflow) != 4 || got.WorkItemTypes[0].Key != "spike" || got.CustomFields[0].Key != "customer" || len(got.Rules.RequiredByState["qa"]) != 1 {
		t.Errorf("applied = %+v", got)
	}
	// Copy straight from a project too.
	var third api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "E", Name: "Third"}, admin, &third)
	if s, _ := call(t, r, http.MethodPost, "/api/v1/projects/"+third.Id.String()+"/apply-process", map[string]any{"fromProjectId": src.Id}, amy, &got); s != http.StatusOK || got.Workflow[2].Key != "qa" {
		t.Errorf("copy from project = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, ap, map[string]any{}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("no source = %d", s)
	}
	var list api.ProcessTemplateList
	call(t, r, http.MethodGet, "/api/v1/process-templates", nil, amy, &list)
	if len(list.Data) != 1 {
		t.Errorf("templates = %d", len(list.Data))
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/process-templates/"+tpl.Id.String(), nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("delete = %d", s)
	}
}
