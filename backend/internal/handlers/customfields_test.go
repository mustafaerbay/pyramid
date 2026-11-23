package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestCustomFields(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "C", Name: "Custom"}, &p)
	if p.CustomFields == nil || len(p.CustomFields) != 0 {
		t.Errorf("default fields = %#v, want []", p.CustomFields)
	}
	path := "/api/v1/projects/" + p.Id.String()
	fields := []map[string]any{
		{"key": "customer", "label": "Customer", "type": "text", "required": true},
		{"key": "risk", "label": "Risk", "type": "select", "options": []string{"Low", "High"}, "required": false},
		{"key": "budget", "label": "Budget", "type": "number", "required": false},
		{"key": "golive", "label": "Go-live", "type": "date", "required": false},
	}
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"customFields": fields}, &p); s != http.StatusOK || len(p.CustomFields) != 4 {
		t.Fatalf("save fields = %d %+v", s, p.CustomFields)
	}
	for name, bad := range map[string][]map[string]any{
		"bad key":        {{"key": "Bad Key", "label": "x", "type": "text", "required": false}},
		"duplicate key":  {{"key": "a", "label": "x", "type": "text", "required": false}, {"key": "a", "label": "y", "type": "text", "required": false}},
		"select without": {{"key": "s", "label": "S", "type": "select", "required": false}},
		"unknown type":   {{"key": "s", "label": "S", "type": "colour", "required": false}},
	} {
		if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"customFields": bad}, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	create := func(values map[string]any) (int, api.Task) {
		var task api.Task
		s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "T", "customValues": values}, &task)
		return s, task
	}
	s, task := create(map[string]any{"customer": "ACME", "risk": "High", "budget": 1200.5, "golive": "2027-01-15"})
	if s != http.StatusCreated || task.CustomValues["customer"] != "ACME" || task.CustomValues["budget"] != 1200.5 {
		t.Fatalf("create = %d %+v", s, task.CustomValues)
	}
	for name, vals := range map[string]map[string]any{
		"missing required": {"risk": "Low"},
		"bad option":       {"customer": "A", "risk": "Medium"},
		"not a number":     {"customer": "A", "budget": "lots"},
		"bad date":         {"customer": "A", "golive": "15/01/2027"},
		"unknown field":    {"customer": "A", "colour": "red"},
	} {
		if s, _ := create(vals); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	var got api.Task
	tp := "/api/v1/tasks/" + task.Id.String()
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"title": "T2"}, &got)
	if got.CustomValues["risk"] != "High" {
		t.Errorf("values after title PATCH = %v", got.CustomValues)
	}
	var e api.Error
	if s := doJSON(t, r, http.MethodPatch, tp, map[string]any{"customValues": map[string]any{"customer": "Globex"}}, &e); s != http.StatusOK {
		t.Fatalf("replace = %d %s", s, e.Message)
	}
	got = api.Task{} // decoding into a used map would keep its old keys
	doJSON(t, r, http.MethodGet, tp, nil, &got)
	if got.CustomValues["customer"] != "Globex" || got.CustomValues["risk"] != nil {
		t.Errorf("values after replace = %v", got.CustomValues)
	}

	// A field removed from the project disappears from items.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"customFields": fields[1:]}, nil)
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"customValues": map[string]any{"customer": "Initech", "risk": "Low"}}, nil)
	got = api.Task{}
	doJSON(t, r, http.MethodGet, tp, nil, &got)
	if _, ok := got.CustomValues["customer"]; ok {
		t.Errorf("removed field still shown: %v", got.CustomValues)
	}
}
