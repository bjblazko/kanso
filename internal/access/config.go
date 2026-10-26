// SPDX-License-Identifier: Apache-2.0

package access

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/bjblazko/kanso/internal/store"
)

// ConfigFile holds the groups, relative to the data directory.
const ConfigFile = "kanso.yaml"

type configFile struct {
	Groups Groups `yaml:"groups"`
}

// LoadGroups reads the groups; a missing file means no groups.
func LoadGroups(dataDir string) (Groups, error) {
	var c configFile
	err := store.ReadYAML(filepath.Join(dataDir, ConfigFile), &c)
	if errors.Is(err, fs.ErrNotExist) {
		return Groups{}, nil
	}
	if c.Groups == nil {
		c.Groups = Groups{}
	}
	return c.Groups, err
}

// SaveGroups writes the groups (the caller commits the file).
func SaveGroups(dataDir string, g Groups) error {
	return store.WriteYAML(filepath.Join(dataDir, ConfigFile), configFile{Groups: g})
}
