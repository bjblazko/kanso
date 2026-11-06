// SPDX-License-Identifier: Apache-2.0

package recipes

import "strings"

// Family tells which units can be added up: within a family, never across.
type Family int

// The unit families.
const (
	FamilyNone Family = iota
	FamilyMass
	FamilyVolume
	FamilyCount
)

type unit struct {
	canonical string
	family    Family
}

// unitWords maps every accepted spelling (lower case, without a final dot) to its unit.
var unitWords = buildUnitWords(map[unit][]string{
	{"g", FamilyMass}:         {"g", "gramm", "gram", "grams"},
	{"kg", FamilyMass}:        {"kg", "kilogramm"},
	{"ml", FamilyVolume}:      {"ml", "milliliter"},
	{"cl", FamilyVolume}:      {"cl"},
	{"l", FamilyVolume}:       {"l", "liter", "litre"},
	{"EL", FamilyCount}:       {"el", "tbsp", "esslöffel"},
	{"TL", FamilyCount}:       {"tl", "tsp", "teelöffel"},
	{"Prise", FamilyCount}:    {"prise", "prisen", "pinch"},
	{"Stück", FamilyCount}:    {"stück", "stk", "pc", "pcs"},
	{"Bund", FamilyCount}:     {"bund", "bunch"},
	{"Dose", FamilyCount}:     {"dose", "dosen", "can", "cans"},
	{"Packung", FamilyCount}:  {"packung", "packungen", "pck", "pkg"},
	{"Becher", FamilyCount}:   {"becher"},
	{"Zehe", FamilyCount}:     {"zehe", "zehen", "clove", "cloves"},
	{"Scheibe", FamilyCount}:  {"scheibe", "scheiben", "slice", "slices"},
	{"Tasse", FamilyCount}:    {"tasse", "tassen", "cup", "cups"},
	{"Glas", FamilyCount}:     {"glas", "gläser", "jar", "jars"},
	{"Zweig", FamilyCount}:    {"zweig", "zweige", "sprig", "sprigs"},
	{"Handvoll", FamilyCount}: {"handvoll", "handful"},
})

func buildUnitWords(m map[unit][]string) map[string]unit {
	words := map[string]unit{}
	for u, spellings := range m {
		for _, s := range spellings {
			words[s] = u
		}
	}
	return words
}

// lookupUnit finds the unit a word names, ignoring case and a final dot.
func lookupUnit(word string) (canonical string, f Family, ok bool) {
	u, ok := unitWords[strings.ToLower(strings.TrimSuffix(word, "."))]
	return u.canonical, u.family, ok
}
