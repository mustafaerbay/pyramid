package handlers

import (
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Notifications go to accounts (users). Who did the thing never gets told
// about it. When SMTP is set up each one is also emailed, in the
// background so a slow mail server never slows the app down.

const notificationsShown = 50

// oneLine keeps user text (task titles…) from breaking mail headers.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(s)), " ")
}

func smtpSender(cfg config.Config) func(to, subject, body string) error {
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return nil
	}
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil {
		log.Printf("SMTP_FROM %q is not an email address; email is off", cfg.SMTPFrom)
		return nil
	}
	addr := net.JoinHostPort(cfg.SMTPHost, cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}
	return func(to, subject, body string) error {
		msg := "From: " + from.String() + "\r\n" +
			"To: " + oneLine(to) + "\r\n" +
			"Subject: " + mime.QEncoding.Encode("utf-8", oneLine(subject)) + "\r\n" +
			"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" +
			strings.ReplaceAll(body, "\n", "\r\n")
		return smtp.SendMail(addr, auth, from.Address, []string{oneLine(to)}, []byte(msg))
	}
}

// actor is the signed-in person's id and name (zero id when signed out).
func (h *Handler) actor(c *gin.Context) (uuid.UUID, string) {
	u, ok := currentUser(c)
	if !ok {
		return uuid.Nil, "Someone"
	}
	var p models.Person
	if err := h.db.First(&p, "id = ?", u.PersonID).Error; err != nil {
		return u.PersonID, "Someone"
	}
	return u.PersonID, strings.TrimSpace(p.FirstName + " " + p.LastName)
}

// notify tells the accounts of `people` (minus whoever did it). Best
// effort: errors are logged, never returned to the caller's request.
func (h *Handler) notify(c *gin.Context, people []uuid.UUID, typ api.NotificationType, title string, body *string, link string) {
	actorID, _ := h.actor(c)
	seen := map[uuid.UUID]bool{actorID: true}
	ids := make([]uuid.UUID, 0, len(people))
	for _, p := range people {
		if !seen[p] {
			seen[p] = true
			ids = append(ids, p)
		}
	}
	if len(ids) == 0 {
		return
	}
	var users []models.User
	if err := h.db.Where("person_id IN ? AND is_active = ?", ids, true).Find(&users).Error; err != nil {
		log.Printf("notify: %v", err)
		return
	}
	for _, u := range users {
		n := models.Notification{UserID: u.ID, Type: typ, Title: title, Body: body}
		if link != "" {
			n.LinkURL = &link
		}
		if err := h.db.Create(&n).Error; err != nil {
			log.Printf("notify: %v", err)
			continue
		}
		if h.mail != nil {
			text := title + "\n"
			if body != nil {
				text += "\n" + *body + "\n"
			}
			if link != "" {
				text += "\n" + strings.TrimRight(h.cfg.PublicURL, "/") + link + "\n"
			}
			go func(to, subject, text string) {
				if err := h.mail(to, subject, text); err != nil {
					log.Printf("notification email to %s failed: %v", to, err)
				}
			}(u.Email, title, text)
		}
	}
}

func taskLink(t models.Task) string { return fmt.Sprintf("/projects/%s?task=%s", t.ProjectID, t.ID) }
func taskRef(t models.Task) string  { return fmt.Sprintf("#%d %s", t.Number, oneLine(t.Title)) }

