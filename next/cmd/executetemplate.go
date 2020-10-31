package cmd

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type executeTemplateCmdConfig struct {
	init         bool
	promptBool   map[string]string
	promptInt    map[string]int
	promptString map[string]string
}

func (c *Config) newExecuteTemplateCmd() *cobra.Command {
	executeTemplateCmd := &cobra.Command{
		Use:     "execute-template [template]...",
		Short:   "Execute the given template(s)",
		Long:    mustGetLongHelp("execute-template"),
		Example: getExample("execute-template"),
		RunE:    c.makeRunEWithSourceState(c.runExecuteTemplateCmd),
	}

	persistentFlags := executeTemplateCmd.PersistentFlags()
	persistentFlags.BoolVarP(&c.executeTemplate.init, "init", "i", c.executeTemplate.init, "simulate chezmoi init")
	persistentFlags.StringToStringVar(&c.executeTemplate.promptBool, "promptBool", c.executeTemplate.promptBool, "simulate promptBool")
	persistentFlags.StringToIntVar(&c.executeTemplate.promptInt, "promptInt", c.executeTemplate.promptInt, "simulate promptInt")
	persistentFlags.StringToStringVarP(&c.executeTemplate.promptString, "promptString", "p", c.executeTemplate.promptString, "simulate promptString")

	return executeTemplateCmd
}

func (c *Config) runExecuteTemplateCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	promptBool := make(map[string]bool)
	for key, valueStr := range c.executeTemplate.promptBool {
		value, err := parseBool(valueStr)
		if err != nil {
			return err
		}
		promptBool[key] = value
	}
	if c.executeTemplate.init {
		for name, f := range map[string]interface{}{
			"promptBool": func(prompt string) bool {
				return promptBool[prompt]
			},
			"promptInt": func(prompt string) int {
				return c.executeTemplate.promptInt[prompt]
			},
			"promptString": func(prompt string) string {
				if value, ok := c.executeTemplate.promptString[prompt]; ok {
					return value
				}
				return prompt
			},
		} {
			c.templateFuncs[name] = f
		}
	}

	if len(args) == 0 {
		data, err := ioutil.ReadAll(c.stdin)
		if err != nil {
			return err
		}
		output, err := sourceState.ExecuteTemplateData("stdin", data)
		if err != nil {
			return err
		}
		return c.writeOutput(output)
	}

	output := strings.Builder{}
	for i, arg := range args {
		result, err := sourceState.ExecuteTemplateData("arg"+strconv.Itoa(i+1), []byte(arg))
		if err != nil {
			return err
		}
		if _, err := output.Write(result); err != nil {
			return err
		}
	}
	return c.writeOutputString(output.String())
}
