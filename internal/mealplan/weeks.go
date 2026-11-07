// SPDX-License-Identifier: Apache-2.0

package mealplan

import (
	"fmt"
	"time"
)

// WeekKey names the ISO week of t, e.g. "2026-W46".
func WeekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}

// parseKey returns the Monday of an ISO week key, if the week exists.
func parseKey(key string) (time.Time, bool) {
	var y, w int
	if n, err := fmt.Sscanf(key, "%04d-W%02d", &y, &w); err != nil || n != 2 || len(key) != 8 || w < 1 || w > 53 {
		return time.Time{}, false
	}
	jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
	monday := jan4.AddDate(0, 0, -int((jan4.Weekday()+6)%7)+7*(w-1))
	return monday, WeekKey(monday) == key
}

// ParseWeek reads a week key; anything invalid means the week of now.
func ParseWeek(key string, now time.Time) (start time.Time, canonical string) {
	if m, ok := parseKey(key); ok {
		return m, key
	}
	m, _ := parseKey(WeekKey(now))
	return m, WeekKey(now)
}

// Days are the seven days of the week that starts on start.
func Days(start time.Time) []time.Time {
	out := make([]time.Time, 7)
	for i := range out {
		out[i] = start.AddDate(0, 0, i)
	}
	return out
}

// Shift moves a valid week key by n weeks.
func Shift(key string, n int) string {
	m, ok := parseKey(key)
	if !ok {
		return key
	}
	return WeekKey(m.AddDate(0, 0, 7*n))
}
