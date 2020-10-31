package chezmoi

import (
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v2"
	vfs "github.com/twpayne/go-vfs"
)

// A stringSet is a set of strings.
type stringSet map[string]struct{}

// An PatternSet is a set of patterns.
type PatternSet struct {
	includePatterns stringSet
	excludePatterns stringSet
}

// A PatternSetOption sets an option on a pattern set.
type PatternSetOption func(*PatternSet)

// NewPatternSet returns a new PatternSet.
func NewPatternSet(options ...PatternSetOption) *PatternSet {
	ps := &PatternSet{
		includePatterns: newStringSet(),
		excludePatterns: newStringSet(),
	}
	for _, option := range options {
		option(ps)
	}
	return ps
}

// Add adds a pattern to ps.
func (ps *PatternSet) Add(pattern string, include bool) error {
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return err
	}
	if include {
		ps.includePatterns.Add(pattern)
	} else {
		ps.excludePatterns.Add(pattern)
	}
	return nil
}

// Glob returns all matches in fs.
func (ps *PatternSet) Glob(fs vfs.FS, prefix string) ([]string, error) {
	vos := doubleStarOS{FS: fs}
	allMatches := newStringSet()
	for includePattern := range ps.includePatterns {
		matches, err := doublestar.GlobOS(vos, prefix+includePattern)
		if err != nil {
			return nil, err
		}
		allMatches.Add(matches...)
	}
	for match := range allMatches {
		for excludePattern := range ps.excludePatterns {
			exclude, err := doublestar.PathMatchOS(vos, prefix+excludePattern, match)
			if err != nil {
				return nil, err
			}
			if exclude {
				delete(allMatches, match)
			}
		}
	}
	matchesSlice := allMatches.Elements()
	for i, match := range matchesSlice {
		matchesSlice[i] = mustTrimPrefix(filepath.ToSlash(match), prefix)
	}
	sort.Strings(matchesSlice)
	return matchesSlice, nil
}

// Match returns if name matches any pattern in ps.
func (ps *PatternSet) Match(name string) bool {
	for pattern := range ps.excludePatterns {
		if ok, _ := doublestar.Match(pattern, name); ok {
			return false
		}
	}
	for pattern := range ps.includePatterns {
		if ok, _ := doublestar.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

// newStringSet returns a new StringSet containing elements.
func newStringSet(elements ...string) stringSet {
	s := make(stringSet)
	s.Add(elements...)
	return s
}

// Add adds elements to s.
func (s stringSet) Add(elements ...string) {
	for _, element := range elements {
		s[element] = struct{}{}
	}
}

// Elements returns all the elements of s.
func (s stringSet) Elements() []string {
	elements := make([]string, 0, len(s))
	for element := range s {
		elements = append(elements, element)
	}
	return elements
}
