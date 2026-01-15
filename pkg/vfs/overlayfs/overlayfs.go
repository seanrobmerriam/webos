// Package overlayfs provides a layered filesystem implementation.
// It combines a read-only lower filesystem with a read-write upper filesystem,
// using copy-on-write semantics for modifications.
package overlayfs

import (
	"errors"
	"os"
	"strings"
	"time"

	vfs "webos/pkg/vfs"
)

// ErrNotSupported is returned when an operation is not supported in overlay.
var ErrNotSupported = errors.New("overlayfs: operation not supported")

// FS represents a layered filesystem with lower (read-only) and upper (read-write) layers.
type FS struct {
	upper vfs.FileSystem
	lower vfs.FileSystem
}

// New creates a new overlay filesystem with the given upper and lower layers.
// The lower layer is typically read-only, and the upper layer receives all writes.
func New(upper, lower vfs.FileSystem) *FS {
	return &FS{
		upper: upper,
		lower: lower,
	}
}

// Open implements vfs.FileSystem.Open.
func (fs *FS) Open(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDONLY, 0)
}

// OpenFile implements vfs.FileSystem.OpenFile.
func (fs *FS) OpenFile(path string, flags int, perm os.FileMode) (vfs.File, error) {
	// Determine which layer to use
	existsUpper, err := fs.existsInUpper(path)
	if err != nil {
		return nil, err
	}

	// Check if file exists in upper layer
	if existsUpper {
		return fs.upper.OpenFile(path, flags, perm)
	}

	// Check if file exists in lower layer
	existsLower, err := fs.existsInLower(path)
	if err != nil {
		return nil, err
	}

	if !existsLower {
		// File doesn't exist in either layer
		if (flags & vfs.O_CREATE) == 0 {
			return nil, os.ErrNotExist
		}

		// Create in upper layer
		return fs.upper.OpenFile(path, flags|vfs.O_CREATE, perm)
	}

	// File exists in lower layer but not upper
	// For read-only access, open from lower
	if (flags & (vfs.O_WRONLY | vfs.O_RDWR)) == 0 {
		return fs.lower.Open(path)
	}

	// For write access, we need to copy up first (copy-on-write)
	if err := fs.copyUp(path); err != nil {
		return nil, err
	}

	return fs.upper.OpenFile(path, flags, perm)
}

// Stat implements vfs.FileSystem.Stat.
func (fs *FS) Stat(path string) (vfs.FileInfo, error) {
	// Check for whiteout first (files hidden by upper layer)
	if fs.isWhiteout(path) {
		return vfs.FileInfo{}, os.ErrNotExist
	}

	// Check upper layer first
	info, err := fs.upper.Stat(path)
	if err == nil {
		return info, nil
	}

	// Check lower layer
	return fs.lower.Stat(path)
}

// isWhiteout checks if a path has a whiteout in the upper layer.
func (fs *FS) isWhiteout(path string) bool {
	dir := vfs.Dir(path)
	base := vfs.Base(path)
	whiteoutName := ".wh." + base
	whiteoutPath := vfs.Join(dir, whiteoutName)

	// Use Lstat to check for whiteout file (don't follow symlinks)
	_, err := fs.upper.Lstat(whiteoutPath)
	return err == nil
}

// Lstat implements vfs.FileSystem.Lstat.
func (fs *FS) Lstat(path string) (vfs.FileInfo, error) {
	// Check upper layer first
	info, err := fs.upper.Lstat(path)
	if err == nil {
		return info, nil
	}

	// Check lower layer
	return fs.lower.Lstat(path)
}

// Mkdir implements vfs.FileSystem.Mkdir.
func (fs *FS) Mkdir(path string, perm os.FileMode) error {
	// Create in upper layer
	return fs.upper.Mkdir(path, perm)
}

// MkdirAll implements vfs.FileSystem.MkdirAll.
func (fs *FS) MkdirAll(path string, perm os.FileMode) error {
	// Create in upper layer
	return fs.upper.MkdirAll(path, perm)
}

