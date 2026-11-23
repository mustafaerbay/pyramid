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

// DateRange is an inclusive [From, To] span of calendar days.
type DateRange struct{ From, To time.Time }

// LeaveDaysInMonth counts the Mon–Fri days of the month containing m that
// fall in at least one of ranges (overlapping ranges count a day once).
func LeaveDaysInMonth(ranges []DateRange, m time.Time) int {
	first := MonthStart(m)
	last := first.AddDate(0, 1, -1)
	n := 0
	for day := first; !day.After(last); day = day.AddDate(0, 0, 1) {
		if !isWeekday(day) {
			continue
		}
		if onLeave(day, ranges) {
			n++
		}
	}
	return n
}

// MonthCapacity is a person's capacity for one month after leave: the
// monthly capacity scaled by the share of working days they are not on
// leave. Never negative; 0 when the month has no working days.
func MonthCapacity(capacity float64, workingDays, leaveDays int) float64 {
	if workingDays <= 0 || leaveDays >= workingDays {
		return 0
	}
	return capacity * float64(workingDays-leaveDays) / float64(workingDays)
}

func onLeave(day time.Time, leave []DateRange) bool {
	for _, r := range leave {
		if !day.Before(dateOnly(r.From)) && !day.After(dateOnly(r.To)) {
			return true
		}
	}
	return false
}

// freeDaysBetween counts Mon–Fri days in [from, to] that aren't on leave.
func freeDaysBetween(from, to time.Time, leave []DateRange) int {
	n := 0
	for day := dateOnly(from); !day.After(dateOnly(to)); day = day.AddDate(0, 0, 1) {
		if isWeekday(day) && !onLeave(day, leave) {
			n++
		}
	}
	return n
}

// AvailableDays is how many working days (Mon–Fri, not on leave) a person
// has between a task's start and due dates. ok is false when the task
// doesn't have both dates, so there is no window to check against.
func AvailableDays(start, due *time.Time, leave []DateRange) (days int, ok bool) {
	if start == nil || due == nil {
		return 0, false
	}
	return freeDaysBetween(*start, *due, leave), true
}

// SpreadManDaysAroundLeave is SpreadManDays for one person's share of a
// task, skipping the days that person is on leave: the man-days are spread
// over the working days in [start, due] they are actually there. When the
// task has a single date, or every working day in its window is on leave,
// it falls back to SpreadManDays so the work still shows up somewhere.
func SpreadManDaysAroundLeave(manDays float64, start, due *time.Time, leave []DateRange, m time.Time) float64 {
	if len(leave) == 0 || start == nil || due == nil || dateOnly(*start).Equal(dateOnly(*due)) {
		return SpreadManDays(manDays, start, due, m)
	}
	total := freeDaysBetween(*start, *due, leave)
	if manDays == 0 || total == 0 {
		return SpreadManDays(manDays, start, due, m)
	}
	from, to := dateOnly(*start), dateOnly(*due)
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
	return manDays * float64(freeDaysBetween(from, to, leave)) / float64(total)
}
