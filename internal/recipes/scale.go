// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"math"
	"strconv"
	"strings"
)

// MaxServings is the most servings the cook view scales to.
const MaxServings = 99

var quarters = map[int]string{1: "¼", 2: "½", 3: "¾"}

// Scale returns the ingredient with its amount multiplied by factor and
// written anew in lang ("de" or "en"); an ingredient without an amount stays as it is.
func (i Ingredient) Scale(factor float64, lang string) Ingredient {
	if i.Qty == nil {
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
// grams and millilitres whole (from 100 on to 5), kilograms and litres with
// up to two decimals, everything counted to a quarter.
func FormatAmount(v float64, f Family, unit string, lang string) string {
	switch {
	case unit == "kg" || unit == "l":
		s := strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
		if lang != "en" {
			s = strings.Replace(s, ".", ",", 1)
		}
		return s
	case f == FamilyMass || f == FamilyVolume:
		if v >= 100 {
			v = math.Round(v/5) * 5
		}
		return strconv.Itoa(int(math.Round(v)))
	}
	return formatQuarters(v)
}

func formatQuarters(v float64) string {
	n := int(math.Round(v * 4))
	if n == 0 {
		n = 1
	}
	whole, frac := n/4, quarters[n%4]
	if whole == 0 {
		return frac
	}
	return strconv.Itoa(whole) + frac
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
