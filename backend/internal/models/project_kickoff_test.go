package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewProjectKickoffFromCreate_MinimalOnlyProjectId(t *testing.T) {
	projectID := uuid.New()
	k := NewProjectKickoffFromCreate(api.ProjectKickoffCreate{ProjectId: projectID})

	if k.ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %v", k.ProjectID, projectID)
	}
	if k.MeetingDate != nil {
		t.Errorf("MeetingDate = %v, want nil when not supplied", k.MeetingDate)
	}
	if k.FacilitatorID != nil {
		t.Errorf("FacilitatorID = %v, want nil when not supplied", k.FacilitatorID)
	}
	if len(k.ActionItems) != 0 {
		t.Errorf("ActionItems = %v, want empty — the handler inserts them separately", k.ActionItems)
	}
}

func TestNewProjectKickoffFromCreate_AllFields(t *testing.T) {
	projectID, facilitatorID := uuid.New(), uuid.New()
	meetingDate := openapi_types.Date{Time: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)}
	nextSession := openapi_types.Date{Time: time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)}
	attendees := "PM, Lead Dev, UX Designer, Client Stakeholders"
	businessProblem := "Manual scheduling is error-prone."
	success := "50% fewer scheduling conflicts within 3 months."
	primaryUsers := "Ops coordinators"
	painPoints := "No visibility into who's overbooked."
	mvp := "A single shared calendar view."
	constraints := "Must ship before Q4, budget capped at $50k."
	risk := "Vendor API rate limits."
	signOff := "VP of Operations"
	cadence := "Weekly Tuesday syncs"
	tools := "Slack, Jira"

	k := NewProjectKickoffFromCreate(api.ProjectKickoffCreate{
		ProjectId:         projectID,
		MeetingDate:       &meetingDate,
		FacilitatorId:     &facilitatorID,
		Attendees:         &attendees,
		BusinessProblem:   &businessProblem,
		SuccessDefinition: &success,
		PrimaryUsers:      &primaryUsers,
		PainPoints:        &painPoints,
		MvpFeature:        &mvp,
		ConstraintsNotes:  &constraints,
		BiggestRisk:       &risk,
		SignOffOwner:      &signOff,
		MeetingCadence:    &cadence,
		PrimaryTools:      &tools,
		NextSessionDate:   &nextSession,
	})

	if k.MeetingDate == nil || !k.MeetingDate.Equal(meetingDate.Time) {
		t.Errorf("MeetingDate = %v, want %v", k.MeetingDate, meetingDate.Time)
	}
	if k.FacilitatorID == nil || *k.FacilitatorID != facilitatorID {
		t.Errorf("FacilitatorID = %v, want %v", k.FacilitatorID, facilitatorID)
	}
	if k.Attendees == nil || *k.Attendees != attendees {
		t.Errorf("Attendees = %v, want %q", k.Attendees, attendees)
	}
	if k.BusinessProblem == nil || *k.BusinessProblem != businessProblem {
		t.Errorf("BusinessProblem = %v, want %q", k.BusinessProblem, businessProblem)
	}
	if k.SuccessDefinition == nil || *k.SuccessDefinition != success {
		t.Errorf("SuccessDefinition = %v, want %q", k.SuccessDefinition, success)
	}
	if k.PrimaryUsers == nil || *k.PrimaryUsers != primaryUsers {
		t.Errorf("PrimaryUsers = %v, want %q", k.PrimaryUsers, primaryUsers)
	}
	if k.PainPoints == nil || *k.PainPoints != painPoints {
		t.Errorf("PainPoints = %v, want %q", k.PainPoints, painPoints)
	}
	if k.MvpFeature == nil || *k.MvpFeature != mvp {
		t.Errorf("MvpFeature = %v, want %q", k.MvpFeature, mvp)
	}
	if k.ConstraintsNotes == nil || *k.ConstraintsNotes != constraints {
		t.Errorf("ConstraintsNotes = %v, want %q", k.ConstraintsNotes, constraints)
	}
	if k.BiggestRisk == nil || *k.BiggestRisk != risk {
		t.Errorf("BiggestRisk = %v, want %q", k.BiggestRisk, risk)
	}
	if k.SignOffOwner == nil || *k.SignOffOwner != signOff {
		t.Errorf("SignOffOwner = %v, want %q", k.SignOffOwner, signOff)
	}
	if k.MeetingCadence == nil || *k.MeetingCadence != cadence {
		t.Errorf("MeetingCadence = %v, want %q", k.MeetingCadence, cadence)
	}
	if k.PrimaryTools == nil || *k.PrimaryTools != tools {
		t.Errorf("PrimaryTools = %v, want %q", k.PrimaryTools, tools)
	}
	if k.NextSessionDate == nil || !k.NextSessionDate.Equal(nextSession.Time) {
		t.Errorf("NextSessionDate = %v, want %v", k.NextSessionDate, nextSession.Time)
	}
}

