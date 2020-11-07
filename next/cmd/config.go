package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/twpayne/go-vfs"
	vfsafero "github.com/twpayne/go-vfsafero"
	"github.com/twpayne/go-xdg/v3"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/twpayne/chezmoi/internal/git"
	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type templateConfig struct {
	Options []string
}

// A Config represents a configuration.
// FIXME organize this better, e.g. move stdin & co next to homeDir & co.
type Config struct {
	version     *semver.Version
	versionInfo VersionInfo
	versionStr  string

	bds *xdg.BaseDirectorySpecification

	configFile   string
	err          error
	fs           vfs.FS
	baseSystem   chezmoi.System
	sourceSystem chezmoi.System
	destSystem   chezmoi.System
	color        bool

	// Global configuration, settable in the config file.
	SourceDir osPath
	DestDir   osPath
	Umask     fileMode
	Format    string
	Remove    bool
	Color     string
	Data      map[string]interface{}
	Template  templateConfig

	// Global configuration, not settable in the config file.
	debug         bool
	dryRun        bool
	force         bool
	keepGoing     bool
	output        string
	verbose       bool
	templateFuncs template.FuncMap

	// Password manager configurations, settable in the config file.
	Bitwarden   bitwardenConfig
	Gopass      gopassConfig
	Keepassxc   keepassxcConfig
	Lastpass    lastpassConfig
	Onepassword onepasswordConfig
	Pass        passConfig
	Secret      secretConfig
	Vault       vaultConfig

	// Password manager data.
	keyring keyringData

	// Command configurations, settable in the config file.
	CD    cdCmdConfig
	Diff  diffCmdConfig
	Edit  editCmdConfig
	Git   gitCmdConfig
	Merge mergeCmdConfig

	// Command configurations, not settable in the config file.
	add             addCmdConfig
	apply           applyCmdConfig
	archive         archiveCmdConfig
	dump            dumpCmdConfig
	executeTemplate executeTemplateCmdConfig
	init            initCmdConfig
	managed         managedCmdConfig
	update          updateCmdConfig
	verify          verifyCmdConfig

	absSourceDir string
	absDestDir   string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	//nolint:structcheck,unused
	ioregData ioregData
}

// A configOption sets and option on a Config.
type configOption func(*Config) error

var (
	persistentStateFilename    = "chezmoistate.boltdb"
	commitMessageTemplateAsset = "assets/templates/COMMIT_MESSAGE.tmpl"

	wellKnownAbbreviations = map[string]struct{}{
		"ANSI": {},
		"CPE":  {},
		"ID":   {},
		"URL":  {},
	}

	identifierRegexp = regexp.MustCompile(`\A[\pL_][\pL\p{Nd}_]*\z`)
	whitespaceRegexp = regexp.MustCompile(`\s+`)

	assets = make(map[string][]byte)
)

// withVersionInfo sets the version information.
func withVersionInfo(versionInfo VersionInfo) configOption {
	return func(c *Config) error {
		var version *semver.Version
		var versionElems []string
		if versionInfo.Version != "" {
			var err error
			version, err = semver.NewVersion(strings.TrimPrefix(versionInfo.Version, "v"))
			if err != nil {
				return err
			}
			versionElems = append(versionElems, version.String())
		} else {
			versionElems = append(versionElems, "dev")
		}
		if versionInfo.Commit != "" {
			versionElems = append(versionElems, "commit "+versionInfo.Commit)
		}
		if versionInfo.Date != "" {
			versionElems = append(versionElems, "built at "+versionInfo.Date)
		}
		if versionInfo.BuiltBy != "" {
			versionElems = append(versionElems, "built by "+versionInfo.BuiltBy)
		}
		c.version = version
		c.versionInfo = versionInfo
		c.versionStr = strings.Join(versionElems, ", ")
		return nil
	}
}

