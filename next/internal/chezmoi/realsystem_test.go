package chezmoi

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vfs "github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"

	"github.com/twpayne/chezmoi/next/internal/chezmoitest"
)

var _ System = &RealSystem{}

func TestRealSystemGlob(t *testing.T) {
	fs, cleanup, err := vfst.NewTestFS(map[string]interface{}{
		"/home/user": map[string]interface{}{
			"bar":            "",
			"baz":            "",
			"foo":            "",
			"dir/bar":        "",
			"dir/foo":        "",
			"dir/subdir/foo": "",
		},
	})
	require.NoError(t, err)
	defer cleanup()

	s := newTestRealSystem(fs)
	for _, tc := range []struct {
		pattern         string
		expectedMatches []string
	}{
		{
			pattern: "/home/user/foo",
			expectedMatches: []string{
				"/home/user/foo",
			},
		},
		{
			pattern: "/home/user/**/foo",
			expectedMatches: []string{
				"/home/user/dir/foo",
				"/home/user/dir/subdir/foo",
				"/home/user/foo",
			},
		},
		{
			pattern: "/home/user/**/ba*",
			expectedMatches: []string{
				"/home/user/bar",
				"/home/user/baz",
				"/home/user/dir/bar",
			},
		},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			actualMatches, err := s.Glob(tc.pattern)
			require.NoError(t, err)
			sort.Strings(actualMatches)
			assert.Equal(t, tc.expectedMatches, pathsToSlashes(actualMatches))
		})
	}
}

func newTestRealSystem(fs vfs.FS) *RealSystem {
	return NewRealSystem(fs, chezmoitest.NewPersistentState())
}

func pathsToSlashes(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		result = append(result, filepath.ToSlash(path))
	}
	return result
}
