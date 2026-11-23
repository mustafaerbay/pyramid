package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestRouter spins up a fresh in-memory SQLite database, creates the
// (hand-authored, SQLite-flavored) test schema below, and returns a real
// Gin engine wired to the production handlers via Handler.RegisterRoutes —
// tests exercise the actual HTTP request/response path (routing, JSON
// binding, status codes), not just direct Go function calls.
//
// This deliberately does NOT use GORM's AutoMigrate against the production
// model structs: their `gorm` tags carry Postgres-specific DDL (e.g.
// `default:gen_random_uuid()`, `type:task_status`) that SQLite's DDL parser
// rejects outright (a bare `DEFAULT gen_random_uuid()` isn't valid SQLite
// syntax — SQLite only accepts literals, NULL, CURRENT_* keywords, or a
// fully parenthesized `(expr)` in a DEFAULT clause). See idDefaultDDL below
// for the SQLite-native stand-in this test schema uses instead.
// testDBSeq keeps database names unique even if a test calls newTestRouter
// more than once or tests run with -count > 1.
var testDBSeq atomic.Int64

func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	// Each test gets its own named in-memory database. A bare
	// "file::memory:?cache=shared" is ONE database shared by every
	// connection in the process, so the second test's CREATE TABLE failed
	// with "table people already exists". The name is unique per test and
	// the DB is closed (and so discarded) when the test finishes.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared",
		url.PathEscape(fmt.Sprintf("%s-%d", t.Name(), testDBSeq.Add(1))))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// A ":memory:" SQLite DB is private per-connection; without capping the
	// pool to one connection, GORM could hand out a second connection that
	// sees an empty database even with cache=shared in some driver/version
	// combinations. One connection keeps this fully deterministic.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Exec(testSchema).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Handler tests run signed out; auth_test.go covers sign-in itself.
	h := New(db, config.Config{CORSOrigin: "*", AuthOff: true})
	h.RegisterRoutes(r)
	return r, db
}

// idDefaultDDL is a SQLite-native replacement for Postgres's
// gen_random_uuid(): it builds an 8-4-4-4-12 hex string (via
// randomblob()/hex(), both SQLite builtins) shaped like a UUID. Nothing in
// this app depends on the RFC 4122 version/variant bits being correct, only
// on the value round-tripping cleanly through uuid.Parse, so this is good
// enough for a test double without needing a real UUID extension.
const idDefaultDDL = `(
	lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' ||
	lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) || '-' ||
	lower(hex(randomblob(6)))
)`

