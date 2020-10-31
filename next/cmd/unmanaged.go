package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	vfs "github.com/twpayne/go-vfs"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

func (c *Config) newUnmanagedCmd() *cobra.Command {
	unmanagedCmd := &cobra.Command{
		Use:     "unmanaged",
		Short:   "List the unmanaged files in the destination directory",
		Long:    mustGetLongHelp("unmanaged"),
		Example: getExample("unmanaged"),
		Args:    cobra.NoArgs,
		RunE:    c.makeRunEWithSourceState(c.runUnmanagedCmd),
	}
	return unmanagedCmd
}

func (c *Config) runUnmanagedCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	sb := strings.Builder{}
	if err := vfs.WalkSlash(c.destSystem, c.absDestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == c.absDestDir {
			return nil
		}
		targetName := strings.TrimPrefix(path, c.absDestDir+"/")
		_, managed := sourceState.Entry(targetName)
		ignored := sourceState.Ignored(targetName)
		if !managed && !ignored {
			sb.WriteString(targetName + "\n")
		}
		if info.IsDir() && (!managed || ignored) {
			return vfs.SkipDir
		}
		return nil
	}); err != nil {
		return err
	}
	return c.writeOutputString(sb.String())
}
