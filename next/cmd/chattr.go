package cmd

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type boolModifier int

const (
	boolModifierSet            boolModifier = 1
	boolModifierLeaveUnchanged boolModifier = 0
	boolModifierClear          boolModifier = -1
)

type orderModifier int

const (
	orderModifierSetFirst       orderModifier = -2
	orderModifierClearFirst     orderModifier = -1
	orderModifierLeaveUnchanged orderModifier = 0
	orderModifierClearLast      orderModifier = 1
	orderModifierSetLast        orderModifier = 2
)

type attributesModifier struct {
	empty      boolModifier
	encrypted  boolModifier
	exact      boolModifier
	executable boolModifier
	once       boolModifier
	order      orderModifier
	private    boolModifier
	template   boolModifier
}

func (c *Config) newChattrCmd() *cobra.Command {
	attributes := []string{
		"empty", "e",
		"encrypted",
		"exact",
		"executable", "x",
		"first", "f",
		"last", "l",
		"once", "o",
		"private", "p",
		"template", "t",
	}
	validArgs := make([]string, 0, 4*len(attributes))
	for _, attribute := range attributes {
		validArgs = append(validArgs, attribute, "-"+attribute, "+"+attribute, "no"+attribute)
	}

	chattrCmd := &cobra.Command{
		Use:       "chattr attributes target...",
		Short:     "Change the attributes of a target in the source state",
		Long:      mustGetLongHelp("chattr"),
		Example:   getExample("chattr"),
		Args:      cobra.MinimumNArgs(2),
		ValidArgs: validArgs,
		RunE:      c.makeRunEWithSourceState(c.runChattrCmd),
		Annotations: map[string]string{
			modifiesSourceDirectory: "true",
		},
	}

	return chattrCmd
}

