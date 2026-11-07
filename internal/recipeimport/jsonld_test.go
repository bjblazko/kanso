// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"slices"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/recipes"
)

const simple = `<html><head><script type="application/ld+json">
{"@context":"https://schema.org","@type":"Recipe","name":"Kartoffelcurry &amp; Reis","description":"Mild und schnell.",
 "recipeYield":"4 Portionen","totalTime":"PT1H5M",
 "recipeIngredient":["800 g Kartoffeln","2 Zwiebeln","1 Dose Kokosmilch"],
 "recipeInstructions":[{"@type":"HowToStep","text":"Kartoffeln schälen."},{"@type":"HowToStep","text":"Kochen."}]}
</script></head><body>x</body></html>`

const graph = `<script type='application/ld+json'>{"@graph":[{"@type":"WebPage"},{"@type":["Recipe","NewsArticle"],"name":"Brot",
 "recipeYield":["2","2 Laibe"],"prepTime":"PT20M","cookTime":"PT40M","recipeIngredient":["500 g Mehl"],
 "recipeInstructions":[{"@type":"HowToSection","name":"Teig","itemListElement":[{"@type":"HowToStep","text":"Kneten."}]},
  {"@type":"HowToSection","name":"Backen","itemListElement":[{"@type":"HowToStep","text":"Backen."}]}]}]}</script>`

const list = `<script type="application/ld+json">[{"@type":"Organization"},{"@type":"Recipe","name":"Suppe","recipeYield":3,
 "recipeIngredient":["1 l Brühe"],"recipeInstructions":"Alles kochen.\nServieren."}]</script>`

func TestFromJSONLD(t *testing.T) {
	d, ok := FromJSONLD(simple, "https://beispiel.de/curry")
	if !ok || d.Title != "Kartoffelcurry & Reis" || d.Servings != 4 || d.Time != 65 || d.Description != "Mild und schnell." ||
		len(d.Ingredients) != 3 || !slices.Equal(d.Steps, []string{"Kartoffeln schälen.", "Kochen."}) || d.Source != "https://beispiel.de/curry" {
		t.Fatalf("simple %+v", d)
	}
	d, ok = FromJSONLD(graph, "https://b.de")
	if !ok || d.Title != "Brot" || d.Servings != 2 || d.Time != 60 || !slices.Equal(d.Steps, []string{"Kneten.", "Backen."}) {
		t.Fatalf("graph %+v", d)
	}
	d, ok = FromJSONLD(list, "https://c.de")
	if !ok || d.Servings != 3 || !slices.Equal(d.Steps, []string{"Alles kochen.", "Servieren."}) {
		t.Fatalf("list %+v", d)
	}
	if _, ok := FromJSONLD(`<script type="application/ld+json">{"@type":"Article"}</script>`, "x"); ok {
		t.Error("article taken as recipe")
	}
	if _, ok := FromJSONLD(`<script type="application/ld+json">{broken</script>`, "x"); ok {
		t.Error("broken json")
	}
}

func TestDraftMarkdownParses(t *testing.T) {
	d, _ := FromJSONLD(simple, "https://beispiel.de/curry")
	md := d.Markdown("de")
	r := recipes.Parse(md)
	if len(r.Ingredients()) != 3 || len(r.Steps) != 2 || !strings.Contains(md, "Mild und schnell.") ||
		!strings.Contains(md, "Quelle: [beispiel.de](https://beispiel.de/curry). Für den privaten Gebrauch übernommen.") {
		t.Fatalf("%s", md)
	}
	if en := d.Markdown("en"); !strings.Contains(en, "## Ingredients") || !strings.Contains(en, "## Method") {
		t.Errorf("%s", en)
	}
}

func TestText(t *testing.T) {
	got := Text(`<html><head><style>.x{}</style><script>var a=1</script></head><body><nav>Menü</nav>
<h1>Brot &amp; Butter</h1><p>500 g Mehl<br>Wasser</p><footer>Impressum</footer></body></html>`)
	if !strings.Contains(got, "Brot & Butter") || !strings.Contains(got, "500 g Mehl") || strings.Contains(got, "var a") ||
		strings.Contains(got, "Menü") || strings.Contains(got, "Impressum") || strings.Contains(got, ".x{}") {
		t.Fatalf("%q", got)
	}
}
