package handlers

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

type sentMail struct{ to, subject, body string }

type mailbox struct {
	mu   sync.Mutex
	sent []sentMail
}

func (m *mailbox) send(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMail{to, subject, body})
	return nil
}

// waitFor polls until n mails arrived (sending runs in the background).
func (m *mailbox) waitFor(t *testing.T, n int) []sentMail {
	t.Helper()
	for i := 0; i < 100; i++ {
		m.mu.Lock()
		got := append([]sentMail(nil), m.sent...)
		m.mu.Unlock()
		if len(got) >= n {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %d mails", n)
	return nil
}

func inbox(t *testing.T, r *gin.Engine, c *http.Cookie) api.NotificationList {
	t.Helper()
	var l api.NotificationList
	if s, _ := call(t, r, http.MethodGet, "/api/v1/notifications", nil, c, &l); s != http.StatusOK {
		t.Fatalf("notifications: %d", s)
	}
	return l
}

func personOf(t *testing.T, r *gin.Engine, c *http.Cookie) uuid.UUID {
	t.Helper()
	var me api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, c, &me)
	return me.Person.Id
}

func TestNotifications_TaskEvents(t *testing.T) {
	box := &mailbox{}
	r, _ := newAuthRouter(t, func(h *Handler) { h.mail = box.send })
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")
	amyID, bobID := personOf(t, r, amy), personOf(t, r, bob)

	// Assigning amy (by admin) notifies her, not the admin.
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Ship it", AssigneeIds: &[]uuid.UUID{amyID},
	}, admin, &task)
	l := inbox(t, r, amy)
	if l.UnreadCount != 1 || len(l.Data) != 1 || l.Data[0].Type != api.NotificationTypeAssignment {
		t.Fatalf("amy after assignment = %+v", l)
	}
	if !strings.Contains(l.Data[0].Title, "Ship it") || l.Data[0].Link == nil || !strings.Contains(*l.Data[0].Link, task.Id.String()) {
		t.Errorf("assignment notification = %+v", l.Data[0])
	}
	if n := inbox(t, r, admin).UnreadCount; n != 0 {
		t.Errorf("admin got %d notifications for their own change", n)
	}
	mails := box.waitFor(t, 1)
	if mails[0].to != "amy@x.dev" || !strings.Contains(mails[0].subject, "Ship it") {
		t.Errorf("mail = %+v", mails[0])
	}

	path := "/api/v1/tasks/" + task.Id.String()
	// amy moving her own task: nobody else is assigned, no notification.
	call(t, r, http.MethodPatch, path, map[string]any{"status": "in_progress"}, amy, nil)
	if n := inbox(t, r, amy).UnreadCount; n != 1 {
		t.Errorf("amy unread after own status change = %d, want 1", n)
	}
	// Someone else changing its status notifies her.
	call(t, r, http.MethodPatch, path, map[string]any{"status": "blocked"}, bob, nil)
	l = inbox(t, r, amy)
	if l.UnreadCount != 2 || l.Data[0].Type != api.NotificationTypeStatusChange {
		t.Errorf("amy after bob's status change = %+v", l.Data)
	}
	// Re-saving the same assignees doesn't notify again.
	call(t, r, http.MethodPatch, path, map[string]any{"assigneeIds": []uuid.UUID{amyID}}, admin, nil)
	if n := inbox(t, r, amy).UnreadCount; n != 2 {
		t.Errorf("amy unread after same assignees = %d, want 2", n)
	}

	// A comment reaches the assignee and anyone @mentioned, not its author.
	var adminMe api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, admin, &adminMe)
	call(t, r, http.MethodPost, path+"/comments", api.TaskCommentInput{Body: "Any news? cc @P bob@x.dev"}, admin, nil)
	if l = inbox(t, r, amy); l.UnreadCount != 3 || l.Data[0].Type != api.NotificationTypeMention {
		t.Errorf("amy after comment = %+v", l.Data)
	}
	if l = inbox(t, r, bob); l.UnreadCount != 1 || l.Data[0].Type != api.NotificationTypeMention {
		t.Errorf("bob after @mention = %+v", l.Data)
	}
	if n := inbox(t, r, admin).UnreadCount; n != 0 {
		t.Errorf("admin got %d for own comment", n)
	}
	_ = bobID

	// Reading.
	first := inbox(t, r, amy).Data[0].Id
	if s, _ := call(t, r, http.MethodPost, "/api/v1/notifications/"+first.String()+"/read", nil, bob, nil); s != http.StatusNotFound {
		t.Errorf("bob reads amy's notification = %d, want 404", s)
	}
	call(t, r, http.MethodPost, "/api/v1/notifications/"+first.String()+"/read", nil, amy, nil)
	if l = inbox(t, r, amy); l.UnreadCount != 2 || l.Data[0].ReadAt == nil {
		t.Errorf("after reading one: unread %d, first %+v", l.UnreadCount, l.Data[0])
	}
	call(t, r, http.MethodPost, "/api/v1/notifications/read-all", nil, amy, nil)
	if n := inbox(t, r, amy).UnreadCount; n != 0 {
		t.Errorf("after read-all unread = %d", n)
	}
}

func TestNotifications_Leave(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "vic@x.dev", api.SystemRoleViewer)
	amy := login(t, r, "amy@x.dev", "first-pass")
	vic := login(t, r, "vic@x.dev", "first-pass")

	vacation := api.LeaveTypeVacation
	var leave api.LeaveRequest
	call(t, r, http.MethodPost, "/api/v1/leave-requests", api.LeaveRequestCreate{
		PersonId: personOf(t, r, amy), Type: &vacation, StartDate: *mdDate("2026-12-01"), EndDate: *mdDate("2026-12-03"),
	}, amy, &leave)
	// Managers and admins hear about new requests; viewers don't.
	if l := inbox(t, r, admin); l.UnreadCount != 1 || l.Data[0].Type != api.NotificationTypeApproval {
		t.Errorf("admin after leave request = %+v", l.Data)
	}
	if n := inbox(t, r, vic).UnreadCount; n != 0 {
		t.Errorf("viewer got %d", n)
	}
	// Viewers can still mark their own notifications read.
	if s, _ := call(t, r, http.MethodPost, "/api/v1/notifications/read-all", nil, vic, nil); s != http.StatusNoContent {
		t.Errorf("viewer read-all = %d, want 204", s)
	}

	call(t, r, http.MethodPatch, "/api/v1/leave-requests/"+leave.Id.String(), map[string]any{"status": "approved"}, admin, nil)
	l := inbox(t, r, amy)
	if l.UnreadCount != 1 || l.Data[0].Type != api.NotificationTypeApproval || !strings.Contains(l.Data[0].Title, "approved") {
		t.Errorf("amy after approval = %+v", l.Data)
	}
}

func TestNotifications_DescriptionMentions(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")

	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Spec", Description: ptrTo("**Please** review, @P bob@x.dev"),
	}, amy, &task)
	l := inbox(t, r, bob)
	if l.UnreadCount != 1 || l.Data[0].Type != api.NotificationTypeMention || !strings.Contains(l.Data[0].Title, "mentioned you") {
		t.Fatalf("bob after mention = %+v", l.Data)
	}
	// Editing the text without a new mention doesn't notify again.
	call(t, r, http.MethodPatch, "/api/v1/tasks/"+task.Id.String(), map[string]any{"description": "Please review now, @P bob@x.dev"}, amy, nil)
	if n := inbox(t, r, bob).UnreadCount; n != 1 {
		t.Errorf("bob unread after edit = %d, want 1", n)
	}
}
