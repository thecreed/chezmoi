package chezmoi

import (
	"testing"

	"github.com/muesli/combinator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirAttributes tests dirAttributes by round-tripping between directory
// names and dirAttributes.
func TestDirAttributes(t *testing.T) {
	testData := struct {
		Name    []string
		Exact   []bool
		Private []bool
	}{
		Name: []string{
			".dir",
			"dir.tmpl",
			"dir",
			"empty_dir",
			"encrypted_dir",
			"executable_dir",
			"once_dir",
			"run_dir",
			"run_once_dir",
			"symlink_dir",
		},
		Exact:   []bool{false, true},
		Private: []bool{false, true},
	}
	var das []DirAttributes
	require.NoError(t, combinator.Generate(&das, testData))
	for _, da := range das {
		actualSourceName := da.BaseName()
		actualDA := parseDirAttributes(actualSourceName)
		assert.Equal(t, da, actualDA)
		assert.Equal(t, actualSourceName, actualDA.BaseName())
	}
}

// TestFileAttributes tests fileAttributes by round-tripping between file names
// and fileAttributes.
func TestFileAttributes(t *testing.T) {
	var fas []FileAttributes
	require.NoError(t, combinator.Generate(&fas, struct {
		Type       SourceFileTargetType
		Name       []string
		Empty      []bool
		Encrypted  []bool
		Executable []bool
		Private    []bool
		Template   []bool
	}{
		Type: SourceFileTypeFile,
		Name: []string{
			".name",
			"exact_name",
			"name",
		},
		Empty:      []bool{false, true},
		Encrypted:  []bool{false, true},
		Executable: []bool{false, true},
		Private:    []bool{false, true},
		Template:   []bool{false, true},
	}))
	require.NoError(t, combinator.Generate(&fas, struct {
		Type       SourceFileTargetType
		Name       []string
		Encrypted  []bool
		Executable []bool
		Private    []bool
		Template   []bool
	}{
		Type: SourceFileTypePresent,
		Name: []string{
			".name",
			"exact_name",
			"name",
		},
		Encrypted:  []bool{false, true},
		Executable: []bool{false, true},
		Private:    []bool{false, true},
		Template:   []bool{false, true},
	}))
	require.NoError(t, combinator.Generate(&fas, struct {
		Type  SourceFileTargetType
		Name  []string
		Once  []bool
		Order []int
	}{
		Type: SourceFileTypeScript,
		Name: []string{
			".name",
			"exact_name",
			"name",
		},
		Once:  []bool{false, true},
		Order: []int{-1, 0, 1},
	}))
	require.NoError(t, combinator.Generate(&fas, struct {
		Type SourceFileTargetType
		Name []string
	}{
		Type: SourceFileTypeSymlink,
		Name: []string{
			".name",
			"exact_name",
			"name",
		},
	}))
	for _, fa := range fas {
		actualSourceName := fa.BaseName()
		actualFA := parseFileAttributes(actualSourceName)
		assert.Equal(t, fa, actualFA)
		assert.Equal(t, actualSourceName, actualFA.BaseName())
	}
}
