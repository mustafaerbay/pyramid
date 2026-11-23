package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func fakeArti(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/versions/Shop/App", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"package": "Shop/App", "version": "v1.2.0", "created": "2026-11-20T10:00:00Z", "has_notes": true, "locked": true,
				"files": []map[string]any{{"name": "app.msi", "size": 1234, "sha256": "ab"}}},
			{"package": "Shop/App", "version": "v1.1.0", "created": "2026-11-01T10:00:00Z", "has_notes": false, "files": []map[string]any{}},
		})
	})
	mux.HandleFunc("GET /api/v1/changelog/Shop/App/v1.2.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("## Added\n- Login"))
	})
	return httptest.NewServer(mux)
}

func TestArtifacts(t *testing.T) {
	arti := fakeArti(t)
	defer arti.Close()
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "AR", Name: "Art"}, &p)
	var v api.Version
	doJSON(t, r, http.MethodPost, "/api/v1/versions", api.VersionCreate{ProjectId: p.Id, Name: "Release 1"}, &v)
	var m api.Milestone
	doJSON(t, r, http.MethodPost, "/api/v1/milestones", api.MilestoneCreate{ProjectId: p.Id, Name: "M1", VersionId: &v.Id}, &m)
	done := api.TaskStatusDone
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Login shipped", MilestoneId: &m.Id, Status: &done})
	newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Not done", MilestoneId: &m.Id})

	var pkg api.ProjectPackage
	if s := doJSON(t, r, http.MethodPost, "/api/v1/project-packages", api.ProjectPackageInput{ProjectId: p.Id, ServerUrl: ptrTo(arti.URL + "/"), Package: "Shop/App"}, &pkg); s != http.StatusCreated {
		t.Fatalf("link = %d", s)
	}
	if pkg.ServerUrl != arti.URL || pkg.UpdateUrl != arti.URL+"/api/v1/packages/Shop/App" {
		t.Errorf("pkg = %+v", pkg)
	}
	path := "/api/v1/project-packages/" + pkg.Id.String()
	if s := doJSON(t, r, http.MethodPut, path+"/links", map[string]any{"artifactVersion": "v1.2.0", "versionId": v.Id}, nil); s != http.StatusNoContent {
		t.Fatalf("link version = %d", s)
	}
	var list api.ArtifactVersionList
	if s := doJSON(t, r, http.MethodGet, path+"/versions", nil, &list); s != http.StatusOK {
		t.Fatalf("versions = %d", s)
	}
	if len(list.Data) != 2 {
		t.Fatalf("versions = %+v", list.Data)
	}
	top := list.Data[0]
	if top.Version != "v1.2.0" || top.Notes != "## Added\n- Login" || !top.Locked || top.VersionName == nil || *top.VersionName != "Release 1" ||
		len(top.Items) != 1 || top.Items[0].Title != "Login shipped" || len(top.Files) != 1 || top.Files[0].Url != arti.URL+"/api/v1/packages/Shop/App/v1.2.0/app.msi" {
		t.Errorf("v1.2.0 = %+v", top)
	}
	if list.Data[1].VersionId != nil || list.Data[1].Notes != "" {
		t.Errorf("v1.1.0 = %+v", list.Data[1])
	}
	doJSON(t, r, http.MethodPut, path+"/links", map[string]any{"artifactVersion": "v1.2.0", "versionId": nil}, nil)
	doJSON(t, r, http.MethodGet, path+"/versions", nil, &list)
	if list.Data[0].VersionId != nil {
		t.Errorf("unlink kept %+v", list.Data[0])
	}

	for name, in := range map[string]api.ProjectPackageInput{
		"bad name":   {ProjectId: p.Id, ServerUrl: ptrTo(arti.URL), Package: "../etc"},
		"bad server": {ProjectId: p.Id, ServerUrl: ptrTo("ftp://x"), Package: "A"},
		"no server":  {ProjectId: p.Id, Package: "A"},
	} {
		if s := doJSON(t, r, http.MethodPost, "/api/v1/project-packages", in, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d", name, s)
		}
	}
	// arti down → 502.
	var dead api.ProjectPackage
	doJSON(t, r, http.MethodPost, "/api/v1/project-packages", api.ProjectPackageInput{ProjectId: p.Id, ServerUrl: ptrTo("http://127.0.0.1:1"), Package: "X"}, &dead)
	if s := doJSON(t, r, http.MethodGet, "/api/v1/project-packages/"+dead.Id.String()+"/versions", nil, nil); s != http.StatusBadGateway {
		t.Errorf("dead arti = %d", s)
	}
	var pl api.ProjectPackageList
	doJSON(t, r, http.MethodGet, "/api/v1/project-packages?projectId="+p.Id.String(), nil, &pl)
	if len(pl.Data) != 2 {
		t.Errorf("list = %+v", pl)
	}
	if s := doJSON(t, r, http.MethodDelete, path, nil, nil); s != http.StatusNoContent {
		t.Errorf("unlink = %d", s)
	}
}
