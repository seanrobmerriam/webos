package vfs

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

// ErrClosedFile is returned when operations are performed on a closed file.
var ErrClosedFile = errors.New("vfs: file is closed")

// ErrNotImplemented is returned when a filesystem operation is not supported.
var ErrNotImplemented = errors.New("vfs: operation not implemented")

// ErrPermissionDenied is returned when the operation is not permitted.
var ErrPermissionDenied = errors.New("vfs: permission denied")

// ErrInvalidSeek is returned for an invalid seek operation.
var ErrInvalidSeek = errors.New("vfs: invalid seek")

// vfsFile is the concrete implementation of the File interface.
type vfsFile struct {
	mu         sync.RWMutex
	closed     bool
	readOnly   bool
	path       string
	filesystem FileSystem
	data       []byte
	offset     int64
	append     bool
}

// newVFSFile creates a new file instance.
func newVFSFile(path string, data []byte, fs FileSystem, readOnly bool) *vfsFile {
	return &vfsFile{
		path:       path,
		filesystem: fs,
		data:       data,
		offset:     0,
		readOnly:   readOnly,
		append:     false,
	}
}

// Read implements the io.Reader interface.
func (f *vfsFile) Read(b []byte) (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return 0, ErrClosedFile
	}

	if len(b) == 0 {
		return 0, nil
	}

	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}

	n := copy(b, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

// Write implements the io.Writer interface.
func (f *vfsFile) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return 0, ErrClosedFile
	}

	if f.readOnly {
		return 0, ErrPermissionDenied
	}

	if f.append {
		f.offset = int64(len(f.data))
	}

	n := len(b)
	needed := f.offset + int64(n)

	if needed > int64(len(f.data)) {
		newData := make([]byte, needed)
		copy(newData, f.data)
		f.data = newData
	}

	copy(f.data[f.offset:], b)
	f.offset += int64(n)
	return n, nil
}

// Seek implements the io.Seeker interface.
func (f *vfsFile) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return 0, ErrClosedFile
	}

	var newOffset int64
	switch whence {
	case SEEK_SET:
		newOffset = offset
	case SEEK_CUR:
		newOffset = f.offset + offset
	case SEEK_END:
		newOffset = int64(len(f.data)) + offset
	default:
		return 0, ErrInvalidSeek
	}

	if newOffset < 0 {
		return 0, ErrInvalidSeek
	}

	f.offset = newOffset
	return f.offset, nil
}

// Close implements the io.Closer interface.
func (f *vfsFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	f.closed = true
	return nil
}

// Stat returns the FileInfo structure for the file.
func (f *vfsFile) Stat() (FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return FileInfo{}, ErrClosedFile
	}

	return FileInfo{
		Name:    f.path,
		Size:    int64(len(f.data)),
		Mode:    0666,
		ModTime: time.Now(),
		IsDir:   false,
	}, nil
}

// Truncate changes the size of the file.
func (f *vfsFile) Truncate(size int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return ErrClosedFile
	}

	if f.readOnly {
		return ErrPermissionDenied
	}

	if size < 0 {
		return ErrInvalidSeek
	}

	if size < int64(len(f.data)) {
		f.data = f.data[:size]
	} else {
		newData := make([]byte, size)
		copy(newData, f.data)
		f.data = newData
	}

	if f.offset > size {
		f.offset = size
	}

	return nil
}

// Sync commits the current contents of the file to stable storage.
// For in-memory filesystems, this is a no-op.
func (f *vfsFile) Sync() error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return ErrClosedFile
	}

	return nil
}

// fileData returns the underlying file data (for internal use).
func (f *vfsFile) fileData() []byte {
	return f.data
}

// setData sets the underlying file data (for internal use).
func (f *vfsFile) setData(data []byte) {
	f.data = data
}

// GetOffset returns the current file offset.
func (f *vfsFile) GetOffset() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.offset
}

// isClosed returns true if the file is closed.
func (f *vfsFile) isClosed() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.closed
}

// isReadOnly returns true if the file is read-only.
func (f *vfsFile) isReadOnly() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.readOnly
}

// DirEntry represents a directory entry.
type dirEntry struct {
	name  string
	info  FileInfo
	isDir bool
}

// Name returns the base name of the directory entry.
func (d *dirEntry) Name() string {
	return d.name
}

// IsDir returns true if the entry is a directory.
func (d *dirEntry) IsDir() bool {
	return d.isDir
}

// Type returns the type of the entry as a FileMode bit mask.
func (d *dirEntry) Type() os.FileMode {
	if d.isDir {
		return os.ModeDir
	}
	return 0
}

// Info returns a FileInfo structure describing the entry.
func (d *dirEntry) Info() (FileInfo, error) {
	return d.info, nil
}

// newDirEntry creates a new directory entry.
func newDirEntry(name string, isDir bool, info FileInfo) *dirEntry {
	return &dirEntry{
		name:  name,
		isDir: isDir,
		info:  info,
	}
}

// FileModeFromPerm converts permission bits to a FileMode.
func FileModeFromPerm(perm os.FileMode) os.FileMode {
	return perm & 0777
}
