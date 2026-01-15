package memfs

import (
	"os"
	"testing"
	"time"

	vfs "webos/pkg/vfs"
)

func TestNew(t *testing.T) {
	fs := New()
	if fs == nil {
		t.Fatal("New() returned nil")
	}

	// Check root directory exists
	info, err := fs.Stat("/")
	if err != nil {
		t.Fatalf("Stat(\"/\") failed: %v", err)
	}

	if !info.IsDir {
		t.Error("root should be a directory")
	}
}

func TestCreateAndOpen(t *testing.T) {
	fs := New()

	// Create a file
	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	file.Close()

	// Open the file
	file, err = fs.Open("/test.txt")
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer file.Close()

	// Check it's not a directory
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.IsDir {
		t.Error("file should not be a directory")
	}
}

func TestReadWrite(t *testing.T) {
	fs := New()

	// Create and write to file
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
		t.Errorf("Write() wrote %d bytes, expected %d", n, len(testData))
	}

	file.Close()

	// Read the file
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
		t.Errorf("Read() returned %q, expected %q", string(buf[:n]), "Hello, World!")
	}
}

func TestSeek(t *testing.T) {
	fs := New()

	// Create a file
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

	// Seek to position 2
	pos, err := file.Seek(2, vfs.SEEK_SET)
	if err != nil {
		t.Fatalf("Seek() failed: %v", err)
	}

	if pos != 2 {
		t.Errorf("Seek() returned %d, expected 2", pos)
	}

	// Read from position 2
	buf := make([]byte, 3)
	n, err := file.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if string(buf[:n]) != "234" {
		t.Errorf("Read() returned %q, expected %q", string(buf[:n]), "234")
	}
}

func TestMkdir(t *testing.T) {
	fs := New()

	// Create a directory
	err := fs.Mkdir("/mydir", 0755)
	if err != nil {
		t.Fatalf("Mkdir() failed: %v", err)
	}

	// Check it exists
	info, err := fs.Stat("/mydir")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if !info.IsDir {
		t.Error("/mydir should be a directory")
	}
}

func TestMkdirAll(t *testing.T) {
	fs := New()

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
	fs := New()

	// Create a file
	fs.Create("/test.txt")

	// Remove it
	err := fs.Remove("/test.txt")
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Check it no longer exists
	_, err = fs.Stat("/test.txt")
	if err == nil {
		t.Error("file should not exist after Remove()")
	}
}

func TestRename(t *testing.T) {
	fs := New()

	// Create a file
	fs.WriteFile("/old.txt", []byte("content"), 0644)

	// Rename it
	err := fs.Rename("/old.txt", "/new.txt")
	if err != nil {
		t.Fatalf("Rename() failed: %v", err)
	}

	// Check old name doesn't exist
	_, err = fs.Stat("/old.txt")
	if err == nil {
		t.Error("old name should not exist")
	}

	// Check new name exists
	info, err := fs.Stat("/new.txt")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.Size != 7 {
		t.Errorf("file size is %d, expected 7", info.Size)
	}
}

func TestReadDir(t *testing.T) {
	fs := New()

	// Create some files and directories
	fs.Create("/file1.txt")
	fs.Create("/file2.txt")
	fs.Mkdir("/dir1", 0755)

	// Read root directory
	entries, err := fs.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("ReadDir() returned %d entries, expected 3", len(entries))
	}

	// Check entry names
	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
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
	fs := New()

	// Create a file
	fs.WriteFile("/target.txt", []byte("target content"), 0644)

	// Create a symlink
	err := fs.Symlink("/target.txt", "/link.txt")
	if err != nil {
		t.Fatalf("Symlink() failed: %v", err)
	}

	// Read through symlink
	data, err := fs.ReadFile("/link.txt")
	if err != nil {
		t.Fatalf("ReadFile() through symlink failed: %v", err)
	}

	if string(data) != "target content" {
		t.Errorf("ReadFile() returned %q, expected %q", string(data), "target content")
	}
}

func TestChmod(t *testing.T) {
	fs := New()

	// Create a file
	fs.Create("/test.txt")

	// Change mode
	err := fs.Chmod("/test.txt", 0600)
	if err != nil {
		t.Fatalf("Chmod() failed: %v", err)
	}

	// Check mode
	info, err := fs.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	if info.Mode != 0600 {
		t.Errorf("file mode is %o, expected 0600", info.Mode)
	}
}

