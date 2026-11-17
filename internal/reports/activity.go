// SPDX-License-Identifier: Apache-2.0

package reports

import "time"

// Event is one change someone made at a time.
type Event struct {
	When   time.Time
	Person string
}

// Day is how many changes happened on Date (local time).
type Day struct {
	Date time.Time
	N    int
}

// calendarDays is how far the activity calendar reaches back.
const calendarDays = 365

// Calendar counts events per day for the year up to and including to's day.
func Calendar(events []Event, to time.Time) []Day {
	end := dayOf(to)
	first := end.AddDate(0, 0, -(calendarDays - 1))
	days := make([]Day, calendarDays)
	for i := range days {
		days[i].Date = first.AddDate(0, 0, i)
	}
	for _, e := range events {
		d := dayOf(e.When.In(to.Location()))
		if d.Before(first) || d.After(end) {
			continue
		}
		days[daysBetween(first, d)].N++
	}
	return days
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// daysBetween counts calendar days, safe across daylight saving changes.
func daysBetween(a, b time.Time) int {
	ua := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	ub := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}

// WeekHours counts events per weekday (Monday first) and hour, in local time.
func WeekHours(events []Event) [7][24]int {
	var wh [7][24]int
	for _, e := range events {
		t := e.When.Local()
		wh[(int(t.Weekday())+6)%7][t.Hour()]++
	}
	return wh
}
