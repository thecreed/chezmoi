package chezmoi

import (
	"os"
	"os/exec"

	vfs "github.com/twpayne/go-vfs"
)

// DryRunSystem is an System that reads from, but does not write to, to
// a wrapped System.
type DryRunSystem struct {
	s        System
	ps       *dryRunPersistentState
	modified bool
}

// NewDryRunSystem returns a new DryRunSystem that wraps fs.
func NewDryRunSystem(s System) *DryRunSystem {
	return &DryRunSystem{
		s:  s,
		ps: newDryRunPersistentState(s.PersistentState()),
	}
}

// Chmod implements System.Chmod.
func (s *DryRunSystem) Chmod(name string, mode os.FileMode) error {
	s.modified = true
	return nil
}

// Glob implements System.Glob.
func (s *DryRunSystem) Glob(pattern string) ([]string, error) {
	return s.s.Glob(pattern)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *DryRunSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return s.s.IdempotentCmdOutput(cmd)
}

// Lstat implements System.Lstat.
func (s *DryRunSystem) Lstat(name string) (os.FileInfo, error) {
	return s.s.Lstat(name)
}

// Mkdir implements System.Mkdir.
func (s *DryRunSystem) Mkdir(name string, perm os.FileMode) error {
	s.modified = true
	return nil
}

// Modified returns true if a method that would have modified the wrapped system
// has been called.
func (s *DryRunSystem) Modified() bool {
	return s.modified
}

// PersistentState implements System.PersistentState.
func (s *DryRunSystem) PersistentState() PersistentState {
	return s.ps
}

// RawPath implements System.RawPath.
func (s *DryRunSystem) RawPath(path string) (string, error) {
	return s.s.RawPath(path)
}

// ReadDir implements System.ReadDir.
func (s *DryRunSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return s.s.ReadDir(dirname)
}

// ReadFile implements System.ReadFile.
func (s *DryRunSystem) ReadFile(filename string) ([]byte, error) {
	return s.s.ReadFile(filename)
}

// Readlink implements System.Readlink.
func (s *DryRunSystem) Readlink(name string) (string, error) {
	return s.s.Readlink(name)
}

// RemoveAll implements System.RemoveAll.
func (s *DryRunSystem) RemoveAll(string) error {
	s.modified = true
	return nil
}

// Rename implements System.Rename.
func (s *DryRunSystem) Rename(oldpath, newpath string) error {
	s.modified = true
	return nil
}

// RunCmd implements System.RunCmd.
func (s *DryRunSystem) RunCmd(cmd *exec.Cmd) error {
	s.modified = true
	return nil
}

// RunScript implements System.RunScript.
func (s *DryRunSystem) RunScript(scriptname, dir string, data []byte) error {
	s.modified = true
	return nil
}

// Stat implements System.Stat.
func (s *DryRunSystem) Stat(name string) (os.FileInfo, error) {
	return s.s.Stat(name)
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *DryRunSystem) UnderlyingFS() vfs.FS {
	return s.s.UnderlyingFS()
}

// WriteFile implements System.WriteFile.
func (s *DryRunSystem) WriteFile(string, []byte, os.FileMode) error {
	s.modified = true
	return nil
}

// WriteSymlink implements System.WriteSymlink.
func (s *DryRunSystem) WriteSymlink(string, string) error {
	s.modified = true
	return nil
}
