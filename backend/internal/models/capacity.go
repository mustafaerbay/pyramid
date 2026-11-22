package models

import "time"

// Man-day planning: a task assignment's man-days are spread evenly over the
// working days (Mon–Fri) between the task's start and due dates, so a
// two-month task loads both months instead of piling onto one.

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func isWeekday(t time.Time) bool {
	wd := t.Weekday()
	return wd != time.Saturday && wd != time.Sunday
}

func sameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// weekdaysBetween counts Mon–Fri days in [from, to] (both inclusive).
func weekdaysBetween(from, to time.Time) int {
	n := 0
	for d := dateOnly(from); !d.After(dateOnly(to)); d = d.AddDate(0, 0, 1) {
		if isWeekday(d) {
			n++
		}
	}
	return n
}

// MonthStart returns the first day (UTC) of t's month.
func MonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// WorkingDaysInMonth counts Mon–Fri days in the month that contains m.
func WorkingDaysInMonth(m time.Time) int {
	first := MonthStart(m)
	return weekdaysBetween(first, first.AddDate(0, 1, -1))
}

// SpreadManDays returns the share of manDays that falls in the month
// containing m:
//   - no dates: 0 (undated work isn't placed in any month);
//   - only one date, or start == due: everything in that date's month;
//   - otherwise manDays × (weekdays of [start, due] inside the month) ÷
//     (weekdays of [start, due]); a range without weekdays goes to the due
//     month.
func SpreadManDays(manDays float64, start, due *time.Time, m time.Time) float64 {
	if manDays == 0 || (start == nil && due == nil) {
		return 0
	}
	if start == nil || due == nil || dateOnly(*start).Equal(dateOnly(*due)) {
		at := due
		if at == nil {
			at = start
		}
		if sameMonth(*at, m) {
			return manDays
		}
		return 0
	}
	from, to := dateOnly(*start), dateOnly(*due)
	total := weekdaysBetween(from, to)
	if total == 0 {
		if sameMonth(to, m) {
			return manDays
		}
		return 0
	}
	first := MonthStart(m)
	last := first.AddDate(0, 1, -1)
	if from.After(last) || to.Before(first) {
		return 0
	}
	if from.Before(first) {
		from = first
	}
	if to.After(last) {
		to = last
	}
	return manDays * float64(weekdaysBetween(from, to)) / float64(total)
}
