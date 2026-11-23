// Command seed wipes Pyramid's data and loads a demo with history: people,
// projects with their own workflows/types/fields, epics down to tasks,
// past and current sprints, and backdated status changes so burndowns,
// cumulative flow, velocity and cycle times have something to show.
//
// Sign-in accounts and the people they belong to are kept. Run it inside
// the backend container (same DB settings as the server):
//
//	/app/seed --wipe
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"github.com/mustafaerbay/pyramid/backend/internal/database"
	"gorm.io/gorm"
)

var (
	db    *gorm.DB
	rng   = rand.New(rand.NewSource(42))
	today = time.Now().UTC().Truncate(24 * time.Hour)
	actor *uuid.UUID // the first admin account: who "made" the changes
)

func day(offset int) time.Time { return today.AddDate(0, 0, offset) }
func date(t time.Time) string  { return t.Format("2006-01-02") }
func js(v any) string          { b, _ := json.Marshal(v); return string(b) }
func pick[T any](xs []T) T     { return xs[rng.Intn(len(xs))] }

func ins(table string, row map[string]any) uuid.UUID {
	id, ok := row["id"].(uuid.UUID)
	if !ok {
		id = uuid.New()
		row["id"] = id
	}
	if err := db.Table(table).Create(row).Error; err != nil {
		log.Fatalf("insert into %s: %v (%v)", table, err, row)
	}
	return id
}

func wipe() {
	// Keep accounts and their people; everything else goes.
	for _, q := range []string{
		"TRUNCATE projects, labels, notifications, activity_log, saved_queries, dashboards, attachments, password_resets, goals, contributions, leave_requests RESTART IDENTITY CASCADE",
		"DELETE FROM people WHERE id NOT IN (SELECT person_id FROM users)",
	} {
		if err := db.Exec(q).Error; err != nil {
			log.Fatalf("%s: %v", q, err)
		}
	}
}

type person struct {
	id    uuid.UUID
	first string
}

func seedPeople() []person {
	rows := []struct {
		first, last, title, seniority string
		cap                           float64
	}{
		{"Ada", "Kim", "Frontend Engineer", "mid", 20}, {"Bruno", "Silva", "Backend Engineer", "senior", 20},
		{"Chen", "Wei", "Engineering Manager", "manager", 12}, {"Dilara", "Yilmaz", "Product Manager", "senior", 16},
		{"Elena", "Rossi", "Backend Engineer", "mid", 20}, {"Farid", "Haddad", "Fullstack Engineer", "senior", 20},
		{"Grace", "Okafor", "Product Designer", "senior", 18}, {"Hugo", "Martins", "Backend Engineer", "junior", 20},
		{"Ivy", "Novak", "Data Engineer", "mid", 20}, {"Julia", "Costa", "QA Engineer", "mid", 20},
		{"Noah", "Berg", "Mobile Engineer", "senior", 20}, {"Priya", "Nair", "Tech Lead", "lead", 16},
	}
	out := make([]person, len(rows))
	for i, r := range rows {
		email := strings.ToLower(r.first+"."+r.last) + "@pyramid.dev"
		// People who have an account were kept: refresh them instead.
		var kept []uuid.UUID
		db.Table("people").Where("LOWER(email) = ?", email).Pluck("id", &kept)
		if len(kept) > 0 {
			db.Table("people").Where("id = ?", kept[0]).Updates(map[string]any{"title": r.title, "seniority": r.seniority, "monthly_capacity_man_days": r.cap, "status": "active"})
			out[i] = person{kept[0], r.first}
			continue
		}
		out[i] = person{ins("people", map[string]any{
			"first_name": r.first, "last_name": r.last, "title": r.title, "seniority": r.seniority,
			"email": email, "status": "active",
			"monthly_capacity_man_days": r.cap, "hire_date": date(day(-400 - 30*i)), "location": pick([]string{"Istanbul", "Berlin", "Remote", "Lisbon"}),
		}), r.first}
	}
	return out
}

// History of one item: (time, from, to) status changes.
type change struct {
	at       time.Time
	from, to string
}

