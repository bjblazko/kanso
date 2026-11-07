// SPDX-License-Identifier: Apache-2.0

package shopping

import "github.com/bjblazko/kanso/internal/recipes"

// toBase is how many grams or millilitres one unit holds.
var toBase = map[string]float64{"g": 1, "kg": 1000, "ml": 1, "cl": 10, "l": 1000}

// sum adds b to a when both have an amount in compatible units: grams and
// kilograms, millilitres to litres, or the same counted unit. The result
// is in grams or millilitres, from 1000 on in kilograms or litres.
func sum(a, b recipes.Ingredient, lang string) (recipes.Ingredient, bool) {
	if a.Qty == nil || b.Qty == nil || a.Family != b.Family {
		return a, false
	}
	fa, fb := 1.0, 1.0
	if a.Family == recipes.FamilyMass || a.Family == recipes.FamilyVolume {
		fa, fb = toBase[a.Unit], toBase[b.Unit]
	} else if a.Unit != b.Unit {
		return a, false
	}
	q := recipes.Quantity{Low: a.Qty.Low*fa + b.Qty.Low*fb, High: a.Qty.High*fa + b.Qty.High*fb}
	a.Qty = &q
	if fa != 1 || fb != 1 || a.Family != recipes.FamilyCount {
		a.Unit, a.UnitText = baseUnit(&q, a.Family)
	}
	q.Text = recipes.FormatAmount(q.Low, a.Family, a.Unit, lang)
	if q.High != q.Low {
		q.Text += "–" + recipes.FormatAmount(q.High, a.Family, a.Unit, lang)
	}
	return a, true
}

// baseUnit picks g or kg (ml or l) for an amount in grams (millilitres) and scales it.
func baseUnit(q *recipes.Quantity, f recipes.Family) (unit, text string) {
	small, big := "g", "kg"
	if f == recipes.FamilyVolume {
		small, big = "ml", "l"
	}
	if q.High < 1000 {
		return small, small
	}
	q.Low, q.High = q.Low/1000, q.High/1000
	return big, big
}
