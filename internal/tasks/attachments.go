// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bjblazko/kanso/internal/store"
)

// ErrTooLarge means an upload exceeded the size limit.
var ErrTooLarge = store.ErrTooLarge

const attachmentsDir = "attachments"

// File is one upload: its original name and its content.
type File struct {
	Name string
	R    io.Reader
}

// attachmentRel is the folder of a task's attachments, relative to the data
// directory, next to the board's tasks so the task file keeps its place.
func attachmentRel(boardID, taskID string) string {
	return filepath.Join(Dir, boardID, attachmentsDir, taskID)
}

// AddAttachment stores one file next to a task the person sees.
func (s *Store) AddAttachment(a Actor, boardID, taskID, filename string, r io.Reader, max int64) (string, error) {
	names, err := s.AddAttachments(a, boardID, taskID, []File{{Name: filename, R: r}}, max)
	if err != nil {
		return "", err
	}
	return names[0], nil
}

// AddAttachments stores files next to a task as one commit and returns their
// stored names in order.
func (s *Store) AddAttachments(a Actor, boardID, taskID string, files []File, max int64) ([]string, error) {
	var names []string
	err := s.repo.Locked(func() error {
		t, err := s.Get(a, boardID, taskID)
		if err != nil {
			return err
		}
		rel := attachmentRel(boardID, taskID)
		for _, f := range files {
			name, err := store.SaveUpload(filepath.Join(s.dataDir, rel), f.Name, f.R, max)
			if err != nil {
				return err
			}
			names = append(names, name)
		}
		return s.repo.Commit(a.Author, commitMsg(t, "Anhang hinzugefügt"), rel)
	})
	return names, err
}

// OpenAttachment opens a task's file; an invisible board or a name that
// could leave the folder is ErrNotFound.
func (s *Store) OpenAttachment(a Actor, boardID, taskID, name string) (*os.File, error) {
	if _, err := s.Get(a, boardID, taskID); err != nil {
		return nil, err
	}
	if !store.ValidAttachmentName(name) {
		return nil, ErrNotFound
	}
	abs, err := store.Join(s.dataDir, attachmentRel(boardID, taskID), name)
	if err != nil {
		return nil, ErrNotFound
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, ErrNotFound
	}
	return f, nil
}

// hasAttachments reports whether a task's attachment folder exists.
func (s *Store) hasAttachments(boardID, taskID string) bool {
	_, err := os.Stat(filepath.Join(s.dataDir, attachmentRel(boardID, taskID)))
	return err == nil
}

// moveAttachments takes a task's folder to its new board; the caller holds the repo lock.
func (s *Store) moveAttachments(a Actor, t Task, fromBoard string) error {
	if !s.hasAttachments(fromBoard, t.Meta.ID) {
		return nil
	}
	return s.repo.Move(a.Author, attachmentRel(fromBoard, t.Meta.ID), attachmentRel(t.Board, t.Meta.ID), commitMsg(t, "Anhänge verschoben"))
}

// removeAttachments deletes a task's folder and returns its path when there
// was one, for the caller's commit.
func (s *Store) removeAttachments(boardID, taskID string) ([]string, error) {
	if !s.hasAttachments(boardID, taskID) {
		return nil, nil
	}
	rel := attachmentRel(boardID, taskID)
	return []string{rel}, os.RemoveAll(filepath.Join(s.dataDir, rel))
}
