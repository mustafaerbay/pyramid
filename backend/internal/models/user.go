package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// User is a sign-in account (users table), always tied to one person.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PersonID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Email        string    `gorm:"not null"`
	PasswordHash *string
	SystemRole   api.SystemRole `gorm:"type:system_role;not null;default:member"`
	IsActive     bool           `gorm:"not null;default:true"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string { return "users" }

func (u User) ToAPI() api.User {
	return api.User{
		Id:          u.ID,
		PersonId:    u.PersonID,
		Email:       u.Email,
		Role:        u.SystemRole,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

// RoleRank orders roles from least (viewer) to most (admin) access.
func RoleRank(r api.SystemRole) int {
	switch r {
	case api.SystemRoleAdmin:
		return 3
	case api.SystemRoleManager:
		return 2
	case api.SystemRoleMember:
		return 1
	}
	return 0
}