func logChanges(taskID uuid.UUID, created time.Time, changes []change, stateOf func(string) string) {
	ins("activity_log", map[string]any{"actor_id": actor, "action": "created", "entity_type": "task", "entity_id": taskID, "created_at": created})
	for _, c := range changes {
		ins("activity_log", map[string]any{
			"actor_id": actor, "action": "updated", "entity_type": "task", "entity_id": taskID, "created_at": c.at,
			"changes": js(map[string]any{
				"status": map[string]string{"from": c.from, "to": c.to},
				"state":  map[string]string{"from": stateOf(c.from), "to": stateOf(c.to)},
			}),
		})
	}
}

type sprint struct {
	id         uuid.UUID
	start, end time.Time
	status     string
}

type projectSpec struct {
	key, name, desc, status, priority, health string
	workflow                                  []map[string]string
	types                                     []map[string]string
	fields                                    []map[string]any
	areas                                     []string
	epics                                     []string
}

var defaultFlow = []map[string]string{
	{"key": "backlog", "label": "Backlog", "category": "backlog"}, {"key": "todo", "label": "To do", "category": "todo"},
	{"key": "in_progress", "label": "In progress", "category": "in_progress"}, {"key": "in_review", "label": "In review", "category": "in_review"},
	{"key": "blocked", "label": "Blocked", "category": "blocked"}, {"key": "done", "label": "Done", "category": "done"},
	{"key": "cancelled", "label": "Cancelled", "category": "cancelled"},
}

