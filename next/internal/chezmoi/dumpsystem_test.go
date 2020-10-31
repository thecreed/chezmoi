package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

var _ System = &DumpSystem{}

func TestDumpSystem(t *testing.T) {
	fs, cleanup, err := vfst.NewTestFS(map[string]interface{}{
		"/home/user/.local/share/chezmoi": map[string]interface{}{
			".chezmoiignore":  "README.md\n",
			".chezmoiremove":  "*.txt\n",
			".chezmoiversion": "1.2.3\n",
			".chezmoitemplates": map[string]interface{}{
				"foo": "bar",
			},
			"README.md": "",
			"dir": map[string]interface{}{
				"foo": "bar",
			},
			"run_script":      "#!/bin/sh\n",
			"symlink_symlink": "bar",
		},
	})
	require.NoError(t, err)
	defer cleanup()

	s := NewSourceState(
		WithSourceDir("/home/user/.local/share/chezmoi"),
		WithSystem(newTestRealSystem(fs)),
	)
	require.NoError(t, s.Read())
	require.NoError(t, s.Evaluate())

	dumpSystem := NewDumpSystem()
	require.NoError(t, s.applyAll(dumpSystem, "", ApplyOptions{}))
	expectedData := map[string]interface{}{
		"dir": &dirData{
			Type: dataTypeDir,
			Name: "dir",
			Perm: 0o777,
		},
		"dir/foo": &fileData{
			Type:     dataTypeFile,
			Name:     "dir/foo",
			Contents: "bar",
			Perm:     0o666,
		},
		"script": &scriptData{
			Type:     dataTypeScript,
			Name:     "script",
			Contents: "#!/bin/sh\n",
		},
		"symlink": &symlinkData{
			Type:     dataTypeSymlink,
			Name:     "symlink",
			Linkname: "bar",
		},
	}
	actualData := dumpSystem.Data()
	assert.Equal(t, expectedData, actualData)
}
