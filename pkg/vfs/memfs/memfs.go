// Package memfs provides an in-memory filesystem implementation.
// It is useful for ephemeral storage, testing, or as a temporary cache.
package memfs

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	vfs "webos/pkg/vfs"
)

// ErrFileNotFound is returned when a file is not found.
var ErrFileNotFound = errors.New("memfs: file not found")

// ErrFileExists is returned when a file already exists.
var ErrFileExists = errors.New("memfs: file already exists")

// ErrNotDirectory is returned when a path is not a directory.
var ErrNotDirectory = errors.New("memfs: not a directory")

// ErrIsDirectory is returned when an operation requires a non-directory.
var ErrIsDirectory = errors.New("memfs: is a directory")

// ErrReadOnly is returned when writing to a read-only filesystem.
var ErrReadOnly = errors.New("memfs: read-only filesystem")

// memNode represents a node in the filesystem (file or directory).
type memNode struct {
	mu       sync.RWMutex
	data     []byte
	isDir    bool
	children map[string]*memNode
	mode     os.FileMode
	uid      int
	gid      int
	atime    time.Time
	mtime    time.Time
	ctime    time.Time
	symlink  string // Target if this is a symlink
}

// newMemNode creates a new memory node.
func newMemNode(isDir bool) *memNode {
	now := time.Now()
	return &memNode{
		isDir:    isDir,
		children: make(map[string]*memNode),
		mode:     0666 &^ umask,
		atime:    now,
		mtime:    now,
		ctime:    now,
	}
}

// FS represents an in-memory filesystem.
type FS struct {
	mu       sync.RWMutex
	root     *memNode
	readOnly bool
}

// umask is the default umask for new files.
var umask os.FileMode = 022

// New creates a new in-memory filesystem.
func New() *FS {
	fs := &FS{
		root: newMemNode(true),
	}
	fs.root.isDir = true
	return fs
}

// NewReadOnly creates a new read-only in-memory filesystem.
func NewReadOnly() *FS {
	return &FS{
		root:     newMemNode(true),
		readOnly: true,
	}
}

// nodeFromPath walks the filesystem and returns the node at the given path.
func (fs *FS) nodeFromPath(path string) (*memNode, error) {
	path = vfs.Clean(path)

	if path == "/" {
		return fs.root, nil
	}

	parts := splitPath(path)
	node := fs.root

	for _, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			return nil, ErrFileNotFound
		}

		node = child
	}

	return node, nil
}

// nodeFromPathWalk walks the filesystem, following symlinks.
// It returns the final node and the resolved path.
func (fs *FS) nodeFromPathWalk(path string) (*memNode, string, error) {
	path = vfs.Clean(path)

	if path == "/" {
		return fs.root, path, nil
	}

	parts := splitPath(path)
	node := fs.root
	var resolvedPath string

	for i, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			return nil, "", ErrFileNotFound
		}

		// Check for symlink
		if child.symlink != "" {
			// Resolve symlink
			target := child.symlink
			if !vfs.IsAbs(target) {
				// Relative symlink - resolve relative to parent
				parentPath := "/" + joinParts(parts[:i])
				target = vfs.Clean(vfs.Join(parentPath, target))
			}
			child, err := fs.nodeFromPath(target)
			if err != nil {
				return nil, "", err
			}
			if child.isDir && i < len(parts)-1 {
				// Continue resolving remaining path components
				remaining := "/" + joinParts(parts[i+1:])
				targetChild, _, err := fs.nodeFromPathWalk(vfs.Join(target, remaining))
				return targetChild, vfs.Join(target, remaining), err
			}
			return child, vfs.Join(target, joinParts(parts[i+1:])), nil
		}

		node = child
		resolvedPath = "/" + joinParts(parts[:i+1])
	}

	return node, resolvedPath, nil
}

// Open implements vfs.FileSystem.Open.
func (fs *FS) Open(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDONLY, 0)
}

