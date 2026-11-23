package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestAreas(t *testing.T) {
	r, _ := newTestRouter(t)
	p := uuid.New()
	mk := func(path string) (api.Area, int) {
		var a api.Area
		s := doJSON(t, r, http.MethodPost, "/api/v1/areas", api.AreaCreate{ProjectId: p, Path: path}, &a)
		return a, s
	}
	web, _ := mk(" Web / Checkout ")
	if web.Path != "Web/Checkout" {
		t.Errorf("path = %q, want trimmed Web/Checkout", web.Path)
	}
	webRoot, _ := mk("Web")
	mk("Web/Checkout/Payments")
	api1, _ := mk("API")
	for name, path := range map[string]string{"duplicate": "web/checkout", "empty level": "Web//X", "blank": "  ", "too deep": "A/B/C/D/E"} {
		if _, s := mk(path); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
	var list api.AreaList
	doJSON(t, r, http.MethodGet, "/api/v1/areas?projectId="+p.String(), nil, &list)
	got := []string{}
	for _, a := range list.Data {
		got = append(got, a.Path)
	}
	if len(got) != 4 || got[0] != "API" || got[1] != "Web" || got[2] != "Web/Checkout" || got[3] != "Web/Checkout/Payments" {
		t.Errorf("list = %v", got)
	}

	// Items in Web/Checkout and API; filtering by Web includes sub-areas.
	a := newTask(t, r, api.TaskCreate{ProjectId: p, Title: "A", AreaId: &web.Id})
	newTask(t, r, api.TaskCreate{ProjectId: p, Title: "B", AreaId: &api1.Id})
	var tl api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?areaId="+webRoot.Id.String(), nil, &tl)
	if len(tl.Data) != 1 || tl.Data[0].Title != "A" {
		t.Errorf("tasks under Web = %v", tl.Data)
	}
	other := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "X"})
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+other.Id.String(), map[string]any{"areaId": web.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("area of another project = %d, want 400", s)
	}

	// Renaming Web renames its sub-areas too.
	var renamed api.Area
	doJSON(t, r, http.MethodPatch, "/api/v1/areas/"+webRoot.Id.String(), api.AreaUpdate{Path: "Frontend"}, &renamed)
	doJSON(t, r, http.MethodGet, "/api/v1/areas?projectId="+p.String(), nil, &list)
	got = got[:0]
	for _, x := range list.Data {
		got = append(got, x.Path)
	}
	if len(got) != 4 || got[1] != "Frontend" || got[2] != "Frontend/Checkout" || got[3] != "Frontend/Checkout/Payments" {
		t.Errorf("after rename = %v", got)
	}

	// Can't delete an area with sub-areas; deleting a leaf clears its items.
	if s := doJSON(t, r, http.MethodDelete, "/api/v1/areas/"+webRoot.Id.String(), nil, nil); s != http.StatusBadRequest {
		t.Errorf("delete with sub-areas = %d, want 400", s)
	}
	var payments api.Area
	for _, x := range list.Data {
		if x.Path == "Frontend/Checkout/Payments" {
			payments = x
		}
	}
	doJSON(t, r, http.MethodDelete, "/api/v1/areas/"+payments.Id.String(), nil, nil)
	if s := doJSON(t, r, http.MethodDelete, "/api/v1/areas/"+web.Id.String(), nil, nil); s != http.StatusNoContent {
		t.Errorf("delete leaf = %d", s)
	}
	var back api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+a.Id.String(), nil, &back)
	if back.AreaId != nil {
		t.Errorf("A area after delete = %v", back.AreaId)
	}
}