func TestProjectKickoff_ApplyUpdate_OnlyTouchesSuppliedFields(t *testing.T) {
	problem := "old problem"
	k := ProjectKickoff{ProjectID: uuid.New(), BusinessProblem: &problem}

	newRisk := "New risk surfaced mid-project."
	k.ApplyUpdate(api.ProjectKickoffUpdate{BiggestRisk: &newRisk})

	if k.BiggestRisk == nil || *k.BiggestRisk != newRisk {
		t.Errorf("BiggestRisk = %v, want %q", k.BiggestRisk, newRisk)
	}
	// Untouched fields survive the partial patch.
	if k.BusinessProblem == nil || *k.BusinessProblem != problem {
		t.Errorf("BusinessProblem = %v, want unchanged %q", k.BusinessProblem, problem)
	}
}

func TestProjectKickoff_ApplyUpdate_EmptyPatchChangesNothing(t *testing.T) {
	problem := "unchanged"
	k := ProjectKickoff{ProjectID: uuid.New(), BusinessProblem: &problem}
	before := k

	k.ApplyUpdate(api.ProjectKickoffUpdate{})

	if k.ProjectID != before.ProjectID || k.BusinessProblem != before.BusinessProblem {
		t.Errorf("empty patch mutated the record: got %+v, want %+v", k, before)
	}
}

func TestProjectKickoff_ApplyUpdate_AllFields(t *testing.T) {
	k := ProjectKickoff{ProjectID: uuid.New()}

	facilitatorID := uuid.New()
	meetingDate := openapi_types.Date{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	nextSession := openapi_types.Date{Time: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)}
	attendees := "Everyone"
	businessProblem := "problem"
	success := "success"
	primaryUsers := "users"
	painPoints := "pain"
	mvp := "mvp"
	constraints := "constraints"
	risk := "risk"
	signOff := "signoff"
	cadence := "cadence"
	tools := "tools"

	k.ApplyUpdate(api.ProjectKickoffUpdate{
		MeetingDate:       &meetingDate,
		FacilitatorId:     &facilitatorID,
		Attendees:         &attendees,
		BusinessProblem:   &businessProblem,
		SuccessDefinition: &success,
		PrimaryUsers:      &primaryUsers,
		PainPoints:        &painPoints,
		MvpFeature:        &mvp,
		ConstraintsNotes:  &constraints,
		BiggestRisk:       &risk,
		SignOffOwner:      &signOff,
		MeetingCadence:    &cadence,
		PrimaryTools:      &tools,
		NextSessionDate:   &nextSession,
	})

	if k.MeetingDate == nil || !k.MeetingDate.Equal(meetingDate.Time) {
		t.Errorf("MeetingDate = %v, want %v", k.MeetingDate, meetingDate.Time)
	}
	if k.FacilitatorID == nil || *k.FacilitatorID != facilitatorID {
		t.Errorf("FacilitatorID = %v, want %v", k.FacilitatorID, facilitatorID)
	}
	if k.Attendees == nil || *k.Attendees != attendees {
		t.Errorf("Attendees = %v, want %q", k.Attendees, attendees)
	}
	if k.SuccessDefinition == nil || *k.SuccessDefinition != success {
		t.Errorf("SuccessDefinition = %v, want %q", k.SuccessDefinition, success)
	}
	if k.MvpFeature == nil || *k.MvpFeature != mvp {
		t.Errorf("MvpFeature = %v, want %q", k.MvpFeature, mvp)
	}
	if k.ConstraintsNotes == nil || *k.ConstraintsNotes != constraints {
		t.Errorf("ConstraintsNotes = %v, want %q", k.ConstraintsNotes, constraints)
	}
	if k.SignOffOwner == nil || *k.SignOffOwner != signOff {
		t.Errorf("SignOffOwner = %v, want %q", k.SignOffOwner, signOff)
	}
	if k.NextSessionDate == nil || !k.NextSessionDate.Equal(nextSession.Time) {
		t.Errorf("NextSessionDate = %v, want %v", k.NextSessionDate, nextSession.Time)
	}
}

