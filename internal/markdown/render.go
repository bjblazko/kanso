// SPDX-License-Identifier: Apache-2.0

// Package markdown turns page Markdown into safe HTML.
package markdown

import (
	"bytes"
	"html/template"
	"io"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
)

// PageCtx is what rendering needs to know about the page.
type PageCtx struct {
	PageID string
}

// Info describes what the rendered page contains.
type Info struct {
	HasMermaid bool
}

// Renderer converts Markdown; raw HTML is never passed through.
type Renderer struct {
	parser   parser.Parser
	renderer html.Renderer
}

// New creates a renderer with GitHub-flavoured Markdown.
func New() *Renderer {
	return &Renderer{
		parser: parser.New(parser.WithAutoHeadingID(), parser.WithExtensions(extension.GFMParser)),
		renderer: html.New(html.WithExtensions(extension.GFMHTMLRenderer),
			html.WithNodeRendererDecorator(ast.KindCodeBlock, func(html.NodeRenderer) html.NodeRenderer {
				return html.NodeRendererFunc(renderCodeBlock)
			})),
	}
}

// Render returns the HTML for src.
func (r *Renderer) Render(src []byte, pc PageCtx) (template.HTML, Info, error) {
	doc := r.parser.Parse(src)
	info := rewrite(doc, src, pc)
	var buf bytes.Buffer
	if err := r.renderer.Render(&buf, src, doc); err != nil {
		return "", info, err
	}
	// The renderer escapes all text and drops raw HTML and dangerous URLs;
	// rewrite() limits link targets further.
	return template.HTML(buf.String()), info, nil //nolint:gosec // see above
}

// rewrite fixes link and image targets and collects page facts.
func rewrite(doc ast.Node, src []byte, pc PageCtx) Info {
	var info Info
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Link:
			node.Destination = safeTarget(node.Destination.Value(src), pc)
		case *ast.Image:
			node.Destination = safeTarget(node.Destination.Value(src), pc)
		case *ast.CodeBlock:
			lang, _ := node.Language(src)
			info.HasMermaid = info.HasMermaid || lang == "mermaid"
		}
		return ast.WalkContinue, nil
	})
	return info
}

var (
	pageLink       = regexp.MustCompile(`^/(p/[a-z0-9]+(/attachments/[^/?#]+)?|s/[a-z0-9-]+)$`)
	attachmentLink = regexp.MustCompile(`^attachments/[^/?#]+$`)
)

// safeTarget allows web and mail links, anchors, Kanso pages and the page's
// own attachments; everything else becomes "#".
func safeTarget(dest string, pc PageCtx) text.SingleLineValue {
	d := strings.TrimSpace(dest)
	lower := strings.ToLower(d)
	switch {
	case strings.Contains(d, ".."):
		d = "#"
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "http://"),
		strings.HasPrefix(lower, "mailto:"), strings.HasPrefix(d, "#"), pageLink.MatchString(d):
	case attachmentLink.MatchString(d):
		d = "/p/" + pc.PageID + "/" + d
	default:
		d = "#"
	}
	return text.NewSingleLineValue(d, text.IdentityDecoder)
}

// renderCodeBlock writes Mermaid blocks for the diagram script and all other
// blocks as escaped code.
func renderCodeBlock(w io.Writer, src []byte, n ast.Node, entering bool, _ renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	cb := n.(*ast.CodeBlock)
	code := template.HTMLEscapeString(string(cb.Value.Bytes(src)))
	lang, _ := cb.Language(src)
	var err error
	switch {
	case lang == "mermaid":
		_, err = io.WriteString(w, `<pre class="mermaid">`+code+"</pre>\n")
	case lang != "":
		_, err = io.WriteString(w, `<pre><code class="language-`+template.HTMLEscapeString(lang)+`">`+code+"</code></pre>\n")
	default:
		_, err = io.WriteString(w, "<pre><code>"+code+"</code></pre>\n")
	}
	return ast.WalkSkipChildren, err
}
