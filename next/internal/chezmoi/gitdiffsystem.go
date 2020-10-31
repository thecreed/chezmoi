package chezmoi

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
	vfs "github.com/twpayne/go-vfs"
)

// A GitDiffSystem wraps a System and logs all of the actions executed as a git
// diff.
type GitDiffSystem struct {
	s              System
	prefix         string
	unifiedEncoder *diff.UnifiedEncoder
}

// NewGitDiffSystem returns a new GitDiffSystem that writes to w.
func NewGitDiffSystem(s System, w io.Writer, prefix string, color bool) *GitDiffSystem {
	unifiedEncoder := diff.NewUnifiedEncoder(w, diff.DefaultContextLines)
	if color {
		unifiedEncoder.SetColor(diff.NewColorConfig())
	}
	return &GitDiffSystem{
		s:              s,
		prefix:         prefix,
		unifiedEncoder: unifiedEncoder,
	}
}

// Chmod implements System.Chmod.
func (s *GitDiffSystem) Chmod(name string, mode os.FileMode) error {
	fromFileMode, info, err := s.getFileMode(name)
	if err != nil {
		return err
	}
	// Assume that we're only changing permissions.
	toFileMode, err := filemode.NewFromOSFileMode(info.Mode()&^os.ModePerm | mode)
	if err != nil {
		return err
	}
	path := s.trimPrefix(name)
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				from: &gitDiffFile{
					fileMode: fromFileMode,
					path:     path,
					hash:     plumbing.ZeroHash,
				},
				to: &gitDiffFile{
					fileMode: toFileMode,
					path:     path,
					hash:     plumbing.ZeroHash,
				},
			},
		},
	}); err != nil {
		return err
	}
	return s.s.Chmod(name, mode)
}

// Glob implements System.Glob.
func (s *GitDiffSystem) Glob(pattern string) ([]string, error) {
	return s.s.Glob(pattern)
}

// Lstat implements System.Lstat.
func (s *GitDiffSystem) Lstat(name string) (os.FileInfo, error) {
	return s.s.Lstat(name)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *GitDiffSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return s.s.IdempotentCmdOutput(cmd)
}

// Mkdir implements System.Mkdir.
func (s *GitDiffSystem) Mkdir(name string, perm os.FileMode) error {
	toFileMode, err := filemode.NewFromOSFileMode(os.ModeDir | perm)
	if err != nil {
		return err
	}
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				to: &gitDiffFile{
					fileMode: toFileMode,
					path:     s.trimPrefix(name),
					hash:     plumbing.ZeroHash,
				},
			},
		},
	}); err != nil {
		return err
	}
	return s.s.Mkdir(name, perm)
}

// PersistentState implements System.PersistentState.
func (s *GitDiffSystem) PersistentState() PersistentState {
	return s.s.PersistentState()
}

// RawPath implements System.RawPath.
func (s *GitDiffSystem) RawPath(path string) (string, error) {
	return s.s.RawPath(path)
}

// ReadDir implements System.ReadDir.
func (s *GitDiffSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return s.s.ReadDir(dirname)
}

// ReadFile implements System.ReadFile.
func (s *GitDiffSystem) ReadFile(filename string) ([]byte, error) {
	return s.s.ReadFile(filename)
}

// Readlink implements System.Readlink.
func (s *GitDiffSystem) Readlink(name string) (string, error) {
	return s.s.Readlink(name)
}

// RemoveAll implements System.RemoveAll.
func (s *GitDiffSystem) RemoveAll(name string) error {
	fromFileMode, _, err := s.getFileMode(name)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				from: &gitDiffFile{
					fileMode: fromFileMode,
					path:     s.trimPrefix(name),
					hash:     plumbing.ZeroHash,
				},
			},
		},
	}); err != nil {
		return err
	}
	return s.s.RemoveAll(name)
}

// RunCmd implements System.RunCmd.
func (s *GitDiffSystem) RunCmd(cmd *exec.Cmd) error {
	return s.s.RunCmd(cmd)
}

// RunScript implements System.RunScript.
func (s *GitDiffSystem) RunScript(scriptname, dir string, data []byte) error {
	isBinary := isBinary(data)
	var chunks []diff.Chunk
	if !isBinary {
		chunk := &gitDiffChunk{
			content:   string(data),
			operation: diff.Add,
		}
		chunks = append(chunks, chunk)
	}
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				isBinary: isBinary,
				to: &gitDiffFile{
					fileMode: filemode.Executable,
					path:     s.trimPrefix(scriptname),
					hash:     plumbing.ComputeHash(plumbing.BlobObject, data),
				},
				chunks: chunks,
			},
		},
	}); err != nil {
		return err
	}
	return s.s.RunScript(scriptname, dir, data)
}

// Stat implements System.Stat.
func (s *GitDiffSystem) Stat(name string) (os.FileInfo, error) {
	return s.s.Stat(name)
}

