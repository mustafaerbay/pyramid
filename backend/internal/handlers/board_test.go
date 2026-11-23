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
