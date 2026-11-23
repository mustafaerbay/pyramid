package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (h *Handler) projectFields(projectID uuid.UUID) []api.CustomField {
	var p models.Project
	if h.db.Select("id, custom_fields").First(&p, "id = ?", projectID).Error != nil {
		return []api.CustomField{}
	}
	return p.Fields()
}

// checkCustomValues validates values against the project's fields and
// returns them cleaned (blank values dropped, text trimmed).
func checkCustomValues(fields []api.CustomField, values map[string]any) (map[string]any, error) {
	byKey := map[string]api.CustomField{}
	for _, f := range fields {
		byKey[f.Key] = f
	}
	out := map[string]any{}
	for k, v := range values {
		f, ok := byKey[k]
		if !ok {
			return nil, fmt.Errorf("this project has no field %q", k)
		}
		if v == nil {
			continue
		}
		switch f.Type {
		case api.CustomFieldTypeText:
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be text", f.Label)
			}
			if s = strings.TrimSpace(s); s == "" {
				continue
			} else if utf8.RuneCountInString(s) > 500 {
				return nil, fmt.Errorf("%s can be at most 500 characters", f.Label)
			}
			out[k] = strings.TrimSpace(s)
		case api.CustomFieldTypeNumber:
			n, ok := v.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
				return nil, fmt.Errorf("%s must be a number", f.Label)
			}
			out[k] = n
		case api.CustomFieldTypeDate:
			s, ok := v.(string)
			if s == "" && ok {
				continue
			}
			if _, err := time.Parse("2006-01-02", s); !ok || !isoDate.MatchString(s) || err != nil {
				return nil, fmt.Errorf("%s must be a date like 2026-12-31", f.Label)
			}
			out[k] = s
		case api.CustomFieldTypeSelect:
			s, ok := v.(string)
			if s == "" && ok {
				continue
			}
			valid := false
			if ok && f.Options != nil {
				for _, o := range *f.Options {
					valid = valid || o == s
				}
			}
			if !valid {
				return nil, fmt.Errorf("%s must be one of its options", f.Label)
			}
			out[k] = s
		}
	}
	for _, f := range fields {
		if _, ok := out[f.Key]; f.Required && !ok {
			return nil, fmt.Errorf("%s is required", f.Label)
		}
	}
	return out, nil
}

func encodeValues(values map[string]any) *string {
	if len(values) == 0 {
		return nil
	}
	b, _ := json.Marshal(values)
	s := string(b)
	return &s
}

// attachCustom fills Values from the stored JSON, keeping only keys the
// item's project still defines.
func (h *Handler) attachCustom(tasks ...*models.Task) error {
	fields := map[uuid.UUID]map[string]bool{}
	types := map[uuid.UUID][]api.WorkItemTypeDef{}
	for _, t := range tasks {
		t.Values = map[string]any{}
		if t.CustomType != nil {
			if _, ok := types[t.ProjectID]; !ok {
				var p models.Project
				h.db.Select("id, work_item_types").First(&p, "id = ?", t.ProjectID)
				types[t.ProjectID] = p.ItemTypes()
			}
			for _, d := range types[t.ProjectID] {
				if d.Key == *t.CustomType {
					label, color := d.Label, d.Color
					t.TypeLabel, t.TypeColor = &label, &color
				}
			}
		}
		if t.CustomValues == nil {
			continue
		}
		if _, ok := fields[t.ProjectID]; !ok {
			keys := map[string]bool{}
			for _, f := range h.projectFields(t.ProjectID) {
				keys[f.Key] = true
			}
			fields[t.ProjectID] = keys
		}
		var raw map[string]any
		if json.Unmarshal([]byte(*t.CustomValues), &raw) != nil {
			continue
		}
		for k, v := range raw {
			if fields[t.ProjectID][k] {
				t.Values[k] = v
			}
		}
	}
	return nil
}