func main() {
	wipeFlag := flag.Bool("wipe", false, "required: deletes all data except sign-in accounts")
	flag.Parse()
	if !*wipeFlag {
		log.Fatal("this deletes all projects, items, people without accounts and their history; run with --wipe to go ahead")
	}
	var err error
	if db, err = database.Connect(config.Load()); err != nil {
		log.Fatal(err)
	}
	var admins []uuid.UUID
	db.Table("users").Where("system_role = ?", "admin").Order("created_at").Limit(1).Pluck("id", &admins)
	if len(admins) > 0 {
		actor = &admins[0]
	}

	wipe()
	people := seedPeople()
	var adminPerson []uuid.UUID
	db.Table("users").Where("system_role = ?", "admin").Pluck("person_id", &adminPerson)

	specs := []projectSpec{
		{"CAD", "Cadence Notifications", "Digest emails and in-app alerts for approvals, mentions and due dates.", "active", "high", "at_risk",
			[]map[string]string{
				{"key": "new", "label": "New", "category": "backlog"}, {"key": "ready", "label": "Ready", "category": "todo"},
				{"key": "doing", "label": "Doing", "category": "in_progress"}, {"key": "code_review", "label": "Code review", "category": "in_review"},
				{"key": "qa", "label": "QA", "category": "in_review"}, {"key": "blocked", "label": "Blocked", "category": "blocked"},
				{"key": "shipped", "label": "Shipped", "category": "done"}, {"key": "dropped", "label": "Dropped", "category": "cancelled"},
			},
			[]map[string]string{{"key": "spike", "label": "Spike", "base": "task", "color": "pink"}},
			[]map[string]any{{"key": "customer", "label": "Customer", "type": "text", "required": false}, {"key": "risk", "label": "Risk", "type": "select", "options": []string{"Low", "Medium", "High"}, "required": false}},
			[]string{"Web", "Web/Settings", "API", "Mobile"},
			[]string{"Daily digest", "In-app alert centre"}},
		{"ATL", "Atlas Data Pipeline", "Event ingestion and a warehouse the reporting team can trust.", "active", "critical", "on_track",
			defaultFlow, []map[string]string{{"key": "tech_debt", "label": "Tech debt", "base": "task", "color": "slate"}}, nil,
			[]string{"Ingest", "Warehouse", "Warehouse/Models"},
			[]string{"Streaming ingestion", "Self-serve warehouse"}},
		{"ORB", "Orbit Design System", "Shared components and tokens for every product surface.", "active", "medium", "on_track",
			defaultFlow, nil, nil, []string{"Tokens", "Components"},
			[]string{"Component library v2", "Accessibility pass"}},
		{"HLX", "Helix Search", "Search across projects, people and documents.", "planning", "medium", "on_track",
			defaultFlow, nil, nil, nil, []string{"Search MVP"}},
	}
	teams := [][]int{{0, 1, 5, 7, 9, 11}, {1, 4, 8, 2, 7}, {0, 6, 9, 10}, {3, 5, 8}}

	for pi, sp := range specs {
		team := teams[pi]
		pid := ins("projects", map[string]any{
			"key": sp.key, "name": sp.name, "description": sp.desc, "status": sp.status, "priority": sp.priority, "health": sp.health,
			"lead_id": people[team[0]].id, "start_date": date(day(-90)), "target_date": date(day(75 + 20*pi)),
			"workflow": js(sp.workflow), "work_item_types": js(sp.types), "custom_fields": js(orEmpty(sp.fields)),
			"board_settings": js(map[string]any{"wipLimits": map[string]int{}, "swimlanes": "none", "cardColor": "type"}),
		})
		stateOf := func(cat string) string {
			for _, s := range sp.workflow {
				if s["category"] == cat {
					return s["key"]
				}
			}
			return cat
		}
		roles := []string{"lead", "backend_engineer", "frontend_engineer", "contributor", "reviewer", "contributor"}
		for i, m := range team {
			ins("project_members", map[string]any{"project_id": pid, "person_id": people[m].id, "role": roles[i%len(roles)], "allocation_pct": pick([]int{50, 80, 100}), "joined_at": date(day(-90))})
		}
		areaIDs := []uuid.UUID{}
		for _, a := range sp.areas {
			areaIDs = append(areaIDs, ins("areas", map[string]any{"project_id": pid, "path": a}))
		}
		v1 := ins("project_versions", map[string]any{"project_id": pid, "name": "v1.0", "status": "released", "target_date": date(day(-30)), "released_at": date(day(-28))})
		v2 := ins("project_versions", map[string]any{"project_id": pid, "name": "v1.1", "status": "in_progress", "target_date": date(day(25))})
		milestones := []uuid.UUID{
			ins("milestones", map[string]any{"project_id": pid, "name": "M1 — Foundations", "status": "completed", "start_date": date(day(-90)), "due_date": date(day(-35)), "completed_at": date(day(-33)), "version_id": v1}),
			ins("milestones", map[string]any{"project_id": pid, "name": "M2 — First users", "status": "in_progress", "start_date": date(day(-34)), "due_date": date(day(20)), "version_id": v2}),
			ins("milestones", map[string]any{"project_id": pid, "name": "M3 — General release", "status": "pending", "start_date": date(day(21)), "due_date": date(day(70))}),
		}

		// Two-week sprints: five done, one running, one planned (not for a planning project).
		sprints := []sprint{}
		if sp.status == "active" {
			// The running sprint started on the Monday on or before 6 days ago.
			start := day(-6)
			for start.Weekday() != time.Monday {
				start = start.AddDate(0, 0, -1)
			}
			first := start.AddDate(0, 0, -14*5)
			for i := 0; i < 7; i++ {
				s := first.AddDate(0, 0, 14*i)
				e := s.AddDate(0, 0, 13)
				st := "completed"
				if !today.After(e) && !today.Before(s) {
					st = "active"
				} else if s.After(today) {
					st = "planned"
				}
				id := ins("sprints", map[string]any{"project_id": pid, "name": fmt.Sprintf("Sprint %d", i+1), "status": st,
					"goal":       pick([]string{"Ship the happy path", "Pay down review debt", "Get it in front of the pilot team", "Close the open bugs"}),
					"start_date": date(s), "end_date": date(e)})
				sprints = append(sprints, sprint{id, s, e, st})
			}
		}

		number, rank := 0, 0.0
		mk := func(row map[string]any) uuid.UUID {
			number++
			rank += 1000
			row["project_id"], row["number"], row["rank"] = pid, number, rank
			return ins("tasks", row)
		}
		tagNames := []string{"frontend", "backend", "api", "ux", "perf", "security"}
		labelIDs := map[string]uuid.UUID{}
		for _, t := range tagNames {
			var id []uuid.UUID
			db.Table("labels").Where("name = ? AND project_id IS NULL", t).Pluck("id", &id)
			if len(id) == 0 {
				labelIDs[t] = ins("labels", map[string]any{"name": t, "color": "#888888"})
			} else {
				labelIDs[t] = id[0]
			}
		}
		var leaves []uuid.UUID
		for ei, epicTitle := range sp.epics {
			epic := mk(map[string]any{"title": epicTitle, "type": "epic", "status": "in_progress", "state": stateOf("in_progress"),
				"priority": "high", "start_date": date(day(-80 + 10*ei)), "due_date": date(day(40 + 20*ei)), "created_at": day(-85), "updated_at": day(-85),
				"milestone_id": milestones[1]})
			logChanges(epic, day(-85), []change{{day(-80), "backlog", "in_progress"}}, stateOf)
			for fi := 0; fi < 2; fi++ {
				feat := mk(map[string]any{"title": fmt.Sprintf("%s — %s", epicTitle, []string{"core flow", "settings and polish"}[fi]), "type": "feature",
					"status": "in_progress", "state": stateOf("in_progress"), "priority": "medium", "parent_id": epic,
					"start_date": date(day(-70 + 25*fi)), "due_date": date(day(10 + 25*fi)), "created_at": day(-80), "updated_at": day(-80),
					"milestone_id": milestones[1+fi%2]})
				logChanges(feat, day(-80), []change{{day(-70 + 25*fi), "backlog", "in_progress"}}, stateOf)
				for si := 0; si < 5; si++ {
					sprintIdx := (ei*10 + fi*5 + si) % max(len(sprints), 1)
					var spr *sprint
					if len(sprints) > 0 {
						spr = &sprints[sprintIdx]
					}
					kind := "story"
					if si >= 3 && rng.Intn(2) == 0 {
						kind = "bug"
					}
					title := fmt.Sprintf("%s: %s", []string{"As a user I can", "As an admin I can", "Fix"}[map[string]int{"story": rng.Intn(2), "bug": 2}[kind]],
						pick([]string{"see what changed since yesterday", "choose how often I'm notified", "mute a noisy project", "retry failed sends",
							"filter by team", "export the list", "undo a mistake", "share a saved view", "find items by tag", "see who approved"}))
					story := storyFor(sp, spr, kind, title, feat, mk, stateOf, people, team, areaIDs, labelIDs, tagNames, &leaves)
					_ = story
				}
			}
		}
		// Dependencies and related links between a few leaves.
		for i := 0; i+3 < len(leaves); i += 4 {
			ins("task_dependencies", map[string]any{"task_id": leaves[i+1], "depends_on_id": leaves[i]})
			a, b := leaves[i+2], leaves[i+3]
			if a.String() > b.String() {
				a, b = b, a
			}
			db.Exec("INSERT INTO task_related (task_id, other_id) VALUES (?, ?)", a, b)
		}
		// Backlog items not planned yet.
		for i := 0; i < 4; i++ {
			mk(map[string]any{"title": pick([]string{"Dark mode for emails", "Weekly report PDF", "Bulk archive", "Keyboard shortcuts", "Audit log export", "SSO for partners"}) + fmt.Sprintf(" (%d)", i+1),
				"type": "story", "status": "backlog", "state": stateOf("backlog"), "priority": pick([]string{"low", "medium"}),
				"story_points": pick([]float64{2, 3, 5, 8}), "created_at": day(-20 + i), "updated_at": day(-20 + i)})
		}
		for w := 0; w < 4; w++ {
			ins("project_weekly_summaries", map[string]any{"project_id": pid, "week_of": date(day(-7*w - int(today.Weekday()) + 1)),
				"summary": pick([]string{"Review queue is the bottleneck; two items waited 3+ days.", "Pilot feedback is positive; one blocker on the retry path.", "Scope trimmed to protect the M2 date.", "New joiner onboarded; velocity back to normal."})})
		}
		ins("project_kickoffs", map[string]any{"project_id": pid, "meeting_date": date(day(-90)), "facilitator_id": people[team[0]].id,
			"business_problem": "People miss what needs their attention.", "success_definition": "Fewer missed approvals in the pilot team.",
			"mvp_feature": sp.epics[0], "biggest_risk": "Notification fatigue", "meeting_cadence": "Weekly", "primary_tools": "Pyramid, Slack"})
		_ = adminPerson
	}

	// Leave: some approved (in the past, now and soon), one waiting.
	for i, l := range []struct {
		p        int
		from, to int
		typ, st  string
	}{{4, -20, -16, "vacation", "approved"}, {7, -2, 2, "sick", "approved"}, {10, 8, 15, "vacation", "approved"}, {0, 30, 34, "vacation", "pending"}, {8, 12, 12, "personal", "approved"}} {
		row := map[string]any{"person_id": people[l.p].id, "type": l.typ, "status": l.st, "start_date": date(day(l.from)), "end_date": date(day(l.to)),
			"reason": pick([]string{"Family trip", "Flu", "Moving house", "Conference"}), "created_at": day(l.from - 20 + i)}
		if l.st == "approved" {
			row["approver_id"], row["decided_at"] = people[2].id, day(l.from-15)
		}
		ins("leave_requests", row)
	}
	for _, g := range []struct {
		t, scope, st string
		owner        int
		pct          float64
	}{
		{"Cut missed approvals by half", "team", "active", 11, 45}, {"Warehouse freshness under 15 minutes", "team", "active", 8, 70},
		{"Become the go-to reviewer for the API", "individual", "active", 4, 30}, {"Ship the component library v2", "department", "active", 0, 55},
	} {
		ins("goals", map[string]any{"title": g.t, "scope": g.scope, "status": g.st, "owner_person_id": people[g.owner].id, "progress_pct": g.pct, "start_date": date(day(-60)), "due_date": date(day(90))})
	}
	for i := 0; i < 40; i++ {
		ins("contributions", map[string]any{"person_id": pick(people).id, "type": pick([]string{"commit", "pull_request", "code_review", "documentation"}),
			"title":  pick([]string{"Retry with backoff", "Digest template", "Fix flaky test", "Warehouse model for orders", "Button tokens", "Docs: alerts API"}),
			"weight": pick([]float64{1, 1, 2, 3}), "occurred_at": day(-rng.Intn(60)), "metadata": "{}"})
	}
	log.Printf("seeded %d people and %d projects with history up to %s", len(people), len(specs), date(today))
}

