package chezmoi

import (
	"os"
	"testing"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"

	"github.com/twpayne/chezmoi/next/internal/chezmoitest"
)

func TestSourceStateAdd(t *testing.T) {
	for _, tc := range []struct {
		name       string
		destPaths  []string
		addOptions AddOptions
		extraRoot  interface{}
		tests      []interface{}
	}{
		{
			name: "file",
			destPaths: []string{
				"/home/user/.bashrc",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_bashrc",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .bashrc\n"),
				),
			},
		},
		{
			name: "replace_file_contents",
			destPaths: []string{
				"/home/user/.bashrc",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/dot_bashrc": "# old contents of .bashrc\n",
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_bashrc",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .bashrc\n"),
				),
			},
		},
		{
			name: "change_file_attributes",
			destPaths: []string{
				"/home/user/.bashrc",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user/.local/share/chezmoi/executable_dot_bashrc": "# contents of .bashrc\n",
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_bashrc",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .bashrc\n"),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/executable_dot_bashrc",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "dir_unix",
			destPaths: []string{
				"/home/user/.ssh",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh/config",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "dir_windows",
			destPaths: []string{
				"/home/user/.ssh",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh/config",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "change_dir_attributes",
			destPaths: []string{
				"/home/user/dir",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			extraRoot: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi/exact_dir/file": "# contents of file\n",
					"dir/file":                            "# contents of file\n",
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/exact_dir",
					vfst.TestDoesNotExist,
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dir",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dir/file",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of file\n"),
				),
			},
		},
		{
			name: "dir_and_file_windows",
			destPaths: []string{
				"/home/user/.ssh",
				"/home/user/.ssh/config",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
		{
			name: "file_in_dir_unix",
			destPaths: []string{
				"/home/user/.ssh/config",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/private_dot_ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
		{
			name: "file_in_dir_windows",
			destPaths: []string{
				"/home/user/.ssh/config",
			},
			addOptions: AddOptions{
				Include: NewIncludeSet(IncludeAll),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/.local/share/chezmoi/dot_ssh/config",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("# contents of .ssh/config\n"),
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chezmoitest.SkipUnlessGOOS(t, tc.name)

			fs, cleanup, err := vfst.NewTestFS(map[string]interface{}{
				"/home/user": map[string]interface{}{
					".bashrc": "# contents of .bashrc\n",
					".ssh": &vfst.Dir{
						Perm: 0o700,
						Entries: map[string]interface{}{
							"config": "# contents of .ssh/config\n",
						},
					},
					".local/share/chezmoi": &vfst.Dir{Perm: 0o777},
				},
			})
			require.NoError(t, err)
			defer cleanup()
			if tc.extraRoot != nil {
				require.NoError(t, vfst.NewBuilder().Build(fs, tc.extraRoot))
			}
			system := newTestRealSystem(fs)

			s := NewSourceState(
				WithDestDir("/home/user"),
				WithSourceDir("/home/user/.local/share/chezmoi"),
				WithSystem(system),
			)
			require.NoError(t, s.Read())
			require.NoError(t, s.Evaluate())

			destPathInfos := make(map[string]os.FileInfo)
			for _, destPath := range tc.destPaths {
				require.NoError(t, s.AddDestPathInfos(destPathInfos, system, destPath, nil))
			}
			require.NoError(t, s.Add(system, destPathInfos, &tc.addOptions))

			vfst.RunTests(t, fs, "", tc.tests...)
		})
	}
}

func TestSourceStateApplyAll(t *testing.T) {
	// FIXME script tests
	// FIXME script template tests
	for _, tc := range []struct {
		name               string
		root               interface{}
		sourceStateOptions []SourceStateOption
		tests              []interface{}
	}{
		{
			name: "empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": &vfst.Dir{Perm: 0o777},
				},
			},
		},
		{
			name: "dir",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo": &vfst.Dir{Perm: 0o777},
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
			},
		},
		{
			name: "dir_exact",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "",
					},
					".local/share/chezmoi": map[string]interface{}{
						"exact_foo": &vfst.Dir{Perm: 0o777},
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestIsDir,
					vfst.TestModePerm(0o777&^GetUmask()),
				),
				vfst.TestPath("/home/user/foo/bar",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "file",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("bar"),
				),
			},
		},
		{
			name: "file_remove_empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"foo": "",
					".local/share/chezmoi": map[string]interface{}{
						"foo": "",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestDoesNotExist,
				),
			},
		},
		{
			name: "file_create_empty",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"empty_foo": "",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString(""),
				),
			},
		},
		{
			name: "file_template",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"foo.tmpl": "email = {{ .email }}",
					},
				},
			},
			sourceStateOptions: []SourceStateOption{
				WithTemplateData(map[string]interface{}{
					"email": "you@example.com",
				}),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("email = you@example.com"),
				),
			},
		},
		{
			name: "exists_create",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"exists_foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("bar"),
				),
			},
		},
		{
			name: "exists_no_replace",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"exists_foo": "bar",
					},
					"foo": "baz",
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeIsRegular,
					vfst.TestModePerm(0o666&^GetUmask()),
					vfst.TestContentsString("baz"),
				),
			},
		},
		{
			name: "symlink",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"symlink_foo": "bar",
					},
				},
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeType(os.ModeSymlink),
					vfst.TestSymlinkTarget("bar"),
				),
			},
		},
		{
			name: "symlink_template",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					".local/share/chezmoi": map[string]interface{}{
						"symlink_foo.tmpl": "bar_{{ .os }}",
					},
				},
			},
			sourceStateOptions: []SourceStateOption{
				WithTemplateData(map[string]interface{}{
					"os": "linux",
				}),
			},
			tests: []interface{}{
				vfst.TestPath("/home/user/foo",
					vfst.TestModeType(os.ModeSymlink),
					vfst.TestSymlinkTarget("bar_linux"),
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			system := newTestRealSystem(fs)
			sourceStateOptions := []SourceStateOption{
				WithDestDir("/home/user"),
				WithSourceDir("/home/user/.local/share/chezmoi"),
				WithSystem(system),
			}
			sourceStateOptions = append(sourceStateOptions, tc.sourceStateOptions...)
			s := NewSourceState(sourceStateOptions...)
			require.NoError(t, s.Read())
			require.NoError(t, s.Evaluate())
			require.NoError(t, s.applyAll(system, "/home/user", ApplyOptions{
				Umask: GetUmask(),
			}))

			vfst.RunTests(t, fs, "", tc.tests...)
		})
	}
}