// testSchema covers only the columns backend/internal/models actually maps
// — a deliberately smaller, SQLite-typed mirror of db/schema.sql, not a
// full port of it. Column types are SQLite's loose affinities (TEXT/
// INTEGER/REAL); enum/uuid "types" in the real schema are just TEXT here,
// which SQLite is perfectly happy to store and compare.
var testSchema = `
CREATE TABLE people (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	email TEXT NOT NULL,
	phone TEXT,
	avatar_url TEXT,
	title TEXT,
	seniority TEXT,
	employment_type TEXT NOT NULL DEFAULT 'full_time',
	status TEXT NOT NULL DEFAULT 'active',
	department_id TEXT,
	manager_id TEXT,
	location TEXT,
	timezone TEXT,
	bio TEXT,
	hire_date TEXT,
	termination_date TEXT,
	weekly_capacity_hours REAL,
	monthly_capacity_man_days REAL NOT NULL DEFAULT 20,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	person_id TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash TEXT,
	system_role TEXT NOT NULL DEFAULT 'member',
	is_active BOOLEAN NOT NULL DEFAULT 1,
	last_login_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	key TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL DEFAULT 'planning',
	priority TEXT NOT NULL DEFAULT 'medium',
	health TEXT NOT NULL DEFAULT 'on_track',
	lead_id TEXT,
	start_date TEXT,
	target_date TEXT,
	sponsor TEXT,
	project_director TEXT,
	report_period TEXT,
	hq_pm TEXT,
	turkey_pm TEXT,
	chinese_interface TEXT,
	hq_interface TEXT,
	turkey_qa TEXT,
	account_name TEXT,
	hq_se TEXT,
	turkey_se TEXT,
	project_code TEXT,
	team_size_actual INTEGER,
	team_size_planned INTEGER,
	board_settings TEXT,
	custom_fields TEXT,
	workflow TEXT,
	work_item_types TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_members (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	person_id TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'contributor',
	allocation_pct REAL NOT NULL DEFAULT 100,
	joined_at TEXT NOT NULL DEFAULT CURRENT_DATE,
	left_at TEXT
);

CREATE TABLE project_versions (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL DEFAULT 'planned',
	-- DATE (not TEXT) so the sqlite driver parses these back into time.Time.
	target_date DATE,
	released_at DATE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (project_id, name)
);

CREATE TABLE milestones (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL DEFAULT 'pending',
	start_date TEXT,
	due_date TEXT,
	completed_at TEXT,
	version_id TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	number INTEGER NOT NULL,
	rank REAL NOT NULL DEFAULT 0,
	parent_id TEXT,
	sprint_id TEXT,
	area_id TEXT,
	custom_values TEXT,
	duplicate_of_id TEXT,
	milestone_id TEXT,
	type TEXT NOT NULL DEFAULT 'task',
	title TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL DEFAULT 'backlog',
	state TEXT,
	custom_type TEXT,
	priority TEXT NOT NULL DEFAULT 'medium',
	start_date DATE,
	estimate_man_days REAL,
	story_points REAL,
	remaining_man_days REAL,
	completed_man_days REAL,
	due_date DATE,
	completed_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_assignees (
	task_id TEXT NOT NULL,
	person_id TEXT NOT NULL,
	man_days REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (task_id, person_id)
);

CREATE TABLE labels (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	name TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT '#888888',
	project_id TEXT
);

CREATE TABLE task_dependencies (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	task_id TEXT NOT NULL,
	depends_on_id TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_comments (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	task_id TEXT NOT NULL,
	author_id TEXT,
	parent_id TEXT,
	body TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE activity_log (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	actor_id TEXT,
	action TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id TEXT,
	changes TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE notifications (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	user_id TEXT NOT NULL,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	body TEXT,
	link_url TEXT,
	read_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_related (
	task_id TEXT NOT NULL,
	other_id TEXT NOT NULL,
	PRIMARY KEY (task_id, other_id)
);

CREATE TABLE sprints (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	name TEXT NOT NULL,
	goal TEXT,
	status TEXT NOT NULL DEFAULT 'planned',
	start_date DATE,
	end_date DATE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE areas (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	project_id TEXT NOT NULL,
	path TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE saved_queries (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	owner_id TEXT NOT NULL,
	name TEXT NOT NULL,
	filters TEXT NOT NULL DEFAULT '{}',
	group_by TEXT NOT NULL DEFAULT 'status',
	shared BOOLEAN NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE attachments (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	entity_type TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	filename TEXT NOT NULL,
	url TEXT NOT NULL,
	mime_type TEXT,
	size_bytes INTEGER,
	uploaded_by TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE password_resets (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	used_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_labels (
	task_id TEXT NOT NULL,
	label_id TEXT NOT NULL,
	PRIMARY KEY (task_id, label_id)
);

CREATE TABLE goals (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	title TEXT NOT NULL,
	description TEXT,
	scope TEXT NOT NULL DEFAULT 'individual',
	status TEXT NOT NULL DEFAULT 'draft',
	owner_person_id TEXT,
	progress_pct REAL NOT NULL DEFAULT 0,
	start_date TEXT,
	due_date TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE leave_requests (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	person_id TEXT NOT NULL,
	type TEXT NOT NULL DEFAULT 'vacation',
	status TEXT NOT NULL DEFAULT 'pending',
	start_date DATE NOT NULL,
	end_date DATE NOT NULL,
	reason TEXT,
	approver_id TEXT,
	decided_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contributions (
	id TEXT PRIMARY KEY DEFAULT ` + idDefaultDDL + `,
	person_id TEXT NOT NULL,
	project_id TEXT,
	type TEXT NOT NULL,
	title TEXT,
	url TEXT,
	weight REAL NOT NULL DEFAULT 1,
	occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// doJSON issues an HTTP request against the router with an optional JSON
// body, decodes the response body into out (if non-nil), and returns the
// status code. A helper shared by every handler test in this package.
func doJSON(t *testing.T, r *gin.Engine, method, path string, body any, out any) int {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if out != nil && w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("unmarshal response body %q: %v", w.Body.String(), err)
		}
	}
	return w.Code
}

var _ = http.MethodGet // keep net/http imported for readers reaching for method constants in new tests
