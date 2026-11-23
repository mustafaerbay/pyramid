package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
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
	ProjectCode      *string `gorm:"type:text;column:project_code"`
	TeamSizeActual   *int    `gorm:"column:team_size_actual"`
	TeamSizePlanned  *int    `gorm:"column:team_size_planned"`

	// BoardSettings is JSON (api.BoardSettings); nil means the defaults.
	BoardSettings *string `gorm:"type:jsonb"`
	// CustomFields is JSON ([]api.CustomField): extra fields on its items.
	CustomFields *string `gorm:"type:jsonb"`
	// Workflow is JSON ([]api.WorkflowState); nil means the default seven.
	Workflow *string `gorm:"type:jsonb"`
	// WorkItemTypes is JSON ([]api.WorkItemTypeDef): the project's own types.
	WorkItemTypes *string `gorm:"type:jsonb"`
	// Restricted: only members with write access (and managers) change it.
	Restricted bool `gorm:"not null;default:false"`
	// Rules is JSON (api.WorkItemRules).
	Rules *string `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the model to the existing pluralized table.
func (Project) TableName() string { return "projects" }

// ItemTypes returns the project's own work item types ([] when none).
func (p Project) ItemTypes() []api.WorkItemTypeDef {
	out := []api.WorkItemTypeDef{}
	if p.WorkItemTypes != nil {
		_ = json.Unmarshal([]byte(*p.WorkItemTypes), &out)
	}
	return out
}

// ValidateItemTypes checks a project's own types: up to 20, unique keys
// that aren't built-in type names, a known base and colour.
func ValidateItemTypes(types []api.WorkItemTypeDef) error {
	if len(types) > 20 {
		return errors.New("a project can have at most 20 types of its own")
	}
	seen := map[string]bool{}
	for _, t := range types {
		if !fieldKey.MatchString(t.Key) || api.TaskType(t.Key).Valid() {
			return fmt.Errorf("type key %q must be lowercase letters, digits or _ and not a built-in type", t.Key)
		}
		if seen[t.Key] {
			return fmt.Errorf("two types use the key %q", t.Key)
		}
		seen[t.Key] = true
		if l := strings.TrimSpace(t.Label); l == "" || len([]rune(l)) > 30 {
			return errors.New("every type needs a name of 1–30 characters")
		}
		if !t.Base.Valid() {
			return fmt.Errorf("unknown base type %q", t.Base)
		}
		if !t.Color.Valid() {
			return fmt.Errorf("unknown colour %q", t.Color)
		}
	}
	return nil
}

// DefaultWorkflow is one state per status, keyed by the status.
func DefaultWorkflow() []api.WorkflowState {
	labels := []struct {
		s api.TaskStatus
		l string
	}{
		{api.TaskStatusBacklog, "Backlog"}, {api.TaskStatusTodo, "To do"}, {api.TaskStatusInProgress, "In progress"},
		{api.TaskStatusInReview, "In review"}, {api.TaskStatusBlocked, "Blocked"}, {api.TaskStatusDone, "Done"},
		{api.TaskStatusCancelled, "Cancelled"},
	}
	out := make([]api.WorkflowState, len(labels))
	for i, x := range labels {
		out[i] = api.WorkflowState{Key: string(x.s), Label: x.l, Category: x.s}
	}
	return out
}

// States returns the project's workflow, or the default one.
func (p Project) States() []api.WorkflowState {
	if p.Workflow != nil {
		var out []api.WorkflowState
		if json.Unmarshal([]byte(*p.Workflow), &out) == nil && len(out) > 0 {
			return out
		}
	}
	return DefaultWorkflow()
}

// ValidateWorkflow checks a workflow: 1–20 states with unique keys, known
// categories, a done state and one to work in.
func ValidateWorkflow(states []api.WorkflowState) error {
	if len(states) == 0 || len(states) > 20 {
		return errors.New("a workflow needs 1–20 states")
	}
	seen := map[string]bool{}
	done, open := false, false
	for _, s := range states {
		if !fieldKey.MatchString(s.Key) {
			return fmt.Errorf("state key %q must be lowercase letters, digits or _ (starting with a letter)", s.Key)
		}
		if seen[s.Key] {
			return fmt.Errorf("two states use the key %q", s.Key)
		}
		seen[s.Key] = true
		if l := strings.TrimSpace(s.Label); l == "" || len([]rune(l)) > 40 {
			return errors.New("every state needs a name of 1–40 characters")
		}
		if !s.Category.Valid() {
			return fmt.Errorf("unknown category %q", s.Category)
		}
		done = done || s.Category == api.TaskStatusDone
		open = open || (s.Category != api.TaskStatusDone && s.Category != api.TaskStatusCancelled)
	}
	if !done || !open {
		return errors.New("a workflow needs a done state and at least one state to work in")
	}
	return nil
}

// Fields returns the project's custom field definitions ([] when none).
func (p Project) Fields() []api.CustomField {
	out := []api.CustomField{}
	if p.CustomFields != nil {
		_ = json.Unmarshal([]byte(*p.CustomFields), &out)
	}
	return out
}

var fieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// ValidateFields checks custom field definitions.
func ValidateFields(fields []api.CustomField) error {
	if len(fields) > 20 {
		return errors.New("a project can have at most 20 custom fields")
	}
	seen := map[string]bool{}
	for _, f := range fields {
		if !fieldKey.MatchString(f.Key) {
			return fmt.Errorf("field key %q must be lowercase letters, digits or _ (starting with a letter)", f.Key)
		}
		if seen[f.Key] {
			return fmt.Errorf("two fields use the key %q", f.Key)
		}
		seen[f.Key] = true
		if l := strings.TrimSpace(f.Label); l == "" || len([]rune(l)) > 60 {
			return errors.New("every field needs a label of 1–60 characters")
		}
		if !f.Type.Valid() {
			return fmt.Errorf("unknown field type %q", f.Type)
		}
		if f.Type == api.CustomFieldTypeSelect {
			if f.Options == nil || len(*f.Options) == 0 || len(*f.Options) > 50 {
				return fmt.Errorf("select field %q needs 1–50 options", f.Label)
			}
			opts := map[string]bool{}
			for _, o := range *f.Options {
				if strings.TrimSpace(o) == "" || opts[o] {
					return fmt.Errorf("select field %q has an empty or repeated option", f.Label)
				}
				opts[o] = true
			}
		}
	}
	return nil
}

// Board returns the project's board settings, with defaults filled in.
func (p Project) Board() api.BoardSettings {
	out := api.BoardSettings{WipLimits: map[string]int{}, Swimlanes: api.BoardSettingsSwimlanesNone, CardColor: api.BoardSettingsCardColorNone}
	if p.BoardSettings != nil {
		_ = json.Unmarshal([]byte(*p.BoardSettings), &out)
	}
	if out.WipLimits == nil {
		out.WipLimits = map[string]int{}
	}
	if out.Swimlanes == "" {
		out.Swimlanes = api.BoardSettingsSwimlanesNone
	}
	if out.CardColor == "" {
		out.CardColor = api.BoardSettingsCardColorNone
	}
	return out
}

// ValidateBoard checks limits (1–999, known statuses) and the options.
func ValidateBoard(b api.BoardSettings, states []api.WorkflowState, fields []api.CustomField) error {
	known := map[string]bool{}
	category := map[string]api.TaskStatus{}
	for _, s := range states {
		known[s.Key] = true
		category[s.Key] = s.Category
	}
	for state, n := range b.WipLimits {
		if !known[state] {
			return fmt.Errorf("this project has no state %q for a WIP limit", state)
		}
		if n < 1 || n > 999 {
			return errors.New("WIP limits must be between 1 and 999")
		}
	}
	if !b.Swimlanes.Valid() {
		return errors.New("unknown swimlanes option")
	}
	if !b.CardColor.Valid() {
		return errors.New("unknown cardColor option")
	}
	if b.SplitColumns != nil {
		for _, st := range *b.SplitColumns {
			if !known[st] {
				return fmt.Errorf("this project has no state %q to split", st)
			}
			if c := category[st]; c == api.TaskStatusDone || c == api.TaskStatusCancelled {
				return errors.New("done and cancelled columns can't be split")
			}
		}
	}
	if b.DefinitionOfDone != nil {
		for st, text := range *b.DefinitionOfDone {
			if !known[st] {
				return fmt.Errorf("this project has no state %q for a definition of done", st)
			}
			if len([]rune(text)) > 500 {
				return errors.New("a definition of done can be up to 500 characters")
			}
		}
	}
	if b.CardFields != nil {
		fieldKeys := map[string]bool{}
		for _, f := range fields {
			fieldKeys[f.Key] = true
		}
		for _, f := range *b.CardFields {
			if strings.HasPrefix(f, "custom:") {
				if !fieldKeys[strings.TrimPrefix(f, "custom:")] {
					return fmt.Errorf("this project has no custom field %q", strings.TrimPrefix(f, "custom:"))
				}
				continue
			}
			if !cardFields[f] {
				return fmt.Errorf("cards can't show %q", f)
			}
		}
	}
	if b.TagColors != nil {
		for tag, c := range *b.TagColors {
			if !c.Valid() {
				return fmt.Errorf("unknown colour for tag %q", tag)
			}
		}
	}
	if b.CardRules != nil {
		if len(*b.CardRules) > 20 {
			return errors.New("up to 20 card rules")
		}
		for _, r := range *b.CardRules {
			if !r.Field.Valid() {
				return errors.New("unknown card rule field")
			}
			if !r.Color.Valid() {
				return errors.New("unknown card rule colour")
			}
		}
	}
	return nil
}

var cardFields = map[string]bool{"assignees": true, "manDays": true, "points": true, "due": true, "tags": true, "priority": true, "parent": true, "remaining": true}

// ToAPI converts the persistence model into the generated API DTO.
func (p Project) ToAPI() api.Project {
	out := api.Project{
		Rules:            p.RulesOf(),
		Restricted:       p.Restricted,
		WorkItemTypes:    p.ItemTypes(),
		Workflow:         p.States(),
		CustomFields:     p.Fields(),
		BoardSettings:    p.Board(),
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
// RulesOf returns the project's work item rules (empty when none).
func (p Project) RulesOf() api.WorkItemRules {
	out := api.WorkItemRules{}
	if p.Rules != nil {
		_ = json.Unmarshal([]byte(*p.Rules), &out)
	}
	if out.RequiredByState == nil {
		out.RequiredByState = map[string][]string{}
	}
	if out.Defaults == nil {
		out.Defaults = map[string]any{}
	}
	if out.ReadOnly == nil {
		out.ReadOnly = []string{}
	}
	return out
}

func (p *Project) ApplyUpdate(in api.ProjectUpdate) {
	if in.Rules != nil {
		if b, err := json.Marshal(*in.Rules); err == nil {
			v := string(b)
			p.Rules = &v
		}
	}
	if in.Restricted != nil {
		p.Restricted = *in.Restricted
	}
	if in.WorkItemTypes != nil {
		if b, err := json.Marshal(*in.WorkItemTypes); err == nil {
			v := string(b)
			p.WorkItemTypes = &v
		}
	}
	if in.Workflow != nil {
		if b, err := json.Marshal(*in.Workflow); err == nil {
			v := string(b)
			p.Workflow = &v
		}
	}
	if in.CustomFields != nil {
		if b, err := json.Marshal(*in.CustomFields); err == nil {
			v := string(b)
			p.CustomFields = &v
		}
	}
	if in.BoardSettings != nil {
		if b, err := json.Marshal(in.BoardSettings); err == nil {
			v := string(b)
			p.BoardSettings = &v
		}
	}
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
