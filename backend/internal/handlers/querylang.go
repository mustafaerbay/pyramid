package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// The query language: groups of conditions joined by AND / OR, run in Go
// over the items the plain filters picked (so custom fields and tags work
// the same on every database).

const (
	maxQueryDepth      = 3
	maxQueryConditions = 50
	maxQueryCandidates = 5000
)

type matcher func(t *models.Task) bool

type fieldKind int

const (
	kindText fieldKind = iota
	kindKey            // exact, case-insensitive (state, status, type, priority)
	kindID             // one id (sprint, project, parent)
	kindIDs            // many ids (assignee)
	kindTags
	kindArea
	kindNumber
	kindDate
	kindCustom
)

var queryFields = map[string]fieldKind{
	"title": kindText, "description": kindText,
	"state": kindKey, "status": kindKey, "type": kindKey, "priority": kindKey,
	"sprint": kindID, "project": kindID, "parent": kindID,
	"assignee": kindIDs, "tag": kindTags, "area": kindArea,
	"storyPoints": kindNumber, "manDays": kindNumber, "remaining": kindNumber,
	"dueDate": kindDate, "startDate": kindDate, "createdAt": kindDate,
}

var relDate = regexp.MustCompile(`^@today(?:([+-])(\d{1,4}))?$`)

// queryDate turns YYYY-MM-DD or @today±N into YYYY-MM-DD.
func queryDate(v string, now time.Time) (string, error) {
	v = strings.TrimSpace(v)
	if m := relDate.FindStringSubmatch(v); m != nil {
		d := now
		if m[1] != "" {
			n, _ := strconv.Atoi(m[2])
			if m[1] == "-" {
				n = -n
			}
			d = d.AddDate(0, 0, n)
		}
		return d.Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return "", fmt.Errorf("%q isn't a date (YYYY-MM-DD or @today-7)", v)
	}
	return v, nil
}

type queryCompiler struct {
	h      *Handler
	now    time.Time
	custom map[string]bool
	count  int
}

func (qc *queryCompiler) group(g api.QueryGroup, depth int) (matcher, error) {
	if depth > maxQueryDepth {
		return nil, fmt.Errorf("groups can nest up to %d deep", maxQueryDepth)
	}
	if !g.Match.Valid() {
		return nil, errors.New("a group matches all or any")
	}
	var parts []matcher
	for _, c := range g.Conditions {
		qc.count++
		if qc.count > maxQueryConditions {
			return nil, fmt.Errorf("up to %d conditions", maxQueryConditions)
		}
		m, err := qc.condition(c)
		if err != nil {
			return nil, err
		}
		parts = append(parts, m)
	}
	if g.Groups != nil {
		for _, sub := range *g.Groups {
			m, err := qc.group(sub, depth+1)
			if err != nil {
				return nil, err
			}
			parts = append(parts, m)
		}
	}
	any := g.Match == api.QueryGroupMatchAny
	return func(t *models.Task) bool {
		if len(parts) == 0 {
			return true
		}
		for _, p := range parts {
			if p(t) == any {
				return any
			}
		}
		return !any
	}, nil
}

