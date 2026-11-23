package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Project hooks send Slack / Teams messages or signed JSON webhooks when
// things happen in a project. Delivery is in the background with a short
// timeout; the last result is kept on the hook.

type projectHook struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID  uuid.UUID `gorm:"type:uuid;not null"`
	Kind       string    `gorm:"not null"`
	Name       string    `gorm:"not null"`
	URL        string    `gorm:"column:url;not null"`
	Secret     *string
	Events     string `gorm:"type:jsonb;not null"`
	Active     bool   `gorm:"not null"`
	LastStatus *int
	LastError  *string
	LastAt     *time.Time
	CreatedAt  time.Time
}

func (projectHook) TableName() string { return "project_hooks" }

func (k projectHook) events() []api.HookEvent {
	var out []api.HookEvent
	_ = json.Unmarshal([]byte(k.Events), &out)
	return out
}

func (k projectHook) toAPI() api.ProjectHook {
	ev := k.events()
	if ev == nil {
		ev = []api.HookEvent{}
	}
	return api.ProjectHook{
		Id: k.ID, ProjectId: k.ProjectID, Kind: api.ProjectHookKind(k.Kind), Name: k.Name, Url: k.URL,
		HasSecret: k.Secret != nil && *k.Secret != "", Events: ev, Active: k.Active,
		LastStatus: k.LastStatus, LastError: k.LastError, LastAt: k.LastAt, CreatedAt: k.CreatedAt,
	}
}

func validHookURL(s string) bool {
	if len(s) > 2000 {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

func (h *Handler) readHookInput(c *gin.Context, k *projectHook) bool {
	var in api.ProjectHookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return false
	}
	fail := func(msg string) bool {
		c.JSON(http.StatusBadRequest, api.Error{Message: msg})
		return false
	}
	var n int64
	h.db.Model(&models.Project{}).Where("id = ?", in.ProjectId).Count(&n)
	if n == 0 {
		return fail("unknown project")
	}
	if k.ID != uuid.Nil && in.ProjectId != k.ProjectID {
		return fail("a hook stays in its project")
	}
	if !in.Kind.Valid() {
		return fail("a hook is slack, teams or webhook")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return fail("a hook needs a name of 1–100 characters")
	}
	u := strings.TrimSpace(in.Url)
	if !validHookURL(u) {
		return fail("the URL must start with https:// (or http://)")
	}
	if len(in.Events) == 0 {
		return fail("pick at least one event")
	}
	for _, e := range in.Events {
		if !e.Valid() {
			return fail(fmt.Sprintf("unknown event %q", e))
		}
	}
	k.ProjectID, k.Kind, k.Name, k.URL = in.ProjectId, string(in.Kind), name, u
	b, _ := json.Marshal(in.Events)
	k.Events = string(b)
	if in.Secret != nil {
		if *in.Secret == "" {
			k.Secret = nil
		} else {
			s := *in.Secret
			k.Secret = &s
		}
	}
	if in.Kind != api.ProjectHookInputKindWebhook {
		k.Secret = nil
	}
	k.Active = true
	if in.Active != nil {
		k.Active = *in.Active
	}
	return true
}

// ListProjectHooks implements GET /api/v1/project-hooks.
func (h *Handler) ListProjectHooks(c *gin.Context) {
	var rows []projectHook
	if err := h.db.Where("project_id = ?", c.Query("projectId")).Order("created_at").Find(&rows).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to list hooks", Details: strptr(err.Error())})
		return
	}
	out := make([]api.ProjectHook, len(rows))
	for i, k := range rows {
		out[i] = k.toAPI()
	}
	c.JSON(http.StatusOK, api.ProjectHookList{Data: out})
}

// CreateProjectHook implements POST /api/v1/project-hooks.
func (h *Handler) CreateProjectHook(c *gin.Context) {
	var k projectHook
	if !h.readHookInput(c, &k) {
		return
	}
	if err := h.db.Create(&k).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save hook", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, k.toAPI())
}

func (h *Handler) loadHook(c *gin.Context) (projectHook, bool) {
	var k projectHook
	if err := h.db.First(&k, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "hook not found"})
		return k, false
	}
	return k, true
}

// UpdateProjectHook implements PATCH /api/v1/project-hooks/{id}.
func (h *Handler) UpdateProjectHook(c *gin.Context) {
	k, ok := h.loadHook(c)
	if !ok || !h.readHookInput(c, &k) {
		return
	}
	if err := h.db.Save(&k).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save hook", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, k.toAPI())
}

