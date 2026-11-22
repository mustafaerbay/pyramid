# Person workload page — short design

**Why:** See one person's monthly load: how much of their monthly man-day
capacity is filled, month by month, and by which projects/tasks.

**Approved layout** (`/people/{id}`, opened by clicking a person on People):
- Header: name, status, title, email, monthly capacity; Edit button.
- Year chart: 12 vertical bars (planned man-days per month, stacked by
  project), dashed capacity line, over-capacity months red, ‹ year ›.
- Month detail (current month selected; tap a bar to change): donut
  `planned / capacity`, free share, legend by project; list of that month's
  tasks with total man-days, the share falling in the month and dates.
- Phone: bars shrink to fit 375px; donut and list stack.

**Approach:** one endpoint `GET /api/v1/people/{id}/workload?year=YYYY`
returning all 12 months (per-project and per-task shares) computed with the
same `models.SpreadManDays` rule as `/capacity`; the page draws plain
SVG/CSS bars (no chart library, matching the existing chart components) and
reuses `DonutChart`. Rejected: 12 calls to `/capacity` (no per-task detail,
12 round trips).
