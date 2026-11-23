package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"gorm.io/gorm"
)

func TestPasswordReset(t *testing.T) {
	box := &mailbox{}
	var db *gorm.DB
	r, db := newAuthRouter(t, func(h *Handler) { h.mail = box.send; h.cfg.PublicURL = "https://pyramid.test" })
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)

	var opts api.AuthOptions
	if s, _ := call(t, r, http.MethodGet, "/api/v1/auth/options", nil, nil, &opts); s != http.StatusOK || !opts.EmailReset {
		t.Errorf("options = %d %+v", s, opts)
	}
	// Unknown email: same answer, no mail.
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/forgot", api.ForgotPassword{Email: "nobody@x.dev"}, nil, nil); s != http.StatusNoContent {
		t.Errorf("unknown email = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/forgot", api.ForgotPassword{Email: " AMY@x.dev "}, nil, nil); s != http.StatusNoContent {
		t.Errorf("forgot = %d", s)
	}
	mails := box.waitFor(t, 1)
	if len(mails) != 1 || mails[0].to != "amy@x.dev" {
		t.Fatalf("mails = %+v", mails)
	}
	m := regexp.MustCompile(`https://pyramid\.test/reset\?token=([A-Za-z0-9_-]+)`).FindStringSubmatch(mails[0].body)
	if m == nil {
		t.Fatalf("no link in %q", mails[0].body)
	}
	token := m[1]
	var stored string
	db.Raw("SELECT token_hash FROM password_resets").Scan(&stored)
	if stored == "" || strings.Contains(stored, token) {
		t.Errorf("token stored in clear: %q", stored)
	}

	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/reset", api.ResetPassword{Token: token, NewPassword: "short"}, nil, nil); s != http.StatusBadRequest {
		t.Errorf("short password = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/reset", api.ResetPassword{Token: token, NewPassword: "brand-new-pass"}, nil, nil); s != http.StatusNoContent {
		t.Fatalf("reset = %d", s)
	}
	login(t, r, "amy@x.dev", "brand-new-pass")
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/reset", api.ResetPassword{Token: token, NewPassword: "another-pass"}, nil, nil); s != http.StatusBadRequest {
		t.Errorf("token reused = %d, want 400", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/reset", api.ResetPassword{Token: "made-up", NewPassword: "another-pass"}, nil, nil); s != http.StatusBadRequest {
		t.Errorf("made-up token = %d, want 400", s)
	}

	// Expired links don't work.
	call(t, r, http.MethodPost, "/api/v1/auth/forgot", api.ForgotPassword{Email: "amy@x.dev"}, nil, nil)
	mails = box.waitFor(t, 2)
	token2 := regexp.MustCompile(`token=([A-Za-z0-9_-]+)`).FindStringSubmatch(mails[1].body)[1]
	db.Exec("UPDATE password_resets SET expires_at = ?", time.Now().Add(-time.Minute))
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/reset", api.ResetPassword{Token: token2, NewPassword: "another-pass"}, nil, nil); s != http.StatusBadRequest {
		t.Errorf("expired token = %d, want 400", s)
	}
}

func TestPasswordReset_NoEmail(t *testing.T) {
	r, _ := newAuthRouter(t)
	var opts api.AuthOptions
	call(t, r, http.MethodGet, "/api/v1/auth/options", nil, nil, &opts)
	if opts.EmailReset {
		t.Error("emailReset should be off without SMTP")
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/forgot", api.ForgotPassword{Email: "admin@x.dev"}, nil, nil); s != http.StatusNoContent {
		t.Errorf("forgot without email = %d", s)
	}
}
