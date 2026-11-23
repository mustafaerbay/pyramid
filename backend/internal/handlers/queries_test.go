package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func ptrTo[T any](v T) *T { return &v }

func TestQueries_Run(t *testing.T) {
	r, _ := newTestRouter(t)
	p1, p2 := uuid.New(), uuid.New()
	alice := uuid.New()
	var ar api.Area
	doJSON(t, r, http.MethodPost, "/api/v1/areas", api.AreaCreate{ProjectId: p1, Path: "Web"}, &ar)
	var sub api.Area
	doJSON(t, r, http.MethodPost, "/api/v1/areas", api.AreaCreate{ProjectId: p1, Path: "Web/Checkout"}, &sub)
	bug := api.TaskTypeBug
	newTask(t, r, api.TaskCreate{ProjectId: p1, Title: "Login bug", Type: &bug, Tags: tagsPtr("auth"), AssigneeIds: &[]uuid.UUID{alice}, AreaId: &sub.Id, DueDate: mdDate("2026-12-05")})
	newTask(t, r, api.TaskCreate{ProjectId: p1, Title: "Login page", Tags: tagsPtr("ui"), DueDate: mdDate("2026-12-20")})
	newTask(t, r, api.TaskCreate{ProjectId: p2, Title: "Billing export", Type: &bug})

	run := func(f api.QueryFilters) []string {
		t.Helper()
		var res api.QueryResult
		if s := doJSON(t, r, http.MethodPost, "/api/v1/queries/run", f, &res); s != http.StatusOK {
			t.Fatalf("run = %d", s)
		}
		out := []string{}
		for _, x := range res.Data {
			out = append(out, x.Title)
		}
		if res.Total != len(out) || res.Truncated {
			t.Errorf("total %d truncated %v for %d rows", res.Total, res.Truncated, len(out))
		}
		return out
	}
	eq := func(name string, got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s = %v, want %v", name, got, want)
				return
			}
		}
	}
	if all := run(api.QueryFilters{}); len(all) != 3 {
		t.Errorf("everything = %v", all)
	}
	if bugs := run(api.QueryFilters{Types: &[]api.TaskType{api.TaskTypeBug}}); len(bugs) != 2 {
		t.Errorf("bugs = %v", bugs)
	}
	eq("project", run(api.QueryFilters{ProjectIds: &[]uuid.UUID{p2}}), "Billing export")
	eq("text", run(api.QueryFilters{Text: ptrTo("LOGIN")}), "Login bug", "Login page")
	eq("tag", run(api.QueryFilters{Tags: &[]string{"UI"}}), "Login page")
	eq("assignee", run(api.QueryFilters{AssigneeIds: &[]uuid.UUID{alice}}), "Login bug")
	eq("unassigned", run(api.QueryFilters{Unassigned: ptrTo(true), ProjectIds: &[]uuid.UUID{p1}}), "Login page")
	eq("area with sub-areas", run(api.QueryFilters{AreaIds: &[]uuid.UUID{ar.Id}}), "Login bug")
	eq("due range", run(api.QueryFilters{DueAfter: mdDate("2026-12-10"), DueBefore: mdDate("2026-12-31")}), "Login page")
	eq("combined", run(api.QueryFilters{Types: &[]api.TaskType{api.TaskTypeBug}, Text: ptrTo("billing")}), "Billing export")
}

func TestQueries_Saved(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")

	var mine, shared api.SavedQuery
	if s, _ := call(t, r, http.MethodPost, "/api/v1/queries", api.SavedQueryInput{Name: "My bugs", Filters: api.QueryFilters{Types: &[]api.TaskType{api.TaskTypeBug}}}, amy, &mine); s != http.StatusCreated {
		t.Fatalf("save = %d", s)
	}
	call(t, r, http.MethodPost, "/api/v1/queries", api.SavedQueryInput{Name: "Team view", Filters: api.QueryFilters{}, Shared: ptrTo(true), GroupBy: ptrTo(api.QueryGroupByAssignee)}, amy, &shared)
	if mine.GroupBy != api.QueryGroupByStatus || !mine.Mine || mine.Filters.Types == nil {
		t.Errorf("saved = %+v", mine)
	}

	var list api.SavedQueryList
	call(t, r, http.MethodGet, "/api/v1/queries", nil, bob, &list)
	if len(list.Data) != 1 || list.Data[0].Name != "Team view" || list.Data[0].Mine {
		t.Errorf("bob sees %+v, want only the shared one", list.Data)
	}
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/queries/"+shared.Id.String(), api.SavedQueryInput{Name: "Hijack", Filters: api.QueryFilters{}}, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob edits = %d, want 403", s)
	}
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/queries/"+mine.Id.String(), api.SavedQueryInput{Name: "x", Filters: api.QueryFilters{}}, bob, nil); s != http.StatusNotFound {
		t.Errorf("bob edits a private one = %d, want 404", s)
	}
	var upd api.SavedQuery
	call(t, r, http.MethodPatch, "/api/v1/queries/"+mine.Id.String(), api.SavedQueryInput{Name: "Open bugs", Filters: api.QueryFilters{}}, amy, &upd)
	if upd.Name != "Open bugs" {
		t.Errorf("rename = %+v", upd)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/queries", api.SavedQueryInput{Name: " ", Filters: api.QueryFilters{}}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("blank name = %d", s)
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/queries/"+shared.Id.String(), nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("admin deletes shared = %d", s)
	}
	call(t, r, http.MethodGet, "/api/v1/queries", nil, amy, &list)
	if len(list.Data) != 1 {
		t.Errorf("amy has %d left, want 1", len(list.Data))
	}
}
