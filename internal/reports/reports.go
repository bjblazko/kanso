// SPDX-License-Identifier: Apache-2.0

// Package reports turns what Kanso holds into numbers for the "Berichte"
// pages. It knows nothing of HTTP or rights: callers pass only what the
// viewer's scope allows, with titles already cleared where they may not read.
package reports

import "time"

// Kind is what a document is.
type Kind string

// The kinds of documents, in the order reports list them.
const (
	Page       Kind = "page"
	Recipe     Kind = "recipe"
	Note       Kind = "note"
	List       Kind = "list"
	Task       Kind = "task"
	Attachment Kind = "attachment"
)

// Kinds are all kinds in report order.
var Kinds = []Kind{Page, Recipe, Note, List, Task, Attachment}

// Doc is one document as reports see it.
type Doc struct {
	Kind    Kind
	ID      string
	Title   string // "" when the viewer may not read it
	Href    string // "" when the viewer may not read it
	Person  string // a sub, "group:<name>" for a group's things, "" when unknown
	Space   string // the space's slug, or "notes", "lists", "boards"
	Path    string // the file, relative to the data directory
	Created time.Time
	Updated time.Time
	Bytes   int64
}