// Remove implements vfs.FileSystem.Remove.
func (fs *FS) Remove(path string) error {
	// Try to remove from upper layer
	err := fs.upper.Remove(path)
	if err == nil {
		return nil
	}

	// If error is not "not exist", return it
	if err != os.ErrNotExist && err.Error() != "memfs: file not found" {
		return err
	}

	// Check if path exists in lower
	_, lowerErr := fs.lower.Stat(path)
	if lowerErr != nil {
		// File doesn't exist anywhere
		return os.ErrNotExist
	}

	// Create whiteout file in upper layer to hide lower file
	return fs.createWhiteout(path)
}

// RemoveAll implements vfs.FileSystem.RemoveAll.
func (fs *FS) RemoveAll(path string) error {
	// Try upper first
	fs.upper.RemoveAll(path)

	// Check if path exists in lower
	_, err := fs.lower.Stat(path)
	if err != nil {
		if err == os.ErrNotExist || err.Error() == "memfs: file not found" {
			return nil // Doesn't exist in lower
		}
		return err
	}

	// Create whiteout for the path
	return fs.createWhiteout(path)
}

// Rename implements vfs.FileSystem.Rename.
func (fs *FS) Rename(oldpath, newpath string) error {
	// Try to rename in upper layer first
	err := fs.upper.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}

	// Check if oldpath exists in lower
	_, lowerErr := fs.lower.Stat(oldpath)
	if lowerErr != nil {
		return err // Original error
	}

	// Check if newpath exists in lower and create whiteout
	if _, lowerErr := fs.lower.Stat(newpath); lowerErr == nil {
		fs.createWhiteout(newpath)
	}

	// Copy up oldpath and rename
	if err := fs.copyUp(oldpath); err != nil {
		return err
	}

	// Perform rename in upper
	err = fs.upper.Rename(oldpath, newpath)
	if err != nil {
		return err
	}

	// Create whiteout for oldpath to hide lower-layer file
	return fs.createWhiteout(oldpath)
}

// ReadDir implements vfs.FileSystem.ReadDir.
func (fs *FS) ReadDir(path string) ([]vfs.DirEntry, error) {
	// Get entries from upper layer
	upperEntries, upperErr := fs.upper.ReadDir(path)
	lowerEntries, lowerErr := fs.lower.ReadDir(path)

	// If both fail, return the error
	if upperErr != nil && lowerErr != nil {
		return nil, upperErr
	}

	// Merge entries
	entries := make(map[string]vfs.DirEntry)

	// Collect whiteout names from upper
	whiteouts := make(map[string]bool)

	// Add upper entries (skip whiteouts)
	if upperErr == nil {
		for _, e := range upperEntries {
			name := e.Name()
			// Skip whiteout files
			if strings.HasPrefix(name, ".wh.") {
				whiteouts[strings.TrimPrefix(name, ".wh.")] = true
				continue
			}
			entries[name] = e
		}
	}

	// Add lower entries (skip if overridden by whiteout or upper)
	if lowerErr == nil {
		for _, e := range lowerEntries {
			name := e.Name()
			if _, exists := entries[name]; exists {
				continue // Already in upper
			}
			if whiteouts[name] {
				continue // Hidden by whiteout
			}
			entries[name] = e
		}
	}

	// Convert to slice
	result := make([]vfs.DirEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, e)
	}

	return result, nil
}

// ReadFile implements vfs.FileSystem.ReadFile.
func (fs *FS) ReadFile(path string) ([]byte, error) {
	// Try upper first
	data, err := fs.upper.ReadFile(path)
	if err == nil {
		return data, nil
	}

	// Try lower
	return fs.lower.ReadFile(path)
}

// WriteFile implements vfs.FileSystem.WriteFile.
func (fs *FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return fs.upper.WriteFile(path, data, perm)
}

// Create implements vfs.FileSystem.Create.
func (fs *FS) Create(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDWR|vfs.O_CREATE|vfs.O_TRUNC, 0666)
}

// Symlink implements vfs.FileSystem.Symlink.
func (fs *FS) Symlink(target, newpath string) error {
	return fs.upper.Symlink(target, newpath)
}

