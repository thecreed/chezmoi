package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newRemoveCmd() *cobra.Command {
	removeCmd := &cobra.Command{
		Use:     "remove target...",
		Aliases: []string{"rm"},
		Short:   "Remove a target from the source state and the destination directory",
		Long:    mustGetLongHelp("remove"),
		Example: getExample("remove"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runRemoveCmd),
		Annotations: map[string]string{
			modifiesDestinationDirectory: "true",
			modifiesSourceDirectory:      "true",
		},
	}

	return removeCmd
}

func (c *Config) runRemoveCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetNames, err := c.getTargetNames(sourceState, args, getTargetNamesOptions{
		recursive:           false,
		mustBeInSourceState: true,
	})
	if err != nil {
		return err
	}

	for _, targetName := range targetNames {
		destPath := path.Join(c.absDestDir, targetName)
		sourcePath := sourceState.MustEntry(targetName).Path()
		if !c.force {
			choice, err := c.prompt(fmt.Sprintf("Remove %s and %s", destPath, sourcePath), "ynqa")
			if err != nil {
				return err
			}
			switch choice {
			case 'y':
			case 'n':
				continue
			case 'q':
				return nil
			case 'a':
				c.force = true
			}
		}
		if err := c.destSystem.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := c.sourceSystem.RemoveAll(sourcePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
