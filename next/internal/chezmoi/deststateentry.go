package chezmoi

import (
	"os"
)

// An DestStateEntry represents the state of an entry in the destination state.
type DestStateEntry interface {
	Path() string
	Remove(s System) error
}

// A DestStateAbsent represents the absence of an entry in the destination
// state.
type DestStateAbsent struct {
	path string
}

// A DestStateDir represents the state of a directory in the destination state.
type DestStateDir struct {
	path string
	perm os.FileMode
}

// A DestStateFile represents the state of a file in the destination state.
type DestStateFile struct {
	path string
	perm os.FileMode
	*lazyContents
}

// A DestStateSymlink represents the state of a symlink in the destination state.
type DestStateSymlink struct {
	path string
	*lazyLinkname
}

// NewDestStateEntry returns a new DestStateEntry populated with path from fs.
func NewDestStateEntry(s System, path string) (DestStateEntry, error) {
	info, err := s.Lstat(path)
	switch {
	case os.IsNotExist(err):
		return &DestStateAbsent{
			path: path,
		}, nil
	case err != nil:
		return nil, err
	}
	//nolint:exhaustive
	switch info.Mode() & os.ModeType {
	case 0:
		return &DestStateFile{
			path: path,
			perm: info.Mode() & os.ModePerm,
			lazyContents: &lazyContents{
				contentsFunc: func() ([]byte, error) {
					return s.ReadFile(path)
				},
			},
		}, nil
	case os.ModeDir:
		return &DestStateDir{
			path: path,
			perm: info.Mode() & os.ModePerm,
		}, nil
	case os.ModeSymlink:
		return &DestStateSymlink{
			path: path,
			lazyLinkname: &lazyLinkname{
				linknameFunc: func() (string, error) {
					return s.Readlink(path)
				},
			},
		}, nil
	default:
		return nil, &unsupportedFileTypeError{
			path: path,
			mode: info.Mode(),
		}
	}
}

// Path returns d's path.
func (d *DestStateAbsent) Path() string {
	return d.path
}

// Remove removes d.
func (d *DestStateAbsent) Remove(s System) error {
	return nil
}

// Path returns d's path.
func (d *DestStateDir) Path() string {
	return d.path
}

// Remove removes d.
func (d *DestStateDir) Remove(s System) error {
	return s.RemoveAll(d.path)
}

// Path returns d's path.
func (d *DestStateFile) Path() string {
	return d.path
}

// Remove removes d.
func (d *DestStateFile) Remove(s System) error {
	return s.RemoveAll(d.path)
}

// Path returns d's path.
func (d *DestStateSymlink) Path() string {
	return d.path
}

// Remove removes d.
func (d *DestStateSymlink) Remove(s System) error {
	return s.RemoveAll(d.path)
}
