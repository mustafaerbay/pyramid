package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestTOTPVector(t *testing.T) {
	// RFC 6238, SHA-1, T = 59 s → 94287082; the last 6 digits.
	if got := totpAt([]byte("12345678901234567890"), 59/30); got != "287082" {
		t.Errorf("totp = %s", got)
	}
}

func TestTwoFactor(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	amyUser := addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")

	var setup api.TwoFactorSetup
	call(t, r, http.MethodPost, "/api/v1/auth/2fa/setup", nil, amy, &setup)
	if setup.Secret == "" || !strings.HasPrefix(setup.OtpauthUrl, "otpauth://totp/") {
		t.Fatalf("setup = %+v", setup)
	}
	key, _ := decodeSecret(setup.Secret)
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/2fa/enable", map[string]string{"code": "000000"}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("wrong code = %d", s)
	}
	var rec api.TwoFactorRecovery
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/2fa/enable", map[string]string{"code": totpAt(key, time.Now().Unix()/30)}, amy, &rec); s != http.StatusOK || len(rec.RecoveryCodes) != 10 {
		t.Fatalf("enable = %d %+v", s, rec)
	}

	var e api.Error
	s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "first-pass"}, nil, &e)
	if s != http.StatusUnauthorized || e.Details == nil || *e.Details != "totp_required" {
		t.Errorf("login without code = %d %+v", s, e)
	}
	// The code used to enable can't be used again; the next step's can.
	next := totpAt(key, time.Now().Unix()/30+1)
	if s, c := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "first-pass", Code: &next}, nil, nil); s != http.StatusOK || c == nil {
		t.Errorf("login with code = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "first-pass", Code: &next}, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("reused code = %d", s)
	}
	rc := rec.RecoveryCodes[0]
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "first-pass", Code: &rc}, nil, nil); s != http.StatusOK {
		t.Errorf("recovery code = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/login", api.LoginRequest{Email: "amy@x.dev", Password: "first-pass", Code: &rc}, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("recovery code twice = %d", s)
	}
	var st api.TwoFactorStatus
	call(t, r, http.MethodGet, "/api/v1/auth/2fa", nil, amy, &st)
	if !st.Enabled || st.RecoveryCodesLeft != 9 {
		t.Errorf("status = %+v", st)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/auth/2fa/disable", map[string]string{"password": "nope"}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("disable wrong password = %d", s)
	}
	// An admin can turn it off for someone who lost their phone.
	if s, _ := call(t, r, http.MethodPost, "/api/v1/users/"+amyUser.Id.String()+"/reset-2fa", nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("admin reset = %d", s)
	}
	login(t, r, "amy@x.dev", "first-pass")
}

func TestOIDC(t *testing.T) {
	var issuer string
	idp := http.NewServeMux()
	idp.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer, "authorization_endpoint": issuer + "/authorize",
			"token_endpoint": issuer + "/token", "userinfo_endpoint": issuer + "/userinfo"})
	})
	idp.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "good" || r.Form.Get("client_secret") != "shh" || r.Form.Get("code_verifier") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at-123", "token_type": "Bearer"})
	})
	email := "amy@x.dev"
	idp.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"email": email, "email_verified": true})
	})
	srv := httptest.NewServer(idp)
	defer srv.Close()
	issuer = srv.URL

	r, _ := newAuthRouter(t, func(h *Handler) {
		h.cfg.OIDCIssuer, h.cfg.OIDCClientID, h.cfg.OIDCClientSecret, h.cfg.OIDCLabel = issuer, "pyramid", "shh", "Sign in with Acme"
		h.cfg.PublicURL = "https://app.example"
	})
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)

	var opts api.AuthOptions
	call(t, r, http.MethodGet, "/api/v1/auth/options", nil, nil, &opts)
	if opts.Sso == nil || !*opts.Sso || opts.SsoLabel == nil || *opts.SsoLabel != "Sign in with Acme" {
		t.Errorf("options = %+v", opts)
	}

	start := func() (*url.URL, *http.Cookie) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/start", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusFound {
			t.Fatalf("start = %d %s", w.Code, w.Body)
		}
		loc, _ := url.Parse(w.Header().Get("Location"))
		var ck *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == "pyramid_oidc" {
				ck = c
			}
		}
		return loc, ck
	}
	callback := func(q string, ck *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?"+q, nil)
		if ck != nil {
			req.AddCookie(ck)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	loc, ck := start()
	if !strings.HasPrefix(loc.String(), issuer+"/authorize") || loc.Query().Get("code_challenge") == "" || loc.Query().Get("client_id") != "pyramid" || ck == nil {
		t.Fatalf("redirect = %s cookie %v", loc, ck)
	}
	state := loc.Query().Get("state")
	if w := callback("code=good&state=wrong", ck); !strings.Contains(w.Header().Get("Location"), "sso=failed") {
		t.Errorf("bad state → %s", w.Header().Get("Location"))
	}
	w := callback("code=good&state="+url.QueryEscape(state), ck)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "https://app.example/" {
		t.Fatalf("callback = %d %s", w.Code, w.Header().Get("Location"))
	}
	var session *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "pyramid_session" {
			session = c
		}
	}
	var me api.AuthMe
	if s, _ := call(t, r, http.MethodGet, "/api/v1/auth/me", nil, session, &me); s != http.StatusOK || string(me.User.Email) != "amy@x.dev" {
		t.Errorf("me after SSO = %d %+v", s, me)
	}
	email = "stranger@x.dev"
	loc, ck = start()
	if w := callback("code=good&state="+url.QueryEscape(loc.Query().Get("state")), ck); !strings.Contains(w.Header().Get("Location"), "sso=no_account") {
		t.Errorf("unknown email → %s", w.Header().Get("Location"))
	}
}
