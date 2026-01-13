package security

import (
	"errors"
	"strings"
)

// Common unveil permission errors
var (
	// ErrInvalidPermissions indicates the permissions string is invalid.
	ErrInvalidPermissions = errors.New("invalid unveil permissions")
)

// ValidPermissions contains the set of valid permission strings.
var ValidPermissions = map[string]bool{
	"r":   true,
	"w":   true,
	"x":   true,
	"rw":  true,
	"rx":  true,
	"wx":  true,
	"rwx": true,
}

// UnveilPath represents a filesystem path with associated permissions.
// This is inspired by OpenBSD's unveil system which restricts filesystem
// access to only explicitly specified paths.
type UnveilPath struct {
	// Path is the filesystem path that can be accessed.
	Path string
	// Permissions is a string specifying the allowed operations:
	// "r" for read-only, "w" for write-only, "x" for execute-only,
	// or combinations like "rw", "rx", "wx", "rwx".
	Permissions string
}

// NewUnveilPath creates a new UnveilPath with the specified path and permissions.
// Returns an error if the permissions string is invalid.
func NewUnveilPath(path, permissions string) (*UnveilPath, error) {
	if !ValidPermissions[permissions] {
		return nil, ErrInvalidPermissions
	}
	return &UnveilPath{
		Path:        path,
		Permissions: permissions,
	}, nil
}

// CanRead returns true if the permissions include read access.
func (u UnveilPath) CanRead() bool {
	return strings.Contains(u.Permissions, "r")
}

// CanWrite returns true if the permissions include write access.
func (u UnveilPath) CanWrite() bool {
	return strings.Contains(u.Permissions, "w")
}

// CanExecute returns true if the permissions include execute access.
func (u UnveilPath) CanExecute() bool {
	return strings.Contains(u.Permissions, "x")
}

// ValidatePermissions checks if the permissions string is valid.
func ValidatePermissions(permissions string) bool {
	return ValidPermissions[permissions]
}

// UnveilRegistry manages unveil paths for different components.
type UnveilRegistry struct {
	paths map[string][]UnveilPath
}

// NewUnveilRegistry creates a new UnveilRegistry.
func NewUnveilRegistry() *UnveilRegistry {
	return &UnveilRegistry{
		paths: make(map[string][]UnveilPath),
	}
}

// AddPath adds a new unveil path for the specified component.
func (r *UnveilRegistry) AddPath(component string, path UnveilPath) {
	r.paths[component] = append(r.paths[component], path)
}

// GetPaths returns all unveil paths for the specified component.
func (r *UnveilRegistry) GetPaths(component string) []UnveilPath {
	return r.paths[component]
}

// HasPath checks if the component has access to the specified path.
func (r *UnveilRegistry) HasPath(component, path string) bool {
	for _, p := range r.paths[component] {
		if p.Path == path {
			return true
		}
	}
	return false
}

// CanAccessPath checks if the component has the specified access to the path.
func (r *UnveilRegistry) CanAccessPath(component, path string, wantRead, wantWrite, wantExecute bool) bool {
	for _, p := range r.paths[component] {
		if p.Path == path {
			if wantRead && !p.CanRead() {
				continue
			}
			if wantWrite && !p.CanWrite() {
				continue
			}
			if wantExecute && !p.CanExecute() {
				continue
			}
			return true
		}
	}
	return false
}

// RemovePaths removes all unveil paths for the specified component.
func (r *UnveilRegistry) RemovePaths(component string) {
	delete(r.paths, component)
}
