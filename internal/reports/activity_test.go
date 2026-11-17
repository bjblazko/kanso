// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"testing"
	"time"
)

func TestCalendarCountsPerDay(t *testing.T) {
	to := time.Date(2026, 11, 17, 20, 0, 0, 0, time.Local)
	events := []Event{
		{When: to.Add(-time.Hour)}, {When: to.Add(-2 * time.Hour)},
		{When: to.AddDate(0, 0, -1)},
		{When: to.AddDate(-2, 0, 0)}, // too old
	}
	days := Calendar(events, to)
	if len(days) != 365 {
		t.Fatalf("%d days", len(days))
	}
	last, before := days[364], days[363]
	if last.N != 2 || before.N != 1 || last.Date.Day() != 17 {
		t.Errorf("last %+v, before %+v", last, before)
	}
	total := 0
	for _, d := range days {
		total += d.N
	}
	if total != 3 {
		t.Errorf("total %d", total)
	}
}

func TestWeekHoursMondayFirst(t *testing.T) {
	mon := time.Date(2026, 11, 16, 9, 30, 0, 0, time.Local) // a Monday
	sun := time.Date(2026, 11, 22, 23, 5, 0, 0, time.Local)
	wh := WeekHours([]Event{{When: mon}, {When: mon}, {When: sun}})
	if wh[0][9] != 2 || wh[6][23] != 1 {
		t.Errorf("monday 9h %d, sunday 23h %d", wh[0][9], wh[6][23])
	}
}
