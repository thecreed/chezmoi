package cmd

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type managedCmdConfig struct {
	include *chezmoi.IncludeSet
}

func (c *Config) newManagedCmd() *cobra.Command {
	managedCmd := &cobra.Command{
		Use:     "managed",
		Short:   "List the managed entries in the destination directory",
		Long:    mustGetLongHelp("managed"),
		Example: getExample("managed"),
		Args:    cobra.NoArgs,
		RunE:    c.makeRunEWithSourceState(c.runManagedCmd),
	}

	persistentFlags := managedCmd.PersistentFlags()
	persistentFlags.VarP(c.managed.include, "include", "i", "include entry types")

	return managedCmd
}

func (c *Config) runManagedCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	entries := sourceState.Entries()
	targetNames := make([]string, 0, len(entries))
	for targetName, sourceStateEntry := range entries {
		targetStateEntry, err := sourceStateEntry.TargetStateEntry()
		if err != nil {
			return err
		}
		if !c.managed.include.IncludeTargetStateEntry(targetStateEntry) {
			continue
		}
		targetNames = append(targetNames, targetName)
	}

	sort.Strings(targetNames)
	sb := strings.Builder{}
	for _, targetName := range targetNames {
		fmt.Fprintln(&sb, path.Join(c.absDestDir, targetName))
	}
	return c.writeOutputString(sb.String())
}
