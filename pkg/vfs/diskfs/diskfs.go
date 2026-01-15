// Package diskfs provides a disk-based filesystem implementation.
// It wraps the standard library's os functions to provide a VFS-compatible interface.
package diskfs

import (
	"os"
	"path/filepath"
	"time"

	vfs "webos/pkg/vfs"
)

// FS represents a disk-based filesystem.
type FS struct {
	root string
}

// New creates a new disk-based filesystem rooted at the given directory.
func New(root string) *FS {
	return &FS{root: filepath.Clean(root)}
}

// Open implements vfs.FileSystem.Open.
func (fs *FS) Open(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDONLY, 0)
}

// OpenFile implements vfs.FileSystem.OpenFile.
func (fs *FS) OpenFile(path string, flags int, perm os.FileMode) (vfs.File, error) {
	fullPath := fs.fullPath(path)
	file, err := os.OpenFile(fullPath, flags, perm)
	if err != nil {
		return nil, err
	}
	return &diskFile{file: file, path: path}, nil
}

// Stat implements vfs.FileSystem.Stat.
func (fs *FS) Stat(path string) (vfs.FileInfo, error) {
	fullPath := fs.fullPath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return vfs.FileInfo{}, err
	}
	return fileInfoFromOS(path, info), nil
}

// Lstat implements vfs.FileSystem.Lstat.
func (fs *FS) Lstat(path string) (vfs.FileInfo, error) {
	fullPath := fs.fullPath(path)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return vfs.FileInfo{}, err
	}
	return fileInfoFromOS(path, info), nil
}

// Mkdir implements vfs.FileSystem.Mkdir.
func (fs *FS) Mkdir(path string, perm os.FileMode) error {
	fullPath := fs.fullPath(path)
	return os.Mkdir(fullPath, perm)
}

// MkdirAll implements vfs.FileSystem.MkdirAll.
func (fs *FS) MkdirAll(path string, perm os.FileMode) error {
	fullPath := fs.fullPath(path)
	return os.MkdirAll(fullPath, perm)
}

// Remove implements vfs.FileSystem.Remove.
func (fs *FS) Remove(path string) error {
	fullPath := fs.fullPath(path)
	return os.Remove(fullPath)
}

// RemoveAll implements vfs.FileSystem.RemoveAll.
func (fs *FS) RemoveAll(path string) error {
	fullPath := fs.fullPath(path)
	return os.RemoveAll(fullPath)
}

// Rename implements vfs.FileSystem.Rename.
func (fs *FS) Rename(oldpath, newpath string) error {
	oldFull := fs.fullPath(oldpath)
	newFull := fs.fullPath(newpath)
	return os.Rename(oldFull, newFull)
}

// ReadDir implements vfs.FileSystem.ReadDir.
func (fs *FS) ReadDir(path string) ([]vfs.DirEntry, error) {
	fullPath := fs.fullPath(path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	result := make([]vfs.DirEntry, len(entries))
	for i, entry := range entries {
		result[i] = &diskDirEntry{
			name:  entry.Name(),
			entry: entry,
		}
	}
	return result, nil
}

// ReadFile implements vfs.FileSystem.ReadFile.
func (fs *FS) ReadFile(path string) ([]byte, error) {
	fullPath := fs.fullPath(path)

	// Check if it's a symlink and resolve if needed
	info, err := os.Lstat(fullPath)
	if err != nil {
		return nil, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// It's a symlink, resolve the target
		target, err := os.Readlink(fullPath)
		if err != nil {
			return nil, err
		}

		// If target is absolute, make it relative to VFS root
		if filepath.IsAbs(target) {
			// Convert absolute path to VFS relative path
			rel, err := filepath.Rel("/", target)
			if err != nil {
				return nil, err
			}
			target = "/" + rel
		}

		// Read the target file
		return fs.ReadFile(target)
	}

	return os.ReadFile(fullPath)
}

// WriteFile implements vfs.FileSystem.WriteFile.
func (fs *FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	fullPath := fs.fullPath(path)
	return os.WriteFile(fullPath, data, perm)
}

// Create implements vfs.FileSystem.Create.
func (fs *FS) Create(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDWR|vfs.O_CREATE|vfs.O_TRUNC, 0666)
}

// Symlink implements vfs.FileSystem.Symlink.
func (fs *FS) Symlink(target, newpath string) error {
	newFull := fs.fullPath(newpath)
	return os.Symlink(target, newFull)
}

// Readlink implements vfs.FileSystem.Readlink.
func (fs *FS) Readlink(path string) (string, error) {
	fullPath := fs.fullPath(path)
	return os.Readlink(fullPath)
}

// Chmod implements vfs.FileSystem.Chmod.
func (fs *FS) Chmod(path string, mode os.FileMode) error {
	fullPath := fs.fullPath(path)
	return os.Chmod(fullPath, mode)
}

// Chown implements vfs.FileSystem.Chown.
func (fs *FS) Chown(path string, uid, gid int) error {
	fullPath := fs.fullPath(path)
	return os.Chown(fullPath, uid, gid)
}

// Chtimes implements vfs.FileSystem.Chtimes.
func (fs *FS) Chtimes(path string, atime, mtime time.Time) error {
	fullPath := fs.fullPath(path)
	return os.Chtimes(fullPath, atime, mtime)
}

// fullPath converts a VFS path to an absolute filesystem path.
func (fs *FS) fullPath(path string) string {
	cleanPath := vfs.Clean(path)
	if cleanPath == "/" {
		return fs.root
	}
	// Remove leading slash for filepath.Join
	relPath := cleanPath[1:]
	return filepath.Join(fs.root, relPath)
}

// fileInfoFromOS converts an os.FileInfo to a vfs.FileInfo.
func fileInfoFromOS(path string, info os.FileInfo) vfs.FileInfo {
	return vfs.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Sys:     info.Sys(),
	}
}

// diskFile wraps an os.File to implement vfs.File.
type diskFile struct {
	file *os.File
	path string
}

func (f *diskFile) Read(b []byte) (int, error) {
	return f.file.Read(b)
}

func (f *diskFile) Write(b []byte) (int, error) {
	return f.file.Write(b)
}

func (f *diskFile) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

func (f *diskFile) Close() error {
	return f.file.Close()
}

func (f *diskFile) Stat() (vfs.FileInfo, error) {
	info, err := f.file.Stat()
	if err != nil {
		return vfs.FileInfo{}, err
	}
	return fileInfoFromOS(f.path, info), nil
}

func (f *diskFile) Truncate(size int64) error {
	return f.file.Truncate(size)
}

func (f *diskFile) Sync() error {
	return f.file.Sync()
}

// diskDirEntry wraps an os.DirEntry to implement vfs.DirEntry.
type diskDirEntry struct {
	name  string
	entry os.DirEntry
}

func (e *diskDirEntry) Name() string {
	return e.name
}

func (e *diskDirEntry) IsDir() bool {
	return e.entry.IsDir()
}

func (e *diskDirEntry) Type() os.FileMode {
	return e.entry.Type()
}

func (e *diskDirEntry) Info() (vfs.FileInfo, error) {
	info, err := e.entry.Info()
	if err != nil {
		return vfs.FileInfo{}, err
	}
	return fileInfoFromOS(e.name, info), nil
}
