package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// Single sign-on with OpenID Connect (Entra ID, Google, Keycloak, …):
// authorization code flow with PKCE. The provider's userinfo answer (read
// over TLS with the access token we got straight from its token endpoint)
// gives a verified email, which signs in the active account with that
// email. No account is created; an admin adds people first.

const oidcCookie = "pyramid_oidc"

func (h *Handler) oidcOn() bool {
	return h.cfg.OIDCIssuer != "" && h.cfg.OIDCClientID != "" && h.cfg.OIDCClientSecret != ""
}

type oidcConfig struct {
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
	UserInfo      string `json:"userinfo_endpoint"`
}

var (
	oidcMu    sync.Mutex
	oidcCache = map[string]struct {
		c  oidcConfig
		at time.Time
	}{}
)

func discover(ctx context.Context, issuer string) (oidcConfig, error) {
	oidcMu.Lock()
	if e, ok := oidcCache[issuer]; ok && time.Since(e.at) < time.Hour {
		oidcMu.Unlock()
		return e.c, nil
	}
	oidcMu.Unlock()
	var c oidcConfig
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	res, err := hookClient.Do(req)
	if err != nil {
		return c, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return c, errors.New("discovery: " + res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&c); err != nil {
		return c, err
	}
	if c.Authorization == "" || c.Token == "" || c.UserInfo == "" {
		return c, errors.New("discovery: missing endpoints")
	}
	oidcMu.Lock()
	oidcCache[issuer] = struct {
		c  oidcConfig
		at time.Time
	}{c, time.Now()}
	oidcMu.Unlock()
	return c, nil
}

func randB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handler) oidcSign(s string) string {
	m := hmac.New(sha256.New, h.cfg.SessionSecret)
	m.Write([]byte("oidc|" + s))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (h *Handler) redirectURI(c *gin.Context) string {
	if h.cfg.OIDCRedirectURL != "" {
		return h.cfg.OIDCRedirectURL
	}
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Request.Host
	if fh := c.GetHeader("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	return scheme + "://" + host + "/api/v1/auth/oidc/callback"
}

func (h *Handler) appURL(path string) string {
	base := strings.TrimRight(h.cfg.PublicURL, "/")
	if base == "" {
		base = strings.TrimRight(h.cfg.CORSOrigin, "/")
		if base == "*" {
			base = ""
		}
	}
	return base + path
}

// StartOIDC implements GET /api/v1/auth/oidc/start.
func (h *Handler) StartOIDC(c *gin.Context) {
	if !h.oidcOn() {
		c.JSON(http.StatusNotFound, api.Error{Message: "single sign-on isn't set up"})
		return
	}
	conf, err := discover(c.Request.Context(), h.cfg.OIDCIssuer)
	if err != nil {
		c.Redirect(http.StatusFound, h.appURL("/login?sso=unavailable"))
		return
	}
	state, verifier := randB64(24), randB64(48)
	sum := sha256.Sum256([]byte(verifier))
	payload := state + "." + verifier + "." + strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
	http.SetCookie(c.Writer, &http.Cookie{Name: oidcCookie, Value: payload + "." + h.oidcSign(payload), Path: "/api/v1/auth/oidc",
		MaxAge: 600, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	q := url.Values{
		"response_type": {"code"}, "client_id": {h.cfg.OIDCClientID}, "redirect_uri": {h.redirectURI(c)},
		"scope": {"openid email profile"}, "state": {state},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
	}
	sep := "?"
	if strings.Contains(conf.Authorization, "?") {
		sep = "&"
	}
	c.Redirect(http.StatusFound, conf.Authorization+sep+q.Encode())
}

// OIDCCallback implements GET /api/v1/auth/oidc/callback.
func (h *Handler) OIDCCallback(c *gin.Context) {
	fail := func(reason string) { c.Redirect(http.StatusFound, h.appURL("/login?sso="+reason)) }
	if !h.oidcOn() {
		fail("unavailable")
		return
	}
	raw, err := c.Cookie(oidcCookie)
	http.SetCookie(c.Writer, &http.Cookie{Name: oidcCookie, Value: "", Path: "/api/v1/auth/oidc", MaxAge: -1})
	parts := strings.Split(raw, ".")
	if err != nil || len(parts) != 4 {
		fail("failed")
		return
	}
	payload := strings.Join(parts[:3], ".")
	expUnix, _ := strconv.ParseInt(parts[2], 10, 64)
	exp := time.Unix(expUnix, 0)
	if !hmac.Equal([]byte(h.oidcSign(payload)), []byte(parts[3])) || time.Now().After(exp) ||
		!hmac.Equal([]byte(parts[0]), []byte(c.Query("state"))) || c.Query("code") == "" {
		fail("failed")
		return
	}
	ctx := c.Request.Context()
	conf, err := discover(ctx, h.cfg.OIDCIssuer)
	if err != nil {
		fail("unavailable")
		return
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {c.Query("code")}, "redirect_uri": {h.redirectURI(c)},
		"client_id": {h.cfg.OIDCClientID}, "client_secret": {h.cfg.OIDCClientSecret}, "code_verifier": {parts[1]}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, conf.Token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := hookClient.Do(req)
	if err != nil {
		fail("failed")
		return
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&tok)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || tok.AccessToken == "" {
		fail("failed")
		return
	}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, conf.UserInfo, nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err = hookClient.Do(req)
	if err != nil {
		fail("failed")
		return
	}
	var info struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
		UPN           string `json:"preferred_username"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&info)
	res.Body.Close()
	email := info.Email
	if email == "" && strings.Contains(info.UPN, "@") {
		email = info.UPN // Entra ID work accounts
	}
	if res.StatusCode != http.StatusOK || email == "" || (info.EmailVerified != nil && !*info.EmailVerified) {
		fail("failed")
		return
	}
	u, err := h.findUserByEmail(email)
	if err != nil || !u.IsActive {
		h.audit(c, nil, normEmail(email), "failed sign in (SSO, no account)", "session", nil, nil)
		fail("no_account")
		return
	}
	now := time.Now().UTC()
	h.db.Model(&u).Update("last_login_at", now)
	h.setSession(c, u)
	h.audit(c, &u, u.Email, "sign in (SSO)", "session", nil, nil)
	c.Redirect(http.StatusFound, h.appURL("/"))
}
