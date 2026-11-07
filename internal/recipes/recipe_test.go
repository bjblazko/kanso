// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"slices"
	"testing"
)

const curry = `Kurz und mild, die Kinder essen es.

## Zutaten

- 800 g Kartoffeln, festkochend
- 2 Zwiebeln
- 1 Dose Kokosmilch
- 1,5 EL Currypaste
- ½ TL Salz
- Koriander zum Bestreuen

### Für den Reis

- 300 g Basmatireis

## Zubereitung

1. Kartoffeln schälen und würfeln.
2. Zwiebeln anschwitzen, Currypaste dazu.
3. Alles köcheln lassen.

---

Quelle: [beispiel.de](https://beispiel.de/kartoffelcurry). Für den privaten Gebrauch übertragen.
`

func TestParseSections(t *testing.T) {
	r := Parse(curry)
	if len(r.Groups) != 2 || r.Groups[0].Name != "" || len(r.Groups[0].Ingredients) != 6 ||
		r.Groups[1].Name != "Für den Reis" || len(r.Groups[1].Ingredients) != 1 {
		t.Fatalf("groups %+v", r.Groups)
	}
	if len(r.Steps) != 3 || r.Steps[0] != "Kartoffeln schälen und würfeln." {
		t.Fatalf("steps %q", r.Steps)
	}
	for _, ing := range r.Ingredients() {
		if ing.Name == "Kurz und mild" || ing.Key == "quelle" {
			t.Errorf("free text as ingredient: %+v", ing)
		}
	}
	if len(r.Ingredients()) != 7 {
		t.Errorf("flat %d", len(r.Ingredients()))
	}
}

func TestSectionNames(t *testing.T) {
	for _, ing := range []string{"ZUTATEN", "Ingredients", "zutaten "} {
		for _, steps := range []string{"Schritte", "Steps", "Method", "Zubereitung"} {
			r := Parse("## " + ing + "\n- 1 Ei\n## " + steps + "\n1. Kochen\n")
			if len(r.Ingredients()) != 1 || len(r.Steps) != 1 {
				t.Errorf("%s/%s: %+v", ing, steps, r)
			}
		}
	}
	r := Parse("## Zutaten\n- 1 Ei\n## Tipps\n- Salz nach Geschmack\n")
	if len(r.Ingredients()) != 1 {
		t.Errorf("## Tipps did not end the section: %+v", r)
	}
}

func TestStepContinuation(t *testing.T) {
	r := Parse("## Zubereitung\n\n1. Zwiebeln\n   fein hacken.\n2. Braten.\n")
	if len(r.Steps) != 2 || r.Steps[0] != "Zwiebeln fein hacken." {
		t.Fatalf("%q", r.Steps)
	}
	r = Parse("## Schritte\n- Rühren\n- Backen\n")
	if len(r.Steps) != 2 || r.Steps[1] != "Backen" {
		t.Fatalf("bullets %q", r.Steps)
	}
}

func TestParseNoSections(t *testing.T) {
	r := Parse("Nur Text.\r\n\r\n- 200 g Mehl\r\n")
	if len(r.Groups) != 0 || len(r.Steps) != 0 {
		t.Fatalf("%+v", r)
	}
	r = Parse("## Zutaten\r\n- 200 g Mehl\r\n")
	if len(r.Ingredients()) != 1 || r.Ingredients()[0].Name != "Mehl" {
		t.Fatalf("crlf %+v", r)
	}
}

func TestCodeFenceIgnored(t *testing.T) {
	r := Parse("## Zutaten\n- 1 Ei\n```\n- 2 kg Code\n```\n")
	if len(r.Ingredients()) != 1 {
		t.Fatalf("%+v", r.Ingredients())
	}
}

func TestKansoRezeptSkillFormat(t *testing.T) {
	r := Parse("# Brot\n\n- **Portionen:** 2\n\n## Zutaten\n\n- **500 g** Mehl\n- **1 TL** Salz\n\n## Schritte\n\n1. Kneten\n2. Backen\n")
	ings := r.Ingredients()
	if len(ings) != 2 || ings[0].Name != "Mehl" || ings[1].Unit != "TL" || len(r.Steps) != 2 {
		t.Fatalf("%+v %q", ings, r.Steps)
	}
}

func TestParagraphStepsAndLooseIngredients(t *testing.T) {
	r := Parse("## Zutaten\n200 g Mehl\n1 Prise Salz\n\n## Zubereitung\nMehl sieben.\nDann salzen.\n\nBacken.\n")
	if ings := r.Ingredients(); len(ings) != 2 || ings[0].Name != "Mehl" || ings[1].Unit != "Prise" {
		t.Errorf("loose ingredients %+v", ings)
	}
	if len(r.Steps) != 2 || r.Steps[0] != "Mehl sieben. Dann salzen." || r.Steps[1] != "Backen." {
		t.Errorf("paragraph steps %q", r.Steps)
	}
}

func TestHeadingVariantsAndHashtags(t *testing.T) {
	for _, h := range []string{"## Zutaten (für 4 Personen)", "## Zutaten:", "## Zutaten für den Teig", "# Ingredients"} {
		if r := Parse(h + "\n- 1 Ei\n"); len(r.Ingredients()) != 1 {
			t.Errorf("%q not recognised", h)
		}
	}
	r := Parse("## Zutaten\n- 1 Ei\n#vegan #schnell\n- 200 g Mehl\n")
	if len(r.Ingredients()) != 2 {
		t.Errorf("hashtag ended the section: %+v", r.Ingredients())
	}
}

func TestStepImages(t *testing.T) {
	r := Parse("## Zubereitung\n\n1. Schneiden.\n\n   ![Schritt 1](attachments/s1.jpg)\n\n2. Braten.\n3. Servieren.\n   ![](attachments/s3.jpg)\n")
	if !slices.Equal(r.Steps, []string{"Schneiden.", "Braten.", "Servieren."}) {
		t.Fatalf("steps %q", r.Steps)
	}
	if !slices.Equal(r.StepImages, []string{"attachments/s1.jpg", "", "attachments/s3.jpg"}) {
		t.Fatalf("images %q", r.StepImages)
	}
}
