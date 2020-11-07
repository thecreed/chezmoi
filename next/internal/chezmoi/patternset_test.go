package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

func TestpatternSet(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ps            *patternSet
		expectMatches map[string]bool
	}{
		{
			name: "empty",
			ps:   newPatternSet(),
			expectMatches: map[string]bool{
				"foo": false,
			},
		},
		{
			name: "exact",
			ps: mustnewPatternSet(t, map[string]bool{
				"foo": true,
			}),
			expectMatches: map[string]bool{
				"foo": true,
				"bar": false,
			},
		},
		{
			name: "wildcard",
			ps: mustnewPatternSet(t, map[string]bool{
				"b*": true,
			}),
			expectMatches: map[string]bool{
				"foo": false,
				"bar": true,
				"baz": true,
			},
		},
		{
			name: "exclude",
			ps: mustnewPatternSet(t, map[string]bool{
				"b*":  true,
				"baz": false,
			}),
			expectMatches: map[string]bool{
				"foo": false,
				"bar": true,
				"baz": false,
			},
		},
		{
			name: "doublestar",
			ps: mustnewPatternSet(t, map[string]bool{
				"**/foo": true,
			}),
			expectMatches: map[string]bool{
				"foo":         true,
				"bar/foo":     true,
				"baz/bar/foo": true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for s, expectMatch := range tc.expectMatches {
				assert.Equal(t, expectMatch, tc.ps.Match(s))
			}
		})
	}
}

func TestpatternSetGlob(t *testing.T) {
	for _, tc := range []struct {
		name            string
		ps              *patternSet
		root            interface{}
		expectedMatches []string
	}{
		{
			name:            "empty",
			ps:              newPatternSet(),
			root:            nil,
			expectedMatches: []string{},
		},
		{
			name: "simple",
			ps: mustnewPatternSet(t, map[string]bool{
				"f*": true,
			}),
			root: map[string]interface{}{
				"foo": "",
			},
			expectedMatches: []string{
				"foo",
			},
		},
		{
			name: "include_exclude",
			ps: mustnewPatternSet(t, map[string]bool{
				"b*": true,
				"*z": false,
			}),
			root: map[string]interface{}{
				"bar": "",
				"baz": "",
			},
			expectedMatches: []string{
				"bar",
			},
		},
		{
			name: "doublestar",
			ps: mustnewPatternSet(t, map[string]bool{
				"**/f*": true,
			}),
			root: map[string]interface{}{
				"dir1/dir2/foo": "",
			},
			expectedMatches: []string{
				"dir1/dir2/foo",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			actualMatches, err := tc.ps.Glob(fs, "/")
			require.NoError(t, err)
			assert.Equal(t, tc.expectedMatches, actualMatches)
		})
	}
}

func mustnewPatternSet(t *testing.T, patterns map[string]bool) *patternSet {
	ps := newPatternSet()
	for pattern, exclude := range patterns {
		require.NoError(t, ps.Add(pattern, exclude))
	}
	return ps
}
