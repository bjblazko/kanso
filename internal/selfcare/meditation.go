// SPDX-License-Identifier: Apache-2.0

package selfcare

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed meditations.json
var meditationJSON []byte

// Bell is how long the gong at the start and the end rings, in seconds; the
// browser makes it (meditation.js), so it has no file.
const Bell = 6.0

// Voice is one of the narrators the meditations are spoken in.
type Voice struct {
	ID string `json:"id"`
	Text
}

// Line is one spoken passage and the silence after it, in seconds. The line
// marked Fill takes up whatever time the meditation has left.
type Line struct {
	Text
	Pause float64 `json:"pause"`
	Fill  bool    `json:"fill"`
}

// Meditation is a guided meditation: one small spoken clip per line, in every
// voice and language (tools/meditation); the page plays them with the pauses.
type Meditation struct {
	ID      string `json:"id"`
	Minutes int    `json:"minutes"`
	Title   Text   `json:"title"`
	Lead    Text   `json:"lead"`
	Lines   []Line `json:"lines"`
}

var meditationData struct {
	Voices      []Voice      `json:"voices"`
	Meditations []Meditation `json:"meditations"`
}

func init() {
	if err := json.Unmarshal(meditationJSON, &meditationData); err != nil {
		panic("selfcare: meditations.json: " + err.Error())
	}
}

// Voices are the narrators, the first is the default.
func Voices() []Voice { return meditationData.Voices }

// Meditations are the guided meditations, shortest first.
func Meditations() []Meditation { return meditationData.Meditations }

// MeditationByID finds a meditation.
func MeditationByID(id string) (Meditation, bool) {
	i := slices.IndexFunc(meditationData.Meditations, func(m Meditation) bool { return m.ID == id })
	if i < 0 {
		return Meditation{}, false
	}
	return meditationData.Meditations[i], true
}

// AudioPath is where line i's clip for a voice and language lives below web/static.
func (m Meditation) AudioPath(voice, lang string, i int) string {
	return fmt.Sprintf("audio/meditation/%s/%s/%s-%02d.mp3", voice, lang, m.ID, i+1)
}
