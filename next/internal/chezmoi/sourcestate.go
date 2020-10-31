package chezmoi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	vfs "github.com/twpayne/go-vfs"
	"go.uber.org/multierr"
)

// An Lstater implements Lstat.
type Lstater interface {
	Lstat(name string) (os.FileInfo, error)
}

// A SourceState is a source state.
type SourceState struct {
	entries              map[string]SourceStateEntry
	system               System
	sourceDir            string
	destDir              string
	umask                os.FileMode
	encryptionTool       EncryptionTool
	ignore               *PatternSet
	minVersion           semver.Version
	priorityTemplateData map[string]interface{}
	templateData         map[string]interface{}
	templateFuncs        template.FuncMap
	templateOptions      []string
	templates            map[string]*template.Template
}

// A SourceStateOption sets an option on a source state.
type SourceStateOption func(*SourceState)

// WithDestDir sets the destination directory.
func WithDestDir(destDir string) SourceStateOption {
	return func(s *SourceState) {
		s.destDir = destDir
	}
}

// WithEncryptionTool set the encryption tool.
func WithEncryptionTool(encryptionTool EncryptionTool) SourceStateOption {
	return func(s *SourceState) {
		s.encryptionTool = encryptionTool
	}
}

// WithPriorityTemplateData adds priority template data.
func WithPriorityTemplateData(priorityTemplateData map[string]interface{}) SourceStateOption {
	return func(s *SourceState) {
		recursiveMerge(s.priorityTemplateData, priorityTemplateData)
		recursiveMerge(s.templateData, s.priorityTemplateData)
	}
}

// WithSourceDir sets the source directory.
func WithSourceDir(sourceDir string) SourceStateOption {
	return func(s *SourceState) {
		s.sourceDir = sourceDir
	}
}

// WithSystem sets the system.
func WithSystem(system System) SourceStateOption {
	return func(s *SourceState) {
		s.system = system
	}
}

// WithTemplateData adds template data.
func WithTemplateData(templateData map[string]interface{}) SourceStateOption {
	return func(s *SourceState) {
		recursiveMerge(s.templateData, templateData)
		recursiveMerge(s.templateData, s.priorityTemplateData)
	}
}

// WithTemplateFuncs sets the template functions.
func WithTemplateFuncs(templateFuncs template.FuncMap) SourceStateOption {
	return func(s *SourceState) {
		s.templateFuncs = templateFuncs
	}
}

// WithTemplateOptions sets the template options.
func WithTemplateOptions(templateOptions []string) SourceStateOption {
	return func(s *SourceState) {
		s.templateOptions = templateOptions
	}
}

// WithUmask sets the umask.
func WithUmask(umask os.FileMode) SourceStateOption {
	return func(s *SourceState) {
		s.umask = umask
	}
}

