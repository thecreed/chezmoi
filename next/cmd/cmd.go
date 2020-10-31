package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

const (
	doesNotRequireValidConfig    = "chezmoi_annotation_does_not_require_valid_config"
	modifiesConfigFile           = "chezmoi_annotation_modifies_config_file"
	modifiesDestinationDirectory = "chezmoi_annotation_modifies_destination_directory"
	modifiesSourceDirectory      = "chezmoi_annotation_modifies_source_directory"
	requiresConfigDirectory      = "chezmoi_annotation_requires_config_directory"
	requiresSourceDirectory      = "chezmoi_annotation_requires_source_directory"
	runsCommands                 = "chezmoi_annotation_runs_commands"
)

// An ErrExitCode indicates the the main program should exit with the given
// code.
type ErrExitCode int

func (e ErrExitCode) Error() string { return "" }

// A VersionInfo contains a version.
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

// Main runs chezmoi and returns an exit code.
func Main(versionInfo VersionInfo, args []string) int {
	if err := runMain(versionInfo, args); err != nil {
		if s := err.Error(); s != "" {
			fmt.Fprintf(os.Stderr, "chezmoi: %s\n", s)
		}
		errExitCode := ErrExitCode(1)
		_ = errors.As(err, &errExitCode)
		return int(errExitCode)
	}
	return 0
}

func runMain(versionInfo VersionInfo, args []string) error {
	config, err := newConfig(
		withVersionInfo(versionInfo),
	)
	if err != nil {
		return err
	}
	return config.execute(args)
}

func getAsset(name string) ([]byte, error) {
	asset, ok := assets[name]
	if !ok {
		return nil, fmt.Errorf("%s: not found", name)
	}
	return asset, nil
}

func getBoolAnnotation(cmd *cobra.Command, key string) bool {
	value, ok := cmd.Annotations[key]
	if !ok {
		return false
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}
	return boolValue
}

func getExample(command string) string {
	return helps[command].example
}

func mustGetLongHelp(command string) string {
	help, ok := helps[command]
	if !ok {
		panic(fmt.Sprintf("%s: no long help", command))
	}
	return help.long
}
