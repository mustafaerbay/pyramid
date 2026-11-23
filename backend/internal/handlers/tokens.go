package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// --- personal access tokens ---

const ctxVia = "via" // "session" or "token"
const ctxScope = "tokenScope"

type apiToken struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null"`
	Name       string    `gorm:"not null"`
	Prefix     string    `gorm:"not null"`
	TokenHash  string    `gorm:"not null"`
	Scope      string    `gorm:"not null"`
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

func (apiToken) TableName() string { return "api_tokens" }

func (t apiToken) toAPI() api.ApiToken {
	return api.ApiToken{Id: t.ID, Name: t.Name, Prefix: t.Prefix, Scope: api.ApiTokenScope(t.Scope), ExpiresAt: t.ExpiresAt, LastUsedAt: t.LastUsedAt, CreatedAt: t.CreatedAt}
}

// tokenUser checks an "Authorization: Bearer pyr_…" header.
func (h *Handler) tokenUser(c *gin.Context) (models.User, *apiToken, bool) {
	var u models.User
	raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !ok || !strings.HasPrefix(raw, "pyr_") {
		return u, nil, false
	}
	var t apiToken
	if err := h.db.Where("token_hash = ?", hashToken(strings.TrimSpace(raw))).First(&t).Error; err != nil {
		return u, nil, false
	}
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return u, nil, false
	}
	if err := h.db.First(&u, "id = ?", t.UserID).Error; err != nil || !u.IsActive {
		return u, nil, false
	}
	if t.LastUsedAt == nil || time.Since(*t.LastUsedAt) > time.Minute {
		now := time.Now()
		h.db.Model(&apiToken{}).Where("id = ?", t.ID).Update("last_used_at", now)
		t.LastUsedAt = &now
	}
	return u, &t, true
}

// readOnlyPOST are POSTs that only read.
var readOnlyPOST = map[string]bool{"/api/v1/queries/run": true}

func viaToken(c *gin.Context) bool { return c.GetString(ctxVia) == "token" }

// ListApiTokens implements GET /api/v1/me/tokens.
func (h *Handler) ListApiTokens(c *gin.Context) {
	me, _ := currentUser(c)
	var rows []apiToken
	h.db.Where("user_id = ?", me.ID).Order("created_at").Find(&rows)
	out := make([]api.ApiToken, len(rows))
	for i, t := range rows {
		out[i] = t.toAPI()
	}
	c.JSON(http.StatusOK, api.ApiTokenList{Data: out})
}

// CreateApiToken implements POST /api/v1/me/tokens.
func (h *Handler) CreateApiToken(c *gin.Context) {
	if viaToken(c) {
		c.JSON(http.StatusForbidden, api.Error{Message: "sign in to manage tokens"})
		return
	}
	me, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	var in api.ApiTokenInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a token needs a name of 1–100 characters"})
		return
	}
	if !in.Scope.Valid() {
		c.JSON(http.StatusBadRequest, api.Error{Message: "scope is read or write"})
		return
	}
	var n int64
	h.db.Model(&apiToken{}).Where("user_id = ?", me.ID).Count(&n)
	if n >= 20 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "up to 20 tokens; revoke one first"})
		return
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to make a token"})
		return
	}
	secret := "pyr_" + hex.EncodeToString(buf)
	t := apiToken{UserID: me.ID, Name: name, Prefix: secret[:10], TokenHash: hashToken(secret), Scope: string(in.Scope)}
	if in.ExpiresInDays != nil {
		if *in.ExpiresInDays < 1 || *in.ExpiresInDays > 365 {
			c.JSON(http.StatusBadRequest, api.Error{Message: "tokens expire in 1–365 days"})
			return
		}
		exp := time.Now().Add(time.Duration(*in.ExpiresInDays) * 24 * time.Hour)
		t.ExpiresAt = &exp
	}
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to save the token", Details: strptr(err.Error())})
		return
	}
	a := t.toAPI()
	c.JSON(http.StatusCreated, api.ApiTokenCreated{Id: a.Id, Name: a.Name, Prefix: a.Prefix, Scope: api.ApiTokenCreatedScope(a.Scope), ExpiresAt: a.ExpiresAt, CreatedAt: a.CreatedAt, Token: secret})
}