func TestOpenFileFlags(t *testing.T) {
	fs := New()

	// Test O_CREATE
	file, err := fs.OpenFile("/new.txt", os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("OpenFile(O_CREATE) failed: %v", err)
	}
	file.Close()

	// File should exist now
	_, err = fs.Stat("/new.txt")
	if err != nil {
		t.Error("file should exist after O_CREATE")
	}

	// Test O_EXCL
	_, err = fs.OpenFile("/new.txt", os.O_CREATE|os.O_EXCL, 0644)
	if err == nil {
		t.Error("O_EXCL should fail if file exists")
	}

	// Test O_TRUNC
	fs.WriteFile("/test.txt", []byte("original"), 0644)
	file, err = fs.OpenFile("/test.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("OpenFile(O_TRUNC) failed: %v", err)
	}
	file.Write([]byte("truncated"))
	file.Close()

	data, err := fs.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "truncated" {
		t.Errorf("file content is %q, expected %q", string(data), "truncated")
	}
}

func TestAppend(t *testing.T) {
	fs := New()

	// Create and write initial content
	file, err := fs.Create("/test.txt")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	file.Write([]byte("hello"))
	file.Close()

	// Append with O_APPEND
	file, err = fs.OpenFile("/test.txt", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("OpenFile(O_APPEND) failed: %v", err)
	}
	file.Write([]byte(" world"))
	file.Close()

	// Read and verify
	data, err := fs.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "hello world" {
		t.Errorf("file content is %q, expected %q", string(data), "hello world")
	}
}

func TestReadOnlyFS(t *testing.T) {
	// Create a file on a regular filesystem first
	regularFs := New()
	regularFs.WriteFile("/test.txt", []byte("content"), 0644)

	// Create a read-only filesystem (it starts empty, like mounting)
	fs := NewReadOnly()

	// Should not be able to write
	err := fs.WriteFile("/new.txt", []byte("data"), 0644)
	if err == nil {
		t.Error("WriteFile() should fail on read-only FS")
	}

	// Should not be able to create files
	_, err = fs.OpenFile("/another.txt", os.O_CREATE, 0644)
	if err == nil {
		t.Error("OpenFile(O_CREATE) should fail on read-only FS")
	}

	// Should not be able to remove
	err = fs.Remove("/test.txt")
	if err == nil {
		t.Error("Remove() should fail on read-only FS")
	}

	// Should not be able to mkdir
	err = fs.Mkdir("/newdir", 0755)
	if err == nil {
		t.Error("Mkdir() should fail on read-only FS")
	}
}

func TestFileTimes(t *testing.T) {
	fs := New()
	fs.Create("/test.txt")

	// Get initial times
	stat1, _ := fs.Stat("/test.txt")
	initialTime := stat1.ModTime

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Modify file
	file, _ := fs.OpenFile("/test.txt", os.O_WRONLY, 0644)
	file.Write([]byte("data"))
	file.Close()

	// Check times changed
	stat2, _ := fs.Stat("/test.txt")
	if !stat2.ModTime.After(initialTime) {
		t.Error("modification time should have changed")
	}
}

func TestNestedReadWrite(t *testing.T) {
	fs := New()
	fs.MkdirAll("/a/b/c", 0755)

	// Write to nested file
	fs.WriteFile("/a/b/c/file.txt", []byte("nested content"), 0644)

	// Read it back
	data, err := fs.ReadFile("/a/b/c/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(data) != "nested content" {
		t.Errorf("file content is %q, expected %q", string(data), "nested content")
	}
}

func TestPathCleaning(t *testing.T) {
	fs := New()
	fs.Create("/foo/../bar")

	// Both should exist because paths are cleaned
	_, err := fs.Stat("/bar")
	if err != nil {
		t.Errorf("Stat(\"/bar\") failed: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	fs := New()
	file, _ := fs.Create("/test.txt")
	file.Write([]byte("0123456789"))
	file.Close()

	// Truncate to 5 bytes
	file, _ = fs.OpenFile("/test.txt", os.O_RDWR, 0644)
	err := file.Truncate(5)
	if err != nil {
		t.Fatalf("Truncate() failed: %v", err)
	}
	file.Close()

	// Read and verify
	data, _ := fs.ReadFile("/test.txt")
	if len(data) != 5 {
		t.Errorf("file size is %d, expected 5", len(data))
	}
}

func BenchmarkWrite(b *testing.B) {
	fs := New()
	fs.MkdirAll("/test", 0755)

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, _ := fs.Create("/test/file.txt")
		file.Write(data)
		file.Close()
	}
}

func BenchmarkRead(b *testing.B) {
	fs := New()
	fs.WriteFile("/test/file.txt", make([]byte, 1024), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, _ := fs.Open("/test/file.txt")
		buf := make([]byte, 1024)
		file.Read(buf)
		file.Close()
	}
}