// OpenFile implements vfs.FileSystem.OpenFile.
func (fs *FS) OpenFile(path string, flags int, perm os.FileMode) (vfs.File, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return nil, err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly && (flags&vfs.O_CREATE) != 0 {
		return nil, ErrReadOnly
	}

	path = vfs.Clean(path)

	// Check if file exists
	node, err := fs.nodeFromPath(path)
	if err == nil {
		// File exists - check O_EXCL
		if (flags & vfs.O_EXCL) != 0 {
			return nil, ErrFileExists
		}

		// Follow symlinks
		for node.symlink != "" {
			target := node.symlink
			if !vfs.IsAbs(target) {
				dir := vfs.Dir(path)
				target = vfs.Clean(vfs.Join(dir, target))
			}
			node, err = fs.nodeFromPath(target)
			if err != nil {
				return nil, err
			}
			path = target
		}

		if node.isDir {
			return nil, ErrIsDirectory
		}

		readOnly := (flags & (vfs.O_WRONLY | vfs.O_RDWR)) == 0

		// Handle O_TRUNC
		if (flags&vfs.O_TRUNC) != 0 && !readOnly {
			node.data = nil
			node.mtime = time.Now()
		}

		// Handle O_APPEND
		append := (flags & vfs.O_APPEND) != 0

		file := newVFSFileFromNode(path, node, readOnly, append)
		return file, nil
	}

	// File doesn't exist - check if we should create it
	if (flags & vfs.O_CREATE) == 0 {
		return nil, ErrFileNotFound
	}

	// Create new file
	dirPath := vfs.Dir(path)
	dirNode, err := fs.nodeFromPath(dirPath)
	if err != nil {
		return nil, err
	}

	if !dirNode.isDir {
		return nil, ErrNotDirectory
	}

	base := vfs.Base(path)
	if _, exists := dirNode.children[base]; exists {
		return nil, ErrFileExists
	}

	newNode := newMemNode(false)
	newNode.mode = perm & 0777
	dirNode.children[base] = newNode

	readOnly := (flags & (vfs.O_WRONLY | vfs.O_RDWR)) == 0
	append := (flags & vfs.O_APPEND) != 0

	file := newVFSFileFromNode(path, newNode, readOnly, append)
	return file, nil
}

// Stat implements vfs.FileSystem.Stat.
func (fs *FS) Stat(path string) (vfs.FileInfo, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return vfs.FileInfo{}, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return vfs.FileInfo{}, err
	}

	// Follow symlinks
	for node.symlink != "" {
		target := node.symlink
		if !vfs.IsAbs(target) {
			dir := vfs.Dir(path)
			target = vfs.Clean(vfs.Join(dir, target))
		}
		node, err = fs.nodeFromPath(target)
		if err != nil {
			return vfs.FileInfo{}, err
		}
		path = target
	}

	return fs.nodeToFileInfo(path, node), nil
}

// Lstat implements vfs.FileSystem.Lstat.
func (fs *FS) Lstat(path string) (vfs.FileInfo, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return vfs.FileInfo{}, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path = vfs.Clean(path)

	if path == "/" {
		return fs.nodeToFileInfo("/", fs.root), nil
	}

	parts := splitPath(path)
	node := fs.root

	for _, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			return vfs.FileInfo{}, ErrFileNotFound
		}

		node = child
	}

	return fs.nodeToFileInfo(path, node), nil
}

// Mkdir implements vfs.FileSystem.Mkdir.
func (fs *FS) Mkdir(path string, perm os.FileMode) error {
	return fs.mkdir(path, perm, false)
}

// MkdirAll implements vfs.FileSystem.MkdirAll.
func (fs *FS) MkdirAll(path string, perm os.FileMode) error {
	return fs.mkdir(path, perm, true)
}

