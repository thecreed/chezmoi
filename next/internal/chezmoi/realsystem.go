package chezmoi

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/google/renameio"
	vfs "github.com/twpayne/go-vfs"
	"go.uber.org/multierr"
)

// An RealSystem is a System that writes to a filesystem and executes scripts.
type RealSystem struct {
	vfs.FS
	ps           PersistentState
	devCache     map[string]uint // devCache maps directories to device numbers.
	tempDirCache map[uint]string // tempDirCache maps device numbers to renameio temporary directories.
}

// NewRealSystem returns a System that acts on fs.
func NewRealSystem(fs vfs.FS, persistentState PersistentState) *RealSystem {
	return &RealSystem{
		FS:           fs,
		ps:           persistentState,
		devCache:     make(map[string]uint),
		tempDirCache: make(map[uint]string),
	}
}

// Chmod implements System.Glob.
func (s *RealSystem) Chmod(name string, mode os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return s.FS.Chmod(name, mode)
}

// Glob implements System.Glob.
func (s *RealSystem) Glob(pattern string) ([]string, error) {
	return doublestar.GlobOS(doubleStarOS{FS: s}, pattern)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *RealSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// PathSeparator implements doublestar.OS.PathSeparator.
func (s *RealSystem) PathSeparator() rune {
	return '/'
}

// PersistentState implements System.PersistentState.
func (s *RealSystem) PersistentState() PersistentState {
	return s.ps
}

// RunCmd implements System.RunCmd.
func (s *RealSystem) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// RunScript implements System.RunScript.
func (s *RealSystem) RunScript(scriptname, dir string, data []byte) (err error) {
	// Write the temporary script file. Put the randomness at the front of the
	// filename to preserve any file extension for Windows scripts.
	f, err := ioutil.TempFile("", "*."+path.Base(scriptname))
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(f.Name()))
	}()

	// Make the script private before writing it in case it contains any
	// secrets.
	if runtime.GOOS != "windows" {
		if err = f.Chmod(0o700); err != nil {
			return
		}
	}
	_, err = f.Write(data)
	err = multierr.Append(err, f.Close())
	if err != nil {
		return
	}

	// Run the temporary script file.
	//nolint:gosec
	cmd := exec.Command(f.Name())
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = s.RunCmd(cmd)
	return
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *RealSystem) UnderlyingFS() vfs.FS {
	return s.FS
}

// WriteSymlink implements System.WriteSymlink.
func (s *RealSystem) WriteSymlink(oldname, newname string) error {
	// Special case: if writing to the real filesystem, use
	// github.com/google/renameio.
	if s.FS == vfs.OSFS {
		return renameio.Symlink(oldname, newname)
	}
	if err := s.FS.RemoveAll(newname); err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.FS.Symlink(oldname, newname)
}

// WriteFile is like ioutil.WriteFile but always sets perm before writing data.
// ioutil.WriteFile only sets the permissions when creating a new file. We need
// to ensure permissions, so we use our own implementation.
func WriteFile(fs vfs.FS, filename string, data []byte, perm os.FileMode) (err error) {
	// Create a new file, or truncate any existing one.
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, f.Close())
	}()

	// Set permissions after truncation but before writing any data, in case the
	// file contained private data before, but before writing the new contents,
	// in case the contents contain private data after.
	if runtime.GOOS != "windows" {
		if err = f.Chmod(perm); err != nil {
			return
		}
	}

	_, err = f.Write(data)
	return
}
