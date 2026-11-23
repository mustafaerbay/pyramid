package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm/clause"
)

type taskFollower struct {
	TaskID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	PersonID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}

func (taskFollower) TableName() string { return "task_followers" }

func (h *Handler) followersOf(taskID uuid.UUID) []uuid.UUID {
	var ids []uuid.UUID
	h.db.Model(&taskFollower{}).Where("task_id = ?", taskID).Order("created_at").Pluck("person_id", &ids)
	return ids
}

func (h *Handler) followersJSON(c *gin.Context, taskID uuid.UUID) api.TaskFollowers {
	me, _ := currentUser(c)
	ids := h.followersOf(taskID)
	out := api.TaskFollowers{PersonIds: make([]uuid.UUID, 0, len(ids))}
	for _, id := range ids {
		out.PersonIds = append(out.PersonIds, id)
		if id == me.PersonID {
			out.Following = true
		}
	}
	return out
}

func (h *Handler) followTarget(c *gin.Context) (models.Task, bool) {
	var t models.Task
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
		return t, false
	}
	return t, true
}

// GetTaskFollowers implements GET /api/v1/tasks/{id}/followers.
func (h *Handler) GetTaskFollowers(c *gin.Context) {
	if t, ok := h.followTarget(c); ok {
		c.JSON(http.StatusOK, h.followersJSON(c, t.ID))
	}
}

// FollowTask implements PUT /api/v1/tasks/{id}/follow.
func (h *Handler) FollowTask(c *gin.Context) {
	t, ok := h.followTarget(c)
	if !ok {
		return
	}
	me, signedIn := currentUser(c)
	if !signedIn {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&taskFollower{TaskID: t.ID, PersonID: me.PersonID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to follow", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, h.followersJSON(c, t.ID))
}

// UnfollowTask implements DELETE /api/v1/tasks/{id}/follow.
func (h *Handler) UnfollowTask(c *gin.Context) {
	t, ok := h.followTarget(c)
	if !ok {
		return
	}
	me, _ := currentUser(c)
	h.db.Where("task_id = ? AND person_id = ?", t.ID, me.PersonID).Delete(&taskFollower{})
	c.JSON(http.StatusOK, h.followersJSON(c, t.ID))
}

// --- notification settings ---

var notificationTypes = []api.NotificationType{
	api.NotificationTypeMention, api.NotificationTypeAssignment, api.NotificationTypeStatusChange, api.NotificationTypeReviewRequest,
	api.NotificationTypeApproval, api.NotificationTypeDeadline, api.NotificationTypeSystem, api.NotificationTypeFollow,
}

// prefsOf reads a user's notification settings with every type filled in.
func (h *Handler) prefsOf(userID uuid.UUID) map[string]api.NotificationChannels {
	var raw string
	h.db.Table("users").Select("COALESCE(notification_prefs, '{}')").Where("id = ?", userID).Row().Scan(&raw)
	saved := map[string]api.NotificationChannels{}
	_ = json.Unmarshal([]byte(raw), &saved)
	out := map[string]api.NotificationChannels{}
	for _, t := range notificationTypes {
		v, ok := saved[string(t)]
		if !ok {
			v = api.NotificationChannels{InApp: true, Email: true}
		}
		out[string(t)] = v
	}
	return out
}

// GetNotificationSettings implements GET /api/v1/me/notification-settings.
func (h *Handler) GetNotificationSettings(c *gin.Context) {
	me, _ := currentUser(c)
	c.JSON(http.StatusOK, api.NotificationSettings{Types: h.prefsOf(me.ID)})
}

// PutNotificationSettings implements PUT /api/v1/me/notification-settings.
func (h *Handler) PutNotificationSettings(c *gin.Context) {
	me, ok := currentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.Error{Message: "please sign in"})
		return
	}
	var in api.NotificationSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	merged := h.prefsOf(me.ID)
	for k, v := range in.Types {
		if !api.NotificationType(k).Valid() {
			c.JSON(http.StatusBadRequest, api.Error{Message: "unknown notification type " + k})
			return
		}
		merged[k] = v
	}
	b, _ := json.Marshal(merged)
	if err := h.db.Table("users").Where("id = ?", me.ID).Update("notification_prefs", string(b)).Error; err != nil {
		log.Printf("notification settings: %v", err)
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to save settings"})
		return
	}
	c.JSON(http.StatusOK, api.NotificationSettings{Types: merged})
}
