package chezmoi

import (
	"os"
	"os/exec"

	vfs "github.com/twpayne/go-vfs"
)

// A System reads from and writes to a filesystem, executes idempotent commands,
// runs scripts, and persists state.
type System interface {
	Chmod(name string, mode os.FileMode) error
	Glob(pattern string) ([]string, error)
	IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error)
	Lstat(filename string) (os.FileInfo, error)
	Mkdir(name string, perm os.FileMode) error
	PersistentState() PersistentState
	RawPath(path string) (string, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	Readlink(name string) (string, error)
	RemoveAll(name string) error
	Rename(oldpath, newpath string) error
	RunCmd(cmd *exec.Cmd) error
	RunScript(scriptname, dir string, data []byte) error
	Stat(name string) (os.FileInfo, error)
	UnderlyingFS() vfs.FS
	WriteFile(filename string, data []byte, perm os.FileMode) error
	WriteSymlink(oldname, newname string) error
}

// A nullReaderSystem simulates an empty system.
type nullReaderSystem struct{}

func (nullReaderSystem) ForEach(bucket []byte, fn func(k, v []byte) error) error { return nil }
func (nullReaderSystem) Get(bucket, key []byte) ([]byte, error)                  { return nil, nil }
func (nullReaderSystem) Glob(pattern string) ([]string, error)                   { return nil, nil }
func (nullReaderSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error)       { return cmd.Output() }
func (nullReaderSystem) Lstat(name string) (os.FileInfo, error)                  { return nil, os.ErrNotExist }
func (nullReaderSystem) OpenOrCreate() error                                     { return nil }
func (nullReaderSystem) Stat(name string) (os.FileInfo, error)                   { return nil, os.ErrNotExist }
func (nullReaderSystem) RawPath(path string) (string, error)                     { return path, nil }
func (nullReaderSystem) ReadDir(dirname string) ([]os.FileInfo, error)           { return nil, os.ErrNotExist }
func (nullReaderSystem) ReadFile(filename string) ([]byte, error)                { return nil, os.ErrNotExist }
func (nullReaderSystem) Readlink(name string) (string, error)                    { return "", os.ErrNotExist }
