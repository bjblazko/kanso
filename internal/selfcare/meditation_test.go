// SPDX-License-Identifier: Apache-2.0

package selfcare

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeditationsComplete(t *testing.T) {
	if len(Voices()) < 2 || len(Meditations()) != 2 {
		t.Fatalf("voices %d, meditations %d", len(Voices()), len(Meditations()))
	}
	for _, m := range Meditations() {
		fills := 0
		for i, l := range m.Lines {
			if l.De == "" || l.En == "" {
				t.Errorf("%s line %d without both languages", m.ID, i+1)
			}
			if l.Fill {
				fills++
			}
		}
		if fills != 1 || m.Title.En == "" || m.Lead.De == "" {
			t.Errorf("%s: %d fill lines, title or lead missing", m.ID, fills)
		}
	}
}

// Every line exists as a small clip in every language and voice (made with
// tools/meditation, see its README).
func TestMeditationClipsPresentAndSmall(t *testing.T) {
	const most = 120 << 10
	for _, m := range Meditations() {
		for _, lang := range []string{"de", "en"} {
			for _, v := range Voices() {
				for i := range m.Lines {
					fi, err := os.Stat(filepath.Join("..", "..", "web", "static", filepath.FromSlash(m.AudioPath(v.ID, lang, i))))
					if err != nil {
						t.Errorf("missing: %v", err)
						continue
					}
					if fi.Size() > most {
						t.Errorf("%s/%s/%s: %d KiB", v.ID, lang, fi.Name(), fi.Size()>>10)
					}
				}
			}
		}
	}
}
