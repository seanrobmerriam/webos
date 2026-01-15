package overlayfs

import (
	"os"
	"testing"

	"webos/pkg/vfs/memfs"
)

func TestNew(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	fs := New(upper, lower)
	if fs == nil {
		t.Fatal("New() returned nil")
	}
}

func TestReadFromLower(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower layer
	lower.WriteFile("/lower.txt", []byte("from lower"), 0644)

	// Create overlay
	fs := New(upper, lower)

	// Read should work from lower
	data, err := fs.ReadFile("/lower.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "from lower" {
		t.Errorf("got %q, expected %q", string(data), "from lower")
	}
}

func TestWriteToUpper(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create overlay
	fs := New(upper, lower)

	// Write to overlay
	err := fs.WriteFile("/new.txt", []byte("new content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Should be in upper layer
	data, err := upper.ReadFile("/new.txt")
	if err != nil {
		t.Fatalf("ReadFile from upper failed: %v", err)
	}

	if string(data) != "new content" {
		t.Errorf("got %q, expected %q", string(data), "new content")
	}

	// Should also be readable from overlay
	data, err = fs.ReadFile("/new.txt")
	if err != nil {
		t.Fatalf("ReadFile from overlay failed: %v", err)
	}

	if string(data) != "new content" {
		t.Errorf("got %q, expected %q", string(data), "new content")
	}
}

func TestCopyOnWrite(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower layer
	lower.WriteFile("/original.txt", []byte("original"), 0644)

	// Create overlay
	fs := New(upper, lower)

	// Open for writing (should trigger copy-up)
	file, err := fs.OpenFile("/original.txt", os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("OpenFile() failed: %v", err)
	}

	file.Write([]byte("modified"))
	file.Close()

	// Should now be in upper layer
	data, err := upper.ReadFile("/original.txt")
	if err != nil {
		t.Fatalf("ReadFile from upper failed: %v", err)
	}

	if string(data) != "modified" {
		t.Errorf("got %q, expected %q", string(data), "modified")
	}

	// Lower layer should be unchanged
	data, err = lower.ReadFile("/original.txt")
	if err != nil {
		t.Fatalf("ReadFile from lower failed: %v", err)
	}

	if string(data) != "original" {
		t.Errorf("lower should be unchanged, got %q", string(data))
	}
}

func TestRemoveHidesFromLower(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower layer
	lower.WriteFile("/file.txt", []byte("from lower"), 0644)

	// Create overlay
	fs := New(upper, lower)

	// Remove file
	err := fs.Remove("/file.txt")
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Should no longer be visible in overlay
	_, err = fs.Stat("/file.txt")
	if err == nil {
		t.Error("file should not be visible after remove")
	}
}

func TestMkdir(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	fs := New(upper, lower)

	// Create directory
	err := fs.Mkdir("/newdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir() failed: %v", err)
	}

	// Should be in upper layer
	info, err := upper.Stat("/newdir")
	if err != nil {
		t.Fatalf("Stat in upper failed: %v", err)
	}

	if !info.IsDir {
		t.Error("should be a directory")
	}
}

func TestReadDirMerges(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create files in both layers
	upper.WriteFile("/upper.txt", []byte("upper"), 0644)
	lower.WriteFile("/lower.txt", []byte("lower"), 0644)

	// Create overlay
	fs := New(upper, lower)

	// ReadDir should merge both
	entries, err := fs.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("got %d entries, expected 2", len(entries))
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["upper.txt"] {
		t.Error("upper.txt not found")
	}
	if !names["lower.txt"] {
		t.Error("lower.txt not found")
	}
}

func TestOverriddenFile(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create same file in both layers
	upper.WriteFile("/file.txt", []byte("upper"), 0644)
	lower.WriteFile("/file.txt", []byte("lower"), 0644)

	// Create overlay
	fs := New(upper, lower)

	// Should read from upper
	data, err := fs.ReadFile("/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "upper" {
		t.Errorf("got %q, expected upper content", string(data))
	}
}

func TestSymlink(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create symlink in upper
	upper.Symlink("/target", "/link")

	fs := New(upper, lower)

	// Readlink should work
	target, err := fs.Readlink("/link")
	if err != nil {
		t.Fatalf("Readlink() failed: %v", err)
	}

	if target != "/target" {
		t.Errorf("got %q, expected %q", target, "/target")
	}
}

func TestChmod(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower
	lower.WriteFile("/file.txt", []byte("content"), 0644)

	fs := New(upper, lower)

	// Chmod should copy up and modify
	err := fs.Chmod("/file.txt", 0600)
	if err != nil {
		t.Fatalf("Chmod() failed: %v", err)
	}

	// Check upper has the modified file
	info, err := upper.Stat("/file.txt")
	if err != nil {
		t.Fatalf("Stat in upper failed: %v", err)
	}

	if info.Mode != 0600 {
		t.Errorf("mode is %o, expected 0600", info.Mode)
	}
}

func TestRename(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower
	lower.WriteFile("/old.txt", []byte("content"), 0644)

	fs := New(upper, lower)

	// Rename should copy up and rename
	err := fs.Rename("/old.txt", "/new.txt")
	if err != nil {
		t.Fatalf("Rename() failed: %v", err)
	}

	// Should be in upper
	data, err := upper.ReadFile("/new.txt")
	if err != nil {
		t.Fatalf("ReadFile from upper failed: %v", err)
	}

	if string(data) != "content" {
		t.Errorf("got %q, expected %q", string(data), "content")
	}

	// Old should be gone
	_, err = fs.Stat("/old.txt")
	if err == nil {
		t.Error("old file should not exist")
	}
}

func TestMkdirAll(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	fs := New(upper, lower)

	// Create nested directories
	err := fs.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	// Should be in upper
	for _, path := range []string{"/a", "/a/b", "/a/b/c"} {
		info, err := upper.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%s) in upper failed: %v", path, err)
		}

		if !info.IsDir {
			t.Errorf("%s should be a directory", path)
		}
	}
}

