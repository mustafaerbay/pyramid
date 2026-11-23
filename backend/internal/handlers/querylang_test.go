package handlers

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestQueries_Where(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "Q", Name: "Query"}, &p)
	doJSON(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"customFields": []map[string]any{
		{"key": "risk", "label": "Risk", "type": "select", "options": []string{"Low", "High"}, "required": false},
		{"key": "budget", "label": "Budget", "type": "number", "required": false},
	}}, nil)
	alice := uuid.New()
	bug, story := api.TaskTypeBug, api.TaskTypeStory
	urgent := api.TaskPriorityUrgent
	pts := 5.0
	parent := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Checkout epic", Type: &story, StoryPoints: &pts})
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Card fails", Type: &bug, Priority: &urgent, Tags: tagsPtr("payments"), AssigneeIds: &[]uuid.UUID{alice},
		ParentId: &parent.Id, CustomValues: &map[string]any{"risk": "High", "budget": 1200}})
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Copy tweak", ParentId: &parent.Id, CustomValues: &map[string]any{"risk": "Low", "budget": 50}})
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Old bug", Type: &bug, DueDate: mdDate(yesterday)})

	run := func(f api.QueryFilters) ([]string, api.QueryResult) {
		t.Helper()
		f.ProjectIds = &[]uuid.UUID{p.Id}
		var res api.QueryResult
		if s := doJSON(t, r, http.MethodPost, "/api/v1/queries/run", f, &res); s != http.StatusOK {
			t.Fatalf("run = %d", s)
		}
		out := []string{}
		for _, x := range res.Data {
			out = append(out, x.Title)
		}
		return out, res
	}
	cond := func(field string, op api.QueryConditionOp, value string) api.QueryCondition {
		c := api.QueryCondition{Field: field, Op: op}
		if value != "" {
			c.Value = &value
		}
		return c
	}
	all := func(c ...api.QueryCondition) *api.QueryGroup {
		return &api.QueryGroup{Match: api.QueryGroupMatchAll, Conditions: c}
	}
	same := func(name string, got []string, want ...string) {
		t.Helper()
		g := append([]string{}, got...)
		sort.Strings(g)
		sort.Strings(want)
		if len(g) != len(want) {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
		for i := range g {
			if g[i] != want[i] {
				t.Errorf("%s = %v, want %v", name, got, want)
				return
			}
		}
	}

	got, _ := run(api.QueryFilters{Where: &api.QueryGroup{Match: api.QueryGroupMatchAny, Conditions: []api.QueryCondition{
		cond("priority", api.QueryConditionOpEq, "urgent"), cond("title", api.QueryConditionOpContains, "copy"),
	}}})
	same("OR", got, "Card fails", "Copy tweak")

	got, _ = run(api.QueryFilters{Where: &api.QueryGroup{Match: api.QueryGroupMatchAll, Conditions: []api.QueryCondition{cond("type", api.QueryConditionOpEq, "bug")},
		Groups: &[]api.QueryGroup{{Match: api.QueryGroupMatchAny, Conditions: []api.QueryCondition{
			cond("tag", api.QueryConditionOpEq, "PAYMENTS"), cond("dueDate", api.QueryConditionOpLt, "@today"),
		}}}}})
	same("AND with nested OR", got, "Card fails", "Old bug")

	got, _ = run(api.QueryFilters{Where: all(cond("custom:risk", api.QueryConditionOpEq, "high"))})
	same("custom select", got, "Card fails")
	got, _ = run(api.QueryFilters{Where: all(cond("custom:budget", api.QueryConditionOpGte, "100"))})
	same("custom number", got, "Card fails")
	got, _ = run(api.QueryFilters{Where: all(cond("custom:budget", api.QueryConditionOpEmpty, ""))})
	same("custom empty", got, "Checkout epic", "Old bug")
	got, _ = run(api.QueryFilters{Where: all(cond("assignee", api.QueryConditionOpEmpty, ""), cond("storyPoints", api.QueryConditionOpGt, "1"))})
	same("assignee empty and points", got, "Checkout epic")
	got, _ = run(api.QueryFilters{Where: all(cond("parent", api.QueryConditionOpEq, parent.Id.String()), cond("assignee", api.QueryConditionOpNe, alice.String()))})
	same("parent and not assigned", got, "Copy tweak")

	// Tree: matches with their children after them.
	got, res := run(api.QueryFilters{Where: all(cond("title", api.QueryConditionOpContains, "epic")), Tree: ptrTo(true)})
	if len(got) != 3 || got[0] != "Checkout epic" || res.Total != 1 || res.ChildIds == nil || len(*res.ChildIds) != 2 {
		t.Errorf("tree = %v total %d children %v", got, res.Total, res.ChildIds)
	}
	got, _ = run(api.QueryFilters{Where: all(cond("title", api.QueryConditionOpContains, "epic")), Tree: ptrTo(true),
		ChildWhere: all(cond("type", api.QueryConditionOpEq, "bug"))})
	same("tree with child filter", got, "Checkout epic", "Card fails")

	nest := func(depth int) *api.QueryGroup {
		g := all(cond("title", api.QueryConditionOpEq, "x"))
		for i := 1; i < depth; i++ {
			g = &api.QueryGroup{Match: api.QueryGroupMatchAll, Conditions: []api.QueryCondition{}, Groups: &[]api.QueryGroup{*g}}
		}
		return g
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/queries/run", api.QueryFilters{Where: nest(3), ProjectIds: &[]uuid.UUID{p.Id}}, nil); s != http.StatusOK {
		t.Errorf("3 deep = %d", s)
	}
	for name, w := range map[string]*api.QueryGroup{
		"unknown field":  all(cond("mood", api.QueryConditionOpEq, "x")),
		"unknown custom": all(cond("custom:nope", api.QueryConditionOpEq, "x")),
		"bad op":         all(api.QueryCondition{Field: "title", Op: "like"}),
		"bad date":       all(cond("dueDate", api.QueryConditionOpLt, "soon")),
		"bad number":     all(cond("storyPoints", api.QueryConditionOpLt, "many")),
		"too deep":       nest(4),
	} {
		f := api.QueryFilters{Where: w, ProjectIds: &[]uuid.UUID{p.Id}}
		if s := doJSON(t, r, http.MethodPost, "/api/v1/queries/run", f, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}
