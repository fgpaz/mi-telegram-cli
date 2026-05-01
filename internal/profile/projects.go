package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var ErrProjectBindingNotFound = errors.New("project binding not found")

type ProjectBinding struct {
	ProjectRoot  string    `json:"projectRoot"`
	ProfileID    string    `json:"profileId"`
	DisplayName  string    `json:"displayName,omitempty"`
	CreatedAtUTC time.Time `json:"createdAtUtc"`
	UpdatedAtUTC time.Time `json:"updatedAtUtc"`
}

type projectRegistryFile struct {
	Version  int              `json:"version"`
	Projects []ProjectBinding `json:"projects"`
}

func (s *Store) BindProject(root, profileID, displayName string) (ProjectBinding, error) {
	normalized, err := normalizeProjectRoot(root)
	if err != nil {
		return ProjectBinding{}, err
	}
	if strings.TrimSpace(profileID) == "" {
		return ProjectBinding{}, errors.New("profile id is required")
	}

	registry, err := s.readProjectRegistry()
	if err != nil {
		return ProjectBinding{}, err
	}

	now := s.now()
	binding := ProjectBinding{
		ProjectRoot:  normalized,
		ProfileID:    strings.TrimSpace(profileID),
		DisplayName:  strings.TrimSpace(displayName),
		CreatedAtUTC: now,
		UpdatedAtUTC: now,
	}

	found := false
	for i, existing := range registry.Projects {
		if sameProjectRoot(existing.ProjectRoot, normalized) {
			binding.CreatedAtUTC = existing.CreatedAtUTC
			registry.Projects[i] = binding
			found = true
			break
		}
	}
	if !found {
		registry.Projects = append(registry.Projects, binding)
	}
	sortProjectBindings(registry.Projects)

	if err := s.writeProjectRegistry(registry); err != nil {
		return ProjectBinding{}, err
	}
	return binding, nil
}

func (s *Store) ListProjectBindings() ([]ProjectBinding, error) {
	registry, err := s.readProjectRegistry()
	if err != nil {
		return nil, err
	}
	items := append([]ProjectBinding(nil), registry.Projects...)
	sortProjectBindings(items)
	return items, nil
}

func (s *Store) GetProjectBinding(root string) (ProjectBinding, error) {
	normalized, err := normalizeProjectRoot(root)
	if err != nil {
		return ProjectBinding{}, err
	}

	registry, err := s.readProjectRegistry()
	if err != nil {
		return ProjectBinding{}, err
	}
	for _, binding := range registry.Projects {
		if sameProjectRoot(binding.ProjectRoot, normalized) {
			return binding, nil
		}
	}
	return ProjectBinding{}, ErrProjectBindingNotFound
}

func (s *Store) ResolveProjectBinding(cwd string) (ProjectBinding, bool, error) {
	normalized, err := normalizeProjectRoot(cwd)
	if err != nil {
		return ProjectBinding{}, false, err
	}

	registry, err := s.readProjectRegistry()
	if err != nil {
		return ProjectBinding{}, false, err
	}

	var best ProjectBinding
	bestLen := -1
	for _, binding := range registry.Projects {
		root := normalizeProjectRootForCompare(binding.ProjectRoot)
		target := normalizeProjectRootForCompare(normalized)
		if root == "" || !isPathPrefix(root, target) {
			continue
		}
		if len(root) > bestLen {
			best = binding
			bestLen = len(root)
		}
	}
	if bestLen < 0 {
		return ProjectBinding{}, false, nil
	}
	return best, true, nil
}

func (s *Store) RemoveProjectBinding(root string) (ProjectBinding, error) {
	normalized, err := normalizeProjectRoot(root)
	if err != nil {
		return ProjectBinding{}, err
	}

	registry, err := s.readProjectRegistry()
	if err != nil {
		return ProjectBinding{}, err
	}

	for i, binding := range registry.Projects {
		if !sameProjectRoot(binding.ProjectRoot, normalized) {
			continue
		}
		registry.Projects = append(registry.Projects[:i], registry.Projects[i+1:]...)
		if err := s.writeProjectRegistry(registry); err != nil {
			return ProjectBinding{}, err
		}
		return binding, nil
	}
	return ProjectBinding{}, ErrProjectBindingNotFound
}

func (s *Store) readProjectRegistry() (projectRegistryFile, error) {
	path := s.projectsPath()
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return projectRegistryFile{Version: 1, Projects: []ProjectBinding{}}, nil
	}

	var registry projectRegistryFile
	if err := readJSON(path, &registry); err != nil {
		return projectRegistryFile{}, err
	}
	if registry.Version == 0 {
		registry.Version = 1
	}
	if registry.Projects == nil {
		registry.Projects = []ProjectBinding{}
	}
	return registry, nil
}

func (s *Store) writeProjectRegistry(registry projectRegistryFile) error {
	if registry.Version == 0 {
		registry.Version = 1
	}
	if err := os.MkdirAll(s.baseRoot, 0o700); err != nil {
		return err
	}
	return writeJSON(s.projectsPath(), registry)
}

func (s *Store) projectsPath() string {
	return filepath.Join(s.baseRoot, "projects.json")
}

func normalizeProjectRoot(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "", errors.New("project root is required")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		clean = resolved
	}
	return clean, nil
}

func normalizeProjectRootForCompare(root string) string {
	clean := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		clean = strings.ToLower(clean)
	}
	return clean
}

func sameProjectRoot(a, b string) bool {
	return normalizeProjectRootForCompare(a) == normalizeProjectRootForCompare(b)
}

func isPathPrefix(root, target string) bool {
	if root == target {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func sortProjectBindings(items []ProjectBinding) {
	sort.Slice(items, func(i, j int) bool {
		return normalizeProjectRootForCompare(items[i].ProjectRoot) < normalizeProjectRootForCompare(items[j].ProjectRoot)
	})
}
