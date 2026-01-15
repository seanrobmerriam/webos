package vfs

import (
	"io"
	"os"
	"time"
)

// FileSystem is the main interface for virtual file system operations.
// It provides methods for opening, creating, reading, writing, and managing
// files and directories across different storage backends.
//
// Implementations include MemFS (in-memory), DiskFS (disk-based), and
// OverlayFS (layered) filesystems.
type FileSystem interface {
	// Open opens a file for reading. The file must exist.
	Open(path string) (File, error)

	// OpenFile opens a file with the specified flags and permissions.
	// flags can be a combination of os.O_RDONLY, os.O_WRONLY, os.O_RDWR,
	// os.O_CREATE, os.O_EXCL, os.O_TRUNC, os.O_APPEND.
	OpenFile(path string, flags int, perm os.FileMode) (File, error)

	// Stat returns a FileInfo structure describing the file at path.
	// If the file is a symlink, the returned FileInfo describes the symlink.
	Stat(path string) (FileInfo, error)

	// Lstat returns a FileInfo structure describing the file at path.
	// If the file is a symlink, the returned FileInfo describes the symlink itself.
	Lstat(path string) (FileInfo, error)

	// Mkdir creates a new directory at path with the specified permissions.
	Mkdir(path string, perm os.FileMode) error

	// MkdirAll creates a directory at path and any necessary parents.
	MkdirAll(path string, perm os.FileMode) error

	// Remove removes the file or directory at path.
	// If path is a directory, it must be empty.
	Remove(path string) error

	// RemoveAll removes path and any children recursively.
	RemoveAll(path string) error

	// Rename renames (moves) oldpath to newpath.
	// If newpath already exists, behavior is implementation-dependent.
	Rename(oldpath, newpath string) error

	// ReadDir reads and returns all directory entries for path.
	ReadDir(path string) ([]DirEntry, error)

	// ReadFile reads the entire file at path.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to the file at path, creating it if necessary.
	// The file is truncated if it already exists.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// Create creates a new file at path with read-write permissions.
	Create(path string) (File, error)

	// Symlink creates a symbolic link at newpath pointing to target.
	Symlink(target, newpath string) error

	// Readlink returns the destination of the named symbolic link.
	Readlink(path string) (string, error)

	// Chmod changes the mode of the file at path.
	Chmod(path string, mode os.FileMode) error

	// Chown changes the owner and group of the file at path.
	Chown(path string, uid, gid int) error

	// Chtimes changes the access and modification times of the file at path.
	Chtimes(path string, atime, mtime time.Time) error
}

// File represents an open file and provides methods for reading,
// writing, and seeking within the file.
type File interface {
	// Read reads up to len(b) bytes from the file into b.
	// Returns the number of bytes read and any error encountered.
	io.Reader

	// Write writes len(b) bytes from b to the file.
	// Returns the number of bytes written and any error encountered.
	io.Writer

	// Seek sets the offset for the next Read or Write.
	// offset is interpreted relative to whence (0=start, 1=current, 2=end).
	// Returns the new offset and any error encountered.
	Seek(offset int64, whence int) (int64, error)

	// Close closes the file, making it unusable for further I/O.
	// Returns any error encountered during closing.
	Close() error

	// Stat returns a FileInfo structure describing the file.
	// The FileInfo describes the file at the time of the call.
	Stat() (FileInfo, error)

	// Truncate changes the size of the file.
	// If the file is a symbolic link, it changes the size of the link's target.
	Truncate(size int64) error

	// Sync commits the current contents of the file to stable storage.
	Sync() error
}

// FileInfo describes a file and is returned by Stat and Lstat.
type FileInfo struct {
	Name    string      // Base name of the file
	Size    int64       // Length in bytes for regular files
	Mode    os.FileMode // File mode bits
	ModTime time.Time   // Modification time
	IsDir   bool        // True if path is a directory
	Sys     interface{} // Underlying data source (can be nil)
}

// DirEntry is an entry read from a directory, similar to os.DirEntry.
type DirEntry interface {
	// Name returns the base name of the file or directory.
	Name() string

	// IsDir returns true if the entry is a directory.
	IsDir() bool

	// Type returns the type of the entry as a FileMode bit mask.
	Type() os.FileMode

	// Info returns a FileInfo structure describing the entry.
	Info() (FileInfo, error)
}

// Flags for OpenFile operations, matching os package constants.
const (
	O_RDONLY = os.O_RDONLY // Open file read-only.
	O_WRONLY = os.O_WRONLY // Open file write-only.
	O_RDWR   = os.O_RDWR   // Open file read-write.
	O_CREATE = os.O_CREATE // Create file if it does not exist.
	O_EXCL   = os.O_EXCL   // Used with O_CREATE: file must not exist.
	O_TRUNC  = os.O_TRUNC  // Truncate file to zero length if it exists.
	O_APPEND = os.O_APPEND // Append to the file on each write.
	O_SYNC   = os.O_SYNC   // Synchronous I/O on write.
)

// SeekWhence constants for Seek operations.
const (
	SEEK_SET = io.SeekStart   // Relative to start of file.
	SEEK_CUR = io.SeekCurrent // Relative to current position.
	SEEK_END = io.SeekEnd     // Relative to end of file.
)

// FileMode bit masks for file types and permissions.
const (
	ModeDir        = os.ModeDir        // Directory
	ModeSymlink    = os.ModeSymlink    // Symbolic link
	ModeNamedPipe  = os.ModeNamedPipe  // Named pipe (FIFO)
	ModeSocket     = os.ModeSocket     // UNIX domain socket
	ModeCharDevice = os.ModeCharDevice // Character device
	ModeTypeMask   = 0o170000          // Mask for file type bits (S_IFMT)

	ModeAppend    = os.ModeAppend    // Append-only
	ModeExclusive = os.ModeExclusive // Exclusive use
	ModeTemporary = os.ModeTemporary // Temporary file

	ModePerm = os.ModePerm // Permission bits mask

	AllPerm = 0777 // Default permission for new files
)
