package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

func TestPatternSet(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ps            *PatternSet
		expectMatches map[string]bool
	}{
		{
			name: "empty",
			ps:   NewPatternSet(),
			expectMatches: map[string]bool{
				"foo": false,
			},
		},
		{
			name: "exact",
			ps: mustNewPatternSet(t, map[string]bool{
				"foo": true,
			}),
			expectMatches: map[string]bool{
				"foo": true,
				"bar": false,
			},
		},
		{
			name: "wildcard",
			ps: mustNewPatternSet(t, map[string]bool{
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
			ps: mustNewPatternSet(t, map[string]bool{
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
			ps: mustNewPatternSet(t, map[string]bool{
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

func TestPatternSetGlob(t *testing.T) {
	for _, tc := range []struct {
		name            string
		ps              *PatternSet
		root            interface{}
		expectedMatches []string
	}{
		{
			name:            "empty",
			ps:              NewPatternSet(),
			root:            nil,
			expectedMatches: []string{},
		},
		{
			name: "simple",
			ps: mustNewPatternSet(t, map[string]bool{
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
			ps: mustNewPatternSet(t, map[string]bool{
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
			ps: mustNewPatternSet(t, map[string]bool{
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

func mustNewPatternSet(t *testing.T, patterns map[string]bool) *PatternSet {
	ps := NewPatternSet()
	for pattern, exclude := range patterns {
		require.NoError(t, ps.Add(pattern, exclude))
	}
	return ps
}
