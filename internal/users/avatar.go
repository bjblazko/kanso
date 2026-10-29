// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// avatarDir holds the profile pictures, relative to the data directory and
// outside the Git history: a picture is a copy of what the sign-in names.
const avatarDir = "avatars"

// maxAvatar is the largest picture Kanso keeps.
const maxAvatar = 1 << 20

var avatarTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true}

func (s *Store) avatarPath(sub string) string { return filepath.Join(s.dataDir, avatarDir, sub) }

// SyncAvatar keeps a copy of the picture the sign-in names (the "picture"
// claim), so browsers load it from Kanso and not from wherever it lives; no
// picture removes the copy. Only HTTPS, only images, at most maxAvatar.
func (s *Store) SyncAvatar(ctx context.Context, c *http.Client, sub, picture string) error {
	if !validSub(sub) {
		return fmt.Errorf("unsafe subject %q", sub)
	}
	if picture == "" {
		if err := os.Remove(s.avatarPath(sub)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	b, err := fetchImage(ctx, c, picture)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.dataDir, avatarDir), 0o750); err != nil {
		return err
	}
	return store.WriteAtomic(s.avatarPath(sub), b)
}

func fetchImage(ctx context.Context, c *http.Client, picture string) ([]byte, error) {
	if u, err := url.Parse(picture); err != nil || u.Scheme != "https" {
		return nil, errors.New("picture: not an https URL")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, picture, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxAvatar+1))
	switch {
	case err != nil:
		return nil, err
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("picture: %s", resp.Status)
	case len(b) > maxAvatar:
		return nil, errors.New("picture: too large")
	case !avatarTypes[http.DetectContentType(b)]:
		return nil, errors.New("picture: not an image")
	}
	return b, nil
}

// Avatar returns sub's picture and its type, if there is one.
func (s *Store) Avatar(sub string) ([]byte, string, bool) {
	if !validSub(sub) {
		return nil, "", false
	}
	b, err := os.ReadFile(s.avatarPath(sub))
	if err != nil || !avatarTypes[http.DetectContentType(b)] {
		return nil, "", false
	}
	return b, http.DetectContentType(b), true
}

// AvatarVersion changes whenever sub's picture does; "" means there is none.
func (s *Store) AvatarVersion(sub string) string {
	if !validSub(sub) {
		return ""
	}
	fi, err := os.Stat(s.avatarPath(sub))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(fi.ModTime().UnixNano(), 36)
}
