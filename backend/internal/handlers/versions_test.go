package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func testDate(y int, m time.Month, d int) *openapi_types.Date {
	return &openapi_types.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

func mustCreateVersion(t *testing.T, r *gin.Engine, projectID uuid.UUID, name string, target *openapi_types.Date) api.Version {
	t.Helper()
	var v api.Version
	if status := doJSON(t, r, http.MethodPost, "/api/v1/versions",
		api.VersionCreate{ProjectId: projectID, Name: name, TargetDate: target}, &v); status != http.StatusCreated {
		t.Fatalf("create version %q: status = %d, want 201", name, status)
	}
	return v
}

func TestVersions_CRUD(t *testing.T) {
	r, _ := newTestRouter(t)
	pid := uuid.New()

	created := mustCreateVersion(t, r, pid, "v1.0", testDate(2026, 12, 15))
	if created.Status != api.VersionStatusPlanned {
		t.Errorf("Status = %q, want planned", created.Status)
	}

	var got api.Version
	if status := doJSON(t, r, http.MethodGet, "/api/v1/versions/"+created.Id.String(), nil, &got); status != http.StatusOK || got.Name != "v1.0" {
		t.Fatalf("get: status=%d name=%q", status, got.Name)
	}

	name := "v1.0.0"
	var updated api.Version
	if status := doJSON(t, r, http.MethodPatch, "/api/v1/versions/"+created.Id.String(),
		api.VersionUpdate{Name: &name, TargetDate: testDate(2027, 1, 10)}, &updated); status != http.StatusOK {
		t.Fatalf("patch status = %d", status)
	}
	if updated.Name != "v1.0.0" || updated.TargetDate == nil || updated.TargetDate.String() != "2027-01-10" {
		t.Errorf("patch not applied: %+v", updated)
	}

	if status := doJSON(t, r, http.MethodDelete, "/api/v1/versions/"+created.Id.String(), nil, nil); status != http.StatusNoContent {
		t.Fatalf("delete status = %d", status)
	}
	if status := doJSON(t, r, http.MethodGet, "/api/v1/versions/"+created.Id.String(), nil, nil); status != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", status)
	}
	if status := doJSON(t, r, http.MethodDelete, "/api/v1/versions/"+created.Id.String(), nil, nil); status != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", status)
	}
}

func TestVersions_List_FilterAndSort(t *testing.T) {
	r, _ := newTestRouter(t)
	a, b := uuid.New(), uuid.New()
	mustCreateVersion(t, r, a, "v2", testDate(2027, 1, 1))
	mustCreateVersion(t, r, a, "vX", nil)
	mustCreateVersion(t, r, a, "v1", testDate(2026, 12, 1))
	mustCreateVersion(t, r, b, "other", testDate(2026, 1, 1))

	var list api.VersionList
	if status := doJSON(t, r, http.MethodGet, "/api/v1/versions?projectId="+a.String(), nil, &list); status != http.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	var names []string
	for _, v := range list.Data {
		names = append(names, v.Name)
	}
	want := []string{"v1", "v2", "vX"}
	if len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Errorf("names = %v, want %v", names, want)
	}
	if list.Pagination.Total != 3 {
		t.Errorf("total = %d, want 3", list.Pagination.Total)
	}
}

func TestVersions_DuplicateNameReturns409(t *testing.T) {
	r, _ := newTestRouter(t)
	a, b := uuid.New(), uuid.New()
	mustCreateVersion(t, r, a, "v1.0", nil)
	other := mustCreateVersion(t, r, a, "v1.1", nil)

	var errBody api.Error
	if status := doJSON(t, r, http.MethodPost, "/api/v1/versions", api.VersionCreate{ProjectId: a, Name: "v1.0"}, &errBody); status != http.StatusConflict {
		t.Errorf("duplicate create = %d, want 409", status)
	}
	name := "v1.0"
	if status := doJSON(t, r, http.MethodPatch, "/api/v1/versions/"+other.Id.String(), api.VersionUpdate{Name: &name}, nil); status != http.StatusConflict {
		t.Errorf("rename to taken name = %d, want 409", status)
	}
	mustCreateVersion(t, r, b, "v1.0", nil) // same name, other project: fine
}

func TestVersions_RenameToOwnNameIsOK(t *testing.T) {
	r, _ := newTestRouter(t)
	v := mustCreateVersion(t, r, uuid.New(), "v1.0", nil)
	name := "v1.0"
	if status := doJSON(t, r, http.MethodPatch, "/api/v1/versions/"+v.Id.String(), api.VersionUpdate{Name: &name}, nil); status != http.StatusOK {
		t.Errorf("rename to own name = %d, want 200", status)
	}
}

func TestVersions_DeleteUnassignsMilestones(t *testing.T) {
	r, db := newTestRouter(t)
	pid := uuid.New()
	v := mustCreateVersion(t, r, pid, "v1.0", nil)
	if err := db.Exec("INSERT INTO milestones (id, project_id, name, version_id) VALUES (?, ?, 'M1', ?)",
		"m-1", pid.String(), v.Id.String()).Error; err != nil {
		t.Fatal(err)
	}

	if status := doJSON(t, r, http.MethodDelete, "/api/v1/versions/"+v.Id.String(), nil, nil); status != http.StatusNoContent {
		t.Fatalf("delete status = %d", status)
	}
	var versionID *string
	if err := db.Raw("SELECT version_id FROM milestones WHERE id = 'm-1'").Scan(&versionID).Error; err != nil {
		t.Fatal(err)
	}
	if versionID != nil {
		t.Errorf("milestone version_id = %q, want NULL after its version was deleted", *versionID)
	}
}
