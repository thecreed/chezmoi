package chezmoi

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

var _ System = &TARSystem{}

func TestTARSystem(t *testing.T) {
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

	b := &bytes.Buffer{}
	tarSystem := NewTARSystem(b, tar.Header{})
	require.NoError(t, s.applyAll(tarSystem, "", ApplyOptions{}))

	r := tar.NewReader(b)
	for _, tc := range []struct {
		expectedTypeflag byte
		expectedName     string
		expectedMode     int64
		expectedLinkname string
		expectedContents []byte
	}{
		{
			expectedTypeflag: tar.TypeDir,
			expectedName:     "dir/",
			expectedMode:     0o777,
		},
		{
			expectedTypeflag: tar.TypeReg,
			expectedName:     "dir/foo",
			expectedContents: []byte("bar"),
			expectedMode:     0o666,
		},
		{
			expectedTypeflag: tar.TypeReg,
			expectedName:     "script",
			expectedContents: []byte("#!/bin/sh\n"),
			expectedMode:     0o700,
		},
		{
			expectedTypeflag: tar.TypeSymlink,
			expectedName:     "symlink",
			expectedLinkname: "bar",
		},
	} {
		t.Run(tc.expectedName, func(t *testing.T) {
			header, err := r.Next()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTypeflag, header.Typeflag)
			assert.Equal(t, tc.expectedName, header.Name)
			assert.Equal(t, tc.expectedMode, header.Mode)
			assert.Equal(t, tc.expectedLinkname, header.Linkname)
			assert.Equal(t, int64(len(tc.expectedContents)), header.Size)
			if tc.expectedContents != nil {
				actualContents, err := ioutil.ReadAll(r)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedContents, actualContents)
			}
		})
	}
	_, err = r.Next()
	assert.Equal(t, io.EOF, err)
}