func (c *Config) runChattrCmd(cmd *cobra.Command, args []string, sourceState *chezmoi.SourceState) error {
	// FIXME should the core functionality of chattr move to chezmoi.SourceState?

	am, err := parseAttributesModifier(args[0])
	if err != nil {
		return err
	}

	targetNames, err := c.getTargetNames(sourceState, args[1:], getTargetNamesOptions{
		mustBeInSourceState: true,
		recursive:           false,
	})
	if err != nil {
		return err
	}

	// Sort targets in reverse so we update children before their parent
	// directories.
	sort.Sort(sort.Reverse(sort.StringSlice(targetNames)))

	for _, targetName := range targetNames {
		sourceStateEntry := sourceState.MustEntry(targetName)
		sourceName := strings.TrimPrefix(sourceStateEntry.Path(), c.absSourceDir+"/")
		parentDirName, baseName := path.Split(sourceName)
		switch sourceStateEntry := sourceStateEntry.(type) {
		case *chezmoi.SourceStateDir:
			if newBaseName := am.modifyDirAttributes(sourceStateEntry.Attributes).BaseName(); newBaseName != baseName {
				newSourcePath := path.Join(c.absSourceDir, parentDirName, newBaseName)
				if err := c.sourceSystem.Rename(sourceStateEntry.Path(), newSourcePath); err != nil {
					return err
				}
			}
		case *chezmoi.SourceStateFile:
			// FIXME encrypted attribute changes
			// FIXME when changing encrypted attribute add new file before removing old one
			if newBaseName := am.modifyFileAttributes(sourceStateEntry.Attributes).BaseName(); newBaseName != baseName {
				newSourcePath := path.Join(c.absSourceDir, parentDirName, newBaseName)
				if err := c.sourceSystem.Rename(sourceStateEntry.Path(), newSourcePath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m boolModifier) modify(b bool) bool {
	switch m {
	case boolModifierSet:
		return true
	case boolModifierLeaveUnchanged:
		return b
	case boolModifierClear:
		return false
	default:
		panic(fmt.Sprintf("%d: unknown bool modifier", m))
	}
}

func (m orderModifier) modify(order int) int {
	switch m {
	case orderModifierSetFirst:
		return -1
	case orderModifierClearFirst:
		if order < 0 {
			return 0
		}
		return order
	case orderModifierLeaveUnchanged:
		return order
	case orderModifierClearLast:
		if order > 0 {
			return 0
		}
		return order
	case orderModifierSetLast:
		return 1
	default:
		panic(fmt.Sprintf("%d: unknown order modifier", m))
	}
}

func parseAttributesModifier(s string) (*attributesModifier, error) {
	am := &attributesModifier{}
	for _, modifierStr := range strings.Split(s, ",") {
		modifierStr = strings.TrimSpace(modifierStr)
		if modifierStr == "" {
			continue
		}
		var bm boolModifier
		var attribute string
		switch {
		case modifierStr[0] == '-':
			bm = boolModifierClear
			attribute = modifierStr[1:]
		case modifierStr[0] == '+':
			bm = boolModifierSet
			attribute = modifierStr[1:]
		case strings.HasPrefix(modifierStr, "no"):
			bm = boolModifierClear
			attribute = modifierStr[2:]
		default:
			bm = boolModifierSet
			attribute = modifierStr
		}
		switch attribute {
		case "empty", "e":
			am.empty = bm
		case "encrypted":
			am.encrypted = bm
		case "exact":
			am.exact = bm
		case "executable", "x":
			am.executable = bm
		case "first", "f":
			switch bm {
			case boolModifierClear:
				am.order = orderModifierClearFirst
			case boolModifierLeaveUnchanged:
				am.order = orderModifierLeaveUnchanged
			case boolModifierSet:
				am.order = orderModifierSetFirst
			}
		case "last", "l":
			switch bm {
			case boolModifierClear:
				am.order = orderModifierClearLast
			case boolModifierLeaveUnchanged:
				am.order = orderModifierLeaveUnchanged
			case boolModifierSet:
				am.order = orderModifierSetLast
			}
		case "once", "o":
			am.once = bm
		case "private", "p":
			am.private = bm
		case "template", "t":
			am.template = bm
		default:
			return nil, fmt.Errorf("%s: unknown attribute", attribute)
		}
	}
	return am, nil
}

func (am *attributesModifier) modifyDirAttributes(da chezmoi.DirAttributes) chezmoi.DirAttributes {
	return chezmoi.DirAttributes{
		Name:    da.Name,
		Exact:   am.exact.modify(da.Exact),
		Private: am.private.modify(da.Private),
	}
}

func (am *attributesModifier) modifyFileAttributes(fa chezmoi.FileAttributes) chezmoi.FileAttributes {
	switch fa.Type {
	case chezmoi.SourceFileTypeFile:
		return chezmoi.FileAttributes{
			Name:       fa.Name,
			Type:       chezmoi.SourceFileTypeFile,
			Empty:      am.empty.modify(fa.Empty),
			Encrypted:  am.encrypted.modify(fa.Encrypted),
			Executable: am.executable.modify(fa.Executable),
			Private:    am.private.modify(fa.Private),
			Template:   am.template.modify(fa.Template),
		}
	case chezmoi.SourceFileTypePresent:
		return chezmoi.FileAttributes{
			Name:       fa.Name,
			Type:       chezmoi.SourceFileTypePresent,
			Encrypted:  am.encrypted.modify(fa.Encrypted),
			Executable: am.executable.modify(fa.Executable),
			Private:    am.private.modify(fa.Private),
			Template:   am.template.modify(fa.Template),
		}
	case chezmoi.SourceFileTypeScript:
		return chezmoi.FileAttributes{
			Name:  fa.Name,
			Type:  chezmoi.SourceFileTypeScript,
			Once:  am.once.modify(fa.Once),
			Order: am.order.modify(fa.Order),
		}
	case chezmoi.SourceFileTypeSymlink:
		return chezmoi.FileAttributes{
			Name:     fa.Name,
			Type:     chezmoi.SourceFileTypeSymlink,
			Template: am.template.modify(fa.Template),
		}
	default:
		panic(fmt.Sprintf("%d: unknown source file type", fa.Type))
	}
}
