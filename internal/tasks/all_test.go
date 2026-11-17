// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllTasksAcrossBoards(t *testing.T) {
	s := newStore(t)
	anna, ben := actor("anna"), actor("ben")
	for _, a := range []Actor{anna, ben} {
		if _, _, err := s.Create(a, New{Board: ownBoard(t, s, a).ID, Title: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	all := s.AllTasks()
	if len(all) != 2 {
		t.Fatalf("AllTasks = %d", len(all))
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, TaskPath(all[0].Board, all[0].Meta.ID))); err != nil {
		t.Errorf("TaskPath: %v", err)
	}
}
