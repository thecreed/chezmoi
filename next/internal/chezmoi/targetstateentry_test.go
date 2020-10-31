package chezmoi

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

func TestEntryStateMarshal(t *testing.T) {
	es := EntryState{
		Mode:           os.ModeSymlink,
		ContentsSHA256: []byte{0, 1, 2, 3},
	}
	jsonBytes := []byte(`{"mode":134217728,"contentsSHA256":"00010203"}`)

	actualJSONBytes, err := json.Marshal(es)
	require.NoError(t, err)
	assert.Equal(t, jsonBytes, actualJSONBytes)

	var actualEntryState EntryState
	require.NoError(t, json.Unmarshal(jsonBytes, &actualEntryState))
	assert.Equal(t, es, actualEntryState)
}

func TestTargetStateEntryApplyAndEqual(t *testing.T) {
	for _, tc1 := range []struct {
		name             string
		targetStateEntry TargetStateEntry
	}{
		{
			name:             "absent",
			targetStateEntry: &TargetStateAbsent{},
		},
		{
			name: "dir",
			targetStateEntry: &TargetStateDir{
				perm: 0o777,
			},
		},
		{
			name: "file",
			targetStateEntry: &TargetStateFile{
				perm: 0o666,
				lazyContents: &lazyContents{
					contents: []byte("bar"),
				},
			},
		},
		{
			name: "file_empty",
			targetStateEntry: &TargetStateFile{
				perm: 0o666,
			},
		},
		{
			name: "file_empty_ok",
			targetStateEntry: &TargetStateFile{
				perm: 0o666,
			},
		},
		{
			name: "file_executable",
			targetStateEntry: &TargetStateFile{
				perm: 0o777,
				lazyContents: &lazyContents{
					contents: []byte("#!/bin/sh\n"),
				},
			},
		},
		{
			name: "present",
			targetStateEntry: &TargetStatePresent{
				perm: 0o666,
			},
		},
		{
			name: "symlink",
			targetStateEntry: &TargetStateSymlink{
				lazyLinkname: &lazyLinkname{
					linkname: "bar",
				},
			},
		},
	} {
		t.Run(tc1.name, func(t *testing.T) {
			for _, tc2 := range []struct {
				name string
				root interface{}
			}{
				{
					name: "not_present",
					root: map[string]interface{}{
						"/home/user": &vfst.Dir{Perm: 0o777},
					},
				},
				{
					name: "existing_dir",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Dir{Perm: 0o777},
					},
				},
				{
					name: "existing_dir_chmod",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Dir{Perm: 0o666},
					},
				},
				{
					name: "existing_file_empty",
					root: map[string]interface{}{
						"/home/user/foo": "",
					},
				},
				{
					name: "existing_file_contents",
					root: map[string]interface{}{
						"/home/user/foo": "baz",
					},
				},
				{
					name: "existing_file_chmod",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.File{
							Perm: 0o777,
						},
					},
				},
				{
					name: "existing_symlink",
					root: map[string]interface{}{
						"/home/user/bar": "",
						"/home/user/foo": &vfst.Symlink{Target: "bar"},
					},
				},
				{
					name: "existing_symlink_broken",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Symlink{Target: "bar"},
					},
				},
			} {
				t.Run(tc2.name, func(t *testing.T) {
					fs, cleanup, err := vfst.NewTestFS(tc2.root)
					require.NoError(t, err)
					defer cleanup()
					s := newTestRealSystem(fs)

					// Read the initial destination state entry from fs.
					destStateEntry, err := NewDestStateEntry(s, "/home/user/foo")
					require.NoError(t, err)

					// Apply the target state entry.
					targetSystem := newTestRealSystem(fs)
					require.NoError(t, tc1.targetStateEntry.Apply(targetSystem, destStateEntry, GetUmask()))

					// Verify that the destination state entry matches the
					// desired state.
					vfst.RunTests(t, fs, "", vfst.TestPath("/home/user/foo", targetStateTest(t, tc1.targetStateEntry)...))

					// Read the updated destination state entry from fs and
					// verify that it is equal to the target state entry.
					newDestStateEntry, err := NewDestStateEntry(s, "/home/user/foo")
					require.NoError(t, err)
					equal, err := tc1.targetStateEntry.Equal(newDestStateEntry, GetUmask())
					require.NoError(t, err)
					require.True(t, equal)
				})
			}
		})
	}
}

func targetStateTest(t *testing.T, ts TargetStateEntry) []vfst.PathTest {
	switch ts := ts.(type) {
	case *TargetStateAbsent:
		return []vfst.PathTest{
			vfst.TestDoesNotExist,
		}
	case *TargetStateDir:
		return []vfst.PathTest{
			vfst.TestIsDir,
			vfst.TestModePerm(ts.perm &^ GetUmask()),
		}
	case *TargetStateFile:
		expectedContents, err := ts.Contents()
		require.NoError(t, err)
		return []vfst.PathTest{
			vfst.TestModeIsRegular,
			vfst.TestContents(expectedContents),
			vfst.TestModePerm(ts.perm &^ GetUmask()),
		}
	case *TargetStatePresent:
		return []vfst.PathTest{
			vfst.TestModeIsRegular,
			vfst.TestModePerm(ts.perm &^ GetUmask()),
		}
	case *TargetStateRenameDir:
		// FIXME test for presence of newName
		return []vfst.PathTest{
			vfst.TestDoesNotExist,
		}
	case *TargetStateScript:
		return nil // FIXME how to verify scripts?
	case *TargetStateSymlink:
		expectedLinkname, err := ts.Linkname()
		require.NoError(t, err)
		return []vfst.PathTest{
			vfst.TestModeType(os.ModeSymlink),
			vfst.TestSymlinkTarget(expectedLinkname),
		}
	default:
		return nil
	}
}
