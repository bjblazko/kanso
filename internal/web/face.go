// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// face is someone shown by picture: round avatar, initials below, full name on hover.
type face struct {
	Sub, Name, Initials string
	// AvatarVersion names the picture; "" means the placeholder.
	AvatarVersion string
}

func (s *Server) face(sub string) *face {
	name := s.userName(sub)
	return &face{Sub: sub, Name: name, Initials: initials(name), AvatarVersion: s.Users.AvatarVersion(sub)}
}

// initials takes the first letter of the first and the last word of a name.
func initials(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return ""
	}
	out := firstLetter(words[0])
	if len(words) > 1 {
		out += firstLetter(words[len(words)-1])
	}
	return out
}

func firstLetter(w string) string {
	r, _ := utf8.DecodeRuneInString(w)
	return string(unicode.ToUpper(r))
}
