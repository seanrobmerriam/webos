package diskfs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	vfs "webos/pkg/vfs"
)

func TestNew(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	if fs == nil {
		t.Fatal("New() returned nil")
	}

	if fs.root != tmpDir {
		t.Errorf("root is %q, expected %q", fs.root, tmpDir)
	}
}

func TestCreateAndOpen(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create a file
	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	file.Close()

	// Check file exists on disk
	fullPath := filepath.Join(tmpDir, "test.txt")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("file should exist on disk")
	}

	// Open the file
	file, err = fs.Open("/test.txt")
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.IsDir {
		t.Error("should not be a directory")
	}
}

func TestReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create and write
	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	testData := []byte("Hello, World!")
	n, err := file.Write(testData)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("wrote %d bytes, expected %d", n, len(testData))
	}

	file.Close()

	// Read back
	file, err = fs.Open("/test.txt")
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 100)
	n, err = file.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(buf[:n]) != "Hello, World!" {
		t.Errorf("read %q, expected %q", string(buf[:n]), "Hello, World!")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Write using fs
	err := fs.WriteFile("/test.txt", []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Read using fs
	data, err := fs.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "test content" {
		t.Errorf("read %q, expected %q", string(data), "test content")
	}
}

func TestMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create directory
	err := fs.Mkdir("/mydir", 0755)
	if err != nil {
		t.Fatalf("Mkdir() failed: %v", err)
	}

	// Check on disk
	fullPath := filepath.Join(tmpDir, "mydir")
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

func TestMkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create nested directories
	err := fs.MkdirAll("/a/b/c/d", 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	// Check all directories exist
	for _, path := range []string{"/a", "/a/b", "/a/b/c", "/a/b/c/d"} {
		info, err := fs.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) failed: %v", path, err)
		}

		if !info.IsDir {
			t.Errorf("%q should be a directory", path)
		}
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	fs.Create("/test.txt")

	// Remove
	err := fs.Remove("/test.txt")
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Check on disk
	fullPath := filepath.Join(tmpDir, "test.txt")
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file should not exist")
	}
}

func TestRename(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	fs.WriteFile("/old.txt", []byte("content"), 0644)

	// Rename
	err := fs.Rename("/old.txt", "/new.txt")
	if err != nil {
		t.Fatalf("Rename() failed: %v", err)
	}

	// Check old doesn't exist
	oldPath := filepath.Join(tmpDir, "old.txt")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old file should not exist")
	}

	// Check new exists
	newPath := filepath.Join(tmpDir, "new.txt")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file should exist: %v", err)
	}
}

func TestReadDir(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create files and dirs
	fs.Create("/file1.txt")
	fs.Create("/file2.txt")
	fs.Mkdir("/dir1", 0755)

	// ReadDir
	entries, err := fs.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("got %d entries, expected 3", len(entries))
	}

	// Check names
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["file1.txt"] {
		t.Error("file1.txt not found")
	}
	if !names["file2.txt"] {
		t.Error("file2.txt not found")
	}
	if !names["dir1"] {
		t.Error("dir1 not found")
	}
}

func TestSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create target file
	fs.WriteFile("/target.txt", []byte("target"), 0644)

	// Create symlink
	err := fs.Symlink("/target.txt", "/link.txt")
	if err != nil {
		t.Fatalf("Symlink() failed: %v", err)
	}

	// Read through symlink
	data, err := fs.ReadFile("/link.txt")
	if err != nil {
		t.Fatalf("ReadFile() through symlink failed: %v", err)
	}

	if string(data) != "target" {
		t.Errorf("read %q, expected %q", string(data), "target")
	}
}

func TestChmod(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	fs.Create("/test.txt")

	// Chmod
	err := fs.Chmod("/test.txt", 0600)
	if err != nil {
		t.Fatalf("Chmod() failed: %v", err)
	}

	// Check on disk
	fullPath := filepath.Join(tmpDir, "test.txt")
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("mode is %o, expected 0600", info.Mode().Perm())
	}
}

func TestChown(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	fs.Create("/test.txt")

	// Chown (may fail if not running as root)
	fs.Chown("/test.txt", 0, 0)
}

func TestChtimes(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	fs.Create("/test.txt")

	// Set times
	atime := time.Now().Add(-time.Hour)
	mtime := time.Now().Add(-2 * time.Hour)
	err := fs.Chtimes("/test.txt", atime, mtime)
	if err != nil {
		t.Fatalf("Chtimes() failed: %v", err)
	}

	// Check
	info, err := fs.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	// Allow some tolerance
	if info.ModTime.After(mtime.Add(time.Second)) {
		t.Error("mtime should be close to set time")
	}
}

func TestOpenFileFlags(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// O_CREATE
	file, err := fs.OpenFile("/new.txt", os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("OpenFile(O_CREATE) failed: %v", err)
	}
	file.Close()

	newPath := filepath.Join(tmpDir, "new.txt")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("file should exist")
	}

	// O_EXCL
	_, err = fs.OpenFile("/new.txt", os.O_CREATE|os.O_EXCL, 0644)
	if err == nil {
		t.Error("O_EXCL should fail")
	}

	// O_TRUNC
	fs.WriteFile("/test.txt", []byte("original"), 0644)
	file, err = fs.OpenFile("/test.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("OpenFile(O_TRUNC) failed: %v", err)
	}
	file.Write([]byte("truncated"))
	file.Close()

	data, _ := fs.ReadFile("/test.txt")
	if string(data) != "truncated" {
		t.Errorf("got %q, expected %q", string(data), "truncated")
	}
}

func TestSeek(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create and write
	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	file.Write([]byte("0123456789"))
	file.Close()

	// Open and seek
	file, err = fs.Open("/test.txt")
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer file.Close()

	pos, err := file.Seek(2, vfs.SEEK_SET)
	if err != nil {
		t.Fatalf("Seek() failed: %v", err)
	}

	if pos != 2 {
		t.Errorf("seek pos is %d, expected 2", pos)
	}

	buf := make([]byte, 3)
	n, err := file.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(buf[:n]) != "234" {
		t.Errorf("read %q, expected %q", string(buf[:n]), "234")
	}
}

func TestTruncate(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create file
	file, _ := fs.Create("/test.txt")
	file.Write([]byte("0123456789"))
	file.Close()

	// Truncate
	file, _ = fs.OpenFile("/test.txt", os.O_RDWR, 0644)
	err := file.Truncate(5)
	if err != nil {
		t.Fatalf("Truncate() failed: %v", err)
	}
	file.Close()

	// Check
	data, _ := fs.ReadFile("/test.txt")
	if len(data) != 5 {
		t.Errorf("file size is %d, expected 5", len(data))
	}
}

func TestSync(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	file.Write([]byte("data"))
	err = file.Sync()
	if err != nil {
		t.Fatalf("Sync() failed: %v", err)
	}

	file.Close()
}

func TestNestedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir)

	// Create nested structure
	fs.MkdirAll("/a/b/c", 0755)
	fs.WriteFile("/a/b/c/file.txt", []byte("nested"), 0644)

	// Read
	data, err := fs.ReadFile("/a/b/c/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "nested" {
		t.Errorf("got %q, expected %q", string(data), "nested")
	}
}
