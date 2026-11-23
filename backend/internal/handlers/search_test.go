package handlers

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestSearch(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "SR", Name: "Searchable"}, &p)
	a := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Checkout flow"})
	b := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Other", Description: ptrTo("The payment gateway times out on checkout")})
	c := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Third"})
	doJSON(t, r, http.MethodPost, "/api/v1/tasks/"+c.Id.String()+"/comments", map[string]any{"body": "Seen with the new payment gateway, 100%_sure"}, nil)

	find := func(q string) api.SearchResults {
		t.Helper()
		var res api.SearchResults
		if s := doJSON(t, r, http.MethodGet, "/api/v1/search?q="+url.QueryEscape(q), nil, &res); s != http.StatusOK {
			t.Fatalf("search %q = %d", q, s)
		}
		return res
	}
	res := find("checkout")
	if len(res.Items) != 2 || res.Items[0].Id != a.Id || res.Items[0].MatchedIn != api.SearchItemMatchedInTitle || res.Items[1].Id != b.Id || res.Items[1].Snippet == nil {
		t.Errorf("checkout = %+v", res.Items)
	}
	res = find("payment GATEWAY")
	if len(res.Items) != 2 {
		t.Errorf("payment gateway = %+v", res.Items)
	}
	var viaComment *api.SearchItem
	for i := range res.Items {
		if res.Items[i].Id == c.Id {
			viaComment = &res.Items[i]
		}
	}
	if viaComment == nil || viaComment.MatchedIn != api.SearchItemMatchedInComment {
		t.Errorf("comment match = %+v", res.Items)
	}
	if res := find("100%_"); len(res.Items) != 1 {
		t.Errorf("literal %% and _ = %+v", res.Items)
	}
	if res := find("SR-2"); len(res.Items) != 1 || res.Items[0].Id != b.Id || res.Items[0].MatchedIn != api.SearchItemMatchedInNumber {
		t.Errorf("by key = %+v", res.Items)
	}
	if res := find("searchable"); len(res.Projects) != 1 {
		t.Errorf("projects = %+v", res.Projects)
	}
	if s := doJSON(t, r, http.MethodGet, "/api/v1/search?q=", nil, nil); s != http.StatusBadRequest {
		t.Errorf("empty = %d", s)
	}
}
