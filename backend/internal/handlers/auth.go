package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Sign-in: a session is a signed cookie "userID.expiry.signature". The
// signature also covers the end of the password hash, so changing a
// password ends that account's other sessions.

const (
	sessionCookie     = "pyramid_session"
	sessionTTL        = 14 * 24 * time.Hour
	minPasswordLength = 8
	maxFailedLogins   = 10
	loginLockWindow   = 15 * time.Minute
	ctxUser           = "user"
)

// --- sessions ---

func (h *Handler) sign(payload string, u models.User) string {
	mac := hmac.New(sha256.New, h.cfg.SessionSecret)
	mac.Write([]byte(payload))
	mac.Write([]byte{0})
	if u.PasswordHash != nil {
		ph := *u.PasswordHash
		mac.Write([]byte(ph[max(0, len(ph)-16):]))
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handler) setSession(c *gin.Context, u models.User) {
	exp := time.Now().Add(sessionTTL).Unix()
	payload := fmt.Sprintf("%s.%d", u.ID, exp)
	value := payload + "." + h.sign(payload, u)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: value, Path: "/", MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSession(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

// sessionUser returns the active account behind the request's cookie.
func (h *Handler) sessionUser(c *gin.Context) (models.User, bool) {
	var u models.User
	raw, err := c.Cookie(sessionCookie)
	if err != nil {
		return u, false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return u, false
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return u, false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return u, false
	}
	if err := h.db.First(&u, "id = ?", id).Error; err != nil || !u.IsActive {
		return u, false
	}
	want := h.sign(parts[0]+"."+parts[1], u)
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return u, false
	}
	return u, true
}

// --- middleware ---

// requireAuth answers 401 unless the request has a valid session.
func (h *Handler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.cfg.AuthOff {
			c.Next()
			return
		}
		u, ok := h.sessionUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
			return
		}
		c.Set(ctxUser, u)
		c.Next()
	}
}

// denyViewerWrites keeps viewers read-only; they can still sign out and
// change their password.
func (h *Handler) denyViewerWrites() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		p := c.FullPath()
		if m == http.MethodGet || m == http.MethodHead || strings.HasPrefix(p, "/api/v1/auth/") || strings.HasPrefix(p, "/api/v1/notifications") || p == "/api/v1/queries/run" {
			c.Next()
			return
		}
		if !h.hasRole(c, api.SystemRoleMember) {
			c.AbortWithStatusJSON(http.StatusForbidden, api.Error{Message: "your account can only view"})
			return
		}
		c.Next()
	}
}

func (h *Handler) requireRole(role api.SystemRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.hasRole(c, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, api.Error{Message: fmt.Sprintf("needs the %s role", role)})
			return
		}
		c.Next()
	}
}

// hasRole reports whether the signed-in account has at least `role`.
func (h *Handler) hasRole(c *gin.Context, role api.SystemRole) bool {
	if h.cfg.AuthOff {
		return true
	}
	u, ok := currentUser(c)
	return ok && models.RoleRank(u.SystemRole) >= models.RoleRank(role)
}

func currentUser(c *gin.Context) (models.User, bool) {
	v, ok := c.Get(ctxUser)
	if !ok {
		return models.User{}, false
	}
	u, ok := v.(models.User)
	return u, ok
}

// --- failed-login limiter ---

type loginLimiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{fails: map[string][]time.Time{}} }

func (l *loginLimiter) recent(key string) int {
	cut := time.Now().Add(-loginLockWindow)
	kept := l.fails[key][:0]
	for _, t := range l.fails[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	l.fails[key] = kept
	return len(kept)
}

func (l *loginLimiter) locked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.recent(key) >= maxFailedLogins
}

func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recent(key)
	l.fails[key] = append(l.fails[key], time.Now())
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

// --- handlers ---

func normEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func hashPassword(pw string) (string, error) {
	if len(pw) < minPasswordLength {
		return "", fmt.Errorf("passwords need at least %d characters", minPasswordLength)
	}
	if len(pw) > 72 {
		return "", errors.New("passwords can be at most 72 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func (h *Handler) findUserByEmail(email string) (models.User, error) {
	var u models.User
	err := h.db.Where("LOWER(email) = ?", normEmail(email)).First(&u).Error
	return u, err
}

func (h *Handler) authMe(u models.User) (api.AuthMe, error) {
	var p models.Person
	if err := h.db.First(&p, "id = ?", u.PersonID).Error; err != nil {
		return api.AuthMe{}, err
	}
	return api.AuthMe{User: u.ToAPI(), Person: p.ToAPI()}, nil
}

// Login implements POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var in api.LoginRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	key := normEmail(in.Email) + "|" + c.ClientIP()
	if h.logins.locked(key) {
		c.JSON(http.StatusTooManyRequests, api.Error{Message: "too many failed sign-ins, try again in 15 minutes"})
		return
	}
	u, err := h.findUserByEmail(in.Email)
	ok := err == nil && u.IsActive && u.PasswordHash != nil &&
		bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.Password)) == nil
	if !ok {
		h.logins.fail(key)
		c.JSON(http.StatusUnauthorized, api.Error{Message: "wrong email or password"})
		return
	}
	h.logins.reset(key)
	now := time.Now().UTC()
	u.LastLoginAt = &now
	if err := h.db.Model(&u).Update("last_login_at", now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to sign in", Details: strptr(err.Error())})
		return
	}
	me, err := h.authMe(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load your profile", Details: strptr(err.Error())})
		return
	}
	h.setSession(c, u)
	c.JSON(http.StatusOK, me)
}

