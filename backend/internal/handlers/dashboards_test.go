package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestDashboards(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")

	var q, bobQ api.SavedQuery
	call(t, r, http.MethodPost, "/api/v1/queries", api.SavedQueryInput{Name: "Bugs", Filters: api.QueryFilters{}}, amy, &q)
	call(t, r, http.MethodPost, "/api/v1/queries", api.SavedQueryInput{Name: "Private", Filters: api.QueryFilters{}}, bob, &bobQ)
	pie := api.DashboardWidgetChartPie
	widgets := []api.DashboardWidget{
		{Id: "w1", Kind: api.DashboardWidgetKindQuery, Size: api.DashboardWidgetSizeSmall, QueryId: &q.Id, Chart: &pie, GroupBy: ptrTo(api.QueryGroupByStatus)},
		{Id: "w2", Kind: api.DashboardWidgetKindNotes, Size: api.DashboardWidgetSizeWide, Text: ptrTo("**Ship it**")},
	}
	var d api.Dashboard
	if s, _ := call(t, r, http.MethodPost, "/api/v1/dashboards", api.DashboardInput{Name: "Team", Widgets: &widgets}, amy, &d); s != http.StatusCreated {
		t.Fatalf("create = %d", s)
	}
	if !d.Mine || d.Shared || len(d.Widgets) != 2 || d.Widgets[0].QueryId == nil {
		t.Errorf("created = %+v", d)
	}
	path := "/api/v1/dashboards/" + d.Id.String()
	if s, _ := call(t, r, http.MethodGet, path, nil, bob, nil); s != http.StatusNotFound {
		t.Errorf("bob sees a private dashboard: %d", s)
	}
	call(t, r, http.MethodPatch, path, api.DashboardInput{Name: "Team", Shared: ptrTo(true), Widgets: &widgets}, amy, &d)
	var list api.DashboardList
	call(t, r, http.MethodGet, "/api/v1/dashboards", nil, bob, &list)
	if len(list.Data) != 1 || list.Data[0].Mine {
		t.Errorf("bob's list = %+v", list.Data)
	}
	if s, _ := call(t, r, http.MethodPatch, path, api.DashboardInput{Name: "Mine now"}, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob edits = %d, want 403", s)
	}
	// Patching without widgets keeps them.
	call(t, r, http.MethodPatch, path, api.DashboardInput{Name: "Renamed"}, amy, &d)
	if d.Name != "Renamed" || len(d.Widgets) != 2 {
		t.Errorf("rename = %+v", d)
	}

	nope := uuid.New()
	for name, w := range map[string]api.DashboardWidget{
		"query without id":    {Id: "x", Kind: api.DashboardWidgetKindQuery, Size: api.DashboardWidgetSizeSmall},
		"unknown query":       {Id: "x", Kind: api.DashboardWidgetKindQuery, Size: api.DashboardWidgetSizeSmall, QueryId: &nope},
		"someone's private":   {Id: "x", Kind: api.DashboardWidgetKindQuery, Size: api.DashboardWidgetSizeSmall, QueryId: &bobQ.Id},
		"burndown no project": {Id: "x", Kind: api.DashboardWidgetKindBurndown, Size: api.DashboardWidgetSizeSmall},
		"bad kind":            {Id: "x", Kind: "weather", Size: api.DashboardWidgetSizeSmall},
		"bad size":            {Id: "x", Kind: api.DashboardWidgetKindNotes, Size: "huge"},
		"no id":               {Id: "", Kind: api.DashboardWidgetKindNotes, Size: api.DashboardWidgetSizeSmall},
	} {
		ws := []api.DashboardWidget{w}
		if s, _ := call(t, r, http.MethodPost, "/api/v1/dashboards", api.DashboardInput{Name: "Bad", Widgets: &ws}, amy, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/dashboards", api.DashboardInput{Name: " "}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("blank name = %d", s)
	}
	if s, _ := call(t, r, http.MethodDelete, path, nil, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob deletes = %d", s)
	}
	if s, _ := call(t, r, http.MethodDelete, path, nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("admin deletes = %d", s)
	}
}