// DeleteProjectHook implements DELETE /api/v1/project-hooks/{id}.
func (h *Handler) DeleteProjectHook(c *gin.Context) {
	k, ok := h.loadHook(c)
	if !ok {
		return
	}
	h.db.Delete(&k)
	c.Status(http.StatusNoContent)
}

// TestProjectHook implements POST /api/v1/project-hooks/{id}/test.
func (h *Handler) TestProjectHook(c *gin.Context) {
	k, ok := h.loadHook(c)
	if !ok {
		return
	}
	var p models.Project
	h.db.First(&p, "id = ?", k.ProjectID)
	_, who := h.actor(c)
	h.deliver(k, "test", hookPayload{Event: "test", Project: projectRef(p), Actor: who, At: time.Now().UTC()},
		fmt.Sprintf("[%s] Test message from Pyramid, sent by %s", p.Key, who))
	h.db.First(&k, "id = ?", k.ID)
	c.JSON(http.StatusOK, k.toAPI())
}

// --- sending ---

type hookProject struct {
	ID   uuid.UUID `json:"id"`
	Key  string    `json:"key"`
	Name string    `json:"name"`
}

type hookPayload struct {
	Event   string           `json:"event"`
	Project hookProject      `json:"project"`
	Actor   string           `json:"actor"`
	At      time.Time        `json:"at"`
	Item    *api.Task        `json:"item,omitempty"`
	Sprint  *api.Sprint      `json:"sprint,omitempty"`
	Comment *api.TaskComment `json:"comment,omitempty"`
	Changes []string         `json:"changes,omitempty"`
	URL     string           `json:"url,omitempty"`
}

func projectRef(p models.Project) hookProject { return hookProject{ID: p.ID, Key: p.Key, Name: p.Name} }

// emit sends an event to the project's active hooks that asked for it.
func (h *Handler) emit(c *gin.Context, projectID uuid.UUID, event api.HookEvent, pl hookPayload, text string) {
	var hooks []projectHook
	if err := h.db.Where("project_id = ? AND active = ?", projectID, true).Find(&hooks).Error; err != nil || len(hooks) == 0 {
		return
	}
	var p models.Project
	if h.db.First(&p, "id = ?", projectID).Error != nil {
		return
	}
	_, who := h.actor(c)
	pl.Event, pl.Project, pl.Actor, pl.At = string(event), projectRef(p), who, time.Now().UTC()
	if pl.URL != "" {
		pl.URL = strings.TrimRight(h.cfg.PublicURL, "/") + pl.URL
	}
	msg := fmt.Sprintf("[%s] %s %s", p.Key, who, text)
	if pl.URL != "" {
		msg += " — " + pl.URL
	}
	for _, k := range hooks {
		for _, e := range k.events() {
			if e == event {
				go h.deliver(k, string(event), pl, msg)
				break
			}
		}
	}
}

var hookClient = &http.Client{Timeout: 5 * time.Second}

// deliver posts one event to one hook and records how it went.
func (h *Handler) deliver(k projectHook, event string, pl hookPayload, text string) {
	var body []byte
	switch k.Kind {
	case string(api.ProjectHookKindSlack), string(api.ProjectHookKindTeams):
		body, _ = json.Marshal(map[string]string{"text": text})
	default:
		body, _ = json.Marshal(pl)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.URL, bytes.NewReader(body))
	status := 0
	var errText *string
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Pyramid-Hooks/1")
		req.Header.Set("X-Pyramid-Event", event)
		if k.Secret != nil && *k.Secret != "" {
			mac := hmac.New(sha256.New, []byte(*k.Secret))
			mac.Write(body)
			req.Header.Set("X-Pyramid-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
		var res *http.Response
		res, err = hookClient.Do(req)
		if err == nil {
			status = res.StatusCode
			res.Body.Close()
			if status >= 300 {
				err = errors.New(res.Status)
			}
		}
	}
	if err != nil {
		s := err.Error()
		errText = &s
		log.Printf("hook %s (%s): %v", k.Name, k.ID, err)
	}
	now := time.Now()
	h.db.Model(&projectHook{}).Where("id = ?", k.ID).Updates(map[string]any{"last_status": status, "last_error": errText, "last_at": now})
}
