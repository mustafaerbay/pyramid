package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewPersonFromCreate_Defaults(t *testing.T) {
	in := api.PersonCreate{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@example.com",
	}

	p := NewPersonFromCreate(in)

	if p.EmploymentType != api.EmploymentTypeFullTime {
		t.Errorf("EmploymentType = %q, want %q (default)", p.EmploymentType, api.EmploymentTypeFullTime)
	}
	if p.Status != api.PersonStatusActive {
		t.Errorf("Status = %q, want %q (default)", p.Status, api.PersonStatusActive)
	}
	if p.FirstName != "Ada" || p.LastName != "Lovelace" {
		t.Errorf("name = %q %q, want Ada Lovelace", p.FirstName, p.LastName)
	}
	if p.Email != "ada@example.com" {
		t.Errorf("Email = %q, want ada@example.com", p.Email)
	}
	if p.HireDate != nil {
		t.Errorf("HireDate = %v, want nil when not supplied", p.HireDate)
	}
}

func TestNewPersonFromCreate_ExplicitEmploymentTypeAndHireDate(t *testing.T) {
	contractor := api.EmploymentTypeContractor
	hireDate := openapi_types.Date{Time: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)}

	in := api.PersonCreate{
		FirstName:      "Alan",
		LastName:       "Turing",
		Email:          "alan@example.com",
		EmploymentType: &contractor,
		HireDate:       &hireDate,
	}

	p := NewPersonFromCreate(in)

	if p.EmploymentType != api.EmploymentTypeContractor {
		t.Errorf("EmploymentType = %q, want contractor", p.EmploymentType)
	}
	if p.HireDate == nil || !p.HireDate.Equal(hireDate.Time) {
		t.Errorf("HireDate = %v, want %v", p.HireDate, hireDate.Time)
	}
}

func TestPerson_ApplyUpdate_OnlyTouchesSuppliedFields(t *testing.T) {
	p := Person{
		FirstName:      "Ada",
		LastName:       "Lovelace",
		Email:          "ada@example.com",
		EmploymentType: api.EmploymentTypeFullTime,
		Status:         api.PersonStatusActive,
	}

	newLastName := "King"
	onLeave := api.PersonStatusOnLeave
	p.ApplyUpdate(api.PersonUpdate{
		LastName: &newLastName,
		Status:   &onLeave,
	})

	if p.LastName != "King" {
		t.Errorf("LastName = %q, want King", p.LastName)
	}
	if p.Status != api.PersonStatusOnLeave {
		t.Errorf("Status = %q, want on_leave", p.Status)
	}
	// Untouched fields must survive the partial patch unchanged.
	if p.FirstName != "Ada" {
		t.Errorf("FirstName = %q, want unchanged Ada", p.FirstName)
	}
	if p.Email != "ada@example.com" {
		t.Errorf("Email = %q, want unchanged", p.Email)
	}
}

func TestPerson_ApplyUpdate_EmptyPatchChangesNothing(t *testing.T) {
	p := Person{FirstName: "Ada", LastName: "Lovelace", Status: api.PersonStatusActive}
	before := p

	p.ApplyUpdate(api.PersonUpdate{})

	if p != before {
		t.Errorf("empty patch mutated the record: got %+v, want %+v", p, before)
	}
}

func TestPerson_ApplyUpdate_AllFields(t *testing.T) {
	p := Person{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}

	firstName, lastName := "Augusta", "King"
	email := openapi_types.Email("augusta@example.com")
	phone := "+1-555-0100"
	title := "Countess of Lovelace"
	seniority := api.SeniorityLevelPrincipal
	employmentType := api.EmploymentTypePartTime
	status := api.PersonStatusOffboarding
	departmentID, managerID := uuid.New(), uuid.New()
	location, timezone := "London", "Europe/London"
	hireDate := openapi_types.Date{Time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}

	p.ApplyUpdate(api.PersonUpdate{
		FirstName: &firstName, LastName: &lastName, Email: &email, Phone: &phone,
		Title: &title, Seniority: &seniority, EmploymentType: &employmentType, Status: &status,
		DepartmentId: &departmentID, ManagerId: &managerID, Location: &location,
		Timezone: &timezone, HireDate: &hireDate,
	})

	if p.FirstName != firstName || p.LastName != lastName {
		t.Errorf("name = %q %q, want %q %q", p.FirstName, p.LastName, firstName, lastName)
	}
	if p.Email != string(email) {
		t.Errorf("Email = %q, want %q", p.Email, email)
	}
	if p.Phone == nil || *p.Phone != phone {
		t.Errorf("Phone = %v, want %q", p.Phone, phone)
	}
	if p.Title == nil || *p.Title != title {
		t.Errorf("Title = %v, want %q", p.Title, title)
	}
	if p.Seniority == nil || *p.Seniority != seniority {
		t.Errorf("Seniority = %v, want %q", p.Seniority, seniority)
	}
	if p.EmploymentType != employmentType {
		t.Errorf("EmploymentType = %q, want %q", p.EmploymentType, employmentType)
	}
	if p.Status != status {
		t.Errorf("Status = %q, want %q", p.Status, status)
	}
	if p.DepartmentID == nil || *p.DepartmentID != departmentID {
		t.Errorf("DepartmentID = %v, want %v", p.DepartmentID, departmentID)
	}
	if p.ManagerID == nil || *p.ManagerID != managerID {
		t.Errorf("ManagerID = %v, want %v", p.ManagerID, managerID)
	}
	if p.Location == nil || *p.Location != location {
		t.Errorf("Location = %v, want %q", p.Location, location)
	}
	if p.Timezone == nil || *p.Timezone != timezone {
		t.Errorf("Timezone = %v, want %q", p.Timezone, timezone)
	}
	if p.HireDate == nil || !p.HireDate.Equal(hireDate.Time) {
		t.Errorf("HireDate = %v, want %v", p.HireDate, hireDate.Time)
	}
}

func TestPerson_ToAPI_OmitsNilHireDate(t *testing.T) {
	id := uuid.New()
	p := Person{ID: id, FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}

	out := p.ToAPI()

	if out.Id != id {
		t.Errorf("Id = %v, want %v", out.Id, id)
	}
	if out.HireDate != nil {
		t.Errorf("HireDate = %v, want nil", out.HireDate)
	}
}

func TestPerson_ToAPI_RoundTripsHireDate(t *testing.T) {
	hire := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	p := Person{ID: uuid.New(), FirstName: "Ada", LastName: "Lovelace", HireDate: &hire}

	out := p.ToAPI()

	if out.HireDate == nil || !out.HireDate.Equal(hire) {
		t.Errorf("HireDate = %v, want %v", out.HireDate, hire)
	}
}
