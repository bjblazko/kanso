// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/tasks"
)

func TestTaskAttachmentUploadAndServe(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Tasche"})
	resp, out := uploadTo(t, ts, anna, taskURL(tk), "Bon.txt", "hallo")
	if resp.StatusCode != http.StatusOK || out["markdown"] == "" || out["name"] != "bon.txt" {
		t.Fatalf("upload: %d %v", resp.StatusCode, out)
	}
	if resp, body := get(t, anna, ts.URL+taskURL(tk)+"/attachments/bon.txt"); resp.StatusCode != http.StatusOK || body != "hallo\n" { // get reads line by line
		t.Fatalf("serve: %d %q", resp.StatusCode, body)
	}
	if resp, _ := get(t, ben, ts.URL+taskURL(tk)+"/attachments/bon.txt"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ben sees anna's private file: %d", resp.StatusCode)
	}
	if resp, _ := uploadTo(t, ts, ben, taskURL(tk), "x.txt", "x"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ben uploads to anna's task: %d", resp.StatusCode)
	}
}

func TestTaskEditOffersUpload(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Tasche"})
	_, body := get(t, anna, ts.URL+taskURL(tk)+"/edit")
	if !strings.Contains(body, `data-upload-url="`+taskURL(tk)+`/attachments"`) || !strings.Contains(body, `id="file"`) {
		t.Fatalf("no upload in editor:\n%s", body)
	}
}
