package models

import (
	"math"
	"testing"
	"time"
)

func d(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func month(s string) time.Time {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		panic(err)
	}
	return t
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSpreadManDays(t *testing.T) {
	cases := []struct {
		name       string
		manDays    float64
		start, due *time.Time
		month      string
		want       float64
	}{
		{"no dates counts nowhere", 10, nil, nil, "2026-11", 0},
		{"due only, same month", 10, nil, d("2026-11-20"), "2026-11", 10},
		{"due only, other month", 10, nil, d("2026-11-20"), "2026-10", 0},
		{"start only, same month", 6, d("2026-11-03"), nil, "2026-11", 6},
		{"start equals due", 4, d("2026-11-03"), d("2026-11-03"), "2026-11", 4},
		// 2026-10-26 (Mon) .. 2026-11-06 (Fri): 5 weekdays in Oct, 5 in Nov.
		{"split evenly across months, first", 40, d("2026-10-26"), d("2026-11-06"), "2026-10", 20},
		{"split evenly across months, second", 40, d("2026-10-26"), d("2026-11-06"), "2026-11", 20},
		// 2026-10-30 (Fri) .. 2026-11-04 (Wed): 1 weekday in Oct, 3 in Nov (weekend skipped).
		{"weekend days don't count", 8, d("2026-10-30"), d("2026-11-04"), "2026-10", 2},
		{"weekend days don't count, rest", 8, d("2026-10-30"), d("2026-11-04"), "2026-11", 6},
		{"month outside range", 8, d("2026-10-30"), d("2026-11-04"), "2026-12", 0},
		// Sat..Sun only: no weekday, so everything lands in the due month.
		{"weekend-only range goes to due month", 3, d("2026-10-31"), d("2026-11-01"), "2026-11", 3},
		{"weekend-only range, not start month", 3, d("2026-10-31"), d("2026-11-01"), "2026-10", 0},
		{"zero man-days", 0, d("2026-11-02"), d("2026-11-30"), "2026-11", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SpreadManDays(c.manDays, c.start, c.due, month(c.month))
			if !near(got, c.want) {
				t.Fatalf("SpreadManDays(%v, %v, %v, %s) = %v, want %v", c.manDays, c.start, c.due, c.month, got, c.want)
			}
		})
	}
}

func TestSpreadManDaysSumsToTotal(t *testing.T) {
	start, due := d("2026-09-17"), d("2027-02-03")
	total := 0.0
	for m := month("2026-08"); m.Before(month("2027-04")); m = m.AddDate(0, 1, 0) {
		total += SpreadManDays(37, start, due, m)
	}
	if !near(total, 37) {
		t.Fatalf("months sum to %v, want 37", total)
	}
}

func TestWorkingDaysInMonth(t *testing.T) {
	if got := WorkingDaysInMonth(month("2026-11")); got != 21 {
		t.Fatalf("Nov 2026 has %d weekdays, want 21", got)
	}
	if got := WorkingDaysInMonth(month("2026-02")); got != 20 {
		t.Fatalf("Feb 2026 has %d weekdays, want 20", got)
	}
}

func TestLeaveDaysInMonth(t *testing.T) {
	r := func(from, to string) DateRange { return DateRange{From: *d(from), To: *d(to)} }
	cases := []struct {
		name   string
		ranges []DateRange
		month  string
		want   int
	}{
		{"no leave", nil, "2026-11", 0},
		// Mon 2026-11-09 .. Fri 2026-11-13
		{"one working week", []DateRange{r("2026-11-09", "2026-11-13")}, "2026-11", 5},
		// Sat 2026-11-07 .. Sun 2026-11-08
		{"weekend only", []DateRange{r("2026-11-07", "2026-11-08")}, "2026-11", 0},
		// Mon 2026-10-26 .. Fri 2026-11-06: 5 weekdays in each month
		{"crosses into the month", []DateRange{r("2026-10-26", "2026-11-06")}, "2026-11", 5},
		{"crosses out of the month", []DateRange{r("2026-10-26", "2026-11-06")}, "2026-10", 5},
		{"other month", []DateRange{r("2026-12-01", "2026-12-04")}, "2026-11", 0},
		{"covers the whole month", []DateRange{r("2026-10-01", "2026-12-31")}, "2026-11", 21},
		{"overlapping ranges count once", []DateRange{r("2026-11-09", "2026-11-13"), r("2026-11-12", "2026-11-17")}, "2026-11", 7},
	}
	for _, c := range cases {
		if got := LeaveDaysInMonth(c.ranges, month(c.month)); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestMonthCapacity(t *testing.T) {
	cases := []struct {
		capacity       float64
		working, leave int
		want           float64
	}{
		{20, 21, 0, 20},
		{20, 20, 5, 15},
		{20, 21, 21, 0},
		{20, 21, 30, 0}, // never negative
		{20, 0, 0, 0},   // no working days
		{0, 21, 3, 0},
	}
	for _, c := range cases {
		if got := MonthCapacity(c.capacity, c.working, c.leave); !near(got, c.want) {
			t.Errorf("MonthCapacity(%v, %d, %d) = %v, want %v", c.capacity, c.working, c.leave, got, c.want)
		}
	}
}
