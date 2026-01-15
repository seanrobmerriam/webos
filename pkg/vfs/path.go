package vfs

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
)

// Common path-related errors.
var (
	ErrEmptyPath   = errors.New("vfs: empty path")
	ErrNotAbsolute = errors.New("vfs: path is not absolute")
	ErrInvalidPath = errors.New("vfs: invalid path")
	ErrSymlinkLoop = errors.New("vfs: symbolic link loop")
	ErrPathTooLong = errors.New("vfs: path too long")
)

// MaxPathLength is the maximum allowed path length.
const MaxPathLength = 4096

// Clean normalizes the path by removing unnecessary elements
// and handling relative paths. It is similar to filepath.Clean
// but operates on string paths without filesystem access.
func Clean(p string) string {
	if p == "" {
		return "/"
	}

	// Ensure we use forward slashes
	p = strings.ReplaceAll(p, "\\", "/")

	// Handle root
	if p[0] != '/' {
		p = "/" + p
	}

	// Split into components
	components := strings.Split(p, "/")
	var result []string

	for _, comp := range components {
		switch comp {
		case "", ".":
			// Skip empty and current directory
			continue
		case "..":
			// Go up one level, but not past root
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		default:
			result = append(result, comp)
		}
	}

	// Reconstruct path
	if len(result) == 0 {
		return "/"
	}

	return "/" + strings.Join(result, "/")
}

// IsAbs returns true if the path is absolute.
func IsAbs(p string) bool {
	return strings.HasPrefix(p, "/")
}

// Abs returns an absolute path. If the path is already absolute,
// it is returned unchanged. If the path is relative, it is resolved
// relative to root.
func Abs(p, root string) string {
	if IsAbs(p) {
		return Clean(p)
	}

	if root == "" || !IsAbs(root) {
		root = "/"
	}

	return Clean(filepath.Join(root, p))
}

// Dir returns all but the last element of the path.
func Dir(p string) string {
	p = Clean(p)

	lastSlash := strings.LastIndex(p, "/")
	if lastSlash == 0 {
		return "/"
	}

	return p[:lastSlash]
}

// Base returns the last element of the path.
func Base(p string) string {
	p = Clean(p)

	lastSlash := strings.LastIndex(p, "/")
	if lastSlash == len(p)-1 {
		return ""
	}

	return p[lastSlash+1:]
}

// Ext returns the file name extension of the path.
func Ext(p string) string {
	name := Base(p)
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
	}
	return ""
}

// Split splits the path into directory and base components.
func Split(p string) (dir, base string) {
	p = Clean(p)

	lastSlash := strings.LastIndex(p, "/")
	if lastSlash == 0 {
		return "/", p[1:]
	}
	if lastSlash == len(p)-1 {
		return p, ""
	}

	return p[:lastSlash], p[lastSlash+1:]
}

// Join joins any number of path elements into a single path.
func Join(elem ...string) string {
	return Clean(path.Join(elem...))
}

// Rel returns a relative path from base to target.
func Rel(base, target string) (string, error) {
	base = Clean(base)
	target = Clean(target)

	// If target is inside base, return relative path
	if strings.HasPrefix(target, base) {
		rest := strings.TrimPrefix(target, base)
		if rest == "" {
			return ".", nil
		}
		// Remove leading slash
		if rest[0] == '/' {
			rest = rest[1:]
		}
		return rest, nil
	}

	// Find common prefix
	baseParts := strings.Split(strings.TrimPrefix(base, "/"), "/")
	targetParts := strings.Split(strings.TrimPrefix(target, "/"), "/")

	minLen := len(baseParts)
	if len(targetParts) < minLen {
		minLen = len(targetParts)
	}

	common := 0
	for common < minLen && baseParts[common] == targetParts[common] {
		common++
	}

	// Build relative path
	var rel []string

	// Go up from base
	for i := common; i < len(baseParts); i++ {
		rel = append(rel, "..")
	}

	// Go down to target
	for i := common; i < len(targetParts); i++ {
		rel = append(rel, targetParts[i])
	}

	if len(rel) == 0 {
		return ".", nil
	}

	return strings.Join(rel, "/"), nil
}

// ValidatePath checks if the path is valid for use in the VFS.
func ValidatePath(p string) error {
	if p == "" {
		return ErrEmptyPath
	}

	if len(p) > MaxPathLength {
		return ErrPathTooLong
	}

	// Check for null bytes
	if strings.Contains(p, "\x00") {
		return ErrInvalidPath
	}

	return nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. The paths passed to walkFn are
// relative to root.
func Walk(fs FileSystem, root string, walkFn func(path string, info FileInfo, err error) error) error {
	root = Clean(root)

	info, err := fs.Stat(root)
	if err != nil {
		return walkFn(root, info, err)
	}

	return walk(fs, root, info, walkFn)
}

// walk is the internal implementation of Walk.
func walk(fs FileSystem, p string, info FileInfo, walkFn func(path string, info FileInfo, err error) error) error {
	err := walkFn(p, info, nil)
	if err != nil || !info.IsDir {
		return err
	}

	entries, err := fs.ReadDir(p)
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			if err := walkFn(entry.Name(), entryInfo, err); err != nil {
				return err
			}
		} else {
			if err := walk(fs, Join(p, entry.Name()), entryInfo, walkFn); err != nil {
				return err
			}
		}
	}

	return nil
}

// Glob returns the names of all files matching pattern or nil if there is no matching file.
func Glob(fs FileSystem, pattern string) ([]string, error) {
	// Simple glob implementation - supports * and ? wildcards
	parts := strings.Split(pattern, "/")
	return globParts(fs, parts, "")
}

// globParts is the recursive helper for Glob.
func globParts(fs FileSystem, parts []string, prefix string) ([]string, error) {
	if len(parts) == 0 {
		return []string{prefix}, nil
	}

	part := parts[0]
	rest := parts[1:]

	// List directory contents
	dir := prefix
	if dir != "" {
		dir = prefix + "/"
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matches []string

	for _, entry := range entries {
		name := entry.Name()
		if matchPattern(part, name) {
			newPrefix := dir + name
			if len(parts) > 1 {
				subMatches, err := globParts(fs, rest, newPrefix)
				if err != nil {
					return nil, err
				}
				matches = append(matches, subMatches...)
			} else {
				matches = append(matches, newPrefix)
			}
		}
	}

	return matches, nil
}

// matchPattern checks if a name matches a pattern with * and ? wildcards.
func matchPattern(pattern, name string) bool {
	pm, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return pm
}

// SymlinkTarget extracts the target from a symlink path.
// For VFS purposes, this just validates the path format.
func SymlinkTarget(target string) (string, error) {
	if err := ValidatePath(target); err != nil {
		return "", err
	}
	return target, nil
}
