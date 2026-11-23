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
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"
)

// newAuthRouter is newTestRouter with sign-in turned on and one admin
// (admin@x.dev / admin-pass) created the way the server bootstraps it.
func newAuthRouter(t *testing.T, opts ...func(*Handler)) (*gin.Engine, *gorm.DB) {
	t.Helper()
	_, db := newTestRouter(t)
	cfg := config.Config{
		CORSOrigin:    "*",
		SessionSecret: []byte("test-secret-test-secret-test-secret"),
		AdminEmail:    "admin@x.dev",
		AdminPassword: "admin-pass",
		UploadDir:     t.TempDir(),
	}
	h := New(db, cfg)
	for _, o := range opts {
		o(h)
	}
	if err := h.EnsureAdmin(); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	r := gin.New()
	h.RegisterRoutes(r)
	return r, db
}

// call sends a JSON request with an optional session cookie and returns
// the status and any session cookie the response set.
func call(t *testing.T, r *gin.Engine, method, path string, body any, cookie *http.Cookie, out any) (int, *http.Cookie) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if out != nil && w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("unmarshal %q: %v", w.Body.String(), err)
		}
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return w.Code, c
		}
	}
	return w.Code, nil
}

func login(t *testing.T, r *gin.Engine, email, password string) *http.Cookie {
	t.Helper()
	status, c := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: email, Password: password}, nil, nil)
	if status != http.StatusOK || c == nil {
		t.Fatalf("login %s: status %d, cookie %v", email, status, c)
	}
	return c
}

// addAccount creates a person and an account for them as the admin.
func addAccount(t *testing.T, r *gin.Engine, admin *http.Cookie, email string, role api.SystemRole) api.User {
	t.Helper()
	var p api.Person
	if s, _ := call(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "P", LastName: email, Email: openapi_types.Email(email)}, admin, &p); s != http.StatusCreated {
		t.Fatalf("create person %s: %d", email, s)
	}
	var u api.User
	if s, _ := call(t, r, http.MethodPost, "/api/v1/users", api.UserCreate{PersonId: p.Id, Role: &role, Password: "first-pass"}, admin, &u); s != http.StatusCreated {
		t.Fatalf("create user %s: %d", email, s)
	}
	return u
}

func TestAuth_SignedOutGets401ButHealthAndLoginWork(t *testing.T) {
	r, _ := newAuthRouter(t)
	if s, _ := call(t, r, http.MethodGet, "/api/v1/tasks", nil, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("GET /tasks signed out = %d, want 401", s)
	}
	if s, _ := call(t, r, http.MethodGet, "/health", nil, nil, nil); s != http.StatusOK {
		t.Errorf("GET /health = %d, want 200", s)
	}
	if s, _ := call(t, r, http.MethodGet, "/api/v1/tasks", nil, &http.Cookie{Name: sessionCookie, Value: "forged.value.x"}, nil); s != http.StatusUnauthorized {
		t.Errorf("forged cookie = %d, want 401", s)
	}
}

func TestAuth_LoginMeLogout(t *testing.T) {
	r, _ := newAuthRouter(t)
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "admin@x.dev", Password: "nope"}, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("wrong password = %d, want 401", s)
	}
	// Email is matched ignoring case and surrounding spaces.
	c := login(t, r, "  ADMIN@x.dev ", "admin-pass")
	if !c.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	var me api.AuthMe
	if s, _ := call(t, r, http.MethodGet, "/api/v1/auth/me", nil, c, &me); s != http.StatusOK {
		t.Fatalf("me = %d", s)
	}
	if me.User.Role != api.SystemRoleAdmin || me.User.Email != "admin@x.dev" || me.Person.Email != "admin@x.dev" {
		t.Errorf("me = %+v", me)
	}
	if me.User.LastLoginAt == nil {
		t.Error("lastLoginAt not set on login")
	}
	if s, out := call(t, r, http.MethodPost, "/api/v1/auth/logout", nil, c, nil); s != http.StatusNoContent || out == nil || out.MaxAge >= 0 {
		t.Errorf("logout = %d, cookie %v; want 204 and a cleared cookie", s, out)
	}
}

func TestAuth_EnsureAdminReusesPersonAndIsIdempotent(t *testing.T) {
	_, db := newAuthRouter(t)
	cfg := config.Config{SessionSecret: []byte("x"), AdminEmail: "admin@x.dev", AdminPassword: "other"}
	if err := New(db, cfg).EnsureAdmin(); err != nil {
		t.Fatal(err)
	}
	var users, people int64
	db.Table("users").Count(&users)
	db.Table("people").Count(&people)
	if users != 1 || people != 1 {
		t.Errorf("users=%d people=%d after second EnsureAdmin, want 1 and 1", users, people)
	}
}

