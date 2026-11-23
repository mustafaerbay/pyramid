package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestWorkItemRules(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")

	var p api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "R", Name: "Rules"}, admin, &p)
	path := "/api/v1/projects/" + p.Id.String()
	call(t, r, http.MethodPatch, path, map[string]any{"customFields": []map[string]any{
		{"key": "customer", "label": "Customer", "type": "text", "required": false},
		{"key": "risk", "label": "Risk", "type": "select", "options": []string{"Low", "High"}, "required": false},
	}}, admin, nil)
	if p.Rules.RequiredByState == nil || p.Rules.ReadOnly == nil {
		t.Errorf("default rules = %+v", p.Rules)
	}
	rules := map[string]any{
		"requiredByState": map[string][]string{"in_review": {"dueDate", "assignees", "customer"}},
		"defaults":        map[string]any{"priority": "high", "storyPoints": 3, "risk": "Low"},
		"readOnly":        []string{"customer"},
	}
	if s, _ := call(t, r, http.MethodPatch, path, map[string]any{"rules": rules}, admin, &p); s != http.StatusOK {
		t.Fatalf("save rules = %d", s)
	}
	for name, bad := range map[string]map[string]any{
		"unknown state": {"requiredByState": map[string][]string{"nope": {"dueDate"}}, "defaults": map[string]any{}, "readOnly": []string{}},
		"unknown field": {"requiredByState": map[string][]string{"todo": {"colour"}}, "defaults": map[string]any{}, "readOnly": []string{}},
		"bad default":   {"requiredByState": map[string][]string{}, "defaults": map[string]any{"risk": "Huge"}, "readOnly": []string{}},
		"ro built-in":   {"requiredByState": map[string][]string{}, "defaults": map[string]any{}, "readOnly": []string{"dueDate"}},
	} {
		if s, _ := call(t, r, http.MethodPatch, path, map[string]any{"rules": bad}, admin, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	// Defaults fill what isn't sent.
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "T", "customValues": map[string]any{"customer": "ACME"}}, amy, &task)
	if task.Priority != api.TaskPriorityHigh || task.StoryPoints == nil || *task.StoryPoints != 3 || task.CustomValues["risk"] != "Low" {
		t.Errorf("defaults = %s %v %v", task.Priority, task.StoryPoints, task.CustomValues)
	}
	var explicit api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "U", "priority": "low"}, amy, &explicit)
	if explicit.Priority != api.TaskPriorityLow {
		t.Errorf("sent priority overridden: %s", explicit.Priority)
	}

	// Required fields for a state.
	tp := "/api/v1/tasks/" + task.Id.String()
	var e api.Error
	s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"state": "in_review"}, amy, &e)
	if s != http.StatusBadRequest || !strings.Contains(e.Message, "Due date") || !strings.Contains(e.Message, "Assignees") || strings.Contains(e.Message, "Customer") {
		t.Errorf("missing fields = %d %q", s, e.Message)
	}
	if s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"state": "in_review", "dueDate": "2026-12-31", "assigneeIds": []uuid.UUID{uuid.New()}}, amy, nil); s != http.StatusOK {
		t.Errorf("with the fields = %d", s)
	}

	// Read-only for members after creation; managers may change it.
	if s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"customValues": map[string]any{"customer": "Globex", "risk": "Low"}}, amy, &e); s != http.StatusForbidden {
		t.Errorf("member changes read-only = %d", s)
	}
	if s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"customValues": map[string]any{"customer": "ACME", "risk": "High"}}, amy, nil); s != http.StatusOK {
		t.Errorf("member keeps read-only value = %d", s)
	}
	if s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"customValues": map[string]any{"customer": "Globex"}}, admin, nil); s != http.StatusOK {
		t.Errorf("admin changes read-only = %d", s)
	}
}
