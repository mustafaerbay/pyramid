package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestPeople_CreateThenGet(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Person
	status := doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@example.com",
	}, &created)

	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", status, http.StatusCreated)
	}
	if created.Id == uuid.Nil {
		t.Fatal("created.Id is nil, want a generated UUID")
	}
	// Defaults from NewPersonFromCreate should have been applied server-side.
	if created.EmploymentType != api.EmploymentTypeFullTime {
		t.Errorf("EmploymentType = %q, want full_time (default)", created.EmploymentType)
	}
	if created.Status != api.PersonStatusActive {
		t.Errorf("Status = %q, want active (default)", created.Status)
	}

	var fetched api.Person
	status = doJSON(t, r, http.MethodGet, "/api/v1/people/"+created.Id.String(), nil, &fetched)
	if status != http.StatusOK {
		t.Fatalf("get status = %d, want %d", status, http.StatusOK)
	}
	if fetched.Id != created.Id || fetched.Email != "ada@example.com" {
		t.Errorf("fetched = %+v, want id=%v email=ada@example.com", fetched, created.Id)
	}
}

func TestPeople_Create_InvalidBodyReturns400(t *testing.T) {
	r, _ := newTestRouter(t)

	// firstName is required by the OpenAPI schema's binding tags; sending a
	// malformed JSON body (rather than merely an omitted-but-valid field)
	// guarantees ShouldBindJSON itself fails regardless of exact validation
	// tags on PersonCreate.
	req := map[string]any{"firstName": 12345} // wrong type for a string field
	status := doJSON(t, r, http.MethodPost, "/api/v1/people", req, nil)

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
}

func TestPeople_Get_NotFound(t *testing.T) {
	r, _ := newTestRouter(t)

	var errBody api.Error
	status := doJSON(t, r, http.MethodGet, "/api/v1/people/"+uuid.New().String(), nil, &errBody)

	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if errBody.Message == "" {
		t.Error("Error.Message is empty, want a not-found message")
	}
}

func TestPeople_Update_PartialPatch(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com",
	}, &created)

	newLastName := "King"
	var updated api.Person
	status := doJSON(t, r, http.MethodPatch, "/api/v1/people/"+created.Id.String(),
		map[string]any{"lastName": newLastName}, &updated)

	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if updated.LastName != "King" {
		t.Errorf("LastName = %q, want King", updated.LastName)
	}
	// Untouched fields survive the partial patch.
	if updated.FirstName != "Ada" {
		t.Errorf("FirstName = %q, want unchanged Ada", updated.FirstName)
	}
	if updated.Email != "ada@example.com" {
		t.Errorf("Email = %q, want unchanged", updated.Email)
	}
}

func TestPeople_Update_NotFound(t *testing.T) {
	r, _ := newTestRouter(t)

	status := doJSON(t, r, http.MethodPatch, "/api/v1/people/"+uuid.New().String(),
		map[string]any{"lastName": "King"}, nil)

	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
}

func TestPeople_Delete(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{
		FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com",
	}, &created)

	status := doJSON(t, r, http.MethodDelete, "/api/v1/people/"+created.Id.String(), nil, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", status, http.StatusNoContent)
	}

	// Deleting again (already gone) is a not-found, not a second success.
	status = doJSON(t, r, http.MethodDelete, "/api/v1/people/"+created.Id.String(), nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want %d", status, http.StatusNotFound)
	}

	status = doJSON(t, r, http.MethodGet, "/api/v1/people/"+created.Id.String(), nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", status, http.StatusNotFound)
	}
}

func TestPeople_List_PaginationAndStatusFilter(t *testing.T) {
	r, _ := newTestRouter(t)

	for i := 0; i < 3; i++ {
		doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{
			FirstName: "Person", LastName: "Number", Email: "p@example.com",
		}, nil)
	}

	var list api.PersonList
	status := doJSON(t, r, http.MethodGet, "/api/v1/people?pageSize=2", nil, &list)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want %d", status, http.StatusOK)
	}
	if len(list.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2 (pageSize clamp)", len(list.Data))
	}
	if list.Pagination.Total != 3 {
		t.Errorf("Pagination.Total = %d, want 3", list.Pagination.Total)
	}
	if list.Pagination.PageSize != 2 {
		t.Errorf("Pagination.PageSize = %d, want 2", list.Pagination.PageSize)
	}

	// All freshly created people default to status=active, so filtering for
	// a different status should come back empty without erroring.
	var filtered api.PersonList
	status = doJSON(t, r, http.MethodGet, "/api/v1/people?status=on_leave", nil, &filtered)
	if status != http.StatusOK {
		t.Fatalf("filtered list status = %d, want %d", status, http.StatusOK)
	}
	if len(filtered.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0 for status=on_leave filter", len(filtered.Data))
	}
}
