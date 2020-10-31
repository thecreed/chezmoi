package cmd

import (
	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type addCmdConfig struct {
	autoTemplate bool
	empty        bool
	encrypt      bool
	exact        bool
	include      *chezmoi.IncludeSet
	recursive    bool
	template     bool
}

func (c *Config) newAddCmd() *cobra.Command {
	addCmd := &cobra.Command{
		Use:     "add targets...",
		Aliases: []string{"manage"},
		Short:   "Add an existing file, directory, or symlink to the source state",
		Long:    mustGetLongHelp("add"),
		Example: getExample("add"),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.makeRunEWithSourceState(c.runAddCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
			requiresSourceDirectory: "true",
		},
	}

	persistentFlags := addCmd.PersistentFlags()
	persistentFlags.BoolVarP(&c.add.autoTemplate, "autotemplate", "a", c.add.autoTemplate, "auto generate the template when adding files as templates")
	persistentFlags.BoolVarP(&c.add.empty, "empty", "e", c.add.empty, "add empty files")
	persistentFlags.BoolVar(&c.add.encrypt, "encrypt", c.add.encrypt, "encrypt files")
	persistentFlags.BoolVarP(&c.add.exact, "exact", "x", c.add.exact, "add directories exactly")
	persistentFlags.BoolVarP(&c.add.recursive, "recursive", "r", c.add.recursive, "recursive")
	persistentFlags.BoolVarP(&c.add.template, "template", "T", c.add.template, "add files as templates")

	return addCmd
}

func (c *Config) runAddCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	destPathInfos, err := c.getDestPathInfos(sourceState, args, c.add.recursive)
	if err != nil {
		return err
	}

	return sourceState.Add(c.sourceSystem, destPathInfos, &chezmoi.AddOptions{
		AutoTemplate: c.add.autoTemplate,
		Empty:        c.add.empty,
		Encrypt:      c.add.encrypt,
		Exact:        c.add.exact,
		Include:      c.add.include,
		Template:     c.add.template,
	})
}
