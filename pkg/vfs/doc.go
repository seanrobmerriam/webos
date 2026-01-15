// Package vfs provides a Virtual File System (VFS) abstraction layer
// with support for multiple storage backends including in-memory (MemFS),
// disk-based (DiskFS), and layered (OverlayFS) filesystems.
//
// The VFS interface is inspired by OpenBSD's FFS (Fast File System) and
// provides a unified API for file operations across different storage backends.
//
// # Features
//
//   - Multiple storage backends: MemFS, DiskFS, OverlayFS
//   - Path resolution with symlink support
//   - File locking (advisory and mandatory)
//   - Permission system
//   - Crash recovery via journaling
//
// # Usage
//
// To use a filesystem backend, create an instance and use it through
// the vfs.FileSystem interface:
//
//	fs := memfs.New()
//	file, err := fs.Open("/test.txt", os.O_RDWR|os.O_CREATE)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer file.Close()
package vfs
