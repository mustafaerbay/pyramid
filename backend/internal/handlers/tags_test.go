package handlers

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// tagsPtr builds a *[]string for a request; no args gives an empty (not
// nil) slice so it marshals as [] rather than null.
func tagsPtr(tags ...string) *[]string {
	if tags == nil {
		tags = []string{}
	}
	return &tags
}

func TestTags_CreateNormalizesAndSorts(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Task
	status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Tagged",
		Tags: tagsPtr("  frontend ", "Bug", "bug", "API"),
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", status)
	}
	// Trimmed, case-insensitive duplicates dropped (first spelling wins),
	// sorted ignoring case.
	if want := []string{"API", "Bug", "frontend"}; !reflect.DeepEqual(created.Tags, want) {
		t.Errorf("Tags = %v, want %v", created.Tags, want)
	}

	var got api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+created.Id.String(), nil, &got)
	if want := []string{"API", "Bug", "frontend"}; !reflect.DeepEqual(got.Tags, want) {
		t.Errorf("GET Tags = %v, want %v", got.Tags, want)
	}
}

func TestTags_EmptyArrayNotNull(t *testing.T) {
	r, _ := newTestRouter(t)
	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "Plain"}, &created)
	if created.Tags == nil || len(created.Tags) != 0 {
		t.Errorf("Tags = %#v, want empty non-nil slice", created.Tags)
	}
}

func TestTags_ReuseExistingSpellingAcrossProjects(t *testing.T) {
	r, db := newTestRouter(t)
	var a, b api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "A", Tags: tagsPtr("Backend")}, &a)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "B", Tags: tagsPtr("backend")}, &b)

	if want := []string{"Backend"}; !reflect.DeepEqual(b.Tags, want) {
		t.Errorf("second task Tags = %v, want %v (existing spelling reused)", b.Tags, want)
	}
	var n int64
	db.Table("labels").Count(&n)
	if n != 1 {
		t.Errorf("labels rows = %d, want 1 (one shared tag)", n)
	}
}

func TestTags_UpdateReplacesOrLeavesAlone(t *testing.T) {
	r, _ := newTestRouter(t)
	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "T", Tags: tagsPtr("a", "b")}, &created)
	path := "/api/v1/tasks/" + created.Id.String()

	var updated api.Task
	title := "Renamed"
	doJSON(t, r, http.MethodPatch, path, api.TaskUpdate{Title: &title}, &updated)
	if want := []string{"a", "b"}; !reflect.DeepEqual(updated.Tags, want) {
		t.Errorf("after title-only PATCH Tags = %v, want %v", updated.Tags, want)
	}

	doJSON(t, r, http.MethodPatch, path, api.TaskUpdate{Tags: tagsPtr("c")}, &updated)
	if want := []string{"c"}; !reflect.DeepEqual(updated.Tags, want) {
		t.Errorf("after tags PATCH Tags = %v, want %v", updated.Tags, want)
	}

	doJSON(t, r, http.MethodPatch, path, api.TaskUpdate{Tags: tagsPtr()}, &updated)
	if len(updated.Tags) != 0 {
		t.Errorf("after empty tags PATCH Tags = %v, want none", updated.Tags)
	}
}

func TestTags_Validation(t *testing.T) {
	r, _ := newTestRouter(t)
	many := make([]string, 21)
	for i := range many {
		many[i] = strings.Repeat("x", i+1)
	}
	cases := map[string]*[]string{
		"blank":    tagsPtr("ok", "   "),
		"too long": tagsPtr(strings.Repeat("y", 41)),
		"too many": &many,
	}
	for name, tags := range cases {
		status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: name, Tags: tags}, nil)
		if status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, status)
		}
	}
}

func TestTags_ListAndFilter(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	var one, two api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "One", Tags: tagsPtr("ui", "Bug")}, &one)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "Two", Tags: tagsPtr("ui")}, &two)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "Three"}, nil)
	// A tag no task uses any more drops out of the list.
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "Four", Tags: tagsPtr("gone")}, nil)
	var four api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+project.String()+"&tag=gone", nil, &four)
	if len(four.Data) != 1 {
		t.Fatalf("tag=gone matched %d tasks, want 1", len(four.Data))
	}
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+four.Data[0].Id.String(), api.TaskUpdate{Tags: tagsPtr()}, nil)

	var tags api.TagList
	if status := doJSON(t, r, http.MethodGet, "/api/v1/tags", nil, &tags); status != http.StatusOK {
		t.Fatalf("GET /tags status = %d", status)
	}
	if want := []string{"Bug", "ui"}; !reflect.DeepEqual(tags.Data, want) {
		t.Errorf("tags = %v, want %v", tags.Data, want)
	}

	var list api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+project.String()+"&tag=UI", nil, &list)
	if len(list.Data) != 2 || list.Pagination.Total != 2 {
		t.Errorf("tag=UI matched %d (total %d), want 2", len(list.Data), list.Pagination.Total)
	}
	for _, task := range list.Data {
		if len(task.Tags) == 0 {
			t.Errorf("listed task %q has no tags loaded", task.Title)
		}
	}
}
