// SPDX-License-Identifier: Apache-2.0

package recipes

import "testing"

func TestParseIngredient(t *testing.T) {
	cases := []struct {
		in                         string
		low, high                  float64
		text, unit, unitText, name string
		note                       string
	}{
		{"- 200 g Mehl", 200, 200, "200", "g", "g", "Mehl", ""},
		{"200g Mehl", 200, 200, "200", "g", "g", "Mehl", ""},
		{"- **200 g** Mehl", 200, 200, "200", "g", "g", "Mehl", ""},
		{"- 1,5 EL Currypaste", 1.5, 1.5, "1,5", "EL", "EL", "Currypaste", ""},
		{"- 1.5 tbsp oil", 1.5, 1.5, "1.5", "EL", "tbsp", "oil", ""},
		{"- ½ TL Salz", 0.5, 0.5, "½", "TL", "TL", "Salz", ""},
		{"- 1 ½ Tassen Milch", 1.5, 1.5, "1 ½", "Tasse", "Tassen", "Milch", ""},
		{"- 1/2 l Brühe", 0.5, 0.5, "1/2", "l", "l", "Brühe", ""},
		{"- 2-3 Zehen Knoblauch", 2, 3, "2-3", "Zehe", "Zehen", "Knoblauch", ""},
		{"- 2–3 Zwiebeln", 2, 3, "2–3", "Stück", "", "Zwiebeln", ""},
		{"- 800 g Kartoffeln, festkochend", 800, 800, "800", "g", "g", "Kartoffeln", "festkochend"},
		{"- 1 Dose Tomaten (400 g)", 1, 1, "1", "Dose", "Dose", "Tomaten", "400 g"},
		{"- 2 Eier (Größe M)", 2, 2, "2", "Stück", "", "Eier", "Größe M"},
		{"- 1 Pck. Backpulver", 1, 1, "1", "Packung", "Pck.", "Backpulver", ""},
		{"- 1 Bund Petersilie", 1, 1, "1", "Bund", "Bund", "Petersilie", ""},
	}
	for _, c := range cases {
		got := ParseIngredient(c.in)
		if got.Qty == nil {
			t.Errorf("%q: no quantity", c.in)
			continue
		}
		if got.Qty.Low != c.low || got.Qty.High != c.high || got.Qty.Text != c.text ||
			got.Unit != c.unit || got.UnitText != c.unitText || got.Name != c.name || got.Note != c.note {
			t.Errorf("%q: got %+v qty %+v", c.in, got, *got.Qty)
		}
	}
}

func TestParseIngredientWithoutQuantity(t *testing.T) {
	cases := []struct{ in, name, note string }{
		{"- Salz", "Salz", ""},
		{"- Koriander zum Bestreuen", "Koriander zum Bestreuen", ""},
		{"- etwas Öl, zum Braten", "etwas Öl", "zum Braten"},
	}
	for _, c := range cases {
		got := ParseIngredient(c.in)
		if got.Qty != nil || got.Unit != "" || got.Name != c.name || got.Note != c.note {
			t.Errorf("%q: got %+v", c.in, got)
		}
	}
}

func TestIngredientKeyAndLine(t *testing.T) {
	got := ParseIngredient("-   Rote  Zwiebeln ")
	if got.Key != "rote zwiebeln" {
		t.Errorf("key %q", got.Key)
	}
	if got := ParseIngredient("- 200 g Mehl"); got.Line != "200 g Mehl" {
		t.Errorf("line %q", got.Line)
	}
}

func TestUnitFamilies(t *testing.T) {
	cases := map[string]Family{"g": FamilyMass, "kg": FamilyMass, "ml": FamilyVolume, "cl": FamilyVolume,
		"l": FamilyVolume, "EL": FamilyCount, "Prise": FamilyCount}
	for word, want := range cases {
		if _, f, ok := lookupUnit(word); !ok || f != want {
			t.Errorf("%s: %v %v", word, f, ok)
		}
	}
	for word, want := range map[string]string{"Gramm": "g", "esslöffel": "EL", "TSP": "TL", "Stk.": "Stück",
		"cans": "Dose", "pkg": "Packung", "cloves": "Zehe", "Scheiben": "Scheibe", "cups": "Tasse",
		"Gläser": "Glas", "sprigs": "Zweig", "Handvoll": "Handvoll", "Becher": "Becher", "Liter": "l", "Prisen": "Prise"} {
		if c, _, ok := lookupUnit(word); !ok || c != want {
			t.Errorf("%s → %q %v", word, c, ok)
		}
	}
	if _, _, ok := lookupUnit("Zwiebeln"); ok {
		t.Error("Zwiebeln is no unit")
	}
}
