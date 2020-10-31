package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newSourcePathCmd() *cobra.Command {
	sourcePathCmd := &cobra.Command{
		Use:     "source-path [target]...",
		Short:   "Print the path of a target in the source state",
		Long:    mustGetLongHelp("source-path"),
		Example: getExample("source-path"),
		RunE:    c.makeRunEWithSourceState(c.runSourcePathCmd),
	}

	return sourcePathCmd
}

func (c *Config) runSourcePathCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	if len(args) == 0 {
		return c.writeOutputString(c.absSourceDir + "\n")
	}

	sourcePaths, err := c.getSourcePaths(sourceState, args)
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	for _, sourcePath := range sourcePaths {
		fmt.Fprintln(&sb, sourcePath)
	}
	return c.writeOutputString(sb.String())
}