func (fs *FS) mkdir(path string, perm os.FileMode, createParents bool) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	path = vfs.Clean(path)

	if path == "/" {
		return nil
	}

	// Check if already exists
	node, err := fs.nodeFromPath(path)
	if err == nil {
		if node.isDir {
			return nil
		}
		return ErrNotDirectory
	}

	// Create directory - walk the path and create missing parts
	parts := splitPath(path)
	current := fs.root

	for i, part := range parts {
		// Check if this part exists
		current.mu.RLock()
		child, ok := current.children[part]
		current.mu.RUnlock()

		if ok {
			// Part exists
			if !child.isDir {
				return ErrNotDirectory
			}
			current = child
			continue
		}

		// Part doesn't exist - need to create it
		// If we're creating a specific directory (not all), we can only create
		// if we're at the last part OR createParents is true
		if !createParents && i < len(parts)-1 {
			return ErrFileNotFound
		}

		newDir := newMemNode(true)
		newDir.mode = perm & 0777

		current.mu.Lock()
		current.children[part] = newDir
		current.mu.Unlock()

		current = newDir
	}

	return nil
}

// Remove implements vfs.FileSystem.Remove.
func (fs *FS) Remove(path string) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	path = vfs.Clean(path)

	if path == "/" {
		return errors.New("memfs: cannot remove root")
	}

	parts := splitPath(path)
	parent := fs.root
	childName := parts[len(parts)-1]

	for _, part := range parts[:len(parts)-1] {
		parent.mu.RLock()
		child, ok := parent.children[part]
		parent.mu.RUnlock()

		if !ok {
			return ErrFileNotFound
		}

		parent = child
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	child, ok := parent.children[childName]
	if !ok {
		return ErrFileNotFound
	}

	if child.isDir && len(child.children) > 0 {
		return errors.New("memfs: directory not empty")
	}

	delete(parent.children, childName)
	return nil
}

// RemoveAll implements vfs.FileSystem.RemoveAll.
func (fs *FS) RemoveAll(path string) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	path = vfs.Clean(path)

	if path == "/" {
		// Remove everything except root
		fs.root = newMemNode(true)
		return nil
	}

	parts := splitPath(path)
	parent := fs.root
	childName := parts[len(parts)-1]

	for _, part := range parts[:len(parts)-1] {
		parent.mu.RLock()
		child, ok := parent.children[part]
		parent.mu.RUnlock()

		if !ok {
			return ErrFileNotFound
		}

		parent = child
	}

	delete(parent.children, childName)
	return nil
}

// Rename implements vfs.FileSystem.Rename.
func (fs *FS) Rename(oldpath, newpath string) error {
	if err := vfs.ValidatePath(oldpath); err != nil {
		return err
	}
	if err := vfs.ValidatePath(newpath); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	oldpath = vfs.Clean(oldpath)
	newpath = vfs.Clean(newpath)

	if oldpath == newpath {
		return nil
	}

	// Find source node
	srcNode, err := fs.nodeFromPath(oldpath)
	if err != nil {
		return err
	}

	// Find destination parent
	newDirPath := vfs.Dir(newpath)
	destDir, err := fs.nodeFromPath(newDirPath)
	if err != nil {
		return err
	}

	if !destDir.isDir {
		return ErrNotDirectory
	}

	newName := vfs.Base(newpath)
	if newName == "" {
		return errors.New("memfs: invalid new path")
	}

	// Remove from old location
	oldParts := splitPath(oldpath)
	oldParent := fs.root
	for _, part := range oldParts[:len(oldParts)-1] {
		oldParent = oldParent.children[part]
	}
	delete(oldParent.children, oldParts[len(oldParts)-1])

	// Add to new location
	destDir.children[newName] = srcNode

	return nil
}

// ReadDir implements vfs.FileSystem.ReadDir.
func (fs *FS) ReadDir(path string) ([]vfs.DirEntry, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return nil, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return nil, err
	}

	if !node.isDir {
		return nil, ErrNotDirectory
	}

	entries := make([]vfs.DirEntry, 0, len(node.children))
	for name, child := range node.children {
		entries = append(entries, vfsDirEntry{
			name:  name,
			isDir: child.isDir,
			info:  fs.nodeToFileInfo(vfs.Join(path, name), child),
		})
	}

	return entries, nil
}

