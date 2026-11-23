package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestWiki(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "WK", Name: "Wiki"}, &p)
	var home, setup, deep api.WikiPage
	if s := doJSON(t, r, http.MethodPost, "/api/v1/wiki-pages", api.WikiPageInput{ProjectId: p.Id, Title: "Home", Body: ptrTo("# Welcome")}, &home); s != http.StatusCreated {
		t.Fatalf("create = %d", s)
	}
	doJSON(t, r, http.MethodPost, "/api/v1/wiki-pages", api.WikiPageInput{ProjectId: p.Id, Title: "Setup", ParentId: &home.Id}, &setup)
	doJSON(t, r, http.MethodPost, "/api/v1/wiki-pages", api.WikiPageInput{ProjectId: p.Id, Title: "Deep", ParentId: &setup.Id}, &deep)

	var list api.WikiPageList
	doJSON(t, r, http.MethodGet, "/api/v1/wiki-pages?projectId="+p.Id.String(), nil, &list)
	if len(list.Data) != 3 || list.Data[0].Body != nil {
		t.Errorf("list = %+v", list.Data)
	}
	path := "/api/v1/wiki-pages/" + home.Id.String()
	var got api.WikiPage
	doJSON(t, r, http.MethodGet, path, nil, &got)
	if got.Body == nil || *got.Body != "# Welcome" {
		t.Errorf("get = %+v", got)
	}
	base := got.UpdatedAt
	time.Sleep(5 * time.Millisecond)
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"body": "# Welcome v2", "baseUpdatedAt": base}, &got); s != http.StatusOK || *got.Body != "# Welcome v2" {
		t.Fatalf("edit = %d %+v", s, got)
	}
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"body": "stale", "baseUpdatedAt": base}, nil); s != http.StatusConflict {
		t.Errorf("stale edit = %d, want 409", s)
	}
	var revs api.WikiRevisionList
	doJSON(t, r, http.MethodGet, path+"/revisions", nil, &revs)
	if len(revs.Data) != 1 || revs.Data[0].Body != "# Welcome" {
		t.Errorf("revisions = %+v", revs.Data)
	}
	// No cycles: Home can't go under its own grandchild.
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"parentId": deep.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("cycle = %d", s)
	}
	// Deleting Setup moves Deep up to Home.
	doJSON(t, r, http.MethodDelete, "/api/v1/wiki-pages/"+setup.Id.String(), nil, nil)
	doJSON(t, r, http.MethodGet, "/api/v1/wiki-pages/"+deep.Id.String(), nil, &got)
	if got.ParentId == nil || *got.ParentId != home.Id {
		t.Errorf("deep after delete = %+v", got)
	}
	for name, in := range map[string]api.WikiPageInput{
		"no title":     {ProjectId: p.Id, Title: " "},
		"other parent": {ProjectId: p.Id, Title: "x", ParentId: ptrTo(p.Id)},
	} {
		if s := doJSON(t, r, http.MethodPost, "/api/v1/wiki-pages", in, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d", name, s)
		}
	}
}
