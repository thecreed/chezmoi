// +build !windows

package chezmoi

import (
	"errors"
	"os"
	"path"
	"syscall"

	"github.com/google/renameio"
	vfs "github.com/twpayne/go-vfs"
)

// WriteFile implements System.WriteFile.
func (s *RealSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	// Special case: if writing to the real filesystem on a non-Windows system,
	// use github.com/google/renameio.
	if s.FS == vfs.OSFS {
		dir := path.Dir(filename)
		dev, ok := s.devCache[dir]
		if !ok {
			info, err := s.Stat(dir)
			if err != nil {
				return err
			}
			statT, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return errors.New("os.FileInfo.Sys() cannot be converted to a *syscall.Stat_t")
			}
			dev = uint(statT.Dev)
			s.devCache[dir] = dev
		}
		tempDir, ok := s.tempDirCache[dev]
		if !ok {
			tempDir = renameio.TempDir(dir)
			s.tempDirCache[dev] = tempDir
		}
		t, err := renameio.TempFile(tempDir, filename)
		if err != nil {
			return err
		}
		defer func() {
			_ = t.Cleanup()
		}()
		if err := t.Chmod(perm); err != nil {
			return err
		}
		if _, err := t.Write(data); err != nil {
			return err
		}
		return t.CloseAtomicallyReplace()
	}

	return WriteFile(s.FS, filename, data, perm)
}
