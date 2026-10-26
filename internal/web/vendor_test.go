// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

const mermaidDir = "../../web/static/vendor/mermaid"

func TestMermaidIsVendoredWithoutELK(t *testing.T) {
	if _, err := os.Stat(filepath.Join(mermaidDir, "mermaid.esm.min.mjs")); err != nil {
		t.Fatalf("mermaid not vendored: %v (run scripts/vendor-mermaid.sh)", err)
	}
	err := filepath.WalkDir(mermaidDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasPrefix(d.Name(), "elk-") || strings.HasSuffix(d.Name(), ".map") {
			t.Errorf("must not ship %s", d.Name())
		}
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), "org.eclipse.elk") {
			t.Errorf("%s contains ELK code (EPL-2.0)", d.Name())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMermaidLoaderOnlyWhereNeeded(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	with := ts.createPage(t, admin, "haus", "", "Mit Diagramm")
	without := ts.createPage(t, admin, "haus", "", "Ohne Diagramm")
	ts.deps.Wiki.Save(store.Author{Name: "Admin", Email: "a@x"}, with, "Mit Diagramm", "```mermaid\ngraph TD\n A-->B\n```\n")
	_, body := get(t, admin, ts.URL+"/p/"+with)
	if !strings.Contains(body, "/static/js/mermaid-loader.js") {
		t.Error("page with diagram lacks the loader")
	}
	_, body = get(t, admin, ts.URL+"/p/"+without)
	if strings.Contains(body, "mermaid-loader.js") {
		t.Error("page without diagram loads the loader")
	}
	body, _, _ = openEditor(t, ts, admin, without)
	if !strings.Contains(body, "/static/js/mermaid-loader.js") {
		t.Error("editor lacks the loader for previews")
	}
}
