// SPDX-License-Identifier: Apache-2.0

package recipegen

import (
	"errors"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func form(kv ...string) url.Values {
	v := url.Values{"ingredients": {"Reis"}}
	for i := 0; i+1 < len(kv); i += 2 {
		v[kv[i]] = append([]string(nil), kv[i+1])
	}
	return v
}

func TestParseRefusesUnknownChoice(t *testing.T) {
	for _, kv := range [][2]string{{"style", "evil"}, {"temp", "lava"}, {"heat", "x"}, {"time", "5"}, {"without", "joy"}, {"servings", "0"}, {"servings", "abc"}, {"servings", "100"}} {
		if _, err := Parse(form(kv[0], kv[1])); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s=%s: %v", kv[0], kv[1], err)
		}
	}
	w, err := Parse(form("style", "homestyle", "temp", "warm", "heat", "hot", "time", "45", "without", "meat"))
	if err != nil || w.Servings != 4 || w.Style != "homestyle" || !slices.Equal(w.Without, []string{"meat"}) {
		t.Fatalf("%+v %v", w, err)
	}
}

func TestParseCleansAndCuts(t *testing.T) {
	w, err := Parse(url.Values{"ingredients": {"Hähnchen, <b>Paprika</b>\n\x07Reis,,"}})
	if err != nil || !slices.Equal(w.Ingredients, []string{"Hähnchen", "bPaprika/b", "Reis"}) {
		t.Fatalf("%q %v", w.Ingredients, err)
	}
	many := make([]string, 20)
	for i := range many {
		many[i] = "Zutat" + strconv.Itoa(i)
	}
	many[0] = strings.Repeat("x", 60)
	w, _ = Parse(url.Values{"ingredients": {strings.Join(many, ",")}, "extra": {strings.Repeat("ü", 300)}})
	if len(w.Ingredients) != 15 || len([]rune(w.Ingredients[0])) != 40 || len([]rune(w.Extra)) != 200 {
		t.Fatalf("%d %d %d", len(w.Ingredients), len([]rune(w.Ingredients[0])), len([]rune(w.Extra)))
	}
	if _, err := Parse(url.Values{"ingredients": {" , \n"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty: %v", err)
	}
}

func TestValuesRoundTrip(t *testing.T) {
	w := Wishes{Ingredients: []string{"Reis", "Ei"}, Extend: true, Temp: "cold", Heat: "mild", Style: "asian", Time: "20",
		Without: []string{"fish", "gluten"}, Servings: 2, Extra: "für Kinder", Rejected: []string{"Bratreis"}}
	got, err := Parse(w.Values())
	if err != nil || !reflect.DeepEqual(got, w) {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRejectKeepsTenNewest(t *testing.T) {
	var w Wishes
	for i := 1; i <= 12; i++ {
		w = w.Reject("Titel " + strconv.Itoa(i))
	}
	if len(w.Rejected) != 10 || w.Rejected[0] != "Titel 3" || w.Rejected[9] != "Titel 12" {
		t.Fatalf("%q", w.Rejected)
	}
	w = Wishes{}.Reject("<i>" + strings.Repeat("a", 100))
	if w.Rejected[0] != "i"+strings.Repeat("a", 79) {
		t.Fatalf("%q", w.Rejected)
	}
	if w = (Wishes{}).Reject("  "); len(w.Rejected) != 0 {
		t.Fatalf("blank kept: %q", w.Rejected)
	}
}
