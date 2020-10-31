package chezmoi

import (
	"os"
	"os/exec"

	vfs "github.com/twpayne/go-vfs"
)

type dataType string

const (
	dataTypeDir     dataType = "dir"
	dataTypeFile    dataType = "file"
	dataTypeScript  dataType = "script"
	dataTypeSymlink dataType = "symlink"
)

// A DumpSystem is a System that writes to a data file.
type DumpSystem struct {
	nullReaderSystem
	ps   nullPersistentState
	data map[string]interface{}
}

type dirData struct {
	Type dataType    `json:"type" toml:"type" yaml:"type"`
	Name string      `json:"name" toml:"name" yaml:"name"`
	Perm os.FileMode `json:"perm" toml:"perm" yaml:"perm"`
}

type fileData struct {
	Type     dataType    `json:"type" toml:"type" yaml:"type"`
	Name     string      `json:"name" toml:"name" yaml:"name"`
	Contents string      `json:"contents" toml:"contents" yaml:"contents"`
	Perm     os.FileMode `json:"perm" toml:"perm" yaml:"perm"`
}

type scriptData struct {
	Type     dataType `json:"type" toml:"type" yaml:"type"`
	Name     string   `json:"name" toml:"name" yaml:"name"`
	Contents string   `json:"contents" toml:"contents" yaml:"contents"`
}

type symlinkData struct {
	Type     dataType `json:"type" toml:"type" yaml:"type"`
	Name     string   `json:"name" toml:"name" yaml:"name"`
	Linkname string   `json:"linkname" toml:"linkname" yaml:"linkname"`
}

// NewDumpSystem returns a new DumpSystem that accumulates data.
func NewDumpSystem() *DumpSystem {
	return &DumpSystem{
		data: make(map[string]interface{}),
	}
}

// Chmod implements System.Chmod.
func (s *DumpSystem) Chmod(name string, mode os.FileMode) error {
	return os.ErrPermission
}

// Data returns s's data.
func (s *DumpSystem) Data() interface{} {
	return s.data
}

// Mkdir implements System.Mkdir.
func (s *DumpSystem) Mkdir(dirname string, perm os.FileMode) error {
	if _, exists := s.data[dirname]; exists {
		return os.ErrExist
	}
	s.data[dirname] = &dirData{
		Type: dataTypeDir,
		Name: dirname,
		Perm: perm,
	}
	return nil
}

// PersistentState implements System.PersistentState.
func (s *DumpSystem) PersistentState() PersistentState {
	return s.ps
}

// RemoveAll implements System.RemoveAll.
func (s *DumpSystem) RemoveAll(name string) error {
	return os.ErrPermission
}

// Rename implements System.Rename.
func (s *DumpSystem) Rename(oldpath, newpath string) error {
	return os.ErrPermission
}

// RunCmd implements System.RunCmd.
func (s *DumpSystem) RunCmd(cmd *exec.Cmd) error {
	return nil
}

// RunScript implements System.RunScript.
func (s *DumpSystem) RunScript(scriptname, dir string, data []byte) error {
	if _, exists := s.data[scriptname]; exists {
		return os.ErrExist
	}
	s.data[scriptname] = &scriptData{
		Type:     dataTypeScript,
		Name:     scriptname,
		Contents: string(data),
	}
	return nil
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *DumpSystem) UnderlyingFS() vfs.FS {
	return nil
}

// WriteFile implements System.WriteFile.
func (s *DumpSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if _, exists := s.data[filename]; exists {
		return os.ErrExist
	}
	s.data[filename] = &fileData{
		Type:     dataTypeFile,
		Name:     filename,
		Contents: string(data),
		Perm:     perm,
	}
	return nil
}

// WriteSymlink implements System.WriteSymlink.
func (s *DumpSystem) WriteSymlink(oldname, newname string) error {
	if _, exists := s.data[newname]; exists {
		return os.ErrExist
	}
	s.data[newname] = &symlinkData{
		Type:     dataTypeSymlink,
		Name:     newname,
		Linkname: oldname,
	}
	return nil
}