// newConfig creates a new Config with the given options.
func newConfig(options ...configOption) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	bds, err := xdg.NewBaseDirectorySpecification()
	if err != nil {
		return nil, err
	}

	c := &Config{
		bds:        bds,
		fs:         vfs.OSFS,
		configFile: getDefaultConfigFile(bds),
		DestDir:    osPath(homeDir),
		SourceDir:  getDefaultSourceDir(bds),
		Umask:      fileMode(chezmoi.GetUmask()),
		Color:      "auto",
		Format:     "json",
		Diff: diffCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeAll &^ chezmoi.IncludeScripts),
			NoPager: false,
			Pager:   "",
		},
		Edit: editCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeDirs | chezmoi.IncludeFiles | chezmoi.IncludeSymlinks),
		},
		Git: gitCmdConfig{
			Command:    "git",
			AutoAdd:    false,
			AutoCommit: false,
			AutoPush:   false,
		},
		Merge: mergeCmdConfig{
			Command: "vimdiff",
		},
		Template: templateConfig{
			Options: chezmoi.DefaultTemplateOptions,
		},
		templateFuncs: sprig.TxtFuncMap(),
		Bitwarden: bitwardenConfig{
			Command: "bw",
		},
		Gopass: gopassConfig{
			Command: "gopass",
		},
		Keepassxc: keepassxcConfig{
			Command: "keepassxc-cli",
		},
		Lastpass: lastpassConfig{
			Command: "lpass",
		},
		Onepassword: onepasswordConfig{
			Command: "op",
		},
		Pass: passConfig{
			Command: "pass",
		},
		Vault: vaultConfig{
			Command: "vault",
		},
		add: addCmdConfig{
			autoTemplate: false,
			empty:        false,
			encrypt:      false,
			exact:        false,
			include:      chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive:    true,
			template:     false,
		},
		apply: applyCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		archive: archiveCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		dump: dumpCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		managed: managedCmdConfig{
			include: chezmoi.NewIncludeSet(chezmoi.IncludeDirs | chezmoi.IncludeFiles | chezmoi.IncludeSymlinks),
		},
		update: updateCmdConfig{
			apply:     true,
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll),
			recursive: true,
		},
		verify: verifyCmdConfig{
			include:   chezmoi.NewIncludeSet(chezmoi.IncludeAll &^ chezmoi.IncludeScripts),
			recursive: true,
		},
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	for key, value := range map[string]interface{}{
		"bitwarden":                c.bitwardenFunc,
		"bitwardenFields":          c.bitwardenFieldsFunc,
		"gopass":                   c.gopassFunc,
		"include":                  c.includeFunc,
		"ioreg":                    c.ioregFunc,
		"joinPath":                 c.joinPathFunc,
		"keepassxc":                c.keepassxcFunc,
		"keepassxcAttribute":       c.keepassxcAttributeFunc,
		"keyring":                  c.keyringFunc,
		"lastpass":                 c.lastpassFunc,
		"lastpassRaw":              c.lastpassRawFunc,
		"lookPath":                 c.lookPathFunc,
		"onepassword":              c.onepasswordFunc,
		"onepasswordDetailsFields": c.onepasswordDetailsFieldsFunc,
		"onepasswordDocument":      c.onepasswordDocumentFunc,
		"pass":                     c.passFunc,
		"secret":                   c.secretFunc,
		"secretJSON":               c.secretJSONFunc,
		"stat":                     c.statFunc,
		"vault":                    c.vaultFunc,
	} {
		c.addTemplateFunc(key, value)
	}

	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Config) addTemplateFunc(key string, value interface{}) {
	if _, ok := c.templateFuncs[key]; ok {
		panic(fmt.Sprintf("%s: already defined", key))
	}
	c.templateFuncs[key] = value
}

