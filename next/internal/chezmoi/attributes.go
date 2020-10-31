package chezmoi

import (
	"os"
	"strings"
)

// A SourceFileTargetType is a the type of a target represented by a file in the
// source state. A file in the source state can represent a file, script, or
// symlink in the target state.
type SourceFileTargetType int

// Source file types.
const (
	SourceFileTypeFile SourceFileTargetType = iota
	SourceFileTypePresent
	SourceFileTypeScript
	SourceFileTypeSymlink
)

// DirAttributes holds attributes parsed from a source directory name.
type DirAttributes struct {
	Name    string
	Exact   bool
	Private bool
}

// A FileAttributes holds attributes parsed from a source file name.
type FileAttributes struct {
	Name       string
	Type       SourceFileTargetType
	Empty      bool
	Encrypted  bool
	Executable bool
	Once       bool
	Order      int
	Private    bool
	Template   bool
}

// parseDirAttributes parses a single directory name in the source state.
func parseDirAttributes(sourceName string) DirAttributes {
	var (
		name    = sourceName
		exact   = false
		private = false
	)
	if strings.HasPrefix(name, exactPrefix) {
		name = mustTrimPrefix(name, exactPrefix)
		exact = true
	}
	if strings.HasPrefix(name, privatePrefix) {
		name = mustTrimPrefix(name, privatePrefix)
		private = true
	}
	if strings.HasPrefix(name, dotPrefix) {
		name = "." + mustTrimPrefix(name, dotPrefix)
	}
	return DirAttributes{
		Name:    name,
		Exact:   exact,
		Private: private,
	}
}

// BaseName returns da's source name.
func (da DirAttributes) BaseName() string {
	sourceName := ""
	if da.Exact {
		sourceName += exactPrefix
	}
	if da.Private {
		sourceName += privatePrefix
	}
	if strings.HasPrefix(da.Name, ".") {
		sourceName += dotPrefix + mustTrimPrefix(da.Name, ".")
	} else {
		sourceName += da.Name
	}
	return sourceName
}

// Perm returns da's file mode.
func (da DirAttributes) Perm() os.FileMode {
	perm := os.FileMode(0o777)
	if da.Private {
		perm &^= 0o77
	}
	return perm
}

// parseFileAttributes parses a source file name in the source state.
func parseFileAttributes(sourceName string) FileAttributes {
	var (
		typ        = SourceFileTypeFile
		name       = sourceName
		empty      = false
		encrypted  = false
		executable = false
		once       = false
		private    = false
		template   = false
		order      = 0
	)
	switch {
	case strings.HasPrefix(name, existsPrefix):
		typ = SourceFileTypePresent
		name = mustTrimPrefix(name, existsPrefix)
		if strings.HasPrefix(name, encryptedPrefix) {
			name = mustTrimPrefix(name, encryptedPrefix)
			encrypted = true
		}
		if strings.HasPrefix(name, privatePrefix) {
			name = mustTrimPrefix(name, privatePrefix)
			private = true
		}
		if strings.HasPrefix(name, executablePrefix) {
			name = mustTrimPrefix(name, executablePrefix)
			executable = true
		}
	case strings.HasPrefix(name, runPrefix):
		typ = SourceFileTypeScript
		name = mustTrimPrefix(name, runPrefix)
		switch {
		case strings.HasPrefix(name, firstPrefix):
			name = mustTrimPrefix(name, firstPrefix)
			order = -1
		case strings.HasPrefix(name, lastPrefix):
			name = mustTrimPrefix(name, lastPrefix)
			order = 1
		}
		if strings.HasPrefix(name, oncePrefix) {
			name = mustTrimPrefix(name, oncePrefix)
			once = true
		}
	case strings.HasPrefix(name, symlinkPrefix):
		typ = SourceFileTypeSymlink
		name = mustTrimPrefix(name, symlinkPrefix)
	default:
		if strings.HasPrefix(name, encryptedPrefix) {
			name = mustTrimPrefix(name, encryptedPrefix)
			encrypted = true
		}
		if strings.HasPrefix(name, privatePrefix) {
			name = mustTrimPrefix(name, privatePrefix)
			private = true
		}
		if strings.HasPrefix(name, emptyPrefix) {
			name = mustTrimPrefix(name, emptyPrefix)
			empty = true
		}
		if strings.HasPrefix(name, executablePrefix) {
			name = mustTrimPrefix(name, executablePrefix)
			executable = true
		}
	}
	if strings.HasPrefix(name, dotPrefix) {
		name = "." + mustTrimPrefix(name, dotPrefix)
	}
	if strings.HasSuffix(name, TemplateSuffix) {
		name = mustTrimSuffix(name, TemplateSuffix)
		template = true
	}
	return FileAttributes{
		Name:       name,
		Type:       typ,
		Empty:      empty,
		Encrypted:  encrypted,
		Executable: executable,
		Once:       once,
		Private:    private,
		Template:   template,
		Order:      order,
	}
}

// BaseName returns fa's source name.
func (fa FileAttributes) BaseName() string {
	sourceName := ""
	switch fa.Type {
	case SourceFileTypeFile:
		if fa.Encrypted {
			sourceName += encryptedPrefix
		}
		if fa.Private {
			sourceName += privatePrefix
		}
		if fa.Empty {
			sourceName += emptyPrefix
		}
		if fa.Executable {
			sourceName += executablePrefix
		}
	case SourceFileTypePresent:
		sourceName = existsPrefix
		if fa.Encrypted {
			sourceName += encryptedPrefix
		}
		if fa.Private {
			sourceName += privatePrefix
		}
		if fa.Executable {
			sourceName += executablePrefix
		}
	case SourceFileTypeScript:
		sourceName = runPrefix
		switch fa.Order {
		case -1:
			sourceName += firstPrefix
		case 1:
			sourceName += lastPrefix
		}
		if fa.Once {
			sourceName += oncePrefix
		}
	case SourceFileTypeSymlink:
		sourceName = symlinkPrefix
	}
	if strings.HasPrefix(fa.Name, ".") {
		sourceName += dotPrefix + mustTrimPrefix(fa.Name, ".")
	} else {
		sourceName += fa.Name
	}
	if fa.Template {
		sourceName += TemplateSuffix
	}
	return sourceName
}

// Perm returns fa's permissions.
func (fa FileAttributes) Perm() os.FileMode {
	perm := os.FileMode(0o666)
	if fa.Executable {
		perm |= 0o111
	}
	if fa.Private {
		perm &^= 0o77
	}
	return perm
}
