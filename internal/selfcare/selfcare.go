// SPDX-License-Identifier: Apache-2.0

// Package selfcare holds Kanso's first aid for the mind: small steps to pick
// at random, a battery check that suggests some, and guided exercises. It is
// content only; nothing a person does here is stored, except a note they ask
// for. It is no substitute for help, and the page says where help is.
package selfcare

import "slices"

// Level is one answer to "how full is your battery?".
type Level struct {
	Key   string
	Label Text
	tips  []string
}

// Levels go from full to empty.
var Levels = []Level{
	{"100", Text{"Voll – klar und stabil", "Full – clear and steady"}, []string{"good", "walk", "message", "song"}},
	{"75", Text{"Gut – aber wachsam", "Good – but watchful"}, []string{"slow", "stretch", "song", "tidy"}},
	{"50", Text{"Halb – dünnhäutig", "Half – thin-skinned"}, []string{"window", "breath", "see", "tenyears"}},
	{"25", Text{"Knapp – ich funktioniere nur noch", "Low – just functioning"}, []string{"water", "shoulders", "feet", "no"}},
	{"10", Text{"Leer – es geht kaum noch", "Empty – barely coping"}, []string{"water", "breath", "cold", "no"}},
}

// Tips are all first-aid steps.
func Tips() []Tip { return tips }

// Exercises are the guided exercises in menu order.
func Exercises() []Exercise { return exercises }

// ExerciseByID finds an exercise.
func ExerciseByID(id string) (Exercise, bool) {
	i := slices.IndexFunc(exercises, func(e Exercise) bool { return e.ID == id })
	if i < 0 {
		return Exercise{}, false
	}
	return exercises[i], true
}

// LevelByKey finds a battery level.
func LevelByKey(key string) (Level, bool) {
	i := slices.IndexFunc(Levels, func(l Level) bool { return l.Key == key })
	if i < 0 {
		return Level{}, false
	}
	return Levels[i], true
}

// Suggestions are the tips that fit a level.
func (l Level) Suggestions() []Tip {
	out := make([]Tip, 0, len(l.tips))
	for _, id := range l.tips {
		if i := slices.IndexFunc(tips, func(t Tip) bool { return t.ID == id }); i >= 0 {
			out = append(out, tips[i])
		}
	}
	return out
}

// Empty tells whether the level calls for help from outside first.
func (l Level) Empty() bool { return l.Key == "10" }

// Pick returns a random built-in tip other than the one shown before; n(k)
// must return a number in [0, k), e.g. rand.IntN.
func Pick(n func(int) int, before string) Tip { return PickFrom(tips, n, before) }
