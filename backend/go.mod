module github.com/mustafaerbay/pyramid/backend

go 1.25

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/google/uuid v1.6.0
	github.com/oapi-codegen/runtime v1.6.0
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
)

// gorm.io/driver/sqlite is a test-only dependency (backend/internal/handlers
// httptest suite runs against an in-memory SQLite DB, never Postgres). Its
// go.sum entries could not be computed in the sandbox this was authored in
// (network access to gorm.io is blocked there) — run `go mod tidy` once
// after pulling this change; see TESTING.md.
require gorm.io/driver/sqlite v1.6.0
