package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type mergeCmdConfig struct {
	Command string
	Args    []string
}

func (c *Config) newMergeCmd() *cobra.Command {
	mergeCmd := &cobra.Command{
		Use:     "merge target...",
		Args:    cobra.MinimumNArgs(1),
		Short:   "Perform a three-way merge between the destination state, the source state, and the target state",
		Long:    mustGetLongHelp("merge"),
		Example: getExample("merge"),
		RunE:    c.makeRunEWithSourceState(c.runMergeCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
			requiresSourceDirectory: "true",
		},
	}

	return mergeCmd
}

func (c *Config) runMergeCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetNames, err := c.getTargetNames(sourceState, args, getTargetNamesOptions{
		mustBeInSourceState: false,
		recursive:           true,
	})
	if err != nil {
		return err
	}

	// Create a temporary directory to store the target state and ensure that it
	// is removed afterwards. We cannot use fs as it lacks TempDir
	// functionality.
	tempDir, err := ioutil.TempDir("", "chezmoi")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	for _, targetName := range targetNames {
		sourceStateEntry := sourceState.MustEntry(targetName)
		// FIXME sourceStateEntry.TargetStateEntry eagerly evaluates the return
		// targetStateEntry's contents, which means that we cannot fallback to a
		// two-way merge if the source state's contents cannot be decrypted or
		// are an invalid template
		targetStateEntry, err := sourceStateEntry.TargetStateEntry()
		if err != nil {
			return fmt.Errorf("%s: %w", targetName, err)
		}
		targetStateFile, ok := targetStateEntry.(*chezmoi.TargetStateFile)
		if !ok {
			// FIXME consider handling symlinks?
			return fmt.Errorf("%s: not a file", targetName)
		}
		contents, err := targetStateFile.Contents()
		if err != nil {
			return err
		}
		targetStatePath := path.Join(tempDir, path.Base(targetName))
		if err := ioutil.WriteFile(targetStatePath, contents, 0o600); err != nil {
			return err
		}
		args := append(
			append([]string{}, c.Merge.Args...),
			path.Join(c.absDestDir, targetName),
			sourceStateEntry.Path(),
			targetStatePath,
		)
		if err := c.run(c.absDestDir, c.Merge.Command, args); err != nil {
			return fmt.Errorf("%s: %w", targetName, err)
		}
	}

	return nil
}
