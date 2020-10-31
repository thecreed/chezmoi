package chezmoi

import (
	"os"
	"os/exec"

	vfs "github.com/twpayne/go-vfs"
)

// A ReadOnlySystem is a system that may only be read from.
type ReadOnlySystem struct {
	s  System
	ps PersistentState
}

// NewReadOnlySystem returns a new ReadOnlySystem that wraps system.
func NewReadOnlySystem(system System) *ReadOnlySystem {
	return &ReadOnlySystem{
		s:  system,
		ps: newReadOnlyPersistentState(system.PersistentState()),
	}
}

// Chmod implements System.Chmod.
func (s *ReadOnlySystem) Chmod(name string, perm os.FileMode) error {
	return os.ErrPermission
}

// Glob implements System.Glob.
func (s *ReadOnlySystem) Glob(pattern string) ([]string, error) {
	return s.s.Glob(pattern)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *ReadOnlySystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return s.s.IdempotentCmdOutput(cmd)
}

// Lstat implements System.Lstat.
func (s *ReadOnlySystem) Lstat(filename string) (os.FileInfo, error) {
	return s.s.Lstat(filename)
}

// Mkdir implements System.Mkdir.
func (s *ReadOnlySystem) Mkdir(name string, perm os.FileMode) error {
	return os.ErrPermission
}

// PersistentState implements System.PersistentState.
func (s *ReadOnlySystem) PersistentState() PersistentState {
	return s.ps
}

// RawPath implements System.RawPath.
func (s *ReadOnlySystem) RawPath(path string) (string, error) {
	return s.s.RawPath(path)
}

// ReadDir implements System.ReadDir.
func (s *ReadOnlySystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return s.s.ReadDir(dirname)
}

// ReadFile implements System.ReadFile.
func (s *ReadOnlySystem) ReadFile(filename string) ([]byte, error) {
	return s.s.ReadFile(filename)
}

// Readlink implements System.Readlink.
func (s *ReadOnlySystem) Readlink(name string) (string, error) {
	return s.s.Readlink(name)
}

// RemoveAll implements System.RemoveAll.
func (s *ReadOnlySystem) RemoveAll(name string) error {
	return os.ErrPermission
}

// Rename implements System.Rename.
func (s *ReadOnlySystem) Rename(oldpath, newpath string) error {
	return os.ErrPermission
}

// RunCmd implements System.RunCmd.
func (s *ReadOnlySystem) RunCmd(cmd *exec.Cmd) error {
	return os.ErrPermission
}

// RunScript implements System.RunScript.
func (s *ReadOnlySystem) RunScript(scriptname, dir string, data []byte) error {
	return os.ErrPermission
}

// Stat implements System.Stat.
func (s *ReadOnlySystem) Stat(name string) (os.FileInfo, error) {
	return s.s.Stat(name)
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *ReadOnlySystem) UnderlyingFS() vfs.FS {
	return s.s.UnderlyingFS()
}

// WriteFile implements System.WriteFile.
func (s *ReadOnlySystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.ErrPermission
}

// WriteSymlink implements System.WriteSymlink.
func (s *ReadOnlySystem) WriteSymlink(oldname, newname string) error {
	return os.ErrPermission
}