// ReadFile implements vfs.FileSystem.ReadFile.
func (fs *FS) ReadFile(path string) ([]byte, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return nil, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return nil, err
	}

	// Follow symlinks
	for node.symlink != "" {
		target := node.symlink
		if !vfs.IsAbs(target) {
			// Relative symlink - resolve relative to parent directory
			dir := vfs.Dir(path)
			target = vfs.Clean(vfs.Join(dir, target))
		}
		node, err = fs.nodeFromPath(target)
		if err != nil {
			return nil, err
		}
		path = target
	}

	if node.isDir {
		return nil, ErrIsDirectory
	}

	result := make([]byte, len(node.data))
	copy(result, node.data)
	return result, nil
}

// WriteFile implements vfs.FileSystem.WriteFile.
func (fs *FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	// Check if file exists
	node, err := fs.nodeFromPath(path)
	if err == nil {
		if node.isDir {
			return ErrIsDirectory
		}
		node.data = data
		node.mtime = time.Now()
		return nil
	}

	// Create new file
	dirPath := vfs.Dir(path)
	dirNode, err := fs.nodeFromPath(dirPath)
	if err != nil {
		return err
	}

	if !dirNode.isDir {
		return ErrNotDirectory
	}

	newNode := newMemNode(false)
	newNode.data = data
	newNode.mode = perm & 0777

	dirNode.children[vfs.Base(path)] = newNode
	return nil
}

// Create implements vfs.FileSystem.Create.
func (fs *FS) Create(path string) (vfs.File, error) {
	return fs.OpenFile(path, vfs.O_RDWR|vfs.O_CREATE|vfs.O_TRUNC, 0666)
}

// Symlink implements vfs.FileSystem.Symlink.
func (fs *FS) Symlink(target, newpath string) error {
	if err := vfs.ValidatePath(newpath); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	newpath = vfs.Clean(newpath)

	// Check if newpath already exists
	_, err := fs.nodeFromPath(newpath)
	if err == nil {
		return ErrFileExists
	}

	// Create symlink
	dirPath := vfs.Dir(newpath)
	dirNode, err := fs.nodeFromPath(dirPath)
	if err != nil {
		return err
	}

	if !dirNode.isDir {
		return ErrNotDirectory
	}

	link := newMemNode(false)
	link.symlink = target
	link.mode = 0777 | os.ModeSymlink

	dirNode.children[vfs.Base(newpath)] = link
	return nil
}

// Readlink implements vfs.FileSystem.Readlink.
func (fs *FS) Readlink(path string) (string, error) {
	if err := vfs.ValidatePath(path); err != nil {
		return "", err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return "", err
	}

	if node.symlink == "" {
		return "", errors.New("memfs: not a symbolic link")
	}

	return node.symlink, nil
}

// Chmod implements vfs.FileSystem.Chmod.
func (fs *FS) Chmod(path string, mode os.FileMode) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return err
	}

	node.mode = mode & 0777
	return nil
}

// Chown implements vfs.FileSystem.Chown.
func (fs *FS) Chown(path string, uid, gid int) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.readOnly {
		return ErrReadOnly
	}

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return err
	}

	node.uid = uid
	node.gid = gid
	return nil
}

// Chtimes implements vfs.FileSystem.Chtimes.
func (fs *FS) Chtimes(path string, atime, mtime time.Time) error {
	if err := vfs.ValidatePath(path); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	node, err := fs.nodeFromPath(path)
	if err != nil {
		return err
	}

	node.atime = atime
	node.mtime = mtime
	return nil
}