func orEmpty(v []map[string]any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	return v
}

// storyFor makes a story (or bug) with 1–3 tasks under it, planned in spr,
// with a status history that fits the sprint's dates.
func storyFor(sp projectSpec, spr *sprint, kind, title string, feat uuid.UUID, mk func(map[string]any) uuid.UUID,
	stateOf func(string) string, people []person, team []int, areaIDs []uuid.UUID, labels map[string]uuid.UUID, tagNames []string, leaves *[]uuid.UUID) uuid.UUID {
	points := pick([]float64{1, 2, 3, 5, 8})
	status, created, changes := lifecycle(spr)
	row := map[string]any{"title": title, "type": kind, "status": status, "state": stateOf(status), "priority": pick([]string{"low", "medium", "medium", "high", "urgent"}),
		"parent_id": feat, "story_points": points, "created_at": created, "updated_at": created}
	if spr != nil {
		row["sprint_id"] = spr.id
		row["start_date"], row["due_date"] = date(spr.start), date(spr.end)
	}
	if len(areaIDs) > 0 {
		row["area_id"] = pick(areaIDs)
	}
	if status == "done" {
		row["completed_at"] = changes[len(changes)-1].at
	}
	if len(sp.fields) > 0 {
		row["custom_values"] = js(map[string]any{"customer": pick([]string{"ACME Bank", "Globex", "Initech"}), "risk": pick([]string{"Low", "Medium", "High"})})
	}
	story := mk(row)
	logChanges(story, created, changes, stateOf)
	for _, t := range []string{pick(tagNames), pick(tagNames)} {
		db.Exec("INSERT INTO task_labels (task_id, label_id) VALUES (?, ?) ON CONFLICT DO NOTHING", story, labels[t])
	}

	n := 1 + rng.Intn(3)
	for i := 0; i < n; i++ {
		tStatus, tCreated, tChanges := lifecycle(spr)
		md := pick([]float64{0.5, 1, 1.5, 2, 3})
		trow := map[string]any{"title": pick([]string{"Write the handler", "Build the screen", "Add tests", "Wire up the API", "Write the migration", "Update the docs", "Review copy"}),
			"type": "task", "status": tStatus, "state": stateOf(tStatus), "priority": "medium", "parent_id": story,
			"created_at": tCreated, "updated_at": tCreated, "remaining_man_days": map[bool]any{true: 0.0, false: md}[tStatus == "done"]}
		if spr != nil {
			trow["sprint_id"] = spr.id
			trow["start_date"], trow["due_date"] = date(spr.start), date(spr.end)
		}
		if tStatus == "done" {
			trow["completed_at"] = tChanges[len(tChanges)-1].at
			trow["completed_man_days"] = md
		}
		if i == 0 && sp.key == "CAD" && rng.Intn(4) == 0 {
			trow["custom_type"], trow["title"] = "spike", "Spike: "+trow["title"].(string)
		}
		tid := mk(trow)
		logChanges(tid, tCreated, tChanges, stateOf)
		who := people[team[rng.Intn(len(team))]].id
		db.Exec("INSERT INTO task_assignees (task_id, person_id, man_days) VALUES (?, ?, ?)", tid, who, md)
		*leaves = append(*leaves, tid)
	}
	return story
}