func (qc *queryCompiler) condition(c api.QueryCondition) (matcher, error) {
	op := c.Op
	if !op.Valid() {
		return nil, fmt.Errorf("unknown operator %q", op)
	}
	val := ""
	if c.Value != nil {
		val = strings.TrimSpace(*c.Value)
	}
	kind, ok := queryFields[c.Field]
	key := ""
	if strings.HasPrefix(c.Field, "custom:") {
		key = strings.TrimPrefix(c.Field, "custom:")
		if !qc.custom[key] {
			return nil, fmt.Errorf("no custom field %q in these projects", key)
		}
		kind, ok = kindCustom, true
	}
	if !ok {
		return nil, fmt.Errorf("can't query on %q", c.Field)
	}
	isEmptyOp := op == api.QueryConditionOpEmpty || op == api.QueryConditionOpNotEmpty
	ordered := op == api.QueryConditionOpGt || op == api.QueryConditionOpGte || op == api.QueryConditionOpLt || op == api.QueryConditionOpLte
	textOp := op == api.QueryConditionOpContains || op == api.QueryConditionOpNotContains
	if !isEmptyOp && val == "" {
		return nil, fmt.Errorf("%s needs a value", c.Field)
	}

	// Each kind reads the item's value(s) as strings; numbers and dates
	// are compared as such.
	var num float64
	switch kind {
	case kindNumber:
		if !isEmptyOp {
			n, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("%q isn't a number", val)
			}
			num = n
		}
		if textOp {
			return nil, errors.New("numbers can't use contains")
		}
	case kindDate:
		if !isEmptyOp {
			d, err := queryDate(val, qc.now)
			if err != nil {
				return nil, err
			}
			val = d
		}
		if textOp {
			return nil, errors.New("dates can't use contains")
		}
	case kindKey, kindID, kindIDs, kindTags, kindArea:
		if ordered {
			return nil, fmt.Errorf("%s can't use greater/less than", c.Field)
		}
	}
	var area map[uuid.UUID]bool
	if kind == kindArea && !isEmptyOp {
		ids, err := qc.h.areaAndBelow(val)
		if err != nil {
			return nil, errors.New("unknown area in query")
		}
		area = map[uuid.UUID]bool{}
		for _, id := range ids {
			area[id] = true
		}
	}
	lv := strings.ToLower(val)

	values := func(t *models.Task) []string {
		s := func(p *string) []string {
			if p == nil || *p == "" {
				return nil
			}
			return []string{*p}
		}
		id := func(p *uuid.UUID) []string {
			if p == nil {
				return nil
			}
			return []string{p.String()}
		}
		f := func(p *float64) []string {
			if p == nil {
				return nil
			}
			return []string{strconv.FormatFloat(*p, 'f', -1, 64)}
		}
		d := func(p *time.Time) []string {
			if p == nil {
				return nil
			}
			return []string{p.Format("2006-01-02")}
		}
		switch c.Field {
		case "title":
			return []string{t.Title}
		case "description":
			return s(t.Description)
		case "state":
			if t.State == "" {
				return []string{string(t.Status)}
			}
			return []string{t.State}
		case "status":
			return []string{string(t.Status)}
		case "type":
			out := []string{string(t.Type)}
			if t.CustomType != nil {
				out = append(out, *t.CustomType)
			}
			return out
		case "priority":
			return []string{string(t.Priority)}
		case "sprint":
			return id(t.SprintID)
		case "project":
			return []string{t.ProjectID.String()}
		case "parent":
			return id(t.ParentID)
		case "area":
			return id(t.AreaID)
		case "assignee":
			out := make([]string, len(t.AssigneeIDs))
			for i, a := range t.AssigneeIDs {
				out[i] = a.String()
			}
			return out
		case "tag":
			return t.Tags
		case "storyPoints":
			return f(t.StoryPoints)
		case "manDays":
			if t.EstimateManDays == nil && len(t.ManDays) > 0 {
				sum := 0.0
				for _, v := range t.ManDays {
					sum += v
				}
				return f(&sum)
			}
			return f(t.EstimateManDays)
		case "remaining":
			return f(t.RemainingManDays)
		case "dueDate":
			return d(t.DueDate)
		case "startDate":
			return d(t.StartDate)
		case "createdAt":
			ct := t.CreatedAt
			return d(&ct)
		}
		// custom field
		v, ok := t.Values[key]
		if !ok || v == nil {
			return nil
		}
		switch x := v.(type) {
		case string:
			return s(&x)
		case float64:
			return f(&x)
		case bool:
			return []string{strconv.FormatBool(x)}
		case []any:
			out := []string{}
			for _, e := range x {
				out = append(out, fmt.Sprint(e))
			}
			return out
		}
		return []string{fmt.Sprint(v)}
	}

	valNum, valIsNum := num, kind == kindNumber
	if kind == kindCustom {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			valNum, valIsNum = n, true
		}
	}
	// one says whether a single value satisfies a positive operator.
	one := func(v string, op api.QueryConditionOp) bool {
		switch op {
		case api.QueryConditionOpEq:
			if area != nil {
				id, err := uuid.Parse(v)
				return err == nil && area[id]
			}
			return strings.ToLower(v) == lv
		case api.QueryConditionOpContains:
			return strings.Contains(strings.ToLower(v), lv)
		}
		cmp := strings.Compare(v, val) // dates (YYYY-MM-DD) and text
		if valIsNum {
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return false
			}
			cmp = 0
			if n < valNum {
				cmp = -1
			} else if n > valNum {
				cmp = 1
			}
		}
		switch op {
		case api.QueryConditionOpGt:
			return cmp > 0
		case api.QueryConditionOpGte:
			return cmp >= 0
		case api.QueryConditionOpLt:
			return cmp < 0
		case api.QueryConditionOpLte:
			return cmp <= 0
		}
		return false
	}
	anyHit := func(vs []string, op api.QueryConditionOp) bool {
		for _, v := range vs {
			if one(v, op) {
				return true
			}
		}
		return false
	}

	return func(t *models.Task) bool {
		vs := values(t)
		switch op {
		case api.QueryConditionOpEmpty:
			return len(vs) == 0
		case api.QueryConditionOpNotEmpty:
			return len(vs) > 0
		case api.QueryConditionOpNe:
			return !anyHit(vs, api.QueryConditionOpEq)
		case api.QueryConditionOpNotContains:
			return !anyHit(vs, api.QueryConditionOpContains)
		}
		return anyHit(vs, op)
	}, nil
}

// compileQuery checks a where-group against the custom fields of the
// projects in scope (all projects when none are picked).
func (h *Handler) compileQuery(g *api.QueryGroup, projectIDs *[]uuid.UUID) (matcher, error) {
	if g == nil {
		return func(*models.Task) bool { return true }, nil
	}
	var projects []models.Project
	q := h.db.Model(&models.Project{})
	if nonEmpty(projectIDs) {
		q = q.Where("id IN ?", *projectIDs)
	}
	if err := q.Find(&projects).Error; err != nil {
		return nil, err
	}
	custom := map[string]bool{}
	for _, p := range projects {
		for _, f := range p.Fields() {
			custom[f.Key] = true
		}
	}
	qc := &queryCompiler{h: h, now: time.Now(), custom: custom}
	return qc.group(*g, 1)
}
