package cmd

import (
	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type editCmdConfig struct {
	apply     bool
	include   *chezmoi.IncludeSet
	recursive bool
}

func (c *Config) newEditCmd() *cobra.Command {
	editCmd := &cobra.Command{
		Use:     "edit targets...",
		Short:   "Edit the source state of a target",
		Long:    mustGetLongHelp("edit"),
		Example: getExample("edit"),
		RunE:    c.makeRunEWithSourceState(c.runEditCmd),
		Annotations: map[string]string{
			modifiesDestinationDirectory: "true",
			modifiesSourceDirectory:      "true",
			requiresSourceDirectory:      "true",
			runsCommands:                 "true",
		},
	}

	persistentFlags := editCmd.PersistentFlags()
	persistentFlags.BoolVarP(&c.edit.apply, "apply", "a", c.edit.apply, "apply edit after editing")

	return editCmd
}

func (c *Config) runEditCmd(cmd *cobra.Command, args []string, s *chezmoi.SourceState) error {
	var sourcePaths []string
	if len(args) == 0 {
		sourcePaths = []string{c.absSourceDir}
	} else {
		var err error
		sourcePaths, err = c.getSourcePaths(s, args)
		if err != nil {
			return err
		}
	}

	// FIXME transparently decrypt encrypted files

	if err := c.runEditor(sourcePaths); err != nil {
		return err
	}

	if !c.edit.apply {
		return nil
	}

	return c.applyArgs(c.destSystem, c.absDestDir, args, c.edit.include, c.edit.recursive, c.Umask.FileMode())
}
