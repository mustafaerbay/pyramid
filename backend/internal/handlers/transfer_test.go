package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestTransferTask(t *testing.T) {
	r, _ := newTestRouter(t)
	var from, to api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "FR", Name: "From"}, &from)
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "TO", Name: "To"}, &to)
	doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+from.Id.String(), map[string]any{"customFields": []map[string]any{
		{"key": "risk", "label": "Risk", "type": "text", "required": false},
	}}, nil)
	newTask(t, r, api.TaskCreate{ProjectId: to.Id, Title: "Already there"})
	var sprint api.Sprint
	doJSON(t, r, http.MethodPost, "/api/v1/sprints", map[string]any{"projectId": from.Id, "name": "S1", "startDate": "2026-12-01", "endDate": "2026-12-14"}, &sprint)
	story := api.TaskTypeStory
	parent := newTask(t, r, api.TaskCreate{ProjectId: from.Id, Title: "Parent", Type: &story, SprintId: &sprint.Id, Tags: tagsPtr("x"),
		CustomValues: &map[string]any{"risk": "high"}})
	child := newTask(t, r, api.TaskCreate{ProjectId: from.Id, Title: "Child", ParentId: &parent.Id})
	stay := newTask(t, r, api.TaskCreate{ProjectId: from.Id, Title: "Stays"})

	var moved api.Task
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks/"+parent.Id.String()+"/transfer", map[string]any{"projectId": to.Id}, &moved); s != http.StatusOK {
		t.Fatalf("transfer = %d", s)
	}
	if moved.ProjectId != to.Id || moved.Number != 2 || moved.SprintId != nil || len(moved.Tags) != 1 || len(moved.CustomValues) != 0 {
		t.Errorf("moved = %+v", moved)
	}
	var c api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+child.Id.String(), nil, &c)
	if c.ProjectId != to.Id || c.ParentId == nil || *c.ParentId != parent.Id || c.Number != 3 {
		t.Errorf("child = %+v", c)
	}
	var s api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+stay.Id.String(), nil, &s)
	if s.ProjectId != from.Id {
		t.Errorf("unrelated item moved")
	}
	// Moving a child on its own leaves its parent behind (and unlinks it).
	doJSON(t, r, http.MethodPost, "/api/v1/tasks/"+child.Id.String()+"/transfer", map[string]any{"projectId": from.Id}, &c)
	if c.ParentId != nil || c.ProjectId != from.Id {
		t.Errorf("child moved back = %+v", c)
	}
	for name, body := range map[string]map[string]any{
		"same project":    {"projectId": from.Id},
		"unknown project": {"projectId": uuid.New()},
	} {
		if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks/"+child.Id.String()+"/transfer", body, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}

func TestItemTemplates(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "IT", Name: "Templates"}, &p)
	bug := api.TaskTypeBug
	high := api.TaskPriorityHigh
	in := api.ItemTemplateInput{ProjectId: p.Id, Name: "Bug report", Fields: api.ItemTemplateFields{
		Type: &bug, Priority: &high, Title: ptrTo("[Bug] "), Description: ptrTo("## Steps\n1.\n\n## Expected\n"), Tags: &[]string{"triage"},
	}}
	var tpl api.ItemTemplate
	if s := doJSON(t, r, http.MethodPost, "/api/v1/item-templates", in, &tpl); s != http.StatusCreated || tpl.Fields.Type == nil || *tpl.Fields.Type != bug {
		t.Fatalf("create = %d %+v", s, tpl)
	}
	var list api.ItemTemplateList
	doJSON(t, r, http.MethodGet, "/api/v1/item-templates?projectId="+p.Id.String(), nil, &list)
	if len(list.Data) != 1 || list.Data[0].Name != "Bug report" {
		t.Errorf("list = %+v", list)
	}
	in.Name = "Bug"
	doJSON(t, r, http.MethodPatch, "/api/v1/item-templates/"+tpl.Id.String(), in, &tpl)
	if tpl.Name != "Bug" {
		t.Errorf("rename = %+v", tpl)
	}
	for name, bad := range map[string]api.ItemTemplateInput{
		"no name":      {ProjectId: p.Id, Name: " ", Fields: api.ItemTemplateFields{}},
		"no project":   {ProjectId: uuid.New(), Name: "x", Fields: api.ItemTemplateFields{}},
		"bad type":     {ProjectId: p.Id, Name: "x", Fields: api.ItemTemplateFields{Type: ptrTo(api.TaskType("saga"))}},
		"bad priority": {ProjectId: p.Id, Name: "x", Fields: api.ItemTemplateFields{Priority: ptrTo(api.TaskPriority("meh"))}},
	} {
		if s := doJSON(t, r, http.MethodPost, "/api/v1/item-templates", bad, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
	if s := doJSON(t, r, http.MethodDelete, "/api/v1/item-templates/"+tpl.Id.String(), nil, nil); s != http.StatusNoContent {
		t.Errorf("delete = %d", s)
	}
}