func TestSourceStateSortedTargetNames(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		root                      interface{}
		expectedSortedTargetNames []string
	}{
		{
			name:                      "empty",
			root:                      nil,
			expectedSortedTargetNames: []string{},
		},
		{
			name: "scripts",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_first_1first": "",
					"run_first_2first": "",
					"run_first_3first": "",
					"run_1":            "",
					"run_2":            "",
					"run_3":            "",
					"run_last_1last":   "",
					"run_last_2last":   "",
					"run_last_3last":   "",
				},
			},
			expectedSortedTargetNames: []string{
				"1first",
				"2first",
				"3first",
				"1",
				"2",
				"3",
				"1last",
				"2last",
				"3last",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			s := NewSourceState(
				WithSourceDir("/home/user/.local/share/chezmoi"),
				WithSystem(newTestRealSystem(fs)),
			)
			require.NoError(t, s.Read())
			assert.Equal(t, tc.expectedSortedTargetNames, s.AllTargetNames())
		})
	}
}

func TestSourceStateRead(t *testing.T) {
	for _, tc := range []struct {
		name                string
		root                interface{}
		expectedError       string
		expectedSourceState *SourceState
	}{
		{
			name: "empty",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": &vfst.Dir{Perm: 0o777},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": &vfst.Dir{Perm: 0o777},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attributes: DirAttributes{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
				}),
			),
		},
		{
			name: "file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/foo",
						Attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("bar")),
						},
					},
				}),
			),
		},
		{
			name: "duplicate_target_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo":      "bar",
					"foo.tmpl": "bar",
				},
			},
			expectedError: "foo: duplicate target (/home/user/.local/share/chezmoi/foo, /home/user/.local/share/chezmoi/foo.tmpl)",
		},
		{
			name: "duplicate_target_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo":       "bar",
					"exact_foo": &vfst.Dir{Perm: 0o777},
				},
			},
			expectedError: "foo: duplicate target (/home/user/.local/share/chezmoi/exact_foo, /home/user/.local/share/chezmoi/foo)",
		},
		{
			name: "duplicate_target_script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_script":      "#!/bin/sh\n",
					"run_once_script": "#!/bin/sh\n",
				},
			},
			expectedError: "script: duplicate target (/home/user/.local/share/chezmoi/run_once_script, /home/user/.local/share/chezmoi/run_script)",
		},
		{
			name: "symlink_with_attributes",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"bar":            "baz",
					"executable_foo": &vfst.Symlink{Target: "bar"},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/bar",
						Attributes: FileAttributes{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/executable_foo",
						Attributes: FileAttributes{
							Name:       "foo",
							Type:       SourceFileTypeFile,
							Executable: true,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o777,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
				}),
			),
		},
		{
			name: "symlink_script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"bar":     "baz",
					"run_foo": &vfst.Symlink{Target: "bar"},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/bar",
						Attributes: FileAttributes{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateFile{
							perm:         0o666,
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/run_foo",
						Attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeScript,
						},
						lazyContents: newLazyContents([]byte("baz")),
						targetStateEntry: &TargetStateScript{
							name:         "foo",
							lazyContents: newLazyContents([]byte("baz")),
						},
					},
				}),
			),
		},
		{
			name: "script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/run_foo",
						Attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeScript,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateScript{
							name:         "foo",
							lazyContents: newLazyContents([]byte("bar")),
						},
					},
				}),
			),
		},
		{
			name: "symlink",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"symlink_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/symlink_foo",
						Attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeSymlink,
						},
						lazyContents: newLazyContents([]byte("bar")),
						targetStateEntry: &TargetStateSymlink{
							lazyLinkname: newLazyLinkname("bar"),
						},
					},
				}),
			),
		},
		{
			name: "file_in_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "baz",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attributes: DirAttributes{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
					"foo/bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/foo/bar",
						Attributes: FileAttributes{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: &lazyContents{
							contents: []byte("baz"),
						},
						targetStateEntry: &TargetStateFile{
							perm: 0o666,
							lazyContents: &lazyContents{
								contents: []byte("baz"),
							},
						},
					},
				}),
			),
		},
		{
			name: "chezmoiignore",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"README.md": true,
					}),
				),
			),
		},
		{
			name: "chezmoiignore_ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
					"README.md":      "",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"README.md": true,
					}),
				),
			),
		},
		{
			name: "chezmoiignore_exact_dir",
			root: map[string]interface{}{
				"/home/user/dir": map[string]interface{}{
					"bar": "# contents of dir/bar\n",
					"baz": "# contents of dir/baz\n",
					"foo": "# contents of dir/foo\n",
				},
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "dir/baz\n",
					"exact_dir": map[string]interface{}{
						"bar": "# contents of dir/bar\n",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"dir": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/exact_dir",
						Attributes: DirAttributes{
							Name:  "dir",
							Exact: true,
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
					"dir/bar": &SourceStateFile{
						path: "/home/user/.local/share/chezmoi/exact_dir/bar",
						Attributes: FileAttributes{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
						lazyContents: &lazyContents{
							contents: []byte("# contents of dir/bar\n"),
						},
						targetStateEntry: &TargetStateFile{
							perm: 0o666,
							lazyContents: &lazyContents{
								contents: []byte("# contents of dir/bar\n"),
							},
						},
					},
					"dir/foo": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/exact_dir",
					},
				}),
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"dir/baz": true,
					}),
				),
			),
		},
		{
			name: "chezmoiremove",
			root: map[string]interface{}{
				"/home/user/foo": "",
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiremove": "foo\n",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/.chezmoiremove",
					},
				}),
			),
		},
		{
			name: "chezmoiremove_and_ignore",
			root: map[string]interface{}{
				"/home/user": map[string]interface{}{
					"bar": "",
					"baz": "",
				},
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "baz\n",
					".chezmoiremove": "b*\n",
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"bar": &SourceStateRemove{
						path: "/home/user/.local/share/chezmoi/.chezmoiremove",
					},
				}),
				withIgnore(
					mustNewPatternSet(t, map[string]bool{
						"baz": true,
					}),
				),
			),
		},
		{
			name: "chezmoitemplates",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoitemplates": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withTemplates(
					map[string]*template.Template{
						"foo": template.Must(template.New("foo").Option("missingkey=error").Parse("bar")),
					},
				),
			),
		},
		{
			name: "chezmoiversion",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
				},
			},
			expectedSourceState: NewSourceState(
				withMinVersion(
					semver.Version{
						Major: 1,
						Minor: 2,
						Patch: 3,
					},
				),
			),
		},
		{
			name: "chezmoiversion_multiple",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
					"foo": map[string]interface{}{
						".chezmoiversion": "2.3.4\n",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntries(map[string]SourceStateEntry{
					"foo": &SourceStateDir{
						path: "/home/user/.local/share/chezmoi/foo",
						Attributes: DirAttributes{
							Name: "foo",
						},
						targetStateEntry: &TargetStateDir{
							perm: 0o777,
						},
					},
				}),
				withMinVersion(
					semver.Version{
						Major: 2,
						Minor: 3,
						Patch: 4,
					},
				),
			),
		},
		{
			name: "ignore_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": "",
				},
			},
			expectedSourceState: NewSourceState(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			s := NewSourceState(
				WithDestDir("/home/user"),
				WithSourceDir("/home/user/.local/share/chezmoi"),
				WithSystem(newTestRealSystem(fs)),
			)
			err = s.Read()
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
				return
			}
			require.NoError(t, err)
			require.NoError(t, s.Evaluate())
			tc.expectedSourceState.destDir = "/home/user"
			tc.expectedSourceState.sourceDir = "/home/user/.local/share/chezmoi"
			require.NoError(t, tc.expectedSourceState.Evaluate())
			s.system = nil
			assert.Equal(t, tc.expectedSourceState, s)
		})
	}
}

func withEntries(sourceEntries map[string]SourceStateEntry) SourceStateOption {
	return func(s *SourceState) {
		s.entries = sourceEntries
	}
}

func withIgnore(ignore *PatternSet) SourceStateOption {
	return func(s *SourceState) {
		s.ignore = ignore
	}
}

func withMinVersion(minVersion semver.Version) SourceStateOption {
	return func(s *SourceState) {
		s.minVersion = minVersion
	}
}

func withTemplates(templates map[string]*template.Template) SourceStateOption {
	return func(s *SourceState) {
		s.templates = templates
	}
}
