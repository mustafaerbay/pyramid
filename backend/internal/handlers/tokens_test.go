package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func bearer(t *testing.T, r *gin.Engine, method, path, token string, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestApiTokens(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")

	var read, write api.ApiTokenCreated
	if s, _ := call(t, r, http.MethodPost, "/api/v1/me/tokens", api.ApiTokenInput{Name: "CI", Scope: api.ApiTokenInputScopeRead}, amy, &read); s != http.StatusCreated || !strings.HasPrefix(read.Token, "pyr_") {
		t.Fatalf("create = %d %+v", s, read)
	}
	call(t, r, http.MethodPost, "/api/v1/me/tokens", api.ApiTokenInput{Name: "Script", Scope: api.ApiTokenInputScopeWrite, ExpiresInDays: ptrTo(30)}, amy, &write)
	if write.ExpiresAt == nil {
		t.Errorf("no expiry on %+v", write)
	}

	if s := bearer(t, r, http.MethodGet, "/api/v1/projects", read.Token, ""); s != http.StatusOK {
		t.Errorf("read token GET = %d", s)
	}
	if s := bearer(t, r, http.MethodPost, "/api/v1/tasks", read.Token, `{"projectId":"00000000-0000-0000-0000-000000000001","title":"x"}`); s != http.StatusForbidden {
		t.Errorf("read token POST = %d, want 403", s)
	}
	if s := bearer(t, r, http.MethodPost, "/api/v1/tasks", write.Token, `{"projectId":"00000000-0000-0000-0000-000000000001","title":"x"}`); s != http.StatusCreated {
		t.Errorf("write token POST = %d, want 201", s)
	}
	if s := bearer(t, r, http.MethodPost, "/api/v1/me/tokens", write.Token, `{"name":"more","scope":"write"}`); s != http.StatusForbidden {
		t.Errorf("token makes tokens = %d, want 403", s)
	}
	if s := bearer(t, r, http.MethodGet, "/api/v1/projects", "pyr_nope", ""); s != http.StatusUnauthorized {
		t.Errorf("bad token = %d", s)
	}

	var list api.ApiTokenList
	call(t, r, http.MethodGet, "/api/v1/me/tokens", nil, amy, &list)
	if len(list.Data) != 2 || list.Data[0].LastUsedAt == nil && list.Data[1].LastUsedAt == nil {
		t.Errorf("list = %+v", list.Data)
	}
	call(t, r, http.MethodGet, "/api/v1/me/tokens", nil, admin, &list)
	if len(list.Data) != 0 {
		t.Errorf("admin sees amy's tokens")
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/me/tokens/"+read.Id.String(), nil, admin, nil); s != http.StatusNotFound {
		t.Errorf("admin revokes amy's = %d", s)
	}
	call(t, r, http.MethodDelete, "/api/v1/me/tokens/"+read.Id.String(), nil, amy, nil)
	if s := bearer(t, r, http.MethodGet, "/api/v1/projects", read.Token, ""); s != http.StatusUnauthorized {
		t.Errorf("revoked token = %d", s)
	}
	for name, in := range map[string]api.ApiTokenInput{
		"no name":   {Name: " ", Scope: api.ApiTokenInputScopeRead},
		"bad scope": {Name: "x", Scope: "admin"},
		"too long":  {Name: "x", Scope: api.ApiTokenInputScopeRead, ExpiresInDays: ptrTo(400)},
	} {
		if s, _ := call(t, r, http.MethodPost, "/api/v1/me/tokens", in, amy, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d", name, s)
		}
	}

	// The audit log saw the sign-ins, the token and the task, and only admins read it.
	var audit api.AuditList
	if s, _ := call(t, r, http.MethodGet, "/api/v1/audit", nil, amy, nil); s != http.StatusForbidden {
		t.Errorf("member reads audit = %d", s)
	}
	call(t, r, http.MethodGet, "/api/v1/audit?pageSize=100", nil, admin, &audit)
	seen := map[string]bool{}
	for _, e := range audit.Data {
		seen[e.Action+"|"+string(e.Via)] = true
		if e.Action == "create task" && (len(e.Fields) == 0 || e.Fields[0] != "projectId") {
			t.Errorf("task fields = %v", e.Fields)
		}
	}
	for _, want := range []string{"sign in|session", "create token|session", "create task|token", "revoke token|session", "create user|session"} {
		if !seen[want] {
			t.Errorf("audit has no %q: %v", want, seen)
		}
	}
	call(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "amy@x.dev", "password": "wrong"}, nil, nil)
	call(t, r, http.MethodGet, "/api/v1/audit?q=failed", nil, admin, &audit)
	if audit.Total != 1 || audit.Data[0].Actor != "amy@x.dev" {
		t.Errorf("failed sign-in = %+v", audit.Data)
	}
}