func (c *Config) applyArgs(targetSystem chezmoi.System, targetDir string, args []string, include *chezmoi.IncludeSet, recursive bool, umask os.FileMode) error {
	s, err := c.getSourceState()
	if err != nil {
		return err
	}

	applyOptions := chezmoi.ApplyOptions{
		Include: include,
		Umask:   umask,
	}

	var targetNames []string
	if len(args) == 0 {
		targetNames = s.AllTargetNames()
	} else {
		targetNames, err = c.getTargetNames(s, args, getTargetNamesOptions{
			mustBeInSourceState: true,
			recursive:           recursive,
		})
		if err != nil {
			return err
		}
	}

	for _, targetName := range targetNames {
		if err := s.ApplyOne(targetSystem, targetDir, targetName, applyOptions); err != nil {
			if c.keepGoing {
				c.errorf("%v", err)
			} else {
				return err
			}
		}
	}

	return nil
}

func (c *Config) cmdOutput(dir, name string, args []string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		var err error
		cmd.Dir, err = c.baseSystem.RawPath(dir)
		if err != nil {
			return nil, err
		}
	}
	return c.baseSystem.IdempotentCmdOutput(cmd)
}

func (c *Config) errorf(format string, args ...interface{}) {
	fmt.Fprintf(c.stderr, "chezmoi: "+format, args...)
}

