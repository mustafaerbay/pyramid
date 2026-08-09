package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Project is the GORM persistence model for the `projects` table defined in
// db/schema.sql. Only the fields the current UI/API surface uses are mapped;
// the table itself carries more columns (client, department, budget, …) that
// can be added here as those parts of the schema get wired up.
type Project struct {
	ID          uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Key         string              `gorm:"uniqueIndex;not null"`
	Name        string              `gorm:"not null"`
	Description *string             `gorm:"type:text"`
	Status      api.ProjectStatus   `gorm:"type:project_status;not null;default:planning"`
	Priority    api.ProjectPriority `gorm:"type:project_priority;not null;default:medium"`
	Health      api.ProjectHealth   `gorm:"type:project_health;not null;default:on_track"`
	LeadID      *uuid.UUID          `gorm:"type:uuid"`
	StartDate   *time.Time          `gorm:"type:date"`
	TargetDate  *time.Time          `gorm:"type:date"`

	// Stakeholder / reporting metadata block, shown in the project header on
	// the detail page and edited via a dedicated "Edit project info" modal
	// separate from the core fields above. All optional; all free text
	// except the two team-size counts.
	Sponsor          *string `gorm:"type:text;column:sponsor"`
	ProjectDirector  *string `gorm:"type:text;column:project_director"`
	ReportPeriod     *string `gorm:"type:text;column:report_period"`
	HqPm             *string `gorm:"type:text;column:hq_pm"`
	TurkeyPm         *string `gorm:"type:text;column:turkey_pm"`
	ChineseInterface *string `gorm:"type:text;column:chinese_interface"`
	HqInterface      *string `gorm:"type:text;column:hq_interface"`
	TurkeyQa         *string `gorm:"type:text;column:turkey_qa"`
	AccountName      *string `gorm:"type:text;column:account_name"`
	HqSe             *string `gorm:"type:text;column:hq_se"`
	TurkeySe         *string `gorm:"type:text;column:turkey_se"`
	ReleaseVersion   *string `gorm:"type:text;column:release_version"`
	ProjectCode      *string `gorm:"type:text;column:project_code"`
	TeamSizeActual   *int    `gorm:"column:team_size_actual"`
	TeamSizePlanned  *int    `gorm:"column:team_size_planned"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the model to the existing pluralized table.
func (Project) TableName() string { return "projects" }

// ToAPI converts the persistence model into the generated API DTO.
func (p Project) ToAPI() api.Project {
	out := api.Project{
		Id:               p.ID,
		Key:              p.Key,
		Name:             p.Name,
		Description:      p.Description,
		Status:           p.Status,
		Priority:         p.Priority,
		Health:           p.Health,
		LeadId:           p.LeadID,
		Sponsor:          p.Sponsor,
		ProjectDirector:  p.ProjectDirector,
		ReportPeriod:     p.ReportPeriod,
		HqPm:             p.HqPm,
		TurkeyPm:         p.TurkeyPm,
		ChineseInterface: p.ChineseInterface,
		HqInterface:      p.HqInterface,
		TurkeyQa:         p.TurkeyQa,
		AccountName:      p.AccountName,
		HqSe:             p.HqSe,
		TurkeySe:         p.TurkeySe,
		ReleaseVersion:   p.ReleaseVersion,
		ProjectCode:      p.ProjectCode,
		TeamSizeActual:   p.TeamSizeActual,
		TeamSizePlanned:  p.TeamSizePlanned,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	if p.StartDate != nil {
		out.StartDate = &openapi_types.Date{Time: *p.StartDate}
	}
	if p.TargetDate != nil {
		out.TargetDate = &openapi_types.Date{Time: *p.TargetDate}
	}
	return out
}

// NewProjectFromCreate builds a new Project from a create request, applying
// the same defaults the database would use.
func NewProjectFromCreate(in api.ProjectCreate) Project {
	p := Project{
		Key:         in.Key,
		Name:        in.Name,
		Description: in.Description,
		LeadID:      in.LeadId,
		Status:      api.ProjectStatusPlanning,
		Priority:    api.ProjectPriorityMedium,
		Health:      api.ProjectHealthOnTrack,
	}
	if in.Priority != nil {
		p.Priority = *in.Priority
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		p.StartDate = &t
	}
	if in.TargetDate != nil {
		t := in.TargetDate.Time
		p.TargetDate = &t
	}
	return p
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
func (p *Project) ApplyUpdate(in api.ProjectUpdate) {
	if in.Key != nil {
		p.Key = *in.Key
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Description != nil {
		p.Description = in.Description
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.Priority != nil {
		p.Priority = *in.Priority
	}
	if in.Health != nil {
		p.Health = *in.Health
	}
	if in.LeadId != nil {
		p.LeadID = in.LeadId
	}
	if in.StartDate != nil {
		t := in.StartDate.Time
		p.StartDate = &t
	}
	if in.TargetDate != nil {
		t := in.TargetDate.Time
		p.TargetDate = &t
	}
	if in.Sponsor != nil {
		p.Sponsor = in.Sponsor
	}
	if in.ProjectDirector != nil {
		p.ProjectDirector = in.ProjectDirector
	}
	if in.ReportPeriod != nil {
		p.ReportPeriod = in.ReportPeriod
	}
	if in.HqPm != nil {
		p.HqPm = in.HqPm
	}
	if in.TurkeyPm != nil {
		p.TurkeyPm = in.TurkeyPm
	}
	if in.ChineseInterface != nil {
		p.ChineseInterface = in.ChineseInterface
	}
	if in.HqInterface != nil {
		p.HqInterface = in.HqInterface
	}
	if in.TurkeyQa != nil {
		p.TurkeyQa = in.TurkeyQa
	}
	if in.AccountName != nil {
		p.AccountName = in.AccountName
	}
	if in.HqSe != nil {
		p.HqSe = in.HqSe
	}
	if in.TurkeySe != nil {
		p.TurkeySe = in.TurkeySe
	}
	if in.ReleaseVersion != nil {
		p.ReleaseVersion = in.ReleaseVersion
	}
	if in.ProjectCode != nil {
		p.ProjectCode = in.ProjectCode
	}
	if in.TeamSizeActual != nil {
		p.TeamSizeActual = in.TeamSizeActual
	}
	if in.TeamSizePlanned != nil {
		p.TeamSizePlanned = in.TeamSizePlanned
	}
}
