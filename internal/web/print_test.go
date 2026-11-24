// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/tasks"
)

func TestPrintButton(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Drucken"})
	if _, body := get(t, anna, ts.URL+taskURL(tk)); !strings.Contains(body, `type="button" data-print>`) {
		t.Error("task page: no print button")
	}
	if _, body := get(t, anna, ts.URL+"/notes"); strings.Contains(body, "data-print") {
		t.Error("the notes overview needs no print button")
	}
}
