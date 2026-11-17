// SPDX-License-Identifier: Apache-2.0

package recipegen

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/ai"
	"github.com/bjblazko/kanso/internal/recipeimport"
)

//go:embed prompt.md
var prompt string

// phrases say a choice to the AI, in German like the prompt.
var phrases = map[string]string{
	"warm": "warm serviert", "cold": "kalt serviert", "mild": "mild", "hot": "scharf",
	"homestyle": "gutbürgerlich", "italian": "italienisch", "mediterranean": "mediterran", "asian": "asiatisch",
	"indian": "indisch", "oriental": "orientalisch", "mexican": "mexikanisch", "french": "französisch",
	"meat": "Fleisch", "fish": "Fisch", "vegetables": "Gemüse", "gluten": "Gluten", "lactose": "Laktose",
}

var languageNames = map[string]string{"de": "Deutsch", "en": "Englisch"}

const pantry = "Salz, Pfeffer, Öl, Butter, Zwiebeln, Knoblauch, Mehl, Zucker"

// Messages are the system prompt and the wishes as data.
func Messages(w Wishes, lang string) []ai.Message {
	return []ai.Message{{Role: "system", Content: prompt}, {Role: "user", Content: userMessage(w, lang)}}
}

func userMessage(w Wishes, lang string) string {
	name, ok := languageNames[lang]
	if !ok {
		name = languageNames["de"]
	}
	lines := []string{"Sprache: " + name, "Portionen: " + strconv.Itoa(w.Servings)}
	if w.Extend {
		lines = append(lines, "Weitere Zutaten erlaubt.")
	} else {
		lines = append(lines, "Nur diese Zutaten plus Grundvorrat ("+pantry+").")
	}
	lines = append(lines, choiceLines(w)...)
	lines = append(lines, block("zutaten", w.Ingredients))
	if w.Extra != "" {
		lines = append(lines, block("sonst", []string{w.Extra}))
	}
	if len(w.Rejected) > 0 {
		lines = append(lines, block("schon_verworfen", w.Rejected))
	}
	return strings.Join(lines, "\n")
}

func choiceLines(w Wishes) []string {
	var out []string
	for _, c := range [][2]string{{"Temperatur: ", w.Temp}, {"Schärfe: ", w.Heat}, {"Richtung: ", w.Style}} {
		if c[1] != "" {
			out = append(out, c[0]+phrases[c[1]])
		}
	}
	if w.Time != "" {
		out = append(out, "Zeit: höchstens "+w.Time+" Minuten")
	}
	if len(w.Without) > 0 {
		var no []string
		for _, x := range w.Without {
			no = append(no, phrases[x])
		}
		out = append(out, "Ohne: "+strings.Join(no, ", "))
	}
	return out
}

// block marks data; clean has taken < and > out of it, so it cannot close the block.
func block(name string, lines []string) string {
	return "<" + name + ">\n" + strings.Join(lines, "\n") + "\n</" + name + ">"
}

// Generate asks the AI for one recipe; anything that is not one is recipeimport.ErrNoRecipe.
func Generate(ctx context.Context, c ai.Client, cfg ai.Config, w Wishes, lang string) (recipeimport.Draft, error) {
	answer, err := c.Complete(ctx, cfg, Messages(w, lang))
	if err != nil {
		return recipeimport.Draft{}, err
	}
	d := recipeimport.ParseAnswer(answer)
	d.Source = ""
	if d.Title == "" || len(d.Ingredients) == 0 {
		return recipeimport.Draft{}, recipeimport.ErrNoRecipe
	}
	return d, nil
}