// Rename implements System.Rename.
func (s *GitDiffSystem) Rename(oldpath, newpath string) error {
	fileMode, _, err := s.getFileMode(oldpath)
	if err != nil {
		return err
	}
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				from: &gitDiffFile{
					fileMode: fileMode,
					path:     s.trimPrefix(oldpath),
					hash:     plumbing.ZeroHash,
				},
				to: &gitDiffFile{
					fileMode: fileMode,
					path:     s.trimPrefix(newpath),
					hash:     plumbing.ZeroHash,
				},
			},
		},
	}); err != nil {
		return err
	}
	return s.s.Rename(oldpath, newpath)
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *GitDiffSystem) UnderlyingFS() vfs.FS {
	return s.s.UnderlyingFS()
}

// WriteFile implements System.WriteFile.
func (s *GitDiffSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	fromFileMode, _, err := s.getFileMode(filename)
	var fromData []byte
	switch {
	case err == nil:
		fromData, err = s.s.ReadFile(filename)
		if err != nil {
			return err
		}
	case os.IsNotExist(err):
	default:
		return err
	}
	toFileMode, err := filemode.NewFromOSFileMode(perm)
	if err != nil {
		return err
	}
	path := s.trimPrefix(filename)
	isBinary := isBinary(fromData) || isBinary(data)
	var chunks []diff.Chunk
	if !isBinary {
		chunks = diffChunks(string(fromData), string(data))
	}
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				isBinary: isBinary,
				from: &gitDiffFile{
					fileMode: fromFileMode,
					path:     path,
					hash:     plumbing.ComputeHash(plumbing.BlobObject, fromData),
				},
				to: &gitDiffFile{
					fileMode: toFileMode,
					path:     path,
					hash:     plumbing.ComputeHash(plumbing.BlobObject, data),
				},
				chunks: chunks,
			},
		},
	}); err != nil {
		return err
	}
	return s.s.WriteFile(filename, data, perm)
}

// WriteSymlink implements System.WriteSymlink.
func (s *GitDiffSystem) WriteSymlink(oldname, newname string) error {
	if err := s.unifiedEncoder.Encode(&gitDiffPatch{
		filePatches: []diff.FilePatch{
			&gitDiffFilePatch{
				to: &gitDiffFile{
					fileMode: filemode.Symlink,
					path:     s.trimPrefix(newname),
					hash:     plumbing.ComputeHash(plumbing.BlobObject, []byte(oldname)),
				},
				chunks: []diff.Chunk{
					&gitDiffChunk{
						content:   oldname,
						operation: diff.Add,
					},
				},
			},
		},
	}); err != nil {
		return err
	}
	return s.s.WriteSymlink(oldname, newname)
}

func (s *GitDiffSystem) getFileMode(name string) (filemode.FileMode, os.FileInfo, error) {
	info, err := s.s.Stat(name)
	if err != nil {
		return filemode.Empty, nil, err
	}
	fileMode, err := filemode.NewFromOSFileMode(info.Mode())
	return fileMode, info, err
}

func (s *GitDiffSystem) trimPrefix(path string) string {
	return mustTrimPrefix(path, s.prefix)
}

var gitDiffOperation = map[diffmatchpatch.Operation]diff.Operation{
	diffmatchpatch.DiffDelete: diff.Delete,
	diffmatchpatch.DiffEqual:  diff.Equal,
	diffmatchpatch.DiffInsert: diff.Add,
}

type gitDiffChunk struct {
	content   string
	operation diff.Operation
}

func (c *gitDiffChunk) Content() string      { return c.content }
func (c *gitDiffChunk) Type() diff.Operation { return c.operation }

type gitDiffFile struct {
	hash     plumbing.Hash
	fileMode filemode.FileMode
	path     string
}

func (f *gitDiffFile) Hash() plumbing.Hash     { return f.hash }
func (f *gitDiffFile) Mode() filemode.FileMode { return f.fileMode }
func (f *gitDiffFile) Path() string            { return f.path }

type gitDiffFilePatch struct {
	isBinary bool
	from, to diff.File
	chunks   []diff.Chunk
}

func (fp *gitDiffFilePatch) IsBinary() bool                { return fp.isBinary }
func (fp *gitDiffFilePatch) Files() (diff.File, diff.File) { return fp.from, fp.to }
func (fp *gitDiffFilePatch) Chunks() []diff.Chunk          { return fp.chunks }

type gitDiffPatch struct {
	filePatches []diff.FilePatch
	message     string
}

func (p *gitDiffPatch) FilePatches() []diff.FilePatch { return p.filePatches }
func (p *gitDiffPatch) Message() string               { return p.message }

func diffChunks(from, to string) []diff.Chunk {
	dmp := diffmatchpatch.New()
	dmp.DiffTimeout = time.Second
	fromRunes, toRunes, runesToLines := dmp.DiffLinesToRunes(from, to)
	diffs := dmp.DiffCharsToLines(dmp.DiffMainRunes(fromRunes, toRunes, false), runesToLines)
	chunks := make([]diff.Chunk, 0, len(diffs))
	for _, d := range diffs {
		chunk := &gitDiffChunk{
			content:   d.Text,
			operation: gitDiffOperation[d.Type],
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

func isBinary(data []byte) bool {
	return len(data) != 0 && !strings.HasPrefix(http.DetectContentType(data), "text/")
}
