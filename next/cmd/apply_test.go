package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func TestApplyCmd(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extraRoot interface{}
		args      []string
		tests     []interface{}
	}{
		{
			name: "apply_all",
			tests: []interface{}{
				vfst.TestPath("/home/user/.absent",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.hushlogin",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContents(nil),
				),
				vfst.TestPath("/home/user/.binary",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o777&^chezmoi.GetUmask()),
					vfst.TestContentsString("#!/bin/sh\n"),
				),
				vfst.TestPath("/home/user/.gitconfig",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString(""+
						"[core]\n"+
						"  autocrlf = false\n"+
						"[user]\n"+
						"  email = you@example.com\n"+
						"  name = Your Name\n",
					),
				),
				vfst.TestPath("/home/user/.ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o700&^chezmoi.GetUmask()),
				),
				vfst.TestPath("/home/user/.ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
				vfst.TestPath("/home/user/.symlink",
					vfst.TestModeType(os.ModeSymlink),
					vfst.TestSymlinkTarget(".bashrc"),
				),
			},
		},
		{
			name: "apply_all_--dry-run",
			args: []string{"--dry-run"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.absent",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.hushlogin",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.binary",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.gitconfig",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.ssh",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.ssh/config",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.symlink",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "apply_dir",
			args: []string{"/home/user/.ssh"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o700&^chezmoi.GetUmask()),
				),
				vfst.TestPath("/home/user/.ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
		{
			name: "apply_dir_--recursive=false",
			args: []string{"--recursive=false", "/home/user/.ssh"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o700&^chezmoi.GetUmask()),
				),
				vfst.TestPath("/home/user/.ssh/config",
					vfst.TestDoesNotExist,
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"dot_absent":            "",
					"empty_dot_hushlogin":   "",
					"executable_dot_binary": "#!/bin/sh\n",
					"exists_dot_exists":     "",
					"dot_bashrc":            "# contents of .bashrc\n",
					"dot_gitconfig.tmpl": "" +
						"[core]\n" +
						"  autocrlf = false\n" +
						"[user]\n" +
						"  email = {{ \"you@example.com\" }}\n" +
						"  name = Your Name\n",
					"private_dot_ssh": map[string]interface{}{
						"config": "# contents of .ssh/config\n",
					},
					"symlink_dot_symlink": ".bashrc",
				},
			})
			require.NoError(t, err)
			defer cleanup()

			if tc.extraRoot != nil {
				require.NoError(t, vfst.NewBuilder().Build(fs, tc.extraRoot))
			}

			require.NoError(t, newTestConfig(t, fs).execute(append([]string{"apply"}, tc.args...)))

			vfst.RunTests(t, fs, "", tc.tests)
		})
	}
}