// Logout implements POST /api/v1/auth/logout.
func (h *Handler) Logout(c *gin.Context) {
	h.clearSession(c)
	c.Status(http.StatusNoContent)
}

// GetMe implements GET /api/v1/auth/me.
func (h *Handler) GetMe(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	me, err := h.authMe(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load your profile", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, me)
}

// ChangePassword implements POST /api/v1/auth/password.
func (h *Handler) ChangePassword(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	var in api.PasswordChange
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	if u.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.CurrentPassword)) != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "current password is wrong"})
		return
	}
	hash, err := hashPassword(in.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	if err := h.db.Model(&u).Update("password_hash", hash).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to change password", Details: strptr(err.Error())})
		return
	}
	u.PasswordHash = &hash
	h.setSession(c, u) // the old cookie stopped working with the old hash
	c.Status(http.StatusNoContent)
}

// ListUsers implements GET /api/v1/users.
func (h *Handler) ListUsers(c *gin.Context) {
	var users []models.User
	if err := h.db.Order("email").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list accounts", Details: strptr(err.Error())})
		return
	}
	out := make([]api.User, len(users))
	for i, u := range users {
		out[i] = u.ToAPI()
	}
	c.JSON(http.StatusOK, api.UserList{Data: out})
}

// CreateUser implements POST /api/v1/users.
func (h *Handler) CreateUser(c *gin.Context) {
	var in api.UserCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var p models.Person
	if err := h.db.First(&p, "id = ?", in.PersonId).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "person not found"})
		return
	}
	var n int64
	h.db.Model(&models.User{}).Where("person_id = ?", p.ID).Count(&n)
	if n > 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "this person already has an account"})
		return
	}
	email := p.Email
	if in.Email != nil && strings.TrimSpace(*in.Email) != "" {
		email = *in.Email
	}
	email = normEmail(email)
	if _, err := h.findUserByEmail(email); err == nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "another account already uses this email"})
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	role := api.SystemRoleMember
	if in.Role != nil {
		role = *in.Role
	}
	if models.RoleRank(role) == 0 && role != api.SystemRoleViewer {
		c.JSON(http.StatusBadRequest, api.Error{Message: "unknown role"})
		return
	}
	u := models.User{PersonID: p.ID, Email: email, PasswordHash: &hash, SystemRole: role, IsActive: true}
	if err := h.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create account", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, u.ToAPI())
}

// UpdateUser implements PATCH /api/v1/users/{id}.
func (h *Handler) UpdateUser(c *gin.Context) {
	var u models.User
	if err := h.db.First(&u, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load account", Details: strptr(err.Error())})
		return
	}
	var in api.UserUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	losesAdmin := u.SystemRole == api.SystemRoleAdmin && u.IsActive &&
		((in.Role != nil && *in.Role != api.SystemRoleAdmin) || (in.IsActive != nil && !*in.IsActive))
	if losesAdmin {
		var others int64
		h.db.Model(&models.User{}).Where("system_role = ? AND is_active = ? AND id <> ?", api.SystemRoleAdmin, true, u.ID).Count(&others)
		if others == 0 {
			c.JSON(http.StatusBadRequest, api.Error{Message: "there must be at least one active admin"})
			return
		}
	}
	if in.Role != nil {
		if models.RoleRank(*in.Role) == 0 && *in.Role != api.SystemRoleViewer {
			c.JSON(http.StatusBadRequest, api.Error{Message: "unknown role"})
			return
		}
		u.SystemRole = *in.Role
	}
	if in.IsActive != nil {
		u.IsActive = *in.IsActive
	}
	if in.Password != nil {
		hash, err := hashPassword(*in.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
			return
		}
		u.PasswordHash = &hash
	}
	if err := h.db.Save(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update account", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, u.ToAPI())
}

// EnsureAdmin creates the first admin from ADMIN_EMAIL/ADMIN_PASSWORD when
// no account has that email yet, linking it to the person with that email
// (or a new one). It never changes an existing account.
func (h *Handler) EnsureAdmin() error {
	email := normEmail(h.cfg.AdminEmail)
	if email == "" || h.cfg.AdminPassword == "" {
		return nil
	}
	if _, err := h.findUserByEmail(email); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	hash, err := hashPassword(h.cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
	}
	var p models.Person
	err = h.db.Where("LOWER(email) = ?", email).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		name := email[:max(strings.Index(email, "@"), 0)]
		if name == "" {
			name = "Admin"
		}
		p = models.Person{FirstName: strings.ToUpper(name[:1]) + name[1:], LastName: "Admin", Email: email}
		err = h.db.Create(&p).Error
	}
	if err != nil {
		return err
	}
	return h.db.Create(&models.User{PersonID: p.ID, Email: email, PasswordHash: &hash, SystemRole: api.SystemRoleAdmin, IsActive: true}).Error
}
