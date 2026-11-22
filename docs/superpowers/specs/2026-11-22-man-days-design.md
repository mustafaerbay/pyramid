# Man-day capacity and task effort — design

**Why:** The department wants to plan in man-days. Each person can do a fixed
number of man-days a month (e.g. 20 or 22); each task is worth some man-days
(1, 10, 40…), split across its assignees (40 = 20 + 20). From that we see total
monthly capacity (10 people × 20 = 200), how much of it is planned, and how many
man-days each project holds.

**Decided with the user**
- A task's man-days are spread evenly over the working days (Mon–Fri) between
  its start and due date, so monthly load is realistic. Tasks get an optional
  start date. No start date → everything counts in the due-date month.
- Capacity is per person per month, default 20, editable on the person.
- Screens: task form (start date + man-days per assignee), People (this
  month's `planned / capacity` with a month selector, replacing the
  allocation-% bar), Dashboard "this month" box (capacity, planned, free, by
  project), project page totals (total / done man-days, per milestone group).
- Project-member allocation % stays as data but no longer drives the People
  workload bar.

**Non-goals:** holidays/leave reducing capacity, timesheets (actual vs
estimate), per-month capacity overrides, sprint planning.

**Approaches**
- Chosen: man-days live on the assignment (`task_assignees.man_days`); the task
  total is the sum. Monthly numbers are computed in Go by one capacity
  endpoint that both People and Dashboard use, so the spreading rule exists
  once and is unit-tested.
- Rejected: a single `man_days` on the task split equally between assignees —
  can't express 30 + 10. Rejected: computing months in the browser — the rule
  would be duplicated across pages and People pagination would need every task.

## Data

- `tasks.start_date DATE NULL` (must be ≤ due date when both are set).
- `task_assignees.man_days NUMERIC(6,2) NOT NULL DEFAULT 0` (0–999).
- `people.monthly_capacity_man_days NUMERIC(5,2) NOT NULL DEFAULT 20` (0–31).
- Migration `db/migrations/2026-11-22-man-days.sql`; `db/schema.sql` updated.

## API (openapi/openapi.yaml is the source of truth)

- `Task`: `startDate`, `assignments: [{personId, manDays}]`, read-only
  `manDays` (sum). `assigneeIds` stays (same people, for existing callers).
- `TaskCreate`/`TaskUpdate`: `startDate`; `assignments` replaces the whole set
  (wins over `assigneeIds` if both are sent; `assigneeIds` alone → 0 man-days).
- `Person`/`PersonCreate`/`PersonUpdate`: `monthlyCapacityManDays`.
  `Person.workloadPct` now = planned man-days in the current month ÷ capacity
  × 100; plus read-only `monthManDays`.
- `GET /api/v1/capacity?month=YYYY-MM` (default current month) →
  `{month, workingDays, capacityManDays, plannedManDays,
    people: [{personId, capacityManDays, plannedManDays}],
    projects: [{projectId, plannedManDays}]}`. Capacity counts active people
  only; planned counts tasks that are not cancelled, on projects not
  cancelled. Done tasks still count (the work happened in that month).

## Spreading rule (pure Go function, unit-tested)

`spread(manDays, start, due, month) float64`:
- neither date → 0 for every month (still counted in project totals);
- only start or only due, or start == due → all in that date's month;
- otherwise the man-days are divided by the number of Mon–Fri days in
  [start, due] and multiplied by those that fall in the month; a range with no
  weekday puts everything in the due month.

## Build order

1. Migration + schema + OpenAPI + `make generate`.
2. Go: spread function (TDD), task assignments (create/update/get/list),
   person capacity, capacity endpoint, People workload switch — handler tests
   on the SQLite test schema.
3. Frontend: task form, People month view, Dashboard box, project totals.
4. Check at 375px and desktop, `npm run check`, `go test`, push (auto-deploys),
   apply the migration on the server.