func TestProjectKickoff_ApplyUpdate_ProjectIdIsNotEditable(t *testing.T) {
	// ProjectKickoffUpdate has no projectId field at all — this test
	// documents that reassignment is intentionally impossible via PATCH,
	// same convention as Milestone/ProjectMember.
	projectID := uuid.New()
	k := ProjectKickoff{ProjectID: projectID}

	newRisk := "risk"
	k.ApplyUpdate(api.ProjectKickoffUpdate{BiggestRisk: &newRisk})

	if k.ProjectID != projectID {
		t.Errorf("ProjectID changed to %v, want unchanged %v", k.ProjectID, projectID)
	}
}

func TestProjectKickoff_ToAPI_OmitsNilFieldsAndActionItemsIsEmptyNotNil(t *testing.T) {
	k := ProjectKickoff{ID: uuid.New(), ProjectID: uuid.New()}
	out := k.ToAPI()

	if out.MeetingDate != nil {
		t.Errorf("MeetingDate = %v, want nil", out.MeetingDate)
	}
	if out.FacilitatorId != nil {
		t.Errorf("FacilitatorId = %v, want nil", out.FacilitatorId)
	}
	if out.ActionItems == nil {
		t.Error("ActionItems = nil, want a non-nil empty slice so JSON serializes `[]` not `null`")
	}
	if len(out.ActionItems) != 0 {
		t.Errorf("ActionItems = %v, want empty", out.ActionItems)
	}
}

