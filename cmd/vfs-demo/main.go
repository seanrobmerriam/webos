// VFS Demo - Demonstrates the Virtual File System implementation
//
// This program demonstrates the three filesystem backends:
//   - MemFS: In-memory filesystem
//   - DiskFS: Disk-based filesystem
//   - OverlayFS: Layered filesystem (union of MemFS and DiskFS)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vfs "webos/pkg/vfs"
	"webos/pkg/vfs/diskfs"
	"webos/pkg/vfs/memfs"
	"webos/pkg/vfs/overlayfs"
)

func main() {
	fmt.Println("=== Virtual File System Demo ===")
	fmt.Println()

	// Demo MemFS
	fmt.Println("--- MemFS (In-Memory Filesystem) ---")
	demoMemFS()

	// Demo DiskFS
	fmt.Println()
	fmt.Println("--- DiskFS (Disk-Based Filesystem) ---")
	demoDiskFS()

	// Demo OverlayFS
	fmt.Println()
	fmt.Println("--- OverlayFS (Layered Filesystem) ---")
	demoOverlayFS()

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}

// demoMemFS demonstrates the in-memory filesystem.
func demoMemFS() {
	// Create a new in-memory filesystem
	fs := memfs.New()

	// Create a file
	fmt.Println("Creating /hello.txt...")
	file, err := fs.Create("/hello.txt")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}

	// Write to file
	n, err := file.Write([]byte("Hello from MemFS!"))
	if err != nil {
		fmt.Printf("Error writing: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d bytes\n", n)
	file.Close()

	// Read the file
	data, err := fs.ReadFile("/hello.txt")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	fmt.Printf("Read from file: %q\n", string(data))

	// Create a directory
	fmt.Println("Creating /documents directory...")
	err = fs.Mkdir("/documents", 0755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// Create nested file
	fs.WriteFile("/documents/nested.txt", []byte("Nested file content"), 0644)
	data, _ = fs.ReadFile("/documents/nested.txt")
	fmt.Printf("Nested file content: %q\n", string(data))

	// List directory
	entries, err := fs.ReadDir("/")
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}
	fmt.Printf("Root directory entries: ")
	for _, e := range entries {
		fmt.Printf("%s ", e.Name())
	}
	fmt.Println()

	// Test file operations
	fmt.Println("Testing file operations...")
	testFileOperations(fs)
}

// demoDiskFS demonstrates the disk-based filesystem.
func demoDiskFS() {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "vfs-demo-diskfs-*")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Create disk filesystem
	fs := diskfs.New(tmpDir)

	// Create a file
	fmt.Println("Creating /sample.txt in DiskFS...")
	file, err := fs.Create("/sample.txt")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}

	content := "This file is stored on disk!"
	n, err := file.Write([]byte(content))
	if err != nil {
		fmt.Printf("Error writing: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d bytes\n", n)
	file.Close()

	// Read the file
	data, err := fs.ReadFile("/sample.txt")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	fmt.Printf("Read from file: %q\n", string(data))

	// Verify file exists on disk
	fullPath := filepath.Join(tmpDir, "sample.txt")
	stat, err := os.Stat(fullPath)
	if err != nil {
		fmt.Printf("Error checking file on disk: %v\n", err)
		return
	}
	fmt.Printf("File size on disk: %d bytes\n", stat.Size())

	// Create nested directories
	fmt.Println("Creating nested directories...")
	fs.MkdirAll("/a/b/c/d", 0755)
	fs.WriteFile("/a/b/c/d/deep.txt", []byte("Deep nested file"), 0644)

	// List all files
	fmt.Println("Listing all files:")
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		rel, _ := filepath.Rel(tmpDir, path)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			fmt.Printf("  [DIR]  /%s\n", strings.ReplaceAll(rel, "\\", "/"))
		} else {
			fmt.Printf("  [FILE] /%s (%d bytes)\n", strings.ReplaceAll(rel, "\\", "/"), info.Size())
		}
		return nil
	})

	// Test file operations
	fmt.Println("Testing file operations...")
	testFileOperations(fs)
}

