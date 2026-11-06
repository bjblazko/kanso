// SPDX-License-Identifier: Apache-2.0

// Package recipes understands recipe pages: ingredients, groups, steps,
// scaling to other servings. It never drops text it does not understand.
package recipes

import (
	"regexp"
	"strconv"
	"strings"
)

// Quantity is an amount as written, possibly a range.
type Quantity struct {
	Low, High float64 // High == Low without a range
	Text      string  // as written, e.g. "2-3", "1 ½"
}

// Ingredient is one line of a recipe's ingredient list.
type Ingredient struct {
	Line     string    // the line without its list marker
	Qty      *Quantity // nil = no amount
	Unit     string    // canonical unit; "" without an amount
	UnitText string    // as written; "" for an implicit piece
	Family   Family
	Name     string // without the note
	Note     string // after the first comma, or a bracket at the end
	Key      string // lower-case name with single spaces, for adding up
}

const number = `(?:\d+/\d+|\d+(?:[.,]\d+)?|[¼½¾⅓⅔])`

var (
	quantityRE = regexp.MustCompile(`^(` + number + `)(?:\s*([¼½¾⅓⅔])|\s+(\d+/\d+))?(?:\s*[-–]\s*(` + number + `))?`)
	unitRE     = regexp.MustCompile(`^\s*(\pL+\.?)(?:\s|$)`)
	fractions  = map[string]float64{"¼": 0.25, "½": 0.5, "¾": 0.75, "⅓": 1.0 / 3, "⅔": 2.0 / 3}
)

// ParseIngredient reads "[amount] [unit] name[, note]".
func ParseIngredient(line string) Ingredient {
	ing := Ingredient{Line: stripMarker(line)}
	rest := strings.TrimSpace(strings.Replace(strings.Replace(ing.Line, "**", "", 1), "**", "", 1))
	if m := quantityRE.FindStringSubmatchIndex(rest); m != nil {
		ing.Qty = quantityOf(rest, m)
		rest = ing.readUnit(rest[m[1]:])
	}
	ing.Name, ing.Note = splitNote(strings.TrimSpace(rest))
	ing.Key = strings.ToLower(strings.Join(strings.Fields(ing.Name), " "))
	return ing
}

func stripMarker(line string) string {
	s := strings.TrimSpace(line)
	for _, marker := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(s, marker) {
			return strings.TrimSpace(s[len(marker):])
		}
	}
	return s
}

// quantityOf turns the submatches of quantityRE into a Quantity.
func quantityOf(s string, m []int) *Quantity {
	group := func(i int) string {
		if m[2*i] < 0 {
			return ""
		}
		return s[m[2*i]:m[2*i+1]]
	}
	low := value(group(1)) + value(group(2)) + value(group(3))
	q := &Quantity{Low: low, High: low, Text: s[m[0]:m[1]]}
	if hi := group(4); hi != "" {
		q.High = value(hi)
	}
	return q
}

func value(s string) float64 {
	if f, ok := fractions[s]; ok {
		return f
	}
	if num, den, ok := strings.Cut(s, "/"); ok {
		n, _ := strconv.ParseFloat(num, 64)
		d, _ := strconv.ParseFloat(den, 64)
		if d == 0 {
			return 0
		}
		return n / d
	}
	f, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	return f
}

// readUnit takes a unit from the start of rest, or sets an implicit piece; it returns what follows.
func (ing *Ingredient) readUnit(rest string) string {
	if m := unitRE.FindStringSubmatch(rest); m != nil {
		if c, f, ok := lookupUnit(m[1]); ok {
			ing.Unit, ing.UnitText, ing.Family = c, m[1], f
			return rest[len(m[0]):]
		}
	}
	ing.Unit, ing.Family = "Stück", FamilyCount
	return rest
}

// splitNote separates "name, note" or "name (note)".
func splitNote(s string) (name, note string) {
	if name, note, ok := strings.Cut(s, ","); ok {
		return strings.TrimSpace(name), strings.TrimSpace(note)
	}
	if i := strings.LastIndex(s, " ("); i > 0 && strings.HasSuffix(s, ")") {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+2 : len(s)-1])
	}
	return s, ""
}
