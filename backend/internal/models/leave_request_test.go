package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewLeaveRequestFromCreate_DefaultsToVacationPending(t *testing.T) {
	start := openapi_types.Date{Time: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)}
	end := openapi_types.Date{Time: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)}

	l := NewLeaveRequestFromCreate(api.LeaveRequestCreate{
		PersonId: uuid.New(), StartDate: start, EndDate: end,
	})

	if l.Type != api.LeaveTypeVacation {
		t.Errorf("Type = %q, want vacation (default)", l.Type)
	}
	if l.Status != api.LeaveStatusPending {
		t.Errorf("Status = %q, want pending (every request starts pending, approval is a separate step)", l.Status)
	}
	if !l.StartDate.Equal(start.Time) || !l.EndDate.Equal(end.Time) {
		t.Errorf("dates = %v..%v, want %v..%v", l.StartDate, l.EndDate, start.Time, end.Time)
	}
}

func TestLeaveRequest_ApplyUpdate_ApprovalFlow(t *testing.T) {
	l := LeaveRequest{Status: api.LeaveStatusPending}

	approved := api.LeaveStatusApproved
	approverID := uuid.New()
	l.ApplyUpdate(api.LeaveRequestUpdate{Status: &approved, ApproverId: &approverID})

	if l.Status != api.LeaveStatusApproved {
		t.Errorf("Status = %q, want approved", l.Status)
	}
	if l.ApproverID == nil || *l.ApproverID != approverID {
		t.Errorf("ApproverID = %v, want %v", l.ApproverID, approverID)
	}
}

func TestLeaveRequest_ApplyUpdate_PersonIdIsEditable(t *testing.T) {
	// Unlike ProjectMember/Milestone, LeaveRequestUpdate DOES expose PersonId
	// — this test just documents/pins that as intentional, not an oversight,
	// by confirming it actually takes effect.
	original := uuid.New()
	reassigned := uuid.New()
	l := LeaveRequest{PersonID: original}

	l.ApplyUpdate(api.LeaveRequestUpdate{PersonId: &reassigned})

	if l.PersonID != reassigned {
		t.Errorf("PersonID = %v, want %v", l.PersonID, reassigned)
	}
}

func TestLeaveRequest_ApplyUpdate_TypeDatesAndReason(t *testing.T) {
	l := LeaveRequest{Type: api.LeaveTypeVacation}

	sick := api.LeaveTypeSick
	reason := "Flu"
	start := openapi_types.Date{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	end := openapi_types.Date{Time: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)}

	l.ApplyUpdate(api.LeaveRequestUpdate{Type: &sick, Reason: &reason, StartDate: &start, EndDate: &end})

	if l.Type != api.LeaveTypeSick {
		t.Errorf("Type = %q, want sick", l.Type)
	}
	if l.Reason == nil || *l.Reason != reason {
		t.Errorf("Reason = %v, want %q", l.Reason, reason)
	}
	if !l.StartDate.Equal(start.Time) || !l.EndDate.Equal(end.Time) {
		t.Errorf("dates = %v..%v, want %v..%v", l.StartDate, l.EndDate, start.Time, end.Time)
	}
}

func TestLeaveRequest_ToAPI(t *testing.T) {
	id, personID, approverID := uuid.New(), uuid.New(), uuid.New()
	reason := "Family trip"
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	l := LeaveRequest{
		ID: id, PersonID: personID, Type: api.LeaveTypeVacation, Status: api.LeaveStatusApproved,
		StartDate: start, EndDate: end, Reason: &reason, ApproverID: &approverID,
	}

	out := l.ToAPI()

	if out.Id != id || out.PersonId != personID {
		t.Errorf("Id/PersonId = %v/%v, want %v/%v", out.Id, out.PersonId, id, personID)
	}
	if !out.StartDate.Equal(start) || !out.EndDate.Equal(end) {
		t.Errorf("dates = %v..%v, want %v..%v", out.StartDate, out.EndDate, start, end)
	}
	if out.ApproverId == nil || *out.ApproverId != approverID {
		t.Errorf("ApproverId = %v, want %v", out.ApproverId, approverID)
	}
	if out.Reason == nil || *out.Reason != reason {
		t.Errorf("Reason = %v, want %q", out.Reason, reason)
	}
}
