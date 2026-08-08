package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Person is the GORM persistence model for the `people` table defined in
// db/schema.sql. Enum columns reuse the API enum types so there is a single
// definition of each enum across the persistence and transport layers.
type Person struct {
	ID                  uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	FirstName           string              `gorm:"not null"`
	LastName            string              `gorm:"not null"`
	Email               string              `gorm:"type:citext;uniqueIndex;not null"`
	Phone               *string             `gorm:"type:text"`
	AvatarURL           *string             `gorm:"column:avatar_url;type:text"`
	Title               *string             `gorm:"type:text"`
	Seniority           *api.SeniorityLevel `gorm:"type:seniority_level"`
	EmploymentType      api.EmploymentType  `gorm:"type:employment_type;not null;default:full_time"`
	Status              api.PersonStatus    `gorm:"type:person_status;not null;default:active"`
	DepartmentID        *uuid.UUID          `gorm:"type:uuid"`
	ManagerID           *uuid.UUID          `gorm:"type:uuid"`
	Location            *string             `gorm:"type:text"`
	Timezone            *string             `gorm:"type:text"`
	Bio                 *string             `gorm:"type:text"`
	HireDate            *time.Time          `gorm:"type:date"`
	TerminationDate     *time.Time          `gorm:"type:date"`
	WeeklyCapacityHours *float64            `gorm:"type:numeric(5,2);default:40"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TableName pins the model to the existing pluralized table.
func (Person) TableName() string { return "people" }

// ToAPI converts the persistence model into the generated API DTO.
func (p Person) ToAPI() api.Person {
	out := api.Person{
		Id:                  p.ID,
		FirstName:           p.FirstName,
		LastName:            p.LastName,
		Email:               openapi_types.Email(p.Email),
		EmploymentType:      p.EmploymentType,
		Status:              p.Status,
		Phone:               p.Phone,
		AvatarUrl:           p.AvatarURL,
		Title:               p.Title,
		Seniority:           p.Seniority,
		DepartmentId:        p.DepartmentID,
		ManagerId:           p.ManagerID,
		Location:            p.Location,
		Timezone:            p.Timezone,
		WeeklyCapacityHours: p.WeeklyCapacityHours,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
	if p.HireDate != nil {
		out.HireDate = &openapi_types.Date{Time: *p.HireDate}
	}
	return out
}

// NewPersonFromCreate builds a new Person from a create request, applying the
// same defaults the database would use.
func NewPersonFromCreate(in api.PersonCreate) Person {
	p := Person{
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		Email:          string(in.Email),
		Phone:          in.Phone,
		Title:          in.Title,
		Location:       in.Location,
		Timezone:       in.Timezone,
		DepartmentID:   in.DepartmentId,
		ManagerID:      in.ManagerId,
		Seniority:      in.Seniority,
		EmploymentType: api.EmploymentTypeFullTime,
		Status:         api.PersonStatusActive,
	}
	if in.EmploymentType != nil {
		p.EmploymentType = *in.EmploymentType
	}
	if in.HireDate != nil {
		t := in.HireDate.Time
		p.HireDate = &t
	}
	return p
}

// ApplyUpdate mutates the model in place with the non-nil fields of a patch.
func (p *Person) ApplyUpdate(in api.PersonUpdate) {
	if in.FirstName != nil {
		p.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		p.LastName = *in.LastName
	}
	if in.Email != nil {
		p.Email = string(*in.Email)
	}
	if in.Phone != nil {
		p.Phone = in.Phone
	}
	if in.Title != nil {
		p.Title = in.Title
	}
	if in.Seniority != nil {
		p.Seniority = in.Seniority
	}
	if in.EmploymentType != nil {
		p.EmploymentType = *in.EmploymentType
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.DepartmentId != nil {
		p.DepartmentID = in.DepartmentId
	}
	if in.ManagerId != nil {
		p.ManagerID = in.ManagerId
	}
	if in.Location != nil {
		p.Location = in.Location
	}
	if in.Timezone != nil {
		p.Timezone = in.Timezone
	}
	if in.HireDate != nil {
		t := in.HireDate.Time
		p.HireDate = &t
	}
}