// Readlink implements vfs.FileSystem.Readlink.
func (fs *FS) Readlink(path string) (string, error) {
	// Try upper first
	target, err := fs.upper.Readlink(path)
	if err == nil {
		return target, nil
	}

	// Try lower
	return fs.lower.Readlink(path)
}

// Chmod implements vfs.FileSystem.Chmod.
func (fs *FS) Chmod(path string, mode os.FileMode) error {
	// Check if exists in upper
	_, err := fs.upper.Stat(path)
	if err == nil {
		return fs.upper.Chmod(path, mode)
	}

	// Copy up and modify
	if err := fs.copyUp(path); err != nil {
		return err
	}

	return fs.upper.Chmod(path, mode)
}

// Chown implements vfs.FileSystem.Chown.
func (fs *FS) Chown(path string, uid, gid int) error {
	// Check if exists in upper
	_, err := fs.upper.Stat(path)
	if err == nil {
		return fs.upper.Chown(path, uid, gid)
	}

	// Copy up and modify
	if err := fs.copyUp(path); err != nil {
		return err
	}

	return fs.upper.Chown(path, uid, gid)
}

// Chtimes implements vfs.FileSystem.Chtimes.
func (fs *FS) Chtimes(path string, atime, mtime time.Time) error {
	// Check if exists in upper
	_, err := fs.upper.Stat(path)
	if err == nil {
		return fs.upper.Chtimes(path, atime, mtime)
	}

	// Copy up and modify
	if err := fs.copyUp(path); err != nil {
		return err
	}

	return fs.upper.Chtimes(path, atime, mtime)
}

// existsInUpper checks if a path exists in the upper layer.
func (fs *FS) existsInUpper(path string) (bool, error) {
	_, err := fs.upper.Stat(path)
	if err == nil {
		return true, nil
	}
	// Check for file not found (could be ErrNotExist or MemFS's ErrFileNotFound)
	if err == os.ErrNotExist || err.Error() == "memfs: file not found" {
		return false, nil
	}
	return false, err
}

// existsInLower checks if a path exists in the lower layer.
func (fs *FS) existsInLower(path string) (bool, error) {
	_, err := fs.lower.Stat(path)
	if err == nil {
		return true, nil
	}
	// Check for file not found
	if err == os.ErrNotExist || err.Error() == "memfs: file not found" {
		return false, nil
	}
	return false, err
}

// copyUp copies a file from the lower layer to the upper layer.
func (fs *FS) copyUp(path string) error {
	// Get file info from lower
	info, err := fs.lower.Stat(path)
	if err != nil {
		return err
	}

	// If it's a directory, create it in upper
	if info.IsDir {
		return fs.upper.MkdirAll(path, info.Mode)
	}

	// Create parent directories in upper layer
	dir := vfs.Dir(path)
	if dir != "/" && dir != "" {
		fs.upper.MkdirAll(dir, 0755)
	}

	// If it's a symlink, create it in upper
	if info.Mode&os.ModeSymlink != 0 {
		target, err := fs.lower.Readlink(path)
		if err != nil {
			return err
		}
		return fs.upper.Symlink(target, path)
	}

	// If it's a regular file, copy it
	data, err := fs.lower.ReadFile(path)
	if err != nil {
		return err
	}

	return fs.upper.WriteFile(path, data, info.Mode)
}

// createWhiteout creates a whiteout file to hide a lower layer entry.
func (fs *FS) createWhiteout(path string) error {
	// Whiteout files are named ".wh.<name>"
	// For a file at /path/to/file, the whiteout is at /path/to/.wh.file
	dir := vfs.Dir(path)
	base := vfs.Base(path)
	whiteoutName := ".wh." + base
	whiteoutPath := vfs.Join(dir, whiteoutName)

	// Create parent directories in upper layer
	if dir != "/" && dir != "" {
		fs.upper.MkdirAll(dir, 0755)
	}

	return fs.upper.Symlink("/dev/null", whiteoutPath)
}

// whiteoutPrefix is the prefix used for whiteout files.
const whiteoutPrefix = ".wh."
