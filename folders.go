package pommelporter

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// listDir lists the sub-folders of path for the folder chooser. A path that
// does not exist yet lists the deepest folder above it that does, so a folder
// about to be created does not show as an error.
func listDir(p setup.Platform, path string) (setup.Folder, error) {
	if !filepath.IsAbs(path) {
		return setup.Folder{}, errors.New("the folder must be an absolute path")
	}
	listed := filepath.Clean(path)
	for {
		if fi, err := os.Stat(listed); err == nil && fi.IsDir() {
			break
		}
		up := filepath.Dir(listed)
		if up == listed {
			return setup.Folder{}, errors.New("no folder at or above " + path + " exists")
		}
		listed = up
	}
	items, err := os.ReadDir(listed)
	if err != nil {
		return setup.Folder{}, err
	}
	out := setup.Folder{Path: listed, Roots: p.Roots(), Entries: []setup.Entry{}}
	if up := filepath.Dir(listed); up != listed {
		out.Parent = up
	}
	for _, it := range items {
		name := it.Name()
		// Links are left out: the install folder may not sit behind one.
		if !it.IsDir() || it.Type()&os.ModeSymlink != 0 || strings.HasPrefix(name, "$") {
			continue
		}
		out.Entries = append(out.Entries, setup.Entry{Name: name, Path: filepath.Join(listed, name)})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		return strings.ToLower(out.Entries[i].Name) < strings.ToLower(out.Entries[j].Name)
	})
	return out, nil
}

// makeDir creates one folder named by a single plain name inside parent.
func makeDir(parent, name string) (string, error) {
	switch {
	case !filepath.IsAbs(parent):
		return "", errors.New("the folder to create it in must be an absolute path")
	case strings.TrimSpace(name) == "", name == ".", name == "..":
		return "", errors.New("the new folder needs a name")
	case strings.ContainsAny(name, `/\<>:"|?*`):
		return "", errors.New(`a folder name cannot contain / \ < > : " | ? or *`)
	case strings.HasSuffix(name, ".") || strings.HasSuffix(name, " "):
		return "", errors.New("a folder name cannot end in a dot or a space")
	}
	path := filepath.Join(filepath.Clean(parent), name)
	if err := os.Mkdir(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}
