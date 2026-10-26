// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const (
	dirMode  = 0o750
	fileMode = 0o640
)

// WriteAtomic replaces path with data so readers never see a partial file.
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // gone after the rename; cleans up on errors
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), fileMode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Doc is a Markdown document with YAML frontmatter.
type Doc[M any] struct {
	Meta M
	Body string
}

var fence = []byte("---\n")

// ReadDoc reads a document; a file without frontmatter has empty Meta.
func ReadDoc[M any](path string) (Doc[M], error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Doc[M]{}, err
	}
	d, err := ParseDoc[M](b)
	if err != nil {
		return d, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// ParseDoc splits frontmatter and body.
func ParseDoc[M any](b []byte) (Doc[M], error) {
	var d Doc[M]
	if !bytes.HasPrefix(b, fence) {
		d.Body = string(b)
		return d, nil
	}
	head, body, ok := bytes.Cut(b[len(fence):], append([]byte("\n"), fence...))
	if !ok {
		d.Body = string(b)
		return d, nil
	}
	if err := yaml.Unmarshal(head, &d.Meta); err != nil {
		return d, fmt.Errorf("frontmatter: %w", err)
	}
	d.Body = string(bytes.TrimPrefix(body, []byte("\n")))
	return d, nil
}

// WriteDoc writes frontmatter and body atomically.
func WriteDoc[M any](path string, d Doc[M]) error {
	head, err := yaml.Marshal(d.Meta)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	b.Write(fence)
	b.Write(head)
	b.Write(fence)
	b.WriteString("\n")
	b.WriteString(d.Body)
	return WriteAtomic(path, b.Bytes())
}

// ReadYAML decodes a YAML file into v.
func ReadYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, v)
}

// WriteYAML encodes v into a YAML file atomically.
func WriteYAML(path string, v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return WriteAtomic(path, b)
}
