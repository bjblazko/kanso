// SPDX-License-Identifier: Apache-2.0

package recipes

import "testing"

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		v    float64
		f    Family
		unit string
		lang string
		want string
	}{
		{1234, FamilyMass, "g", "de", "1235"},
		{1232, FamilyMass, "g", "de", "1230"},
		{87.4, FamilyMass, "g", "de", "87"},
		{100.2, FamilyVolume, "ml", "de", "100"},
		{1.5, FamilyMass, "kg", "de", "1,5"},
		{1.5, FamilyMass, "kg", "en", "1.5"},
		{1.256, FamilyVolume, "l", "de", "1,26"},
		{2.0, FamilyVolume, "l", "de", "2"},
		{0.5, FamilyCount, "Dose", "de", "½"},
		{1.5, FamilyCount, "Dose", "de", "1½"},
		{1.3, FamilyCount, "Stück", "de", "1⅓"},
		{2, FamilyCount, "Stück", "de", "2"},
		{0.1, FamilyCount, "TL", "de", "¼"},
		{0.75, FamilyCount, "TL", "de", "¾"},
	}
	for _, c := range cases {
		if got := FormatAmount(c.v, c.f, c.unit, c.lang); got != c.want {
			t.Errorf("FormatAmount(%v, %v, %s, %s) = %q, want %q", c.v, c.f, c.unit, c.lang, got, c.want)
		}
	}
}

func TestScale(t *testing.T) {
	got := ParseIngredient("- 800 g Kartoffeln").Scale(1.5, "de")
	if got.Qty.Low != 1200 || got.Qty.Text != "1200" || got.Name != "Kartoffeln" {
		t.Errorf("kartoffeln %+v %+v", got, *got.Qty)
	}
	got = ParseIngredient("- 2-3 Zehen Knoblauch").Scale(2, "de")
	if got.Qty.Low != 4 || got.Qty.High != 6 || got.Qty.Text != "4–6" {
		t.Errorf("range %+v", *got.Qty)
	}
	if got = ParseIngredient("- 1 Dose Kokosmilch").Scale(1.5, "de"); got.Qty.Text != "1½" {
		t.Errorf("dose %q", got.Qty.Text)
	}
	orig := ParseIngredient("- 1 Dose Kokosmilch")
	_ = orig.Scale(3, "de")
	if orig.Qty.Low != 1 {
		t.Error("Scale changed the original")
	}
	if got = ParseIngredient("- Salz").Scale(2, "de"); got.Qty != nil || got.Name != "Salz" {
		t.Errorf("salz %+v", got)
	}
}

func TestClampServings(t *testing.T) {
	cases := []struct {
		raw      string
		fallback int
		want     int
	}{
		{"", 4, 4}, {"0", 4, 1}, {"-3", 4, 1}, {"abc", 4, 4}, {"1000", 4, 99}, {"6", 4, 6}, {"", 0, 1},
	}
	for _, c := range cases {
		if got := ClampServings(c.raw, c.fallback); got != c.want {
			t.Errorf("ClampServings(%q, %d) = %d, want %d", c.raw, c.fallback, got, c.want)
		}
	}
}

func TestScaleKeepsAmountsAtFactorOne(t *testing.T) {
	for _, line := range []string{"- ⅓ TL Zimt", "- 2,5 g Trockenhefe", "- 0,2 g Safran", "- ⅔ Tasse Milch", "- 2-3 Zehen Knoblauch"} {
		orig := ParseIngredient(line)
		if got := orig.Scale(1, "de"); got.Qty.Text != orig.Qty.Text {
			t.Errorf("%s: %q, want %q", line, got.Qty.Text, orig.Qty.Text)
		}
	}
}

func TestFormatSmallAndThirds(t *testing.T) {
	cases := []struct {
		v    float64
		f    Family
		unit string
		want string
	}{
		{0.4, FamilyMass, "g", "0,4"},
		{0.02, FamilyMass, "g", "0,02"},
		{2.5, FamilyMass, "g", "2,5"},
		{7.0, FamilyVolume, "ml", "7"},
		{1.0 / 3, FamilyCount, "TL", "⅓"},
		{2.0 / 3, FamilyCount, "TL", "⅔"},
		{4.0 / 3, FamilyCount, "Tasse", "1⅓"},
	}
	for _, c := range cases {
		if got := FormatAmount(c.v, c.f, c.unit, "de"); got != c.want {
			t.Errorf("%v %s: %q, want %q", c.v, c.unit, got, c.want)
		}
	}
	if got := ParseIngredient("- 0,2 g Safran").Scale(2, "de"); got.Qty.Text != "0,4" {
		t.Errorf("safran ×2: %q", got.Qty.Text)
	}
}
