package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type diffCmdConfig struct {
	include   *chezmoi.IncludeSet
	recursive bool
	NoPager   bool
	Pager     string
}

func (c *Config) newDiffCmd() *cobra.Command {
	diffCmd := &cobra.Command{
		Use:     "diff [target]...",
		Short:   "Print the diff between the target state and the destination state",
		Long:    mustGetLongHelp("diff"),
		Example: getExample("diff"),
		RunE:    c.runDiffCmd,
	}

	persistentFlags := diffCmd.PersistentFlags()
	persistentFlags.VarP(c.Diff.include, "include", "i", "include entry types")
	persistentFlags.BoolVar(&c.Diff.NoPager, "no-pager", c.Diff.NoPager, "disable pager")
	persistentFlags.BoolVarP(&c.Diff.recursive, "recursive", "r", c.Diff.recursive, "recursive")

	return diffCmd
}

func (c *Config) runDiffCmd(cmd *cobra.Command, args []string) error {
	sb := strings.Builder{}
	gitDiffSystem := chezmoi.NewGitDiffSystem(chezmoi.NewDryRunSystem(c.destSystem), &sb, c.absDestDir+"/", c.color)
	if err := c.applyArgs(gitDiffSystem, c.absDestDir, args, c.Diff.include, c.Diff.recursive, c.Umask.FileMode()); err != nil {
		return err
	}
	return c.writeOutputString(sb.String())
}