// nodeToFileInfo converts a memNode to a FileInfo.
func (fs *FS) nodeToFileInfo(path string, node *memNode) vfs.FileInfo {
	return vfs.FileInfo{
		Name:    vfs.Base(path),
		Size:    int64(len(node.data)),
		Mode:    node.mode,
		ModTime: node.mtime,
		IsDir:   node.isDir,
		Sys:     node,
	}
}

// newVFSFileFromNode creates a vfs.File from a memNode.
func newVFSFileFromNode(path string, node *memNode, readOnly, append bool) vfs.File {
	return &memFile{
		path:     path,
		node:     node,
		readOnly: readOnly,
		append:   append,
	}
}

// memFile is a file backed by a memNode.
type memFile struct {
	path     string
	node     *memNode
	readOnly bool
	append   bool
	offset   int64
}

func (f *memFile) Read(b []byte) (int, error) {
	f.node.mu.RLock()
	defer f.node.mu.RUnlock()

	if len(b) == 0 {
		return 0, nil
	}

	if f.offset >= int64(len(f.node.data)) {
		return 0, nil
	}

	n := copy(b, f.node.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *memFile) Write(b []byte) (int, error) {
	if f.readOnly {
		return 0, vfs.ErrPermissionDenied
	}

	f.node.mu.Lock()
	defer f.node.mu.Unlock()

	if f.append {
		f.offset = int64(len(f.node.data))
	}

	n := len(b)
	needed := f.offset + int64(n)

	if needed > int64(len(f.node.data)) {
		newData := make([]byte, needed)
		copy(newData, f.node.data)
		f.node.data = newData
	}

	copy(f.node.data[f.offset:], b)
	f.offset += int64(n)
	f.node.mtime = time.Now()
	return n, nil
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case vfs.SEEK_SET:
		newOffset = offset
	case vfs.SEEK_CUR:
		newOffset = f.offset + offset
	case vfs.SEEK_END:
		newOffset = int64(len(f.node.data)) + offset
	default:
		return 0, vfs.ErrInvalidSeek
	}

	if newOffset < 0 {
		return 0, vfs.ErrInvalidSeek
	}

	f.offset = newOffset
	return f.offset, nil
}

func (f *memFile) Close() error {
	return nil
}

func (f *memFile) Stat() (vfs.FileInfo, error) {
	return vfs.FileInfo{
		Name:    vfs.Base(f.path),
		Size:    int64(len(f.node.data)),
		Mode:    f.node.mode,
		ModTime: f.node.mtime,
		IsDir:   false,
	}, nil
}

func (f *memFile) Truncate(size int64) error {
	if f.readOnly {
		return vfs.ErrPermissionDenied
	}

	f.node.mu.Lock()
	defer f.node.mu.Unlock()

	if size < 0 {
		return vfs.ErrInvalidSeek
	}

	if size < int64(len(f.node.data)) {
		f.node.data = f.node.data[:size]
	} else {
		newData := make([]byte, size)
		copy(newData, f.node.data)
		f.node.data = newData
	}

	if f.offset > size {
		f.offset = size
	}

	return nil
}

func (f *memFile) Sync() error {
	return nil
}

// splitPath splits a path into components.
func splitPath(p string) []string {
	p = vfs.Clean(p)
	if p == "/" {
		return nil
	}
	parts := strings.Split(p[1:], "/")
	return parts
}

// joinParts joins path components.
func joinParts(parts []string) string {
	return "/" + strings.Join(parts, "/")
}

// vfsDirEntry implements vfs.DirEntry.
type vfsDirEntry struct {
	name  string
	isDir bool
	info  vfs.FileInfo
}

func (e vfsDirEntry) Name() string {
	return e.name
}

func (e vfsDirEntry) IsDir() bool {
	return e.isDir
}

func (e vfsDirEntry) Type() os.FileMode {
	if e.isDir {
		return os.ModeDir
	}
	return 0
}

func (e vfsDirEntry) Info() (vfs.FileInfo, error) {
	return e.info, nil
}
