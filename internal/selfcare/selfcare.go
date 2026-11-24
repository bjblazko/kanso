// SPDX-License-Identifier: Apache-2.0

// Package selfcare holds Kanso's care for the mind: small steps to pick at
// random (calming for a bad moment, lifting for a pause), guided exercises
// and meditations. It is
// content only; nothing a person does here is stored, except a note they ask
// for. It is no substitute for help, and the page says where help is.
package selfcare

import "slices"

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

// Only keeps the tips of one kind: lifting ones for a step between two
// things, calming ones for first aid.
func Only(pool []Tip, lift bool) []Tip {
	return slices.DeleteFunc(slices.Clone(pool), func(t Tip) bool {
		if lift {
			return !t.Lift
		}
		return !t.Gentle
	})
}

// Pick returns a random built-in tip other than the one shown before; n(k)
// must return a number in [0, k), e.g. rand.IntN.
func Pick(n func(int) int, before string) Tip { return PickFrom(tips, n, before) }
