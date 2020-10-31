package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
	"github.com/twpayne/chezmoi/next/internal/chezmoitest"
)

func TestAddCmd(t *testing.T) {
	for _, tc := range []struct {
		name  string
		root  interface{}
		args  []string
		tests []interface{}
	}{
		{
			name: "add_file",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".bashrc": "# contents of .bashrc\n",
				},
			},
			args: []string{"/home/user/.bashrc"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_bashrc",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("# contents of .bashrc\n"),
				),
			},
		},
		{
			name: "add_binary_file_unix",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".binary": &vfst.File{
						Perm:     0o777,
						Contents: []byte("#!/bin/sh\n"),
					},
				},
			},
			args: []string{"/home/user/.binary"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/executable_dot_binary",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("#!/bin/sh\n"),
				),
			},
		},
		{
			name: "add_empty_file",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".hushlogin": "",
				},
			},
			args: []string{"/home/user/.hushlogin"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/empty_dot_hushlogin",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "add_empty_file_with_--empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".hushlogin": "",
				},
			},
			args: []string{"--empty", "/home/user/.hushlogin"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/empty_dot_hushlogin",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContents(nil),
				),
			},
		},
		{
			name: "add_symlink",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".symlink": &vfst.Symlink{
						Target: ".bashrc",
					},
				},
			},
			args: []string{"/home/user/.symlink"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/symlink_dot_symlink",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString(".bashrc\n"),
				),
			},
		},
		{
			name: "add_private_dir_unix",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".ssh": &vfst.Dir{
						Perm: 0o700,
						Entries: map[string]interface{}{
							"config": "# contents of .ssh/config\n",
						},
					},
				},
			},
			args: []string{"/home/user/.ssh"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^chezmoi.GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
		{
			name: "add_file_in_private_dir_unix",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".ssh": &vfst.Dir{
						Perm: 0o700,
						Entries: map[string]interface{}{
							"config": "# contents of .ssh/config\n",
						},
					},
				},
			},
			args: []string{"/home/user/.ssh/config"},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^chezmoi.GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^chezmoi.GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.SkipUnlessGOOS(t, tc.name)

			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			require.NoError(t, newTestConfig(t, fs).execute(append([]string{"add"}, tc.args...)))

			vfst.RunTests(t, fs, "", tc.tests...)
		})
	}
}
