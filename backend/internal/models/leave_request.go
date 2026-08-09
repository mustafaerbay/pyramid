package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// LeaveRequest is the GORM persistence model for the `leave_requests` table
// defined in db/schema.sql.
type LeaveRequest struct {
	ID         uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PersonID   uuid.UUID       `gorm:"type:uuid;not null"`
	Type       api.LeaveType   `gorm:"type:leave_type;not null;default:vacation"`
	Status     api.LeaveStatus `gorm:"type:leave_status;not null;default:pending"`
	StartDate  time.Time       `gorm:"type:date;not null"`
	EndDate    time.Time       `gorm:"type:date;not null"`
	Reason     *string         `gorm:"type:text"`
	ApproverID *uuid.UUID      `gorm:"type:uuid"`
	DecidedAt  *time.Time      `gorm:"type:timestamptz"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName pins the model to the existing pluralized table.
func (LeaveRequest) TableName() string { return "leave_requests" }

// ToAPI converts the persistence model into the generated API DTO.
func (l LeaveRequest) ToAPI() api.LeaveRequest {
	return api.LeaveRequest{
		Id:         l.ID,
		PersonId:   l.PersonID,
		Type:       l.Type,
		Status:     l.Status,
		StartDate:  openapi_types.Date{Time: l.StartDate},
		EndDate:    openapi_types.Date{Time: l.EndDate},
		Reason:     l.Reason,
		ApproverId: l.ApproverID,
		CreatedAt:  l.CreatedAt,
		UpdatedAt:  l.UpdatedAt,
	}
}

// NewLeaveRequestFromCreate builds a new LeaveRequest from a create request,
// applying the same defaults the database would use.
func NewLeaveRequestFromCreate(in api.LeaveRequestCreate) LeaveRequest {
	l := LeaveRequest{
		PersonID:  in.PersonId,
		StartDate: in.StartDate.Time,
		EndDate:   in.EndDate.Time,
		Reason:    in.Reason,
		Type:      api.LeaveTypeVacation,
		Status:    api.LeaveStatusPending,
	}
	if in.Type != nil {
		l.Type = *in.Type
	}
	return l
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
func (l *LeaveRequest) ApplyUpdate(in api.LeaveRequestUpdate) {
	if in.PersonId != nil {
		l.PersonID = *in.PersonId
	}
	if in.Type != nil {
		l.Type = *in.Type
	}
	if in.Status != nil {
		l.Status = *in.Status
	}
	if in.StartDate != nil {
		l.StartDate = in.StartDate.Time
	}
	if in.EndDate != nil {
		l.EndDate = in.EndDate.Time
	}
	if in.Reason != nil {
		l.Reason = in.Reason
	}
	if in.ApproverId != nil {
		l.ApproverID = in.ApproverId
	}
}
