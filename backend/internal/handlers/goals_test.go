package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestGoals_List_FilterByOwner(t *testing.T) {
	r, _ := newTestRouter(t)

	var ada, bob api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Ada", LastName: "L", Email: "ada@example.com"}, &ada)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Bob", LastName: "B", Email: "bob@example.com"}, &bob)

	for _, g := range []api.GoalCreate{
		{Title: "Ada one", OwnerPersonId: &ada.Id},
		{Title: "Ada two", OwnerPersonId: &ada.Id},
		{Title: "Bob one", OwnerPersonId: &bob.Id},
		{Title: "Nobody"},
	} {
		if s := doJSON(t, r, http.MethodPost, "/api/v1/goals", g, nil); s != http.StatusCreated {
			t.Fatalf("create %q status = %d", g.Title, s)
		}
	}

	var list api.GoalList
	if s := doJSON(t, r, http.MethodGet, "/api/v1/goals?ownerPersonId="+ada.Id.String(), nil, &list); s != http.StatusOK {
		t.Fatalf("list status = %d", s)
	}
	if list.Pagination.Total != 2 || len(list.Data) != 2 {
		t.Fatalf("got %d goals (total %d), want Ada's 2", len(list.Data), list.Pagination.Total)
	}
	for _, g := range list.Data {
		if g.OwnerPersonId == nil || *g.OwnerPersonId != ada.Id {
			t.Errorf("goal %q owner = %v, want Ada", g.Title, g.OwnerPersonId)
		}
	}

	doJSON(t, r, http.MethodGet, "/api/v1/goals", nil, &list)
	if list.Pagination.Total != 4 {
		t.Errorf("unfiltered total = %d, want 4", list.Pagination.Total)
	}
}
