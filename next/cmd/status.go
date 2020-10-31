package cmd

import (
	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status [target]...",
		Short: "Show the status of targets",
		// Long: mustGetLongHelp("status"), // FIXME
		Example: getExample("status"),
		RunE:    c.makeRunEWithSourceState(c.runStatusCmd),
	}

	return statusCmd
}

func (c *Config) runStatusCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	// FIXME
	return nil
}