// lifecycle picks a status and its history for an item planned in spr:
// done in past sprints (now and then left over), a mix in the running one.
func lifecycle(spr *sprint) (string, time.Time, []change) {
	if spr == nil {
		return "backlog", day(-10), nil
	}
	created := spr.start.AddDate(0, 0, -3-rng.Intn(7))
	switch spr.status {
	case "planned":
		return "todo", created, []change{{created.Add(2 * time.Hour), "backlog", "todo"}}
	case "completed":
		if rng.Intn(8) == 0 { // left over, carried in the backlog
			return "in_progress", created, []change{{spr.start.Add(10 * time.Hour), "backlog", "todo"}, {spr.start.AddDate(0, 0, 3), "todo", "in_progress"}}
		}
		return "done", created, progress(spr.start, spr.end, "done")
	}
	// Running sprint: how far along depends on luck.
	end := today
	switch r := rng.Intn(10); {
	case r < 3:
		return "done", created, progress(spr.start, end, "done")
	case r < 5:
		return "in_review", created, progress(spr.start, end, "in_review")
	case r < 7:
		return "in_progress", created, progress(spr.start, end, "in_progress")
	case r < 8:
		return "blocked", created, progress(spr.start, end, "blocked")
	default:
		return "todo", created, []change{{spr.start.Add(9 * time.Hour), "backlog", "todo"}}
	}
}

// progress walks backlog → todo → in_progress → in_review → done (or stops
// earlier) at random hours between from and until.
func progress(from, until time.Time, last string) []change {
	steps := []string{"backlog", "todo", "in_progress", "in_review", "done"}
	if last == "blocked" {
		steps = []string{"backlog", "todo", "in_progress", "blocked"}
	}
	span := until.Sub(from)
	if span < 24*time.Hour {
		span = 24 * time.Hour
	}
	at := from.Add(9 * time.Hour)
	out := []change{}
	for i := 1; i < len(steps); i++ {
		at = at.Add(time.Duration(rng.Int63n(int64(span) / int64(len(steps)))))
		if at.After(time.Now()) {
			at = time.Now().Add(-time.Duration(len(steps)-i) * time.Hour)
		}
		out = append(out, change{at, steps[i-1], steps[i]})
		if steps[i] == last {
			break
		}
	}
	return out
}
