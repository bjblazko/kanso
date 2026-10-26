// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"strings"
	"testing"
)

func render(t *testing.T, src string) (string, Info) {
	t.Helper()
	html, info, err := New().Render([]byte(src), PageCtx{PageID: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	return string(html), info
}

func TestDangerousLinksAreNeutralised(t *testing.T) {
	for _, src := range []string{"[x](javascript:alert(1))", "[x](JaVaScRiPt:alert(1))", "[x](data:text/html,hi)", "[x](vbscript:x)"} {
		out, _ := render(t, src)
		if strings.Contains(strings.ToLower(out), "script:") || strings.Contains(out, "data:text") {
			t.Errorf("%s → %s", src, out)
		}
	}
}

func TestRawHTMLIsNotRendered(t *testing.T) {
	out, _ := render(t, "<script>alert(1)</script>\n\nText <img src=x onerror=alert(1)>")
	if strings.Contains(out, "<script") || strings.Contains(out, "<img src=x") {
		t.Errorf("raw html: %s", out)
	}
}

func TestAttachmentPathsAreRewritten(t *testing.T) {
	out, _ := render(t, "![Plan](attachments/grundriss.png) [PDF](attachments/rechnung.pdf)")
	if !strings.Contains(out, `src="/p/abc/attachments/grundriss.png"`) || !strings.Contains(out, `href="/p/abc/attachments/rechnung.pdf"`) {
		t.Errorf("attachments: %s", out)
	}
}

func TestEscapingPathsAreRejected(t *testing.T) {
	out, _ := render(t, "![a](../../etc/passwd) [b](attachments/../../x)")
	if strings.Contains(out, "passwd") || strings.Contains(out, "../") {
		t.Errorf("escape: %s", out)
	}
}

func TestAllowedLinksStay(t *testing.T) {
	out, _ := render(t, "[a](https://example.org) [b](mailto:a@b.de) [c](/p/xyz) [d](#abschnitt)")
	for _, want := range []string{`href="https://example.org"`, `href="mailto:a@b.de"`, `href="/p/xyz"`, `href="#abschnitt"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestMermaidBlock(t *testing.T) {
	out, info := render(t, "```mermaid\ngraph TD\n  A-->B\n```\n")
	if !info.HasMermaid || !strings.Contains(out, `<pre class="mermaid">`) || !strings.Contains(out, "A--&gt;B") {
		t.Errorf("mermaid: %v %s", info, out)
	}
	_, info = render(t, "```go\nfmt.Println()\n```\n")
	if info.HasMermaid {
		t.Error("go block reported as mermaid")
	}
}

func TestCodeBlockEscapesAndLabelsLanguage(t *testing.T) {
	out, _ := render(t, "```go\nif a < b {}\n```\n")
	if !strings.Contains(out, `<code class="language-go">`) || !strings.Contains(out, "a &lt; b") {
		t.Errorf("code: %s", out)
	}
}

func TestGFM(t *testing.T) {
	out, _ := render(t, "| A | B |\n|---|---|\n| 1 | 2 |\n\n- [x] erledigt\n- [ ] offen\n\n~~alt~~")
	for _, want := range []string{"<table>", `type="checkbox"`, "<del>alt</del>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}