// demoOverlayFS demonstrates the layered filesystem.
func demoOverlayFS() {
	// Create upper (writable) and lower (read-only) filesystems
	upper := memfs.New()
	lower := memfs.New()

	// Pre-populate lower layer
	fmt.Println("Setting up lower layer (read-only)...")
	lower.WriteFile("/base.txt", []byte("Content from lower layer"), 0644)
	lower.WriteFile("/shared.txt", []byte("Shared in lower"), 0644)
	lower.Mkdir("/lower-dir", 0755)
	lower.WriteFile("/lower-dir/file.txt", []byte("File in lower directory"), 0644)

	// Create overlay
	fmt.Println("Creating overlay filesystem...")
	fs := overlayfs.New(upper, lower)

	// Read from lower layer
	fmt.Println("Reading /base.txt from lower layer...")
	data, err := fs.ReadFile("/base.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Content: %q\n", string(data))

	// Modify a file from lower layer (copy-on-write)
	fmt.Println("Modifying /base.txt (triggers copy-on-write)...")
	file, err := fs.OpenFile("/base.txt", os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	file.Write([]byte("Modified in overlay!"))
	file.Close()

	// Read modified content
	data, _ = fs.ReadFile("/base.txt")
	fmt.Printf("Modified content: %q\n", string(data))

	// Create new file in overlay
	fmt.Println("Creating new file /overlay-only.txt...")
	fs.WriteFile("/overlay-only.txt", []byte("Only in overlay"), 0644)

	// Read from both layers
	fmt.Println("Reading /shared.txt (exists in both layers)...")
	data, _ = fs.ReadFile("/shared.txt")
	fmt.Printf("Content (from upper): %q\n", string(data))

	// Remove a file from lower layer
	fmt.Println("Removing /base.txt from view...")
	fs.Remove("/base.txt")
	_, err = fs.Stat("/base.txt")
	if err == nil {
		fmt.Println("ERROR: File should not exist!")
	} else {
		fmt.Println("File successfully hidden")
	}

	// List merged directory
	fmt.Println("Listing root directory (merged view):")
	entries, _ := fs.ReadDir("/")
	for _, e := range entries {
		fmt.Printf("  - %s\n", e.Name())
	}

	// Test file operations
	fmt.Println("Testing file operations...")
	testFileOperations(fs)
}

// testFileOperations tests various file operations.
func testFileOperations(fs vfs.FileSystem) {
	// Test seek
	fmt.Println("Testing seek operations...")
	file, err := fs.Create("/seek-test.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	file.Write([]byte("0123456789"))
	file.Close()

	file, _ = fs.Open("/seek-test.txt")
	buf := make([]byte, 3)

	// Seek to position 2
	pos, _ := file.Seek(2, vfs.SEEK_SET)
	fmt.Printf("Seeked to position %d\n", pos)

	n, _ := file.Read(buf)
	fmt.Printf("Read %d bytes: %q\n", n, string(buf[:n]))

	file.Close()

	// Test stat
	fmt.Println("Testing stat...")
	info, _ := fs.Stat("/seek-test.txt")
	fmt.Printf("File info: name=%s, size=%d, isDir=%v\n", info.Name, info.Size, info.IsDir)

	// Test chmod
	fmt.Println("Testing chmod...")
	fs.Chmod("/seek-test.txt", 0600)
	info, _ = fs.Stat("/seek-test.txt")
	fmt.Printf("New mode: %o\n", info.Mode)

	// Test times
	fmt.Println("Testing times...")
	fs.Chtimes("/seek-test.txt", time.Now(), time.Now().Add(-time.Hour))
	info, _ = fs.Stat("/seek-test.txt")
	fmt.Printf("ModTime: %v\n", info.ModTime)

	// Test rename
	fmt.Println("Testing rename...")
	fs.Rename("/seek-test.txt", "/renamed.txt")
	_, err = fs.Stat("/seek-test.txt")
	if err != nil {
		fmt.Println("Original file renamed successfully")
	}
	_, err = fs.Stat("/renamed.txt")
	if err == nil {
		fmt.Println("New file exists")
	}

	// Clean up
	fs.Remove("/renamed.txt")
	fs.Remove("/seek-test.txt")
	fs.Remove("/renamed.txt")

	fmt.Println("File operations test complete.")
}
