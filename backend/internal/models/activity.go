package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// TaskComment is a comment on a task (task_comments). AuthorID is a person.
type TaskComment struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TaskID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	AuthorID  *uuid.UUID `gorm:"type:uuid"`
	Body      string     `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TaskComment) TableName() string { return "task_comments" }

func (c TaskComment) ToAPI() api.TaskComment {
	return api.TaskComment{
		Id: c.ID, TaskId: c.TaskID, AuthorPersonId: c.AuthorID, Body: c.Body,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// ActivityLog is one entry in the audit trail (activity_log). Changes is
// JSON: {"field": {"from": …, "to": …}}. ActorID is a user (account).
type ActivityLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ActorID    *uuid.UUID `gorm:"type:uuid"`
	Action     string     `gorm:"not null"`
	EntityType string     `gorm:"not null"`
	EntityID   *uuid.UUID `gorm:"type:uuid"`
	Changes    *string    `gorm:"type:jsonb"`
	CreatedAt  time.Time
}

func (ActivityLog) TableName() string { return "activity_log" }

// Notification is one message for one account (notifications table).
type Notification struct {
	ID        uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID            `gorm:"type:uuid;not null;index"`
	Type      api.NotificationType `gorm:"type:notification_type;not null"`
	Title     string               `gorm:"not null"`
	Body      *string
	LinkURL   *string `gorm:"column:link_url"`
	ReadAt    *time.Time
	CreatedAt time.Time
}

func (Notification) TableName() string { return "notifications" }

func (n Notification) ToAPI() api.Notification {
	return api.Notification{
		Id: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Link: n.LinkURL,
		ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
	}
}
