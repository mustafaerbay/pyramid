package handlers

import (
	"net/http"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestBoardSettings_DefaultsSaveAndValidate(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "B", Name: "Board"}, &p)
	if p.BoardSettings.Swimlanes != api.BoardSettingsSwimlanesNone || p.BoardSettings.CardColor != api.BoardSettingsCardColorNone || p.BoardSettings.WipLimits == nil {
		t.Errorf("defaults = %+v", p.BoardSettings)
	}

	path := "/api/v1/projects/" + p.Id.String()
	var got api.Project
	if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"boardSettings": map[string]any{
		"wipLimits": map[string]int{"in_progress": 3}, "swimlanes": "assignee", "cardColor": "type",
	}}, &got); s != http.StatusOK {
		t.Fatalf("save = %d", s)
	}
	doJSON(t, r, http.MethodGet, path, nil, &got)
	if got.BoardSettings.WipLimits["in_progress"] != 3 || got.BoardSettings.Swimlanes != api.BoardSettingsSwimlanesAssignee || got.BoardSettings.CardColor != api.BoardSettingsCardColorType {
		t.Errorf("saved = %+v", got.BoardSettings)
	}
	// Other project fields don't touch the board settings.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"name": "Board 2"}, &got)
	if got.BoardSettings.WipLimits["in_progress"] != 3 {
		t.Errorf("after name change = %+v", got.BoardSettings)
	}

	for name, bs := range map[string]map[string]any{
		"zero limit":    {"wipLimits": map[string]int{"todo": 0}, "swimlanes": "none", "cardColor": "none"},
		"unknown state": {"wipLimits": map[string]int{"nope": 2}, "swimlanes": "none", "cardColor": "none"},
		"bad lanes":     {"wipLimits": map[string]int{}, "swimlanes": "weather", "cardColor": "none"},
		"bad colour":    {"wipLimits": map[string]int{}, "swimlanes": "none", "cardColor": "rainbow"},
	} {
		if s := doJSON(t, r, http.MethodPatch, path, map[string]any{"boardSettings": bs}, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}

func TestBoardSettings_FineSettings(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "F", Name: "Fine"}, &p)
	path := "/api/v1/projects/" + p.Id.String()
	base := func(extra map[string]any) map[string]any {
		m := map[string]any{"wipLimits": map[string]int{}, "swimlanes": "none", "cardColor": "none"}
		for k, v := range extra {
			m[k] = v
		}
		return map[string]any{"boardSettings": m}
	}
	var got api.Project
	if s := doJSON(t, r, http.MethodPatch, path, base(map[string]any{
		"splitColumns":     []string{"in_progress"},
		"definitionOfDone": map[string]string{"in_progress": "Code reviewed"},
		"cardFields":       []string{"assignees", "due", "remaining"},
		"tagColors":        map[string]string{"ui": "emerald"},
		"cardRules":        []map[string]string{{"field": "priority", "value": "urgent", "color": "red"}},
	}), &got); s != http.StatusOK {
		t.Fatalf("save = %d", s)
	}
	b := got.BoardSettings
	if b.SplitColumns == nil || (*b.SplitColumns)[0] != "in_progress" || (*b.DefinitionOfDone)["in_progress"] != "Code reviewed" ||
		len(*b.CardFields) != 3 || (*b.TagColors)["ui"] != "emerald" || (*b.CardRules)[0].Color != "red" {
		t.Errorf("saved = %+v", b)
	}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	for name, extra := range map[string]map[string]any{
		"split unknown state": {"splitColumns": []string{"nope"}},
		"split done category": {"splitColumns": []string{"done"}},
		"dod unknown state":   {"definitionOfDone": map[string]string{"nope": "x"}},
		"dod too long":        {"definitionOfDone": map[string]string{"todo": string(long)}},
		"bad card field":      {"cardFields": []string{"weather"}},
		"bad custom field":    {"cardFields": []string{"custom:nope"}},
		"bad tag colour":      {"tagColors": map[string]string{"ui": "plaid"}},
		"bad rule field":      {"cardRules": []map[string]string{{"field": "mood", "value": "x", "color": "red"}}},
		"bad rule colour":     {"cardRules": []map[string]string{{"field": "type", "value": "bug", "color": "plaid"}}},
	} {
		if s := doJSON(t, r, http.MethodPatch, path, base(extra), nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}

func TestTask_BoardDoneClearsOnStateChange(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "D", Name: "Done"}, &p)
	var task api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "x", "state": "in_progress"}, &task)
	tp := "/api/v1/tasks/" + task.Id.String()
	var got api.Task
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"boardDone": true}, &got)
	if !got.BoardDone {
		t.Fatalf("boardDone not saved: %+v", got)
	}
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"title": "y"}, &got)
	if !got.BoardDone {
		t.Errorf("boardDone lost on unrelated change")
	}
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"state": "in_review"}, &got)
	if got.BoardDone {
		t.Errorf("boardDone kept after state change")
	}
	// Moving straight into the Done half of another column.
	doJSON(t, r, http.MethodPatch, tp, map[string]any{"state": "in_progress", "boardDone": true}, &got)
	if !got.BoardDone || got.State != "in_progress" {
		t.Errorf("move into done half = %+v", got)
	}
}
