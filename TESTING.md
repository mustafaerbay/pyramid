# Testing Pyramid

This documents the automated test suites added to satisfy the SDLC matrix's
Phase 4 deliverable ("Unit Test Suite, ≥80% coverage") and how they're wired
into CI (Phase 6, "automated CI/CD pipelines"). It's written to be read
top-to-bottom by someone who just cloned the repo and wants to run
everything locally before trusting the green checkmark in CI.

## Backend (Go)

### Model unit tests — `backend/internal/models/*_test.go`

Every GORM model's three conversion functions (`NewXFromCreate`,
`ApplyUpdate`, `ToAPI`) has table-driven tests covering defaults, explicit
values, partial-patch isolation (an `ApplyUpdate` call only touches the
fields present in the patch), and nil-vs-populated `ToAPI` output. Run them:

```bash
cd backend
go test ./internal/models/... -cover
```

This package was compiled and run in the authoring environment — 89.6%
statement coverage, all tests passing, `go vet` and `gofmt -l` clean. It has
zero dependency on Gin, GORM's SQL layer, or a live database (the functions
under test are pure Go structs-in, structs-out), which is why it was safe to
verify directly in a sandboxed environment with restricted network access.

### Handler integration tests — `backend/internal/handlers/*_test.go`

`testutil_test.go` spins up a real `gin.Engine` wired to the production
handlers via `Handler.RegisterRoutes`, backed by an in-memory SQLite
database (`gorm.io/driver/sqlite`) using a hand-authored, SQLite-flavored
schema — not GORM's `AutoMigrate` against the production model structs,
because their `gorm` tags carry Postgres-only DDL (`default:gen_random_uuid()`,
`type:task_status`, …) that SQLite's DDL parser rejects. See the comment
block at the top of `testutil_test.go` for the exact reasoning and the
`idDefaultDDL` UUID-shaped default expression it uses instead.

Three resources are covered, chosen to each exercise a distinct piece of
handler logic:

| File | Resource | Why this one |
|---|---|---|
| `people_test.go` | People | The "template" CRUD shape — full list/create/get/update/delete plus a validation-error and a not-found path, useful as a reference when adding tests for the remaining resources. |
| `projects_test.go` | Projects | `NewProjectFromCreate`'s server-side defaults (status/priority/health) are only reachable through the HTTP layer, not just the model layer — this confirms the handler actually calls it. |
| `tasks_test.go` | Tasks | The most complex handler logic in the app: multi-assignee replace/attach (`replaceAssignees`), the `milestoneId` explicit-null-vs-absent PATCH semantics (`isExplicitNull`), and per-project sequential task numbering. |

**The other five resource handlers — project-members, milestones, goals,
leave-requests, contributions — do not have tests yet.** They follow the
exact same CRUD shape as `people_test.go` (same `New*FromCreate`/
`ApplyUpdate`/`ToAPI` pattern, same route registration style), so adding
`project_members_test.go` etc. following `people_test.go` as a template is
mechanical, low-effort follow-up work — not a redesign. Do that before
claiming full handler coverage.

**Honesty note on verification:** these handler tests could not be compiled
or run in the sandbox they were authored in, because that environment's
network egress blocks `gorm.io/driver/sqlite` (and `gorm.io/gorm` generally
— vanity-domain redirects are blocked there) and also can't upgrade its Go
toolchain past 1.24.7 to satisfy `gin-gonic/gin`'s `go >= 1.25` requirement.
They were written by close manual review against the actual handler source
(`tasks.go`, `projects.go`, `people.go`) rather than by iterating against a
real test run — treat them as "should be correct, needs a first real CI run
to confirm" rather than "proven," until that first run happens.

Run them (on a machine with normal network access):

```bash
cd backend
go test ./internal/handlers/... -v -cover
```

### `go.sum`

`backend/go.sum` is committed (including the test-only
`gorm.io/driver/sqlite`), so `go build ./...` and `go test ./...` work on a
fresh checkout without running `go mod tidy` first.

### Running everything

```bash
cd backend
go build ./...
go vet ./...
gofmt -l .            # should print nothing
go test ./... -cover
```

## Frontend (Vitest)

`frontend/src/lib/dateMath.ts` — the shared UTC-safe date-math helpers
extracted from three separate hand-rolled copies (`projects/[id]/+page.svelte`,
`ProjectCard.svelte`, the home dashboard) — has a full unit test suite at
`frontend/src/lib/dateMath.test.ts`: 12 tests covering `toUtc` (plain parse,
leap years, timezone independence), `todayUtc` (zero time-of-day), and
`daysUntil` (today/future/past, year and leap-day boundaries, the `DAY_MS`
constant, and the default-argument path). All 12 pass; this suite was run
directly in the authoring environment (no network or native-dependency
issues — it's plain TypeScript, zero SvelteKit/Tailwind wiring needed).

`vitest.config.ts` is deliberately minimal (`environment: 'node'`, no
sveltekit/tailwind plugins) — it's scoped to testing plain `.ts` modules
under `src/lib/`. Testing `.svelte` components would need a separate,
heavier config (`jsdom` environment + `@testing-library/svelte`) — worth
adding if/when component-level tests are wanted, but out of scope here.

Run it:

```bash
cd frontend
npm run test          # vitest run — single pass, CI-friendly
npm run test:watch    # vitest --watch, for local development
```

## CI — `.github/workflows/ci.yml`

Runs on every push and PR to `main`, two independent jobs:

- **backend** — `go mod download`, `gofmt -l` check, `go vet ./...`,
  `go build ./...`, `go test ./... -cover`. `go.sum` is committed, so
  `go mod download` only fetches modules.
- **frontend** — `npm ci`, `npm run check` (svelte-check), `npm run test`
  (vitest), `npm run build`.

There's no deploy/CD job yet — Pyramid has no defined hosting target, so a
CD stage would just be unused scaffolding. Add one once that target exists;
the CI job structure here is meant to be a natural place to append it.

## Coverage summary (honesty ledger)

| Area | Status |
|---|---|
| `backend/internal/models` | ✅ Written **and** run-verified. 89.6% statement coverage. |
| `backend/internal/handlers` (people, projects, tasks) | ⚠️ Written, manually reviewed, **not** run-verified in the authoring sandbox. First CI run is the first real signal. |
| `backend/internal/handlers` (project-members, milestones, goals, leave-requests, contributions) | ❌ Not written. Same pattern as `people_test.go`; mechanical follow-up. |
| `frontend/src/lib/dateMath.ts` | ✅ Written **and** run-verified. 12/12 passing. |
| CI pipeline | ✅ Written. Individual commands it invokes (`go build`, `go vet`, `go test`, `npm run check`, `npm run test`, `npm run build`) have each been run successfully in isolation in the authoring environment where possible; the workflow YAML itself has not had a real GitHub Actions run yet — that happens on first push.
