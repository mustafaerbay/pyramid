package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// patchRaw sends a literal JSON body, so tests can send an explicit
// `"versionId": null` (a Go struct would drop or can't express it).
func patchRaw(t *testing.T, r *gin.Engine, path, body string, out any) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if out != nil && w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("unmarshal %q: %v", w.Body.String(), err)
		}
	}
	return w.Code
}

func TestMilestones_VersionId_SetAndClear(t *testing.T) {
	r, _ := newTestRouter(t)
	pid := uuid.New()
	v := mustCreateVersion(t, r, pid, "v1.0", nil)

	var m api.Milestone
	if status := doJSON(t, r, http.MethodPost, "/api/v1/milestones",
		api.MilestoneCreate{ProjectId: pid, Name: "M1", VersionId: &v.Id}, &m); status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	if m.VersionId == nil || *m.VersionId != v.Id {
		t.Fatalf("created VersionId = %v, want %v", m.VersionId, v.Id)
	}

	path := "/api/v1/milestones/" + m.Id.String()
	var renamed api.Milestone
	if status := patchRaw(t, r, path, `{"name":"M1 renamed"}`, &renamed); status != http.StatusOK {
		t.Fatalf("rename status = %d", status)
	}
	if renamed.VersionId == nil || *renamed.VersionId != v.Id {
		t.Errorf("versionId changed by a patch that omitted it: %v", renamed.VersionId)
	}

	var cleared api.Milestone
	if status := patchRaw(t, r, path, `{"versionId":null}`, &cleared); status != http.StatusOK {
		t.Fatalf("clear status = %d", status)
	}
	if cleared.VersionId != nil {
		t.Errorf("versionId = %v after explicit null, want nil", cleared.VersionId)
	}

	var fetched api.Milestone
	doJSON(t, r, http.MethodGet, path, nil, &fetched)
	if fetched.VersionId != nil {
		t.Errorf("persisted versionId = %v, want nil", fetched.VersionId)
	}
}

func TestMilestones_VersionFromOtherProject_Returns400(t *testing.T) {
	r, _ := newTestRouter(t)
	pid, other := uuid.New(), uuid.New()
	foreign := mustCreateVersion(t, r, other, "v9", nil)

	if status := doJSON(t, r, http.MethodPost, "/api/v1/milestones",
		api.MilestoneCreate{ProjectId: pid, Name: "M1", VersionId: &foreign.Id}, nil); status != http.StatusBadRequest {
		t.Errorf("create with foreign version = %d, want 400", status)
	}

	var m api.Milestone
	doJSON(t, r, http.MethodPost, "/api/v1/milestones", api.MilestoneCreate{ProjectId: pid, Name: "M1"}, &m)
	if status := patchRaw(t, r, "/api/v1/milestones/"+m.Id.String(), `{"versionId":"`+foreign.Id.String()+`"}`, nil); status != http.StatusBadRequest {
		t.Errorf("patch with foreign version = %d, want 400", status)
	}
}

func TestMilestones_UnknownVersion_Returns400(t *testing.T) {
	r, _ := newTestRouter(t)
	missing := uuid.New()
	if status := doJSON(t, r, http.MethodPost, "/api/v1/milestones",
		api.MilestoneCreate{ProjectId: uuid.New(), Name: "M1", VersionId: &missing}, nil); status != http.StatusBadRequest {
		t.Errorf("create with unknown version = %d, want 400", status)
	}
}
