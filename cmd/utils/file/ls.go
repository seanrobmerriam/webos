// Package file provides file operation utilities: ls, cat, cp, mv, rm, mkdir, touch, chmod, chown.
package file

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"webos/pkg/vfs"
)

// LSFlags holds command-line flags for ls.
type LSFlags struct {
	All       bool // Show all files including hidden
	Long      bool // Long format output
	Recursive bool // Recursive listing
	Human     bool // Human-readable sizes
	Type      bool // Show type indicators
}

// ParseLSFlags parses command-line flags for ls.
func ParseLSFlags(args []string) (*LSFlags, []string, error) {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: ls [OPTIONS] [DIRECTORY...]

List directory contents.

Options:
`)
		fs.PrintDefaults()
	}

	all := fs.Bool("a", false, "Show all files (including hidden)")
	allLong := fs.Bool("A", false, "Show all files except . and ..")
	long := fs.Bool("l", false, "Use long listing format")
	recursive := fs.Bool("R", false, "Recursively list directories")
	human := fs.Bool("h", false, "Human-readable sizes")
	typeFlag := fs.Bool("p", false, "Append / indicator to directories")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	// Combine -a and -A into single all flag
	showAll := *all || *allLong

	flags := &LSFlags{
		All:       showAll,
		Long:      *long,
		Recursive: *recursive,
		Human:     *human,
		Type:      *typeFlag,
	}

	return flags, fs.Args(), nil
}

// FormatSize formats a file size in human-readable form.
func FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

// FormatMode formats file mode into string like "drwxr-xr-x".
func FormatMode(mode os.FileMode) string {
	var typeChar string
	switch {
	case mode.IsDir():
		typeChar = "d"
	case mode&os.ModeSymlink != 0:
		typeChar = "l"
	case mode&os.ModeSocket != 0:
		typeChar = "s"
	case mode&os.ModeNamedPipe != 0:
		typeChar = "p"
	case mode&os.ModeCharDevice != 0:
		typeChar = "c"
	default:
		typeChar = "-"
	}

	perm := []byte("---------")
	if mode&0100 != 0 {
		perm[0] = 'r'
	}
	if mode&0200 != 0 {
		perm[1] = 'w'
	}
	if mode&0400 != 0 {
		perm[2] = 'x'
	}
	if mode&01000 != 0 {
		perm[2] = 's'
	}
	if mode&0010 != 0 {
		perm[3] = 'r'
	}
	if mode&0020 != 0 {
		perm[4] = 'w'
	}
	if mode&0040 != 0 {
		perm[5] = 'x'
	}
	if mode&01000 != 0 {
		perm[5] = 's'
	}
	if mode&0001 != 0 {
		perm[6] = 'r'
	}
	if mode&0002 != 0 {
		perm[7] = 'w'
	}
	if mode&0004 != 0 {
		perm[8] = 'x'
	}
	if mode&01000 != 0 {
		perm[8] = 't'
	}

	return typeChar + string(perm)
}

// LS lists directory contents using the provided filesystem.
func LS(fs vfs.FileSystem, paths []string, flags *LSFlags) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	showAll := flags.All

	for i, path := range paths {
		if i > 0 {
			fmt.Println()
		}

		if len(paths) > 1 {
			fmt.Printf("%s:\n", path)
		}

		if flags.Recursive {
			return listRecursive(fs, path, showAll, flags)
		}

		if err := listDirectory(fs, path, showAll, flags); err != nil {
			return err
		}
	}

	return nil
}

// listDirectory lists a single directory.
func listDirectory(fs vfs.FileSystem, path string, showAll bool, flags *LSFlags) error {
	entries, err := fs.ReadDir(path)
	if err != nil {
		return fmt.Errorf("cannot access '%s': %v", path, err)
	}

	if flags.Long {
		var total int64
		var filteredEntries []vfs.DirEntry

		for _, entry := range entries {
			if !showAll && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			filteredEntries = append(filteredEntries, entry)
			info, _ := entry.Info()
			total += info.Size
		}

		if showAll {
			fmt.Printf("total %d\n", total/2)
		} else {
			fmt.Printf("total %d\n", total/2)
		}

		for _, entry := range filteredEntries {
			info, _ := entry.Info()
			line := formatLongEntry(info, flags)
			fmt.Println(line)
		}
	} else {
		var names []string
		for _, entry := range entries {
			if !showAll && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := entry.Name()
			if flags.Type && entry.IsDir() {
				name += "/"
			}
			names = append(names, name)
		}
		fmt.Println(strings.Join(names, "  "))
	}

	return nil
}

// listRecursive lists directories recursively.
func listRecursive(fs vfs.FileSystem, path string, showAll bool, flags *LSFlags) error {
	return vfs.Walk(fs, path, func(p string, info vfs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := filepath.Base(p)
		if !showAll && strings.HasPrefix(name, ".") {
			if info.IsDir {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(path, p)
		if relPath != "." {
			fmt.Printf("%s:\n", relPath)
		}

		if info.IsDir {
			entries, _ := fs.ReadDir(p)
			var names []string
			for _, entry := range entries {
				if !showAll && strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				name := entry.Name()
				if flags.Type && entry.IsDir() {
					name += "/"
				}
				names = append(names, name)
			}
			fmt.Println(strings.Join(names, "  "))
			fmt.Println()
		}

		return nil
	})
}

// formatLongEntry formats a single entry in long format.
func formatLongEntry(info vfs.FileInfo, flags *LSFlags) string {
	mode := FormatMode(info.Mode)
	modTime := info.ModTime.Format("Jan 02 15:04")
	size := info.Size

	if flags.Human {
		return fmt.Sprintf("%s %6s %6s %s %s %s",
			mode, "-", "-", modTime, FormatSize(size), info.Name)
	}

	return fmt.Sprintf("%s %6d %6d %s %s",
		mode, info.Size, info.Size, modTime, info.Name)
}

// Cat concatenates files and writes to stdout.
func Cat(fs vfs.FileSystem, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	for _, path := range paths {
		file, err := fs.Open(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}

		_, err = io.Copy(os.Stdout, file)
		file.Close()
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
	}

	return nil
}

// Mkdir creates directories.
func Mkdir(fs vfs.FileSystem, paths []string, parents bool) error {
	for _, path := range paths {
		if parents {
			if err := fs.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("cannot create directory '%s': %v", path, err)
			}
		} else {
			if err := fs.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("cannot create directory '%s': %v", path, err)
			}
		}
	}
	return nil
}

// Touch creates files or updates timestamps.
func Touch(fs vfs.FileSystem, paths []string) error {
	now := time.Now()
	for _, path := range paths {
		_, err := fs.Stat(path)
		if err == nil {
			// File exists, update timestamp
			if err := fs.Chtimes(path, now, now); err != nil {
				return fmt.Errorf("touch: %s: %v", path, err)
			}
		} else {
			// Create new file
			file, err := fs.Create(path)
			if err != nil {
				return fmt.Errorf("touch: %s: %v", path, err)
			}
			file.Close()
		}
	}
	return nil
}

// Remove removes files or directories.
func Remove(fs vfs.FileSystem, paths []string, recursive, force bool) error {
	for _, path := range paths {
		if recursive {
			if err := fs.RemoveAll(path); err != nil && !force {
				return fmt.Errorf("cannot remove '%s': %v", path, err)
			}
		} else {
			if err := fs.Remove(path); err != nil && !force {
				return fmt.Errorf("cannot remove '%s': %v", path, err)
			}
		}
	}
	return nil
}

// Copy copies files or directories.
func Copy(fs vfs.FileSystem, src, dst string, recursive bool) error {
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("cannot stat '%s': %v", src, err)
	}

	if srcInfo.IsDir && !recursive {
		return fmt.Errorf("omitting directory '%s'", src)
	}

	if srcInfo.IsDir && recursive {
		return copyDirectory(fs, src, dst)
	}

	return copyFile(fs, src, dst)
}

// copyFile copies a single file.
func copyFile(fs vfs.FileSystem, src, dst string) error {
	srcFile, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open '%s': %v", src, err)
	}
	defer srcFile.Close()

	dstFile, err := fs.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("cannot create '%s': %v", dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copy failed: %v", err)
	}

	return nil
}

// copyDirectory copies a directory recursively.
func copyDirectory(fs vfs.FileSystem, src, dst string) error {
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return err
	}

	if err := fs.MkdirAll(dst, srcInfo.Mode); err != nil {
		return err
	}

	entries, err := fs.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(fs, srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(fs, srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Move moves or renames files and directories.
func Move(fs vfs.FileSystem, src, dst string) error {
	return fs.Rename(src, dst)
}

// Chmod changes file mode.
func Chmod(fs vfs.FileSystem, mode os.FileMode, paths []string) error {
	for _, path := range paths {
		if err := fs.Chmod(path, mode); err != nil {
			return fmt.Errorf("chmod: %s: %v", path, err)
		}
	}
	return nil
}

// Chown changes file owner.
func Chown(fs vfs.FileSystem, uid, gid int, paths []string) error {
	for _, path := range paths {
		if err := fs.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("chown: %s: %v", path, err)
		}
	}
	return nil
}