// RevokeApiToken implements DELETE /api/v1/me/tokens/{id}.
func (h *Handler) RevokeApiToken(c *gin.Context) {
	if viaToken(c) {
		c.JSON(http.StatusForbidden, api.Error{Message: "sign in to manage tokens"})
		return
	}
	me, _ := currentUser(c)
	res := h.db.Where("id = ? AND user_id = ?", c.Param("id"), me.ID).Delete(&apiToken{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "token not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// --- audit log ---

type auditEntry struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    *uuid.UUID `gorm:"type:uuid"`
	Actor     string     `gorm:"not null"`
	Action    string     `gorm:"not null"`
	Resource  string     `gorm:"not null"`
	EntityID  *string
	Fields    string `gorm:"type:jsonb;not null"`
	Via       string `gorm:"not null"`
	IP        *string
	CreatedAt time.Time
}

func (auditEntry) TableName() string { return "audit_log" }

func (h *Handler) audit(c *gin.Context, u *models.User, actor, action, resource string, entityID *string, fields []string) {
	if fields == nil {
		fields = []string{}
	}
	b, _ := json.Marshal(fields)
	via := c.GetString(ctxVia)
	if via == "" {
		via = "session"
	}
	ip := c.ClientIP()
	e := auditEntry{Actor: actor, Action: action, Resource: resource, EntityID: entityID, Fields: string(b), Via: via, IP: &ip}
	if u != nil && u.ID != uuid.Nil {
		id := u.ID
		e.UserID = &id
	}
	h.db.Create(&e)
}

var auditSkip = map[string]bool{
	"/api/v1/queries/run": true, "/api/v1/notifications/read-all": true, "/api/v1/notifications/:id/read": true,
	"/api/v1/auth/logout": true, "/api/v1/me/notification-settings": false,
}

var auditNames = map[string]string{
	"DELETE /api/v1/me/tokens/:id":         "revoke token",
	"POST /api/v1/me/tokens":               "create token",
	"POST /api/v1/tasks/:id/transfer":      "move task to project",
	"POST /api/v1/tasks/:id/move":          "reorder task",
	"PUT /api/v1/tasks/:id/follow":         "follow task",
	"DELETE /api/v1/tasks/:id/follow":      "unfollow task",
	"POST /api/v1/auth/password":           "change password",
	"POST /api/v1/project-hooks/:id/test":  "test hook",
	"PUT /api/v1/me/notification-settings": "update notification settings",
	"PUT /api/v1/sprints/:id/capacity":     "update sprint capacity",
}

var irregular = map[string]string{"people": "person", "queries": "query", "activities": "activity", "capacity": "capacity", "settings": "settings"}

func singular(s string) string {
	if v, ok := irregular[s]; ok {
		return v
	}
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	return strings.TrimSuffix(s, "s")
}

var paramSeg = regexp.MustCompile(`^:`)

// auditName turns a route into "create task", "update project hook" …
func auditName(method, route string) (action, resource string) {
	segs := strings.Split(strings.TrimPrefix(route, "/api/v1/"), "/")
	res := ""
	for i := len(segs) - 1; i >= 0; i-- {
		if !paramSeg.MatchString(segs[i]) && segs[i] != "me" {
			res = strings.ReplaceAll(singular(segs[i]), "-", " ")
			break
		}
	}
	if name, ok := auditNames[method+" "+route]; ok {
		return name, res
	}
	verb := map[string]string{http.MethodPost: "create", http.MethodPatch: "update", http.MethodPut: "update", http.MethodDelete: "delete"}[method]
	return verb + " " + res, res
}

// auditTrail records every successful change made through the API: who,
// what, which fields (never values), and whether by session or token.
func (h *Handler) auditTrail() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
			c.Next()
			return
		}
		var fields []string
		if strings.HasPrefix(c.ContentType(), "application/json") && c.Request.Body != nil {
			body, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			var obj map[string]json.RawMessage
			if json.Unmarshal(body, &obj) == nil {
				for k := range obj {
					fields = append(fields, k)
				}
			}
		}
		c.Next()
		route := c.FullPath()
		if c.Writer.Status() >= 400 || route == "" || auditSkip[route] {
			return
		}
		sortStrings(fields)
		action, resource := auditName(m, route)
		var id *string
		if p := c.Param("id"); p != "" {
			id = &p
		}
		u, _ := currentUser(c)
		actor := u.Email
		if actor == "" {
			actor = "system"
		}
		h.audit(c, &u, actor, action, resource, id, fields)
	}
}

// ListAudit implements GET /api/v1/audit.
func (h *Handler) ListAudit(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	q := h.db.Model(&auditEntry{})
	if r := c.Query("resource"); r != "" {
		q = q.Where("resource = ?", r)
	}
	if t := strings.TrimSpace(c.Query("q")); t != "" {
		l := likeEsc(t)
		q = q.Where(`(LOWER(actor) LIKE ? ESCAPE '\' OR LOWER(action) LIKE ? ESCAPE '\')`, l, l)
	}
	var total int64
	q.Count(&total)
	var rows []auditEntry
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows)
	out := make([]api.AuditEntry, len(rows))
	for i, e := range rows {
		var f []string
		_ = json.Unmarshal([]byte(e.Fields), &f)
		if f == nil {
			f = []string{}
		}
		out[i] = api.AuditEntry{Id: e.ID, At: e.CreatedAt, Actor: e.Actor, Action: e.Action, Resource: e.Resource, EntityId: e.EntityID, Fields: f,
			Via: api.AuditEntryVia(e.Via), Ip: e.IP}
	}
	c.JSON(http.StatusOK, api.AuditList{Data: out, Total: int(total)})
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