func (c *Config) execute(args []string) error {
	rootCmd, err := c.newRootCmd()
	if err != nil {
		return err
	}
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func (c *Config) getDefaultTemplateData() (map[string]interface{}, error) {
	data := map[string]interface{}{
		"arch":      runtime.GOARCH,
		"os":        runtime.GOOS,
		"sourceDir": c.SourceDir,
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	data["username"] = currentUser.Username

	// user.LookupGroupId is generally unreliable:
	//
	// If CGO is enabled, then this uses an underlying C library call (e.g.
	// getgrgid_r on Linux) and is trustworthy, except on recent versions of Go
	// on Android, where LookupGroupId is not implemented.
	//
	// If CGO is disabled then the fallback implementation only searches
	// /etc/group, which is typically empty if an external directory service is
	// being used, and so the lookup fails.
	//
	// So, only set group if user.LookupGroupId does not return an error.
	group, err := user.LookupGroupId(currentUser.Gid)
	if err == nil {
		data["group"] = group.Name
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data["homedir"] = homedir

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	data["fullHostname"] = hostname
	data["hostname"] = strings.SplitN(hostname, ".", 2)[0]

	osRelease, err := getOSRelease(c.baseSystem)
	if err == nil {
		if osRelease != nil {
			data["osRelease"] = upperSnakeCaseToCamelCaseMap(osRelease)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	kernelInfo, err := getKernelInfo(c.baseSystem)
	if err == nil && kernelInfo != nil {
		data["kernel"] = kernelInfo
	} else if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"chezmoi": data,
	}, nil
}

func (c *Config) getDestPath(arg osPath) (string, error) {
	path, err := arg.AbsSlash()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(path, c.absDestDir+"/") {
		return "", fmt.Errorf("%s: not in destination directory (%s)", arg, c.absDestDir) // FIXME
	}
	return path, nil
}

func (c *Config) getDestPathInfos(sourceState *chezmoi.SourceState, args []string, recursive bool) (map[string]os.FileInfo, error) {
	destPathInfos := make(map[string]os.FileInfo)
	for _, arg := range args {
		destPath, err := c.getDestPath(osPath(arg))
		if err != nil {
			return nil, err
		}
		if recursive {
			if err := vfs.WalkSlash(c.destSystem, destPath, func(destPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				return sourceState.AddDestPathInfos(destPathInfos, c.destSystem, destPath, info)
			}); err != nil {
				return nil, err
			}
		} else {
			if err := sourceState.AddDestPathInfos(destPathInfos, c.destSystem, destPath, nil); err != nil {
				return nil, err
			}
		}
	}
	return destPathInfos, nil
}

// getEditor returns the path to the user's editor and any extra arguments.
func (c *Config) getEditor() (string, []string) {
	// If the user has set and edit command then use it.
	if c.Edit.Command != "" {
		return c.Edit.Command, c.Edit.Args
	}

	// Prefer $VISUAL over $EDITOR and fallback to vi.
	editor := firstNonEmptyString(
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		"vi",
	)

	// If editor is found, return it.
	if path, err := exec.LookPath(editor); err == nil {
		return path, nil
	}

	// Otherwise, if editor contains spaces, then assume that the first word is
	// the editor and the rest are arguments.
	components := whitespaceRegexp.Split(editor, -1)
	if len(components) > 1 {
		if path, err := exec.LookPath(components[0]); err == nil {
			return path, components[1:]
		}
	}

	// Fallback to editor only.
	return editor, nil
}

func (c *Config) getPersistentState(options *bolt.Options) (chezmoi.PersistentState, error) {
	persistentStateFile := c.getPersistentStateFile()
	if options == nil {
		options = &bolt.Options{}
	}
	if options.Timeout == 0 {
		options.Timeout = 2 * time.Second
	}
	if c.dryRun {
		options.ReadOnly = true
	}
	return chezmoi.NewBoltPersistentState(c.fs, persistentStateFile, options)
}

func (c *Config) getPersistentStateFile() string {
	if c.configFile != "" {
		return filepath.Join(filepath.Dir(c.configFile), persistentStateFilename)
	}
	for _, configDir := range c.bds.ConfigDirs {
		persistentStateFile := filepath.Join(configDir, "chezmoi", persistentStateFilename)
		if _, err := os.Stat(persistentStateFile); err == nil {
			return persistentStateFile
		}
	}
	return filepath.Join(filepath.Dir(getDefaultConfigFile(c.bds)), persistentStateFilename)
}

func (c *Config) getSourcePaths(s *chezmoi.SourceState, args []string) ([]string, error) {
	targetNames, err := c.getTargetNames(s, args, getTargetNamesOptions{
		mustBeInSourceState: true,
		recursive:           false,
	})
	if err != nil {
		return nil, err
	}
	sourcePaths := make([]string, 0, len(targetNames))
	for _, targetName := range targetNames {
		sourcePath := s.MustEntry(targetName).Path()
		sourcePaths = append(sourcePaths, sourcePath)
	}
	return sourcePaths, nil
}

func (c *Config) getSourceState() (*chezmoi.SourceState, error) {
	defaultTemplateData, err := c.getDefaultTemplateData()
	if err != nil {
		return nil, err
	}

	s := chezmoi.NewSourceState(
		chezmoi.WithDestDir(c.absDestDir),
		chezmoi.WithPriorityTemplateData(c.Data),
		chezmoi.WithSourceDir(c.absSourceDir),
		chezmoi.WithSystem(c.sourceSystem),
		chezmoi.WithTemplateData(defaultTemplateData),
		chezmoi.WithTemplateFuncs(c.templateFuncs),
		chezmoi.WithTemplateOptions(c.Template.Options),
	)

	if err := s.Read(); err != nil {
		return nil, err
	}

	if minVersion := s.MinVersion(); c.version != nil && c.version.LessThan(minVersion) {
		return nil, fmt.Errorf("source state requires version %s or later, chezmoi is version %s", minVersion, c.version)
	}

	return s, nil
}

func (c *Config) getTargetName(arg osPath) (string, error) {
	destPath, err := c.getDestPath(arg)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(destPath, c.absDestDir+"/"), nil
}

type getTargetNamesOptions struct {
	mustBeInSourceState bool
	recursive           bool
}

func (c *Config) getTargetNames(s *chezmoi.SourceState, args []string, options getTargetNamesOptions) ([]string, error) {
	targetNames := make([]string, 0, len(args))
	for _, arg := range args {
		targetName, err := c.getTargetName(osPath(arg))
		if err != nil {
			return nil, err
		}
		if options.mustBeInSourceState {
			if _, ok := s.Entry(targetName); !ok {
				return nil, fmt.Errorf("%s: not in source state", arg)
			}
		}
		targetNames = append(targetNames, targetName)
		if options.recursive {
			targetNamePrefix := targetName + "/"
			for _, targetName := range s.TargetNames() {
				if strings.HasPrefix(targetName, targetNamePrefix) {
					targetNames = append(targetNames, targetName)
				}
			}
		}
	}

	if len(targetNames) == 0 {
		return nil, nil
	}

	// Sort and de-duplicate targetNames in place.
	sort.Strings(targetNames)
	n := 1
	for i := 1; i < len(targetNames); i++ {
		if targetNames[i] != targetNames[i-1] {
			targetNames[n] = targetNames[i]
			n++
		}
	}
	return targetNames[:n], nil
}

func (c *Config) gitAutoAdd() (*git.Status, error) {
	if err := c.run(c.absSourceDir, c.Git.Command, []string{"add", "."}); err != nil {
		return nil, err
	}
	output, err := c.cmdOutput(c.absSourceDir, c.Git.Command, []string{"status", "--porcelain=v2"})
	if err != nil {
		return nil, err
	}
	return git.ParseStatusPorcelainV2(output)
}

func (c *Config) gitAutoCommit(status *git.Status) error {
	if status.Empty() {
		return nil
	}
	commitMessageText, err := getAsset(commitMessageTemplateAsset)
	if err != nil {
		return err
	}
	commitMessageTmpl, err := template.New("commit_message").Funcs(c.templateFuncs).Parse(string(commitMessageText))
	if err != nil {
		return err
	}
	commitMessage := strings.Builder{}
	if err := commitMessageTmpl.Execute(&commitMessage, status); err != nil {
		return err
	}
	return c.run(c.absSourceDir, c.Git.Command, []string{"commit", "--message", commitMessage.String()})
}

func (c *Config) gitAutoPush(status *git.Status) error {
	if status.Empty() {
		return nil
	}
	return c.run(c.absSourceDir, c.Git.Command, []string{"push"})
}

func (c *Config) makeRunEWithSourceState(runE func(*cobra.Command, []string, *chezmoi.SourceState) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		sourceState, err := c.getSourceState()
		if err != nil {
			return err
		}
		return runE(cmd, args, sourceState)
	}
}

func (c *Config) newRootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:                "chezmoi",
		Short:              "Manage your dotfiles across multiple diverse machines, securely",
		Version:            c.versionStr,
		PersistentPreRunE:  c.persistentPreRunRootE,
		PersistentPostRunE: c.persistentPostRunRootE,
		SilenceErrors:      true,
		SilenceUsage:       true,
	}

	persistentFlags := rootCmd.PersistentFlags()

	persistentFlags.StringVar(&c.Color, "color", c.Color, "colorize diffs")
	persistentFlags.VarP(&c.DestDir, "destination", "D", "destination directory")
	persistentFlags.StringVar(&c.Format, "format", c.Format, "format ("+serializationFormatNamesStr()+")")
	persistentFlags.BoolVar(&c.Remove, "remove", c.Remove, "remove targets")
	persistentFlags.VarP(&c.SourceDir, "source", "S", "source directory")
	for _, key := range []string{
		"color",
		"destination",
		"format",
		"remove",
		"source",
	} {
		if err := viper.BindPFlag(key, persistentFlags.Lookup(key)); err != nil {
			return nil, err
		}
	}

	persistentFlags.StringVarP(&c.configFile, "config", "c", c.configFile, "config file")
	persistentFlags.BoolVarP(&c.dryRun, "dry-run", "n", c.dryRun, "dry run")
	persistentFlags.BoolVar(&c.force, "force", c.force, "force")
	persistentFlags.BoolVarP(&c.keepGoing, "keep-going", "k", c.keepGoing, "keep going as far as possible after an error")
	persistentFlags.BoolVarP(&c.verbose, "verbose", "v", c.verbose, "verbose")
	persistentFlags.StringVarP(&c.output, "output", "o", c.output, "output file")
	persistentFlags.BoolVar(&c.debug, "debug", c.debug, "write debug logs")

	for _, err := range []error{
		rootCmd.MarkPersistentFlagFilename("config"),
		rootCmd.MarkPersistentFlagDirname("destination"),
		rootCmd.MarkPersistentFlagFilename("output"),
		rootCmd.MarkPersistentFlagDirname("source"),
	} {
		if err != nil {
			return nil, err
		}
	}

	// FIXME this shouldn't be global
	// FIXME move it to c.persistentPreRunRootE
	cobra.OnInitialize(func() {
		v := viper.New()
		v.SetConfigFile(c.configFile)
		err := v.ReadInConfig()
		if os.IsNotExist(err) {
			return
		}
		c.err = err
		if c.err == nil {
			c.err = v.Unmarshal(c)
		}
		if c.err == nil {
			c.err = c.validateData()
		}
		if c.err != nil {
			rootCmd.Printf("warning: %s: %v\n", c.configFile, c.err)
		}
	})

	rootCmd.SetHelpCommand(c.newHelpCmd())
	for _, newCmdFunc := range []func() *cobra.Command{
		c.newAddCmd,
		c.newApplyCmd,
		c.newArchiveCmd,
		c.newCatCmd,
		c.newCDCmd,
		c.newChattrCmd,
		c.newCompletionCmd,
		c.newDataCmd,
		c.newDiffCmd,
		c.newDocsCmd,
		c.newDoctorCmd,
		c.newDumpCmd,
		c.newEditCmd,
		c.newEditConfigCmd,
		c.newExecuteTemplateCmd,
		c.newForgetCmd,
		c.newGitCmd,
		// c.newImportCmd, // FIXME
		c.newInitCmd,
		c.newManagedCmd,
		c.newMergeCmd,
		c.newPurgeCmd,
		c.newRemoveCmd,
		c.newSourcePathCmd,
		c.newStateCmd,
		c.newStatusCmd,
		c.newUnmanagedCmd,
		c.newUpdateCmd,
		c.newVerifyCmd,
	} {
		rootCmd.AddCommand(newCmdFunc())
	}

	return rootCmd, nil
}

func (c *Config) persistentPreRunRootE(cmd *cobra.Command, args []string) error {
	if c.Color == "auto" {
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			c.color = false
		} else if stdout, ok := c.stdout.(*os.File); ok {
			c.color = terminal.IsTerminal(int(stdout.Fd()))
		} else {
			c.color = false
		}
	} else if color, err := parseBool(c.Color); err == nil {
		c.color = color
	} else {
		return fmt.Errorf("%s: invalid color value", c.Color)
	}

	if c.color {
		if err := enableVirtualTerminalProcessing(c.stdout); err != nil {
			return err
		}
	}

	var err error
	c.absSourceDir, err = c.SourceDir.AbsSlash()
	if err != nil {
		return err
	}
	c.absDestDir, err = c.DestDir.AbsSlash()
	if err != nil {
		return err
	}

	persistentState, err := c.getPersistentState(nil)
	if err != nil {
		return err
	}
	c.baseSystem = chezmoi.NewRealSystem(c.fs, persistentState)
	c.sourceSystem = c.baseSystem
	c.destSystem = c.baseSystem
	// FIXME maybe re-order this graph of systems?
	if !getBoolAnnotation(cmd, modifiesDestinationDirectory) {
		c.destSystem = chezmoi.NewReadOnlySystem(c.destSystem)
	}
	if !getBoolAnnotation(cmd, modifiesSourceDirectory) {
		c.sourceSystem = chezmoi.NewReadOnlySystem(c.sourceSystem)
	}
	if c.dryRun {
		c.sourceSystem = chezmoi.NewDryRunSystem(c.sourceSystem)
		c.destSystem = chezmoi.NewDryRunSystem(c.destSystem)
	}
	if c.verbose {
		c.sourceSystem = chezmoi.NewGitDiffSystem(c.sourceSystem, c.stdout, c.absSourceDir+"/", c.color)
		c.destSystem = chezmoi.NewGitDiffSystem(c.destSystem, c.stdout, c.absDestDir+"/", c.color)
	}
	if c.debug {
		logger := log.New(c.stderr, "chezmoi: ", log.LstdFlags|log.Lmsgprefix)
		c.baseSystem = chezmoi.NewDebugSystem(c.baseSystem, logger)
		c.sourceSystem = chezmoi.NewDebugSystem(c.sourceSystem, logger)
		c.destSystem = chezmoi.NewDebugSystem(c.destSystem, logger)
	}

	if !getBoolAnnotation(cmd, doesNotRequireValidConfig) {
		if c.err != nil {
			return errors.New("invalid config, aborting")
		}
	}

	if getBoolAnnotation(cmd, requiresConfigDirectory) {
		if err := vfs.MkdirAll(c.baseSystem, filepath.Dir(c.configFile), 0o777); err != nil {
			return err
		}
	}

	if getBoolAnnotation(cmd, requiresSourceDirectory) {
		if err := vfs.MkdirAll(c.baseSystem, c.absSourceDir, 0o777); err != nil {
			return err
		}
	}

	if getBoolAnnotation(cmd, runsCommands) {
		if runtime.GOOS == "linux" && c.bds.RuntimeDir != "" {
			// Snap sets the $XDG_RUNTIME_DIR environment variable to
			// /run/user/$uid/snap.$snap_name, but does not create this
			// directory. Consequently, any spawned processes that need
			// $XDG_DATA_DIR will fail. As a work-around, create the directory
			// if it does not exist. See
			// https://forum.snapcraft.io/t/wayland-dconf-and-xdg-runtime-dir/186/13.
			if err := vfs.MkdirAll(c.baseSystem, c.bds.RuntimeDir, 0o700); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) persistentPostRunRootE(cmd *cobra.Command, args []string) error {
	if getBoolAnnotation(cmd, modifiesConfigFile) {
		// Warn the user of any errors reading the config file.
		v := viper.New()
		v.SetFs(vfsafero.NewAferoFS(c.fs))
		v.SetConfigFile(c.configFile)
		err := v.ReadInConfig()
		if err == nil {
			err = v.Unmarshal(&Config{})
		}
		if err != nil {
			cmd.Printf("warning: %s: %v\n", c.configFile, err)
		}
	}

	if getBoolAnnotation(cmd, modifiesSourceDirectory) {
		var err error
		var status *git.Status
		if c.Git.AutoAdd || c.Git.AutoCommit || c.Git.AutoPush {
			status, err = c.gitAutoAdd()
			if err != nil {
				return err
			}
		}
		if c.Git.AutoCommit || c.Git.AutoPush {
			if err := c.gitAutoCommit(status); err != nil {
				return err
			}
		}
		if c.Git.AutoPush {
			if err := c.gitAutoPush(status); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) prompt(s, choices string) (byte, error) {
	r := bufio.NewReader(c.stdin)
	for {
		_, err := fmt.Printf("%s [%s]? ", s, strings.Join(strings.Split(choices, ""), ","))
		if err != nil {
			return 0, err
		}
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimSpace(line)
		if len(line) == 1 && strings.IndexByte(choices, line[0]) != -1 {
			return line[0], nil
		}
	}
}

func (c *Config) run(dir, name string, args []string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		var err error
		cmd.Dir, err = c.baseSystem.RawPath(dir)
		if err != nil {
			return err
		}
	}
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return c.baseSystem.RunCmd(cmd)
}

func (c *Config) runEditor(args []string) error {
	editor, editorArgs := c.getEditor()
	return c.run("", editor, append(editorArgs, args...))
}

func (c *Config) marshal(data interface{}) error {
	format, ok := chezmoi.Formats[c.Format]
	if !ok {
		return fmt.Errorf("%s: unknown format", c.Format)
	}
	marshaledData, err := format.Marshal(data)
	if err != nil {
		return err
	}
	return c.writeOutput(marshaledData)
}

func (c *Config) validateData() error {
	return validateKeys(c.Data, identifierRegexp)
}

func (c *Config) writeOutput(data []byte) error {
	if c.output == "" || c.output == "-" {
		_, err := c.stdout.Write(data)
		return err
	}
	return c.baseSystem.WriteFile(c.output, data, 0o666)
}

func (c *Config) writeOutputString(data string) error {
	return c.writeOutput([]byte(data))
}
