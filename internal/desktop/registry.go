// Package desktop implements the workspace overlay across many `.flow/` trees.
// It is a lens: reads aggregate across all registered roots, writes always
// resolve back to exactly one project (adr-0001, adr-0003). Its only legitimate
// state is the registry of project roots, kept in its own inspectable file.
package desktop

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Root is one watched project in the registry.
type Root struct {
	Path      string `yaml:"path"`
	ProjectID string `yaml:"project_id"`
}

// Registry is the single piece of app-owned state: the list of project roots to
// watch. Losing it costs nothing but the list; the projects remain the truth.
type Registry struct {
	Version int    `yaml:"version"`
	Roots   []Root `yaml:"roots"`

	path string // where this was loaded from, for Save
}

// DefaultRegistryPath returns ~/.config/flow/registry.yaml (or the OS equivalent).
func DefaultRegistryPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "flow", "registry.yaml"), nil
}

// LoadRegistry reads and validates the registry at path. A missing file yields an
// empty registry (version 1), not an error, so a first run just has no roots yet.
func LoadRegistry(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{Version: 1, path: path}, nil
		}
		return nil, err
	}
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("registry %s: %w", path, err)
	}
	r.path = path
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// Validate enforces the registry invariants: version 1, absolute paths, and
// unique project ids (identity cannot collide across the plane).
func (r *Registry) Validate() error {
	if r.Version != 1 {
		return fmt.Errorf("registry: unsupported version %d (want 1)", r.Version)
	}
	seen := map[string]string{}
	for _, root := range r.Roots {
		if !filepath.IsAbs(root.Path) {
			return fmt.Errorf("registry: root path %q must be absolute", root.Path)
		}
		if root.ProjectID == "" {
			return fmt.Errorf("registry: root %q has no project_id", root.Path)
		}
		if prev, clash := seen[root.ProjectID]; clash {
			return fmt.Errorf("registry: project_id %q used by both %s and %s", root.ProjectID, prev, root.Path)
		}
		seen[root.ProjectID] = root.Path
	}
	return nil
}

// Add registers a root, keeping project ids unique. It does not persist; call Save.
func (r *Registry) Add(path, projectID string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, root := range r.Roots {
		if root.ProjectID == projectID {
			return fmt.Errorf("project_id %q already registered", projectID)
		}
		if root.Path == abs {
			return fmt.Errorf("path %q already registered", abs)
		}
	}
	r.Roots = append(r.Roots, Root{Path: abs, ProjectID: projectID})
	return nil
}

// Save writes the registry atomically (temp file plus rename) to its path.
func (r *Registry) Save() error {
	if r.path == "" {
		return fmt.Errorf("registry has no path")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".registry-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, r.path)
}
