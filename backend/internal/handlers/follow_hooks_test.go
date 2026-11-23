package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestFollowAndNotificationSettings(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")
	bobID := personOf(t, r, bob)

	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "Watch me"}, amy, &task)
	path := "/api/v1/tasks/" + task.Id.String()
	var f api.TaskFollowers
	if s, _ := call(t, r, http.MethodPut, path+"/follow", nil, bob, &f); s != http.StatusOK || !f.Following || len(f.PersonIds) != 1 || f.PersonIds[0] != bobID {
		t.Fatalf("follow = %d %+v", s, f)
	}
	call(t, r, http.MethodPatch, path, map[string]any{"title": "Watch me now"}, amy, nil)
	l := inbox(t, r, bob)
	if l.UnreadCount != 1 || l.Data[0].Type != api.NotificationTypeFollow || !strings.Contains(l.Data[0].Title, "Watch me now") {
		t.Fatalf("bob after change = %+v", l.Data)
	}
	call(t, r, http.MethodPost, path+"/comments", map[string]any{"body": "hello"}, amy, nil)
	if n := inbox(t, r, bob).UnreadCount; n != 2 {
		t.Errorf("bob after comment = %d, want 2", n)
	}
	call(t, r, http.MethodGet, path+"/followers", nil, amy, &f)
	if f.Following || len(f.PersonIds) != 1 {
		t.Errorf("amy's view = %+v", f)
	}
	call(t, r, http.MethodDelete, path+"/follow", nil, bob, &f)
	call(t, r, http.MethodPatch, path, map[string]any{"title": "Quiet"}, amy, nil)
	if n := inbox(t, r, bob).UnreadCount; n != 2 {
		t.Errorf("bob after unfollow = %d, want 2", n)
	}

	// Settings: bob turns off in-app assignment notifications.
	var st api.NotificationSettings
	call(t, r, http.MethodGet, "/api/v1/me/notification-settings", nil, bob, &st)
	if st.Types[string(api.NotificationTypeAssignment)].InApp != true {
		t.Errorf("default settings = %+v", st)
	}
	off := api.NotificationSettings{Types: map[string]api.NotificationChannels{"assignment": {InApp: false, Email: true}}}
	if s, _ := call(t, r, http.MethodPut, "/api/v1/me/notification-settings", off, bob, &st); s != http.StatusOK || st.Types["assignment"].InApp {
		t.Fatalf("save settings = %d %+v", s, st)
	}
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "For bob", AssigneeIds: &[]uuid.UUID{bobID}}, amy, nil)
	if n := inbox(t, r, bob).UnreadCount; n != 2 {
		t.Errorf("bob got an assignment notification he turned off (%d)", n)
	}
	bad := api.NotificationSettings{Types: map[string]api.NotificationChannels{"weather": {InApp: true, Email: true}}}
	if s, _ := call(t, r, http.MethodPut, "/api/v1/me/notification-settings", bad, bob, nil); s != http.StatusBadRequest {
		t.Errorf("unknown type = %d", s)
	}
}

type hookSink struct {
	mu   sync.Mutex
	reqs []struct {
		headers http.Header
		body    []byte
	}
}