func TestAuth_Roles(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "viewer@x.dev", api.SystemRoleViewer)
	addAccount(t, r, admin, "member@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "manager@x.dev", api.SystemRoleManager)
	viewer := login(t, r, "viewer@x.dev", "first-pass")
	member := login(t, r, "member@x.dev", "first-pass")
	manager := login(t, r, "manager@x.dev", "first-pass")

	task := api.TaskCreate{ProjectId: uuid.New(), Title: "T"}
	person := api.PersonCreate{FirstName: "N", LastName: "N", Email: "n@x.dev"}
	cases := []struct {
		name   string
		cookie *http.Cookie
		method string
		path   string
		body   any
		want   int
	}{
		{"viewer reads", viewer, http.MethodGet, "/api/v1/tasks", nil, http.StatusOK},
		{"viewer can't write", viewer, http.MethodPost, "/api/v1/tasks", task, http.StatusForbidden},
		{"member writes tasks", member, http.MethodPost, "/api/v1/tasks", task, http.StatusCreated},
		{"member can't add people", member, http.MethodPost, "/api/v1/people", person, http.StatusForbidden},
		{"manager adds people", manager, http.MethodPost, "/api/v1/people", person, http.StatusCreated},
		{"manager can't manage accounts", manager, http.MethodGet, "/api/v1/users", nil, http.StatusForbidden},
		{"admin manages accounts", admin, http.MethodGet, "/api/v1/users", nil, http.StatusOK},
	}
	for _, c := range cases {
		if s, _ := call(t, r, c.method, c.path, c.body, c.cookie, nil); s != c.want {
			t.Errorf("%s: %d, want %d", c.name, s, c.want)
		}
	}

	// Leave: members may ask, only managers decide.
	var me api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, member, &me)
	var leave api.LeaveRequest
	vacation := api.LeaveTypeVacation
	s, _ := call(t, r, http.MethodPost, "/api/v1/leave-requests", api.LeaveRequestCreate{
		PersonId: me.Person.Id, Type: &vacation, StartDate: *mdDate("2026-12-01"), EndDate: *mdDate("2026-12-02"),
	}, member, &leave)
	if s != http.StatusCreated {
		t.Fatalf("member creates leave: %d", s)
	}
	approve := map[string]any{"status": "approved"}
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/leave-requests/"+leave.Id.String(), approve, member, nil); s != http.StatusForbidden {
		t.Errorf("member approves leave: %d, want 403", s)
	}
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/leave-requests/"+leave.Id.String(), approve, manager, nil); s != http.StatusOK {
		t.Errorf("manager approves leave: %d, want 200", s)
	}
}

func TestAuth_AccountsAndPasswords(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	u := addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	if u.Email != "amy@x.dev" || u.Role != api.SystemRoleMember || !u.IsActive {
		t.Errorf("created user = %+v", u)
	}
	// One account per person, and passwords need 8+ characters.
	short := "short"
	if s, _ := call(t, r, http.MethodPost, "/api/v1/users", api.UserCreate{PersonId: u.PersonId, Password: "long-enough"}, admin, nil); s != http.StatusBadRequest {
		t.Errorf("second account for a person = %d, want 400", s)
	}
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/users/"+u.Id.String(), api.UserUpdate{Password: &short}, admin, nil); s != http.StatusBadRequest {
		t.Errorf("short password = %d, want 400", s)
	}

	amy := login(t, r, "amy@x.dev", "first-pass")
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/password", api.PasswordChange{CurrentPassword: "wrong", NewPassword: "brand-new-pass"}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("change with wrong current = %d, want 400", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/password", api.PasswordChange{CurrentPassword: "first-pass", NewPassword: "brand-new-pass"}, amy, nil); s != http.StatusNoContent {
		t.Errorf("change password = %d, want 204", s)
	}
	login(t, r, "amy@x.dev", "brand-new-pass")

	// Turning the account off ends its sessions and blocks sign-in.
	off := false
	call(t, r, http.MethodPatch, "/api/v1/users/"+u.Id.String(), api.UserUpdate{IsActive: &off}, admin, nil)
	if s, _ := call(t, r, http.MethodGet, "/api/v1/tasks", nil, amy, nil); s != http.StatusUnauthorized {
		t.Errorf("turned-off account's session = %d, want 401", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "brand-new-pass"}, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("turned-off account login = %d, want 401", s)
	}

	// The last admin can't demote or turn off themselves.
	var me api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, admin, &me)
	member := api.SystemRoleMember
	if s, _ := call(t, r, http.MethodPatch, "/api/v1/users/"+me.User.Id.String(), api.UserUpdate{Role: &member}, admin, nil); s != http.StatusBadRequest {
		t.Errorf("demote last admin = %d, want 400", s)
	}
}

func TestAuth_TooManyFailedLogins(t *testing.T) {
	r, _ := newAuthRouter(t)
	var last int
	for i := 0; i < 11; i++ {
		last, _ = call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "admin@x.dev", Password: "bad"}, nil, nil)
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("11th failed login = %d, want 429", last)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "admin@x.dev", Password: "admin-pass"}, nil, nil); s != http.StatusTooManyRequests {
		t.Errorf("right password while locked = %d, want 429", s)
	}
}
