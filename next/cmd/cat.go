package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newCatCmd() *cobra.Command {
	catCmd := &cobra.Command{
		Use:     "cat target...",
		Short:   "Print the target contents of a file or symlink",
		Long:    mustGetLongHelp("cat"),
		Example: getExample("cat"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runCatCmd),
	}

	return catCmd
}

func (c *Config) runCatCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	targetNames, err := c.getTargetNames(sourceState, args, getTargetNamesOptions{
		mustBeInSourceState: true,
		recursive:           false,
	})
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	for _, targetName := range targetNames {
		targetStateEntry, err := sourceState.MustEntry(targetName).TargetStateEntry()
		if err != nil {
			return fmt.Errorf("%s: %w", targetName, err)
		}
		switch targetStateEntry := targetStateEntry.(type) {
		case *chezmoi.TargetStateFile:
			contents, err := targetStateEntry.Contents()
			if err != nil {
				return fmt.Errorf("%s: %w", targetName, err)
			}
			sb.Write(contents)
		case *chezmoi.TargetStatePresent:
			contents, err := targetStateEntry.Contents()
			if err != nil {
				return fmt.Errorf("%s: %w", targetName, err)
			}
			sb.Write(contents)
		case *chezmoi.TargetStateSymlink:
			linkname, err := targetStateEntry.Linkname()
			if err != nil {
				return fmt.Errorf("%s: %w", targetName, err)
			}
			sb.WriteString(linkname + "\n")
		default:
			return fmt.Errorf("%s: not a file or symlink", targetName)
		}
	}
	return c.writeOutputString(sb.String())
}