func (s *hookSink) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.reqs = append(s.reqs, struct {
		headers http.Header
		body    []byte
	}{r.Header.Clone(), b})
	s.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *hookSink) wait(t *testing.T, n int) []struct {
	headers http.Header
	body    []byte
} {
	t.Helper()
	for i := 0; i < 200; i++ {
		s.mu.Lock()
		if len(s.reqs) >= n {
			out := append(s.reqs[:0:0], s.reqs...)
			s.mu.Unlock()
			return out
		}
		s.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %d hook calls", n)
	return nil
}

func TestProjectHooks(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	var p api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "HK", Name: "Hooks"}, admin, &p)

	web, slack := &hookSink{}, &hookSink{}
	ws, ss := httptest.NewServer(web), httptest.NewServer(slack)
	defer ws.Close()
	defer ss.Close()

	in := api.ProjectHookInput{ProjectId: p.Id, Kind: api.ProjectHookInputKindWebhook, Name: "CI", Url: ws.URL, Secret: ptrTo("s3cret"),
		Events: []api.HookEvent{api.HookEventItemCreated, api.HookEventItemStateChanged}}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-hooks", in, amy, nil); s != http.StatusForbidden {
		t.Errorf("member adds hook = %d, want 403", s)
	}
	var hook api.ProjectHook
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-hooks", in, admin, &hook); s != http.StatusCreated || !hook.HasSecret {
		t.Fatalf("add = %d %+v", s, hook)
	}
	call(t, r, http.MethodPost, "/api/v1/project-hooks", api.ProjectHookInput{ProjectId: p.Id, Kind: api.ProjectHookInputKindSlack, Name: "Team chat", Url: ss.URL,
		Events: []api.HookEvent{api.HookEventItemCreated}}, admin, nil)

	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: p.Id, Title: "Hooked"}, amy, &task)
	got := web.wait(t, 1)[0]
	if got.headers.Get("X-Pyramid-Event") != "item_created" {
		t.Errorf("event header = %q", got.headers.Get("X-Pyramid-Event"))
	}
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(got.body)
	if got.headers.Get("X-Pyramid-Signature") != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
		t.Errorf("bad signature")
	}
	var payload map[string]any
	_ = json.Unmarshal(got.body, &payload)
	if payload["event"] != "item_created" || payload["item"].(map[string]any)["title"] != "Hooked" {
		t.Errorf("payload = %s", got.body)
	}
	var msg map[string]any
	_ = json.Unmarshal(slack.wait(t, 1)[0].body, &msg)
	if text, _ := msg["text"].(string); !strings.Contains(text, "Hooked") {
		t.Errorf("slack message = %v", msg)
	}

	// A title change isn't subscribed; a state change is.
	call(t, r, http.MethodPatch, "/api/v1/tasks/"+task.Id.String(), map[string]any{"title": "Hooked 2"}, amy, nil)
	call(t, r, http.MethodPatch, "/api/v1/tasks/"+task.Id.String(), map[string]any{"state": "in_progress"}, amy, nil)
	calls := web.wait(t, 2)
	if calls[1].headers.Get("X-Pyramid-Event") != "item_state_changed" {
		t.Errorf("second call = %s", calls[1].headers.Get("X-Pyramid-Event"))
	}

	var tested api.ProjectHook
	if s, _ := call(t, r, http.MethodPost, "/api/v1/project-hooks/"+hook.Id.String()+"/test", nil, admin, &tested); s != http.StatusOK || tested.LastStatus == nil || *tested.LastStatus != 200 {
		t.Errorf("test = %d %+v", s, tested)
	}
	var list api.ProjectHookList
	call(t, r, http.MethodGet, "/api/v1/project-hooks?projectId="+p.Id.String(), nil, admin, &list)
	if len(list.Data) != 2 {
		t.Errorf("list = %+v", list)
	}
	for name, bad := range map[string]api.ProjectHookInput{
		"bad url":    {ProjectId: p.Id, Kind: api.ProjectHookInputKindSlack, Name: "x", Url: "ftp://x", Events: []api.HookEvent{api.HookEventItemCreated}},
		"no events":  {ProjectId: p.Id, Kind: api.ProjectHookInputKindSlack, Name: "x", Url: ss.URL, Events: []api.HookEvent{}},
		"bad event":  {ProjectId: p.Id, Kind: api.ProjectHookInputKindSlack, Name: "x", Url: ss.URL, Events: []api.HookEvent{"party"}},
		"bad kind":   {ProjectId: p.Id, Kind: "pager", Name: "x", Url: ss.URL, Events: []api.HookEvent{api.HookEventItemCreated}},
		"no project": {ProjectId: uuid.New(), Kind: api.ProjectHookInputKindSlack, Name: "x", Url: ss.URL, Events: []api.HookEvent{api.HookEventItemCreated}},
	} {
		if s, _ := call(t, r, http.MethodPost, "/api/v1/project-hooks", bad, admin, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}