func TestProjectKickoff_ToAPI_RoundTripsDatesAndActionItems(t *testing.T) {
	id, projectID, facilitatorID := uuid.New(), uuid.New(), uuid.New()
	meetingDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	nextSession := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	ownerID, taskID := uuid.New(), uuid.New()

	k := ProjectKickoff{
		ID: id, ProjectID: projectID, FacilitatorID: &facilitatorID,
		MeetingDate: &meetingDate, NextSessionDate: &nextSession,
		ActionItems: []KickoffActionItem{
			{ID: uuid.New(), KickoffID: id, Description: "Send follow-up notes", OwnerID: &ownerID, TaskID: &taskID},
			{ID: uuid.New(), KickoffID: id, Description: "Draft the requirements doc"},
		},
	}

	out := k.ToAPI()

	if out.Id != id || out.ProjectId != projectID {
		t.Errorf("ids = %v/%v, want %v/%v", out.Id, out.ProjectId, id, projectID)
	}
	if out.MeetingDate == nil || !out.MeetingDate.Equal(meetingDate) {
		t.Errorf("MeetingDate = %v, want %v", out.MeetingDate, meetingDate)
	}
	if out.NextSessionDate == nil || !out.NextSessionDate.Equal(nextSession) {
		t.Errorf("NextSessionDate = %v, want %v", out.NextSessionDate, nextSession)
	}
	if len(out.ActionItems) != 2 {
		t.Fatalf("len(ActionItems) = %d, want 2", len(out.ActionItems))
	}
	if out.ActionItems[0].Description != "Send follow-up notes" {
		t.Errorf("ActionItems[0].Description = %q, want %q", out.ActionItems[0].Description, "Send follow-up notes")
	}
	if out.ActionItems[0].OwnerId == nil || *out.ActionItems[0].OwnerId != ownerID {
		t.Errorf("ActionItems[0].OwnerId = %v, want %v", out.ActionItems[0].OwnerId, ownerID)
	}
	if out.ActionItems[0].TaskId == nil || *out.ActionItems[0].TaskId != taskID {
		t.Errorf("ActionItems[0].TaskId = %v, want %v", out.ActionItems[0].TaskId, taskID)
	}
	if out.ActionItems[1].OwnerId != nil {
		t.Errorf("ActionItems[1].OwnerId = %v, want nil (never assigned an owner)", out.ActionItems[1].OwnerId)
	}
	if out.ActionItems[1].TaskId != nil {
		t.Errorf("ActionItems[1].TaskId = %v, want nil (not yet converted to a task)", out.ActionItems[1].TaskId)
	}
}

func TestNewKickoffActionItemFromInput(t *testing.T) {
	kickoffID, ownerID := uuid.New(), uuid.New()

	item := NewKickoffActionItemFromInput(kickoffID, api.KickoffActionItemInput{
		Description: "Schedule the next requirements session",
		OwnerId:     &ownerID,
	})

	if item.KickoffID != kickoffID {
		t.Errorf("KickoffID = %v, want %v", item.KickoffID, kickoffID)
	}
	if item.Description != "Schedule the next requirements session" {
		t.Errorf("Description = %q, want %q", item.Description, "Schedule the next requirements session")
	}
	if item.OwnerID == nil || *item.OwnerID != ownerID {
		t.Errorf("OwnerID = %v, want %v", item.OwnerID, ownerID)
	}
	if item.TaskID != nil {
		t.Errorf("TaskID = %v, want nil when not supplied", item.TaskID)
	}
}

func TestNewKickoffActionItemFromInput_CarriesExistingTaskId(t *testing.T) {
	// The "convert to task" flow round-trips an already-set taskId back
	// through a save (see replaceActionItems in the handler) — this
	// documents that NewKickoffActionItemFromInput preserves it rather than
	// only ever producing not-yet-converted items.
	kickoffID, taskID := uuid.New(), uuid.New()

	item := NewKickoffActionItemFromInput(kickoffID, api.KickoffActionItemInput{
		Description: "Already converted",
		TaskId:      &taskID,
	})

	if item.TaskID == nil || *item.TaskID != taskID {
		t.Errorf("TaskID = %v, want %v", item.TaskID, taskID)
	}
}

func TestKickoffActionItem_ToAPI(t *testing.T) {
	id, kickoffID, ownerID, taskID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	item := KickoffActionItem{
		ID: id, KickoffID: kickoffID, Description: "Follow up with legal",
		OwnerID: &ownerID, TaskID: &taskID,
	}

	out := item.ToAPI()

	if out.Id != id {
		t.Errorf("Id = %v, want %v", out.Id, id)
	}
	if out.Description != "Follow up with legal" {
		t.Errorf("Description = %q, want %q", out.Description, "Follow up with legal")
	}
	if out.OwnerId == nil || *out.OwnerId != ownerID {
		t.Errorf("OwnerId = %v, want %v", out.OwnerId, ownerID)
	}
	if out.TaskId == nil || *out.TaskId != taskID {
		t.Errorf("TaskId = %v, want %v", out.TaskId, taskID)
	}
}
