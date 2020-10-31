package chezmoi

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	vfs "github.com/twpayne/go-vfs"
)

// A DebugSystem wraps a System and logs all of the actions it executes.
type DebugSystem struct {
	s      System
	ps     PersistentState
	logger *log.Logger
}

// NewDebugSystem returns a new DebugSystem.
func NewDebugSystem(system System, logger *log.Logger) *DebugSystem {
	return &DebugSystem{
		s:      system,
		ps:     newDebugPersistentState(system.PersistentState(), logger),
		logger: logger,
	}
}

// Chmod implements System.Chmod.
func (s *DebugSystem) Chmod(name string, mode os.FileMode) error {
	return s.debugf("Chmod(%q, 0o%o)", []interface{}{name, mode}, func() error {
		return s.s.Chmod(name, mode)
	})
}

// Glob implements System.Glob.
func (s *DebugSystem) Glob(name string) ([]string, error) {
	var matches []string
	err := s.debugf("Glob(%q)", []interface{}{name}, func() error {
		var err error
		matches, err = s.s.Glob(name)
		return err
	})
	return matches, err
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *DebugSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	var output []byte
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	err := s.debugf("IdempotentCmdOutput(%q)", []interface{}{cmdStr}, func() error {
		var err error
		output, err = s.s.IdempotentCmdOutput(cmd)
		return err
	})
	return output, err
}

// Lstat implements System.Lstat.
func (s *DebugSystem) Lstat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := s.debugf("Lstat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = s.s.Lstat(name)
		return err
	})
	return info, err
}

// Mkdir implements System.Mkdir.
func (s *DebugSystem) Mkdir(name string, perm os.FileMode) error {
	return s.debugf("Mkdir(%q, 0o%o)", []interface{}{name, perm}, func() error {
		return s.s.Mkdir(name, perm)
	})
}

// PersistentState implements System.PersistentState.
func (s *DebugSystem) PersistentState() PersistentState {
	return s.ps
}

// RawPath implements System.RawPath.
func (s *DebugSystem) RawPath(path string) (string, error) {
	return s.s.RawPath(path)
}

// ReadDir implements System.ReadDir.
func (s *DebugSystem) ReadDir(name string) ([]os.FileInfo, error) {
	var infos []os.FileInfo
	err := s.debugf("ReadDir(%q)", []interface{}{name}, func() error {
		var err error
		infos, err = s.s.ReadDir(name)
		return err
	})
	return infos, err
}

// ReadFile implements System.ReadFile.
func (s *DebugSystem) ReadFile(filename string) ([]byte, error) {
	var data []byte
	err := s.debugf("ReadFile(%q)", []interface{}{filename}, func() error {
		var err error
		data, err = s.s.ReadFile(filename)
		return err
	})
	return data, err
}

// Readlink implements System.Readlink.
func (s *DebugSystem) Readlink(name string) (string, error) {
	var linkname string
	err := s.debugf("Readlink(%q)", []interface{}{name}, func() error {
		var err error
		linkname, err = s.s.Readlink(name)
		return err
	})
	return linkname, err
}

// RemoveAll implements System.RemoveAll.
func (s *DebugSystem) RemoveAll(name string) error {
	return s.debugf("RemoveAll(%q)", []interface{}{name}, func() error {
		return s.s.RemoveAll(name)
	})
}

// Rename implements System.Rename.
func (s *DebugSystem) Rename(oldpath, newpath string) error {
	return s.debugf("Rename(%q, %q)", []interface{}{oldpath, newpath}, func() error {
		return s.s.Rename(oldpath, newpath)
	})
}

// RunCmd implements System.RunCmd.
func (s *DebugSystem) RunCmd(cmd *exec.Cmd) error {
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	return s.debugf("RunCmd(%q)", []interface{}{cmdStr}, func() error {
		return s.s.RunCmd(cmd)
	})
}

// RunScript implements System.RunScript.
func (s *DebugSystem) RunScript(scriptname, dir string, data []byte) error {
	return s.debugf("Run(%q, %q, %s)", []interface{}{scriptname, dir, firstFewBytes(data)}, func() error {
		return s.s.RunScript(scriptname, dir, data)
	})
}

// Stat implements System.Stat.
func (s *DebugSystem) Stat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := s.debugf("Stat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = s.s.Stat(name)
		return err
	})
	return info, err
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *DebugSystem) UnderlyingFS() vfs.FS {
	return s.s.UnderlyingFS()
}

// WriteFile implements System.WriteFile.
func (s *DebugSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return s.debugf("WriteFile(%q, %s, 0o%o)", []interface{}{name, firstFewBytes(data), perm}, func() error {
		return s.s.WriteFile(name, data, perm)
	})
}

// WriteSymlink implements System.WriteSymlink.
func (s *DebugSystem) WriteSymlink(oldname, newname string) error {
	return s.debugf("WriteSymlink(%q, %q)", []interface{}{oldname, newname}, func() error {
		return s.s.WriteSymlink(oldname, newname)
	})
}

// debugf logs debugging information about calling f.
func (s *DebugSystem) debugf(format string, args []interface{}, f func() error) error {
	errChan := make(chan error)
	start := time.Now()
	go func(errChan chan<- error) {
		errChan <- f()
	}(errChan)
	select {
	case err := <-errChan:
		if err == nil {
			s.logger.Printf(format+" (%s)", append(args, time.Since(start))...)
		} else {
			s.logger.Printf(format+" == %v (%s)", append(args, err, time.Since(start))...)
		}
		return err
	case <-time.After(1 * time.Second):
		s.logger.Printf(format, args...)
		err := <-errChan
		if err == nil {
			s.logger.Printf(format+" (%s)", append(args, time.Since(start))...)
		} else {
			s.logger.Printf(format+" == %v (%s)", append(args, err, time.Since(start))...)
		}
		return err
	}
}

// firstFewBytes returns the first few bytes of data in a human-readable form.
func firstFewBytes(data []byte) string {
	const few = 16
	if len(data) < few {
		return strconv.Quote(string(data))
	}
	return strconv.Quote(string(data[:few]) + "...")
}
