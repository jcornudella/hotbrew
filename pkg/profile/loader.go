package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Info describes an available profile manifest.
type Info struct {
	Name        string
	SourceCount int
}

// Load returns the profile specified by name from ~/.config/hotbrew/profiles.
// Falls back to the embedded default profile if not found or invalid.
func Load(name string) *Profile {
	if name == "" {
		name = "default"
	}

	if p, err := loadProfileFile(filepath.Join(dir(), name+".yaml")); err == nil && len(p.Sources) > 0 {
		return p
	}

	// Allow users to drop multiple YAML files; merge them if no explicit name.
	if name == "default" {
		if merged := mergeAll(dir()); len(merged.Sources) > 0 {
			return merged
		}
	}

	return Default()
}

func loadProfileFile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func mergeAll(path string) *Profile {
	entries, err := os.ReadDir(path)
	if err != nil {
		return &Profile{}
	}
	merged := &Profile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		if p, err := loadProfileFile(filepath.Join(path, entry.Name())); err == nil {
			merged.Sources = append(merged.Sources, p.Sources...)
		}
	}
	return merged
}

func dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew", "profiles")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", "profiles")
}

// List returns the available profile manifests from disk.
func List() []Info {
	_ = EnsureDefaultProfile()
	entries, err := os.ReadDir(dir())
	if err != nil {
		return []Info{{Name: "default", SourceCount: len(Default().Sources)}}
	}
	infos := make([]Info, 0, len(entries))
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !(strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
			continue
		}
		trimmed := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		prof, err := loadProfileFile(filepath.Join(dir(), name))
		if err != nil {
			continue
		}
		infos = append(infos, Info{Name: trimmed, SourceCount: len(prof.Sources)})
	}
	if len(infos) == 0 {
		infos = append(infos, Info{Name: "default", SourceCount: len(Default().Sources)})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// Save writes the provided sources to the given profile name.
func Save(name string, sources []SourceSpec) error {
	if name == "" {
		return fmt.Errorf("profile name required")
	}
	path := filepath.Join(dir(), name+".yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(Profile{Sources: sources})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// EnsureDefaultProfile writes the default profile to disk if no user manifests exist.
func EnsureDefaultProfile() error {
	path := filepath.Join(dir(), "default.yaml")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(Default())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, fs.FileMode(0o644))
}
