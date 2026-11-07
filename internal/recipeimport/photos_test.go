// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"slices"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/recipes"
)

func TestStepImagesFromJSONLD(t *testing.T) {
	page := `<script type="application/ld+json">{"@type":"Recipe","name":"X","recipeIngredient":["1 Ei"],"image":"/haupt.jpg",
"recipeInstructions":[{"@type":"HowToStep","text":"Schneiden.","image":"/s1.jpg"},{"@type":"HowToStep","text":"Braten."},
{"@type":"HowToStep","text":"Anrichten.","image":{"url":"https://cdn.example/s3.jpg"}}]}</script>`
	d, _ := FromJSONLD(page, "https://site.example/r")
	if !slices.Equal(d.StepImages, []string{"https://site.example/s1.jpg", "", "https://cdn.example/s3.jpg"}) {
		t.Fatalf("%q", d.StepImages)
	}
}

func TestPagePhotos(t *testing.T) {
	page := `<img src="/logo.svg" width="300"><img src="https://cdn.example/icon.png" width="20" height="20">
<img src="https://static.example/media/a.jpeg/v1/fill/w_147,h_259/a.jpeg">
<img src="https://static.example/media/b.jpeg/v1/fill/w_1536,h_2048/b.jpeg" alt="Teig">
<img data-src="/bilder/c.jpg" src="data:image/gif;base64,R0lGOD" width="800">
<img src="https://static.example/media/b.jpeg/v1/fill/w_600,h_800/b.jpeg">
<img src="/haupt.jpg">`
	got := PagePhotos(page, "https://site.example/r", "https://site.example/haupt.jpg")
	want := []string{"https://static.example/media/b.jpeg/v1/fill/w_1536,h_2048/b.jpeg", "https://site.example/bilder/c.jpg"}
	if !slices.Equal(got, want) {
		t.Fatalf("%q", got)
	}
}

func TestInsertPhotos(t *testing.T) {
	body := "Kurz.\n\n## Zutaten\n\n- 1 Ei\n\n## Zubereitung\n\n1. Schneiden.\n2. Braten,\n   bis braun.\n3. Anrichten.\n\n---\n\nQuelle: x\n"
	got := InsertPhotos(body, []Placement{
		{Place: "top", Markdown: "![X](attachments/foto-1.jpg)"},
		{Place: "step-2", Markdown: "![Schritt 2](attachments/foto-2.jpg)"},
		{Place: "end", Markdown: "![Y](attachments/foto-3.jpg)"},
	})
	if !strings.HasPrefix(got, "![X](attachments/foto-1.jpg)\n\nKurz.") {
		t.Errorf("top:\n%s", got)
	}
	r := recipes.Parse(got)
	if !slices.Equal(r.Steps, []string{"Schneiden.", "Braten, bis braun.", "Anrichten."}) || r.StepImages[1] != "attachments/foto-2.jpg" {
		t.Errorf("step:\n%s\n%q %q", got, r.Steps, r.StepImages)
	}
	if !strings.Contains(got, "![Y](attachments/foto-3.jpg)\n\n---") {
		t.Errorf("end:\n%s", got)
	}
	if got := InsertPhotos("## Zubereitung\n\n1. A\n", []Placement{{Place: "step-9", Markdown: "![](x)"}}); !strings.HasSuffix(strings.TrimSpace(got), "![](x)") {
		t.Errorf("missing step goes to the end:\n%s", got)
	}
}
