package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ProjectKickoff is the GORM persistence model for the `project_kickoffs`
// table defined in db/schema.sql — one record per project (project_id is
// UNIQUE at the DB level). Fields map 1:1 onto the sections of the standard
// kickoff-meeting-agenda template: meeting info, vision & business goals,
// target audience & value proposition, constraints/risks/compliance,
// governance & communication, next steps & action items.
type ProjectKickoff struct {
	ID                uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	MeetingDate       *time.Time `gorm:"type:date"`
	FacilitatorID     *uuid.UUID `gorm:"type:uuid"`
	Attendees         *string    `gorm:"type:text"`
	BusinessProblem   *string    `gorm:"type:text"`
	SuccessDefinition *string    `gorm:"type:text"`
	PrimaryUsers      *string    `gorm:"type:text"`
	PainPoints        *string    `gorm:"type:text"`
	MvpFeature        *string    `gorm:"type:text"`
	ConstraintsNotes  *string    `gorm:"type:text"`
	BiggestRisk       *string    `gorm:"type:text"`
	SignOffOwner      *string    `gorm:"type:text"`
	MeetingCadence    *string    `gorm:"type:text"`
	PrimaryTools      *string    `gorm:"type:text"`
	NextSessionDate   *time.Time `gorm:"type:date"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// ActionItems is transient (gorm:"-", not a DB column) — the handler
	// populates it from a separate `kickoff_action_items` query before
	// calling ToAPI, and replaces the whole set on save. Same pattern as
	// Task.AssigneeIDs / attachAssignees / replaceAssignees.
	ActionItems []KickoffActionItem `gorm:"-"`
}

// TableName pins the model to the existing pluralized table.
func (ProjectKickoff) TableName() string { return "project_kickoffs" }

// KickoffActionItem is the GORM persistence model for the
// `kickoff_action_items` table — action items captured in a kickoff's "Next
// Steps" section. TaskID is set once an item has been converted into a real
// task; conversion is optional and happens after the kickoff is saved.
type KickoffActionItem struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	KickoffID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Description string     `gorm:"not null"`
	OwnerID     *uuid.UUID `gorm:"type:uuid"`
	TaskID      *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

// TableName pins the model to the existing pluralized table.
func (KickoffActionItem) TableName() string { return "kickoff_action_items" }

// ToAPI converts an action item into its generated API DTO.
func (a KickoffActionItem) ToAPI() api.KickoffActionItem {
	return api.KickoffActionItem{
		Id:          a.ID,
		Description: a.Description,
		OwnerId:     a.OwnerID,
		TaskId:      a.TaskID,
		CreatedAt:   a.CreatedAt,
	}
}

// NewKickoffActionItemFromInput builds a not-yet-saved action item from a
// create/update request entry. kickoffID is supplied by the handler since
// the input body itself never carries it (it's always nested under a
// ProjectKickoff request).
func NewKickoffActionItemFromInput(kickoffID uuid.UUID, in api.KickoffActionItemInput) KickoffActionItem {
	return KickoffActionItem{
		KickoffID:   kickoffID,
		Description: in.Description,
		OwnerID:     in.OwnerId,
		TaskID:      in.TaskId,
	}
}

// ToAPI converts the persistence model into the generated API DTO.
// ActionItems is always a non-nil (possibly empty) slice in the output, so
// callers get `[]` rather than `null` when there are none — mirroring
// Task.AssigneeIds.
func (k ProjectKickoff) ToAPI() api.ProjectKickoff {
	items := make([]api.KickoffActionItem, len(k.ActionItems))
	for i := range k.ActionItems {
		items[i] = k.ActionItems[i].ToAPI()
	}

	out := api.ProjectKickoff{
		Id:                k.ID,
		ProjectId:         k.ProjectID,
		FacilitatorId:     k.FacilitatorID,
		Attendees:         k.Attendees,
		BusinessProblem:   k.BusinessProblem,
		SuccessDefinition: k.SuccessDefinition,
		PrimaryUsers:      k.PrimaryUsers,
		PainPoints:        k.PainPoints,
		MvpFeature:        k.MvpFeature,
		ConstraintsNotes:  k.ConstraintsNotes,
		BiggestRisk:       k.BiggestRisk,
		SignOffOwner:      k.SignOffOwner,
		MeetingCadence:    k.MeetingCadence,
		PrimaryTools:      k.PrimaryTools,
		ActionItems:       items,
		CreatedAt:         k.CreatedAt,
		UpdatedAt:         k.UpdatedAt,
	}
	if k.MeetingDate != nil {
		out.MeetingDate = &openapi_types.Date{Time: *k.MeetingDate}
	}
	if k.NextSessionDate != nil {
		out.NextSessionDate = &openapi_types.Date{Time: *k.NextSessionDate}
	}
	return out
}

// NewProjectKickoffFromCreate builds a new ProjectKickoff from a create
// request. ActionItems is deliberately left empty here — the handler
// creates the row first (to get an ID), then inserts action items via
// replaceActionItems, same two-step flow CreateTask uses for assignees.
func NewProjectKickoffFromCreate(in api.ProjectKickoffCreate) ProjectKickoff {
	k := ProjectKickoff{
		ProjectID:         in.ProjectId,
		FacilitatorID:     in.FacilitatorId,
		Attendees:         in.Attendees,
		BusinessProblem:   in.BusinessProblem,
		SuccessDefinition: in.SuccessDefinition,
		PrimaryUsers:      in.PrimaryUsers,
		PainPoints:        in.PainPoints,
		MvpFeature:        in.MvpFeature,
		ConstraintsNotes:  in.ConstraintsNotes,
		BiggestRisk:       in.BiggestRisk,
		SignOffOwner:      in.SignOffOwner,
		MeetingCadence:    in.MeetingCadence,
		PrimaryTools:      in.PrimaryTools,
	}
	if in.MeetingDate != nil {
		t := in.MeetingDate.Time
		k.MeetingDate = &t
	}
	if in.NextSessionDate != nil {
		t := in.NextSessionDate.Time
		k.NextSessionDate = &t
	}
	return k
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
// projectId is deliberately not editable — same rationale as Milestone/
// ProjectMember not allowing their parent-project reassignment.
// ActionItems is NOT handled here — the handler manages it separately via
// replaceActionItems, since replacing a set of child rows needs its own DB
// statements (same split as Task.ApplyUpdate/replaceAssignees).
func (k *ProjectKickoff) ApplyUpdate(in api.ProjectKickoffUpdate) {
	if in.MeetingDate != nil {
		t := in.MeetingDate.Time
		k.MeetingDate = &t
	}
	if in.FacilitatorId != nil {
		k.FacilitatorID = in.FacilitatorId
	}
	if in.Attendees != nil {
		k.Attendees = in.Attendees
	}
	if in.BusinessProblem != nil {
		k.BusinessProblem = in.BusinessProblem
	}
	if in.SuccessDefinition != nil {
		k.SuccessDefinition = in.SuccessDefinition
	}
	if in.PrimaryUsers != nil {
		k.PrimaryUsers = in.PrimaryUsers
	}
	if in.PainPoints != nil {
		k.PainPoints = in.PainPoints
	}
	if in.MvpFeature != nil {
		k.MvpFeature = in.MvpFeature
	}
	if in.ConstraintsNotes != nil {
		k.ConstraintsNotes = in.ConstraintsNotes
	}
	if in.BiggestRisk != nil {
		k.BiggestRisk = in.BiggestRisk
	}
	if in.SignOffOwner != nil {
		k.SignOffOwner = in.SignOffOwner
	}
	if in.MeetingCadence != nil {
		k.MeetingCadence = in.MeetingCadence
	}
	if in.PrimaryTools != nil {
		k.PrimaryTools = in.PrimaryTools
	}
	if in.NextSessionDate != nil {
		t := in.NextSessionDate.Time
		k.NextSessionDate = &t
	}
}
