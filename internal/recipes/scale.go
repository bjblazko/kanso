// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"math"
	"strconv"
	"strings"
)

// MaxServings is the most servings the cook view scales to.
const MaxServings = 99

// fractions a counted amount is rounded to, with how they are written.
var fractionSteps = []struct {
	v    float64
	text string
}{{0, ""}, {0.25, "¼"}, {1.0 / 3, "⅓"}, {0.5, "½"}, {2.0 / 3, "⅔"}, {0.75, "¾"}, {1, ""}}

// Scale returns the ingredient with its amount multiplied by factor and
// written anew in lang ("de" or "en"); an ingredient without an amount, or
// a factor of 1, leaves it exactly as written.
func (i Ingredient) Scale(factor float64, lang string) Ingredient {
	if i.Qty == nil || factor == 1 {
		return i
	}
	q := Quantity{Low: i.Qty.Low * factor, High: i.Qty.High * factor}
	q.Text = FormatAmount(q.Low, i.Family, i.Unit, lang)
	if q.High != q.Low {
		q.Text += "–" + FormatAmount(q.High, i.Family, i.Unit, lang)
	}
	i.Qty = &q
	return i
}

// FormatAmount writes an amount with a precision that suits its unit:
// grams and millilitres whole from 10 on (from 100 on to 5), below that
// with a decimal; kilograms and litres with up to two decimals; everything
// counted to a quarter or a third. A non-zero amount never shows as 0.
func FormatAmount(v float64, f Family, unit string, lang string) string {
	switch {
	case unit == "kg" || unit == "l":
		return decimal(math.Round(v*100)/100, lang)
	case (f == FamilyMass || f == FamilyVolume) && v >= 10:
		if v >= 100 {
			v = math.Round(v/5) * 5
		}
		return strconv.Itoa(int(math.Round(v)))
	case f == FamilyMass || f == FamilyVolume:
		r := math.Round(v*10) / 10
		if r == 0 {
			r = math.Round(v*100) / 100
		}
		return decimal(r, lang)
	}
	return formatFraction(v)
}

func decimal(v float64, lang string) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if lang != "en" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

// formatFraction writes a counted amount to the nearest quarter or third.
func formatFraction(v float64) string {
	whole := math.Floor(v)
	best := fractionSteps[0]
	for _, step := range fractionSteps[1:] {
		if math.Abs(v-whole-step.v) < math.Abs(v-whole-best.v) {
			best = step
		}
	}
	if best.v == 1 {
		whole, best.text = whole+1, ""
	}
	if whole == 0 && best.text == "" {
		return "¼"
	}
	if whole == 0 {
		return best.text
	}
	return strconv.Itoa(int(whole)) + best.text
}

// ClampServings reads a servings value from a request: fallback when it is
// missing or no number, otherwise limited to 1…MaxServings.
func ClampServings(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		n = fallback
	}
	return min(max(n, 1), MaxServings)
}