// NewSourceState creates a new source state with the given options.
func NewSourceState(options ...SourceStateOption) *SourceState {
	s := &SourceState{
		entries:              make(map[string]SourceStateEntry),
		umask:                GetUmask(),
		encryptionTool:       &nullEncryptionTool{},
		ignore:               NewPatternSet(),
		priorityTemplateData: make(map[string]interface{}),
		templateData:         make(map[string]interface{}),
		templateOptions:      DefaultTemplateOptions,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// AddOptions are options to SourceState.Add.
type AddOptions struct {
	AutoTemplate bool
	Empty        bool
	Encrypt      bool
	Exact        bool
	Include      *IncludeSet
	UpdateState  bool
	Template     bool
	umask        os.FileMode
}

// Add adds sourceStateEntry to s.
func (s *SourceState) Add(sourceSystem System, destPathInfos map[string]os.FileInfo, options *AddOptions) error {
	destPaths := make([]string, 0, len(destPathInfos))
	for destPath := range destPathInfos {
		destPaths = append(destPaths, destPath)
	}
	sort.Strings(destPaths)

	newSourceStateEntries := make(map[string]SourceStateEntry)
	newSourceStateEntriesByTargetName := make(map[string]SourceStateEntry)
	for _, destPath := range destPaths {
		destPathInfo := destPathInfos[destPath]
		if !options.Include.IncludeFileInfo(destPathInfo) {
			continue
		}
		targetName := mustTrimPrefix(destPath, s.destDir+"/")

		// Find the target's parent directory.
		var parentDir string
		if parentDirTargetName := path.Dir(targetName); parentDirTargetName == "." {
			parentDir = ""
		} else if parentDirEntry, ok := newSourceStateEntriesByTargetName[parentDirTargetName]; ok {
			parentDir = parentDirEntry.Path()
		} else if parentDirEntry, ok := s.entries[parentDirTargetName]; ok {
			parentDir = parentDirEntry.Path()
		} else {
			return fmt.Errorf("%s: parent directory not in source state", destPath)
		}

		destStateEntry, err := NewDestStateEntry(sourceSystem, destPath)
		if err != nil {
			return err
		}
		newSourceStateEntry, err := s.sourceStateEntry(destStateEntry, destPath, destPathInfo, parentDir, options)
		if err != nil {
			return err
		}
		if newSourceStateEntry == nil {
			continue
		}

		newName := newSourceStateEntry.Path()
		if oldSourceStateEntry, ok := s.entries[targetName]; ok {
			// If both the new and old source state entries are directories but the name has changed,
			// rename to avoid losing the directory's contents. Otherwise,
			// remove the old.
			oldName := mustTrimPrefix(oldSourceStateEntry.Path(), s.sourceDir+"/")
			if newName != oldName {
				_, newIsDir := newSourceStateEntry.(*SourceStateDir)
				_, oldIsDir := oldSourceStateEntry.(*SourceStateDir)
				if newIsDir && oldIsDir {
					newSourceStateEntry = &SourceStateRenameDir{
						oldName: oldName,
						newName: newName,
					}
				} else {
					newSourceStateEntries[oldName] = &SourceStateRemove{}
				}
			}
		}
		newSourceStateEntries[newName] = newSourceStateEntry
		newSourceStateEntriesByTargetName[targetName] = newSourceStateEntry
	}

	targetSourceState := &SourceState{
		entries: newSourceStateEntries,
	}
	return targetSourceState.applyAll(sourceSystem, s.sourceDir, ApplyOptions{
		Include: options.Include,
		Umask:   options.umask,
	})
}

// AddDestPathInfos adds an os.FileInfo to destPathInfos for destPath and any of
// its parents which are not already known.
func (s *SourceState) AddDestPathInfos(destPathInfos map[string]os.FileInfo, lstater Lstater, destPath string, info os.FileInfo) error {
	if !strings.HasPrefix(destPath, s.destDir+"/") {
		return fmt.Errorf("%s: outside destination directory (%s)", destPath, s.destDir) // FIXME
	}

	if _, ok := destPathInfos[destPath]; ok {
		return nil
	}

	if info == nil {
		var err error
		info, err = lstater.Lstat(destPath)
		if err != nil {
			return err
		}
	}

	destPathInfos[destPath] = info
	destPath = path.Dir(destPath)
	if destPath == s.destDir {
		return nil
	}

	for {
		if _, ok := destPathInfos[destPath]; ok {
			return nil
		}
		info, err := lstater.Lstat(destPath)
		if err != nil {
			return err
		}
		destPathInfos[destPath] = info
		parentDir := path.Dir(destPath)
		if parentDir == s.destDir {
			return nil
		}
		if _, ok := s.entries[parentDir]; ok {
			return nil
		}
		destPath = parentDir
	}
}

// ApplyOptions are options to SourceState.ApplyAll and SourceState.ApplyOne.
type ApplyOptions struct {
	Include     *IncludeSet
	Umask       os.FileMode
	UpdateState bool
}

// AllTargetNames returns all of s's target names in order.
func (s *SourceState) AllTargetNames() []string {
	targetNames := make([]string, 0, len(s.entries))
	for targetName := range s.entries {
		targetNames = append(targetNames, targetName)
	}
	sort.Slice(targetNames, func(i, j int) bool {
		orderI := s.entries[targetNames[i]].Order()
		orderJ := s.entries[targetNames[j]].Order()
		switch {
		case orderI < orderJ:
			return true
		case orderI == orderJ:
			return targetNames[i] < targetNames[j]
		default:
			return false
		}
	})
	return targetNames
}

// ApplyOne updates targetName in targetDir on fs to match s using s.
func (s *SourceState) ApplyOne(targetSystem System, targetDir, targetName string, options ApplyOptions) error {
	targetStateEntry, err := s.entries[targetName].TargetStateEntry()
	if err != nil {
		return err
	}

	if options.Include != nil && !options.Include.IncludeTargetStateEntry(targetStateEntry) {
		return nil
	}

	targetPath := path.Join(targetDir, targetName)
	destStateEntry, err := NewDestStateEntry(targetSystem, targetPath)
	if err != nil {
		return err
	}

	if err := targetStateEntry.Apply(targetSystem, destStateEntry, options.Umask); err != nil {
		return err
	}

	if !options.UpdateState {
		return nil
	}

	entryState, err := targetStateEntry.EntryState()
	if err != nil {
		return err
	}
	if entryState == nil {
		return nil
	}
	data, err := json.Marshal(entryState)
	if err != nil {
		return err
	}
	return targetSystem.PersistentState().Set(EntryStateBucket, []byte(targetPath), data)
}

// Entries returns s's source state entries.
func (s *SourceState) Entries() map[string]SourceStateEntry {
	return s.entries
}

// Ignored returns if targetName is ignored.
func (s *SourceState) Ignored(targetName string) bool {
	return s.ignore.Match(targetName)
}

// TargetNames returns all of s's target names in alphabetical order.
func (s *SourceState) TargetNames() []string {
	targetNames := make([]string, 0, len(s.entries))
	for targetName := range s.entries {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	return targetNames
}

// Entry returns the source state entry for targetName.
func (s *SourceState) Entry(targetName string) (SourceStateEntry, bool) {
	sourceStateEntry, ok := s.entries[targetName]
	return sourceStateEntry, ok
}

// Evaluate evaluates every target state entry in s.
func (s *SourceState) Evaluate() error {
	for _, targetName := range s.AllTargetNames() {
		sourceStateEntry := s.entries[targetName]
		if err := sourceStateEntry.Evaluate(); err != nil {
			return err
		}
		targetStateEntry, err := sourceStateEntry.TargetStateEntry()
		if err != nil {
			return err
		}
		if err := targetStateEntry.Evaluate(); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteTemplateData returns the result of executing template data.
func (s *SourceState) ExecuteTemplateData(name string, data []byte) ([]byte, error) {
	tmpl, err := template.New(name).Option(s.templateOptions...).Funcs(s.templateFuncs).Parse(string(data))
	if err != nil {
		return nil, err
	}
	for name, t := range s.templates {
		tmpl, err = tmpl.AddParseTree(name, t.Tree)
		if err != nil {
			return nil, err
		}
	}
	output := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(output, name, s.TemplateData()); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// MinVersion returns the minimum version for which s is valid.
func (s *SourceState) MinVersion() semver.Version {
	return s.minVersion
}

// MustEntry returns the source state entry associated with targetName, and
// panics if it does not exist.
func (s *SourceState) MustEntry(targetName string) SourceStateEntry {
	sourceStateEntry, ok := s.entries[targetName]
	if !ok {
		panic(fmt.Sprintf("%s: no source state entry", targetName))
	}
	return sourceStateEntry
}

// Read reads a source state from sourcePath.
func (s *SourceState) Read() error {
	info, err := s.system.Lstat(s.sourceDir)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("%s: not a directory", s.sourceDir)
	}

	// Read all source entries.
	allSourceStateEntries := make(map[string][]SourceStateEntry)
	if err := vfs.WalkSlash(s.system, s.sourceDir, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if sourcePath == s.sourceDir {
			return nil
		}
		relPath := mustTrimPrefix(sourcePath, s.sourceDir+"/")
		sourceDirName, sourceName := path.Split(relPath)
		targetDirName := getTargetDirName(sourceDirName)
		// Follow symlinks from the source directory.
		if info.Mode()&os.ModeType == os.ModeSymlink {
			info, err = s.system.Stat(sourcePath)
			if err != nil {
				return err
			}
		}
		switch {
		case strings.HasPrefix(info.Name(), dataName):
			return s.addTemplateData(sourcePath)
		case info.Name() == ignoreName:
			// .chezmoiignore is interpreted as a template. vfs.WalkSlash walks
			// in alphabetical order, so, luckily for us, .chezmoidata will be
			// read before .chezmoiignore, so data in .chezmoidata is available
			// to be used in .chezmoiignore. Unluckily for us, .chezmoitemplates
			// will be read afterwards so partial templates will not be
			// available in .chezmoiignore.
			return s.addPatterns(s.ignore, sourcePath, sourceDirName)
		case info.Name() == removeName:
			// The comment about .chezmoiignore and templates applies to
			// .chezmoiremove too.
			removePatterns := NewPatternSet()
			if err := s.addPatterns(removePatterns, sourcePath, targetDirName); err != nil {
				return err
			}
			matches, err := removePatterns.Glob(s.system.UnderlyingFS(), s.destDir+"/")
			if err != nil {
				return err
			}
			n := 0
			for _, match := range matches {
				if !s.ignore.Match(match) {
					matches[n] = match
					n++
				}
			}
			matches = matches[:n]
			sourceStateEntry := &SourceStateRemove{
				path: sourcePath,
			}
			for _, match := range matches {
				allSourceStateEntries[match] = append(allSourceStateEntries[match], sourceStateEntry)
			}
			return nil
		case info.Name() == templatesDirName:
			if err := s.addTemplatesDir(sourcePath); err != nil {
				return err
			}
			return vfs.SkipDir
		case info.Name() == versionName:
			return s.addVersionFile(sourcePath)
		case strings.HasPrefix(info.Name(), Prefix):
			fallthrough
		case strings.HasPrefix(info.Name(), ignorePrefix):
			if info.IsDir() {
				return vfs.SkipDir
			}
			return nil
		case info.IsDir():
			da := parseDirAttributes(sourceName)
			targetName := path.Join(targetDirName, da.Name)
			if s.ignore.Match(targetName) {
				return nil
			}
			sourceStateEntry := s.newSourceStateDir(sourcePath, da)
			allSourceStateEntries[targetName] = append(allSourceStateEntries[targetName], sourceStateEntry)
			return nil
		case info.Mode().IsRegular():
			fa := parseFileAttributes(sourceName)
			targetName := path.Join(targetDirName, fa.Name)
			if s.ignore.Match(targetName) {
				return nil
			}
			sourceStateEntry := s.newSourceStateFile(sourcePath, fa, targetName)
			allSourceStateEntries[targetName] = append(allSourceStateEntries[targetName], sourceStateEntry)
			return nil
		default:
			return &unsupportedFileTypeError{
				path: sourcePath,
				mode: info.Mode(),
			}
		}
	}); err != nil {
		return err
	}

	// Remove all ignored targets.
	for targetName := range allSourceStateEntries {
		if s.ignore.Match(targetName) {
			delete(allSourceStateEntries, targetName)
		}
	}

	// Generate SourceStateRemoves for exact directories.
	for targetName, sourceStateEntries := range allSourceStateEntries {
		if len(sourceStateEntries) != 1 {
			continue
		}
		sourceStateDir, ok := sourceStateEntries[0].(*SourceStateDir)
		if !ok {
			continue
		}
		if !sourceStateDir.Attributes.Exact {
			continue
		}
		sourceStateRemove := &SourceStateRemove{
			path: sourceStateDir.Path(),
		}
		infos, err := s.system.ReadDir(path.Join(s.destDir, targetName))
		switch {
		case err == nil:
			for _, info := range infos {
				name := info.Name()
				if name == "." || name == ".." {
					continue
				}
				targetEntryName := path.Join(targetName, name)
				if _, ok := allSourceStateEntries[targetEntryName]; ok {
					continue
				}
				if s.ignore.Match(targetEntryName) {
					continue
				}
				allSourceStateEntries[targetEntryName] = append(allSourceStateEntries[targetEntryName], sourceStateRemove)
			}
		case os.IsNotExist(err):
			// Do nothing.
		default:
			return err
		}
	}

	// Check for duplicate source entries with the same target name. Iterate
	// over the target names in order so that any error is deterministic.
	targetNames := make([]string, 0, len(allSourceStateEntries))
	for targetName := range allSourceStateEntries {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	for _, targetName := range targetNames {
		sourceStateEntries := allSourceStateEntries[targetName]
		if len(sourceStateEntries) == 1 {
			continue
		}
		sourcePaths := make([]string, 0, len(sourceStateEntries))
		for _, sourceStateEntry := range sourceStateEntries {
			sourcePaths = append(sourcePaths, sourceStateEntry.Path())
		}
		err = multierr.Append(err, &duplicateTargetError{
			targetName:  targetName,
			sourcePaths: sourcePaths,
		})
	}
	if err != nil {
		return err
	}

	// Populate s.Entries with the unique source entry for each target.
	for targetName, sourceEntries := range allSourceStateEntries {
		s.entries[targetName] = sourceEntries[0]
	}

	return nil
}

// TemplateData returns s's template data.
func (s *SourceState) TemplateData() map[string]interface{} {
	return s.templateData
}

func (s *SourceState) addPatterns(patternSet *PatternSet, sourcePath, relPath string) error {
	data, err := s.executeTemplate(sourcePath)
	if err != nil {
		return err
	}
	dir := path.Dir(relPath)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lineNumber int
	for scanner.Scan() {
		lineNumber++
		text := scanner.Text()
		if index := strings.IndexRune(text, '#'); index != -1 {
			text = text[:index]
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		include := true
		if strings.HasPrefix(text, "!") {
			include = false
			text = mustTrimPrefix(text, "!")
		}
		pattern := path.Join(dir, text)
		if err := patternSet.Add(pattern, include); err != nil {
			return fmt.Errorf("%s:%d: %w", sourcePath, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	return nil
}

// addTemplateData adds all template data in sourcePath to s.
func (s *SourceState) addTemplateData(sourcePath string) error {
	_, name := path.Split(sourcePath)
	suffix := mustTrimPrefix(name, dataName+".")
	format, ok := Formats[suffix]
	if !ok {
		return fmt.Errorf("%s: unknown format", sourcePath)
	}
	data, err := s.system.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	var templateData map[string]interface{}
	if err := format.Decode(data, &templateData); err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	recursiveMerge(s.templateData, templateData)
	recursiveMerge(s.templateData, s.priorityTemplateData)
	return nil
}

// addTemplatesDir adds all templates in templateDir to s.
func (s *SourceState) addTemplatesDir(templateDir string) error {
	templateDirPrefix := templateDir + "/"
	return vfs.WalkSlash(s.system, templateDir, func(templatePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			contents, err := s.system.ReadFile(templatePath)
			if err != nil {
				return err
			}
			name := mustTrimPrefix(templatePath, templateDirPrefix)
			tmpl, err := template.New(name).Option(s.templateOptions...).Funcs(s.templateFuncs).Parse(string(contents))
			if err != nil {
				return err
			}
			if s.templates == nil {
				s.templates = make(map[string]*template.Template)
			}
			s.templates[name] = tmpl
			return nil
		case info.IsDir():
			return nil
		default:
			return &unsupportedFileTypeError{
				path: templatePath,
				mode: info.Mode(),
			}
		}
	})
}

// addVersionFile reads a .chezmoiversion file from source path and updates s's
// minimum version if it contains a more recent version than the current minimum
// version.
func (s *SourceState) addVersionFile(sourcePath string) error {
	data, err := s.system.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	version, err := semver.NewVersion(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	if s.minVersion.LessThan(*version) {
		s.minVersion = *version
	}
	return nil
}

// applyAll updates targetDir in fs to match s.
func (s *SourceState) applyAll(targetSystem System, targetDir string, options ApplyOptions) error {
	for _, targetName := range s.AllTargetNames() {
		if err := s.ApplyOne(targetSystem, targetDir, targetName, options); err != nil {
			return err
		}
	}
	return nil
}

func (s *SourceState) executeTemplate(path string) ([]byte, error) {
	data, err := s.system.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.ExecuteTemplateData(path, data)
}

func (s *SourceState) newSourceStateDir(sourcePath string, da DirAttributes) *SourceStateDir {
	targetStateDir := &TargetStateDir{
		perm: da.Perm(),
	}

	return &SourceStateDir{
		path:             sourcePath,
		Attributes:       da,
		targetStateEntry: targetStateDir,
	}
}

func (s *SourceState) newSourceStateFile(sourcePath string, fa FileAttributes, targetName string) *SourceStateFile {
	lazyContents := &lazyContents{
		contentsFunc: func() ([]byte, error) {
			contents, err := s.system.ReadFile(sourcePath)
			if err != nil {
				return nil, err
			}
			if !fa.Encrypted {
				return contents, nil
			}
			// FIXME pass targetName as filenameHint
			return s.encryptionTool.Decrypt(sourcePath, contents)
		},
	}

	var targetStateEntryFunc func() (TargetStateEntry, error)
	switch fa.Type {
	case SourceFileTypeFile:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fa.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			if !fa.Empty && isEmpty(contents) {
				return &TargetStateAbsent{}, nil
			}
			return &TargetStateFile{
				lazyContents: newLazyContents(contents),
				perm:         fa.Perm(),
			}, nil
		}
	case SourceFileTypePresent:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fa.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStatePresent{
				lazyContents: newLazyContents(contents),
				perm:         fa.Perm(),
			}, nil
		}
	case SourceFileTypeScript:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			contents, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fa.Template {
				contents, err = s.ExecuteTemplateData(sourcePath, contents)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStateScript{
				lazyContents: newLazyContents(contents),
				name:         targetName,
				once:         fa.Once,
			}, nil
		}
	case SourceFileTypeSymlink:
		targetStateEntryFunc = func() (TargetStateEntry, error) {
			linknameBytes, err := lazyContents.Contents()
			if err != nil {
				return nil, err
			}
			if fa.Template {
				linknameBytes, err = s.ExecuteTemplateData(sourcePath, linknameBytes)
				if err != nil {
					return nil, err
				}
			}
			return &TargetStateSymlink{
				lazyLinkname: newLazyLinkname(string(bytes.TrimSpace(linknameBytes))),
			}, nil
		}
	default:
		panic(fmt.Sprintf("%d: unsupported type", fa.Type))
	}

	return &SourceStateFile{
		lazyContents:         lazyContents,
		path:                 sourcePath,
		Attributes:           fa,
		targetStateEntryFunc: targetStateEntryFunc,
	}
}

func (s *SourceState) sourceStateEntry(destStateEntry DestStateEntry, destPath string, info os.FileInfo, parentDir string, options *AddOptions) (SourceStateEntry, error) {
	// FIXME IAMHERE need to generate EntryState and update status if needed
	switch destStateEntry := destStateEntry.(type) {
	case *DestStateAbsent:
		return nil, fmt.Errorf("%s: not found", destPath)
	case *DestStateDir:
		attributes := DirAttributes{
			Name:    info.Name(),
			Exact:   options.Exact,
			Private: runtime.GOOS != "windows" && info.Mode().Perm()&0o77 == 0,
		}
		return &SourceStateDir{
			path:       path.Join(parentDir, attributes.BaseName()),
			Attributes: attributes,
			targetStateEntry: &TargetStateDir{
				perm: 0o777,
			},
		}, nil
	case *DestStateFile:
		attributes := FileAttributes{
			Name:       info.Name(),
			Type:       SourceFileTypeFile,
			Empty:      options.Empty,
			Encrypted:  options.Encrypt,
			Executable: runtime.GOOS != "windows" && info.Mode().Perm()&0o111 != 0,
			Private:    runtime.GOOS != "windows" && info.Mode().Perm()&0o77 == 0,
			Template:   options.Template || options.AutoTemplate,
		}
		contents, err := destStateEntry.Contents()
		if err != nil {
			return nil, err
		}
		if options.AutoTemplate {
			contents = autoTemplate(contents, s.TemplateData())
		}
		if len(contents) == 0 && !options.Empty {
			return nil, nil
		}
		lazyContents := &lazyContents{
			contents: contents,
		}
		return &SourceStateFile{
			path:         path.Join(parentDir, attributes.BaseName()),
			Attributes:   attributes,
			lazyContents: lazyContents,
			targetStateEntry: &TargetStateFile{
				lazyContents: lazyContents,
				perm:         0o666,
			},
		}, nil
	case *DestStateSymlink:
		attributes := FileAttributes{
			Name:     info.Name(),
			Type:     SourceFileTypeSymlink,
			Template: options.Template || options.AutoTemplate,
		}
		linkname, err := destStateEntry.Linkname()
		if err != nil {
			return nil, err
		}
		contents := []byte(linkname)
		if options.AutoTemplate {
			contents = autoTemplate(contents, s.TemplateData())
		}
		contents = append(contents, '\n')
		lazyContents := &lazyContents{
			contents: contents,
		}
		return &SourceStateFile{
			path:         path.Join(parentDir, attributes.BaseName()),
			Attributes:   attributes,
			lazyContents: lazyContents,
			targetStateEntry: &TargetStateFile{
				lazyContents: lazyContents,
				perm:         0o666,
			},
		}, nil
	default:
		panic(fmt.Sprintf("%T: unsupported type", destStateEntry))
	}
}

// getTargetDirName returns the target directory name of sourceDirName.
func getTargetDirName(sourceDirName string) string {
	sourceNames := strings.Split(sourceDirName, "/")
	targetNames := make([]string, 0, len(sourceNames))
	for _, sourceName := range sourceNames {
		da := parseDirAttributes(sourceName)
		targetNames = append(targetNames, da.Name)
	}
	return strings.Join(targetNames, "/")
}
