package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newForgetCmd() *cobra.Command {
	forgetCmd := &cobra.Command{
		Use:     "forget target...",
		Aliases: []string{"unmanage"},
		Short:   "Remove a target from the source state",
		Long:    mustGetLongHelp("forget"),
		Example: getExample("forget"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runForgetCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
		},
	}

	return forgetCmd
}

func (c *Config) runForgetCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	sourcePaths, err := c.getSourcePaths(sourceState, args)
	if err != nil {
		return err
	}

	for _, sourcePath := range sourcePaths {
		if !c.force {
			choice, err := c.prompt(fmt.Sprintf("Remove %s", sourcePath), "ynqa")
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
				c.force = false
			}
		}
		if err := c.sourceSystem.RemoveAll(sourcePath); err != nil {
			return err
		}
	}

	return nil
}
