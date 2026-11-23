package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func hook(t *testing.T, r *gin.Engine, repoID string, headers map[string]string, body string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/git/hooks/"+repoID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func ghSign(secret, body string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func TestGitLinking(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	var p api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "GIT", Name: "Git"}, admin, &p)
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: p.Id, Title: "Login"}, admin, &task)

	in := api.ProjectRepoInput{ProjectId: p.Id, Provider: api.ProjectRepoInputProviderGithub, Url: "https://github.com/acme/shop"}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-repos", in, amy, nil); s != http.StatusForbidden {
		t.Errorf("member links repo = %d", s)
	}
	var repo api.ProjectRepo
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-repos", in, admin, &repo); s != http.StatusCreated || repo.Secret == nil || repo.Name != "acme/shop" {
		t.Fatalf("link = %d %+v", s, repo)
	}
	secret := *repo.Secret
	id := repo.Id.String()

	push := `{"ref":"refs/heads/main","commits":[{"id":"abc123","message":"GIT-1 wire up the form\n\nmore","url":"https://github.com/acme/shop/commit/abc123","timestamp":"2026-11-20T10:00:00Z","author":{"name":"Amy"}},{"id":"def456","message":"unrelated","url":"u","timestamp":"2026-11-20T11:00:00Z","author":{"name":"Bo"}}]}`
	if s := hook(t, r, id, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": "sha256=bad"}, push); s != http.StatusUnauthorized {
		t.Errorf("bad signature = %d", s)
	}
	if s := hook(t, r, id, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": ghSign(secret, push)}, push); s != http.StatusAccepted {
		t.Fatalf("push = %d", s)
	}
	pr := `{"action":"closed","pull_request":{"number":7,"title":"Login form","body":"Closes #1","html_url":"https://github.com/acme/shop/pull/7","state":"closed","merged":true,"user":{"login":"amy"},"head":{"ref":"feature/login","sha":"abc123"},"updated_at":"2026-11-21T09:00:00Z"}}`
	hook(t, r, id, map[string]string{"X-GitHub-Event": "pull_request", "X-Hub-Signature-256": ghSign(secret, pr)}, pr)
	run := `{"action":"completed","workflow_run":{"id":99,"name":"CI","status":"completed","conclusion":"failure","html_url":"https://github.com/acme/shop/actions/runs/99","head_branch":"main","head_sha":"abc123","updated_at":"2026-11-21T10:00:00Z"}}`
	hook(t, r, id, map[string]string{"X-GitHub-Event": "workflow_run", "X-Hub-Signature-256": ghSign(secret, run)}, run)
	// A PR or pipeline number isn't an item reference.
	pr2 := `{"action":"opened","pull_request":{"number":1,"title":"Docs","body":"","html_url":"u","state":"open","merged":false,"user":{"login":"bo"},"head":{"ref":"docs","sha":"zzz"},"updated_at":"2026-11-21T09:00:00Z"}}`
	hook(t, r, id, map[string]string{"X-GitHub-Event": "pull_request", "X-Hub-Signature-256": ghSign(secret, pr2)}, pr2)
	// The same push again doesn't duplicate.
	hook(t, r, id, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": ghSign(secret, push)}, push)

	var list api.GitEventList
	call(t, r, http.MethodGet, "/api/v1/git-activity?taskId="+task.Id.String(), nil, amy, &list)
	kinds := map[api.GitEventKind]api.GitEvent{}
	for _, e := range list.Data {
		kinds[e.Kind] = e
	}
	if len(list.Data) != 3 || kinds[api.GitEventKindPullRequest].State == nil || *kinds[api.GitEventKindPullRequest].State != "merged" ||
		*kinds[api.GitEventKindPipeline].State != "failed" || kinds[api.GitEventKindCommit].Title != "GIT-1 wire up the form" {
		t.Errorf("item activity = %+v", list.Data)
	}
	call(t, r, http.MethodGet, "/api/v1/git-activity?projectId="+p.Id.String()+"&kind=commit", nil, amy, &list)
	if len(list.Data) != 2 {
		t.Errorf("project commits = %d", len(list.Data))
	}

	// GitLab: the secret comes as a token.
	var gl api.ProjectRepo
	call(t, r, http.MethodPost, "/api/v1/project-repos", api.ProjectRepoInput{ProjectId: p.Id, Provider: api.ProjectRepoInputProviderGitlab, Url: "https://gitlab.com/acme/api"}, admin, &gl)
	mr := `{"object_kind":"merge_request","user":{"name":"Bo"},"object_attributes":{"iid":3,"title":"API for GIT-1","url":"https://gitlab.com/acme/api/-/merge_requests/3","state":"opened","source_branch":"x","updated_at":"2026-11-22 10:00:00 UTC"}}`
	if s := hook(t, r, gl.Id.String(), map[string]string{"X-Gitlab-Event": "Merge Request Hook", "X-Gitlab-Token": "nope"}, mr); s != http.StatusUnauthorized {
		t.Errorf("bad gitlab token = %d", s)
	}
	if s := hook(t, r, gl.Id.String(), map[string]string{"X-Gitlab-Event": "Merge Request Hook", "X-Gitlab-Token": *gl.Secret}, mr); s != http.StatusAccepted {
		t.Errorf("gitlab mr = %d", s)
	}
	call(t, r, http.MethodGet, "/api/v1/git-activity?taskId="+task.Id.String()+"&kind=pull_request", nil, amy, &list)
	if len(list.Data) != 2 {
		t.Errorf("prs after gitlab = %+v", list.Data)
	}

	var repos api.ProjectRepoList
	call(t, r, http.MethodGet, "/api/v1/project-repos?projectId="+p.Id.String(), nil, amy, &repos)
	if len(repos.Data) != 2 || repos.Data[0].Secret != nil || repos.Data[0].LastEventAt == nil {
		t.Errorf("repos = %+v", repos.Data)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-repos", api.ProjectRepoInput{ProjectId: p.Id, Provider: api.ProjectRepoInputProviderGithub, Url: "ftp://x"}, admin, nil); s != http.StatusBadRequest {
		t.Errorf("bad url = %d", s)
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/project-repos/"+id, nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("unlink = %d", s)
	}
	if s := hook(t, r, id, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": ghSign(secret, push)}, push); s != http.StatusNotFound {
		t.Errorf("hook after unlink = %d", s)
	}
}