func excerpt(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// notifyTaskChange covers assignments and status changes after a task was
// created (before == nil) or updated.
func (h *Handler) notifyTaskChange(c *gin.Context, before, after map[string]any, t models.Task) {
	_, who := h.actor(c)
	had := map[string]bool{}
	if before != nil {
		for id := range before["assignments"].(map[string]float64) {
			had[id] = true
		}
	}
	var added, kept []uuid.UUID
	for _, id := range t.AssigneeIDs {
		if had[id.String()] {
			kept = append(kept, id)
		} else {
			added = append(added, id)
		}
	}
	h.notify(c, added, api.NotificationTypeAssignment, fmt.Sprintf("%s assigned you to %s", who, taskRef(t)), nil, taskLink(t))
	// People newly @mentioned in the description.
	desc := func(m map[string]any) string {
		if m == nil {
			return ""
		}
		v, _ := m["description"].(string)
		return v
	}
	now := ""
	if t.Description != nil {
		now = *t.Description
	}
	if mentions := h.mentioned(now); len(mentions) > 0 {
		was := map[uuid.UUID]bool{}
		for _, id := range h.mentioned(desc(before)) {
			was[id] = true
		}
		var fresh []uuid.UUID
		for _, id := range mentions {
			if !was[id] {
				fresh = append(fresh, id)
			}
		}
		body := excerpt(now, 300)
		h.notify(c, fresh, api.NotificationTypeMention, fmt.Sprintf("%s mentioned you in %s", who, taskRef(t)), &body, taskLink(t))
	}
	if before != nil && before["status"] != after["status"] {
		h.notify(c, kept, api.NotificationTypeStatusChange,
			fmt.Sprintf("%s moved %s to %s", who, taskRef(t), strings.ReplaceAll(string(t.Status), "_", " ")), nil, taskLink(t))
	}
}

// mentioned finds people with an account written as "@First Last" in text.
func (h *Handler) mentioned(text string) []uuid.UUID {
	if !strings.Contains(text, "@") {
		return nil
	}
	var rows []struct {
		ID        uuid.UUID
		FirstName string
		LastName  string
	}
	h.db.Table("people p").Select("p.id, p.first_name, p.last_name").
		Joins("JOIN users u ON u.person_id = p.id AND u.is_active = ?", true).Scan(&rows)
	lower := strings.ToLower(text)
	var out []uuid.UUID
	for _, r := range rows {
		if strings.Contains(lower, "@"+strings.ToLower(strings.TrimSpace(r.FirstName+" "+r.LastName))) {
			out = append(out, r.ID)
		}
	}
	return out
}

func (h *Handler) notifyComment(c *gin.Context, t models.Task, cm models.TaskComment) {
	if err := h.attachAssigneesOne(&t); err != nil {
		log.Printf("notify comment: %v", err)
		return
	}
	_, who := h.actor(c)
	people := append(append([]uuid.UUID{}, t.AssigneeIDs...), h.mentioned(cm.Body)...)
	body := excerpt(cm.Body, 300)
	h.notify(c, people, api.NotificationTypeMention, fmt.Sprintf("%s commented on %s", who, taskRef(t)), &body, taskLink(t))
}

func leaveSpan(l models.LeaveRequest) string {
	return fmt.Sprintf("%s – %s", l.StartDate.Format("2 Jan"), l.EndDate.Format("2 Jan 2006"))
}

// notifyLeaveRequested tells managers and admins someone asked for leave.
func (h *Handler) notifyLeaveRequested(c *gin.Context, l models.LeaveRequest) {
	var p models.Person
	h.db.First(&p, "id = ?", l.PersonID)
	var approvers []uuid.UUID
	h.db.Model(&models.User{}).Where("system_role IN ? AND is_active = ?", []string{"admin", "manager"}, true).Pluck("person_id", &approvers)
	h.notify(c, approvers, api.NotificationTypeApproval,
		fmt.Sprintf("%s %s asked for %s leave, %s", p.FirstName, p.LastName, l.Type, leaveSpan(l)), nil, "/time-off")
}

func (h *Handler) notifyLeaveDecided(c *gin.Context, l models.LeaveRequest) {
	h.notify(c, []uuid.UUID{l.PersonID}, api.NotificationTypeApproval,
		fmt.Sprintf("Your %s leave, %s, was %s", l.Type, leaveSpan(l), l.Status), nil, "/time-off")
}

// --- endpoints ---

// ListNotifications implements GET /api/v1/notifications.
func (h *Handler) ListNotifications(c *gin.Context) {
	u, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusOK, api.NotificationList{Data: []api.Notification{}})
		return
	}
	var rows []models.Notification
	if err := h.db.Where("user_id = ?", u.ID).Order("created_at DESC").Limit(notificationsShown).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load notifications", Details: strptr(err.Error())})
		return
	}
	var unread int64
	h.db.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", u.ID).Count(&unread)
	out := make([]api.Notification, len(rows))
	for i, n := range rows {
		out[i] = n.ToAPI()
	}
	c.JSON(http.StatusOK, api.NotificationList{Data: out, UnreadCount: int(unread)})
}

// MarkNotificationRead implements POST /api/v1/notifications/{id}/read.
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	u, _ := currentUser(c)
	res := h.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", c.Param("id"), u.ID).
		Update("read_at", time.Now().UTC())
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to update", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		var n int64
		h.db.Model(&models.Notification{}).Where("id = ? AND user_id = ?", c.Param("id"), u.ID).Count(&n)
		if n == 0 {
			c.JSON(http.StatusNotFound, api.Error{Message: "notification not found"})
			return
		}
	}
	c.Status(http.StatusNoContent)
}

// MarkAllNotificationsRead implements POST /api/v1/notifications/read-all.
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	u, _ := currentUser(c)
	if err := h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", u.ID).
		Update("read_at", time.Now().UTC()).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to update", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