func TestRemoveAll(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create nested structure in lower
	lower.MkdirAll("/dir", 0755)
	lower.WriteFile("/dir/file.txt", []byte("content"), 0644)

	fs := New(upper, lower)

	// RemoveAll
	err := fs.RemoveAll("/dir")
	if err != nil {
		t.Fatalf("RemoveAll() failed: %v", err)
	}

	// Should no longer be visible
	_, err = fs.Stat("/dir")
	if err == nil {
		t.Error("dir should not exist")
	}
}

func TestCreate(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	fs := New(upper, lower)

	// Create file
	file, err := fs.Create("/new.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	file.Write([]byte("created"))
	file.Close()

	// Should be in upper
	data, err := upper.ReadFile("/new.txt")
	if err != nil {
		t.Fatalf("ReadFile from upper failed: %v", err)
	}

	if string(data) != "created" {
		t.Errorf("got %q, expected %q", string(data), "created")
	}
}

func TestStat(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create file in lower
	lower.WriteFile("/file.txt", []byte("content"), 0644)

	fs := New(upper, lower)

	// Stat should work from lower
	info, err := fs.Stat("/file.txt")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.Name != "file.txt" {
		t.Errorf("name is %q, expected %q", info.Name, "file.txt")
	}
}

func TestLstat(t *testing.T) {
	upper := memfs.New()
	lower := memfs.New()

	// Create symlink in lower
	lower.Symlink("/target", "/link")

	fs := New(upper, lower)

	// Lstat should return symlink info
	info, err := fs.Lstat("/link")
	if err != nil {
		t.Fatalf("Lstat() failed: %v", err)
	}

	if info.Mode&os.ModeSymlink == 0 {
		t.Error("should be a symlink")
	}
}
