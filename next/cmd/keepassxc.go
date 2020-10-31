package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/coreos/go-semver/semver"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type keepassxcAttributeCacheKey struct {
	entry     string
	attribute string
}

type keepassxcConfig struct {
	Command        string
	Database       string
	Args           []string
	version        *semver.Version
	cache          map[string]map[string]string
	attributeCache map[keepassxcAttributeCacheKey]string
	password       string
}

var (
	keepassxcPairRegexp                  = regexp.MustCompile(`^([^:]+): (.*)$`)
	keepassxcNeedShowProtectedArgVersion = semver.Version{Major: 2, Minor: 5, Patch: 1}
)

func (c *Config) getKeepassxcVersion() *semver.Version {
	if c.Keepassxc.version != nil {
		return c.Keepassxc.version
	}
	name := c.Keepassxc.Command
	args := []string{"--version"}
	cmd := exec.Command(name, args...)
	output, err := c.baseSystem.IdempotentCmdOutput(cmd)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w", name, chezmoi.ShellQuoteArgs(args), err))
	}
	c.Keepassxc.version, err = semver.NewVersion(string(bytes.TrimSpace(output)))
	if err != nil {
		panic(fmt.Errorf("cannot parse version %q: %w", output, err))
	}
	return c.Keepassxc.version
}

func (c *Config) keepassxcFunc(entry string) map[string]string {
	if data, ok := c.Keepassxc.cache[entry]; ok {
		return data
	}
	if c.Keepassxc.Database == "" {
		panic(errors.New("keepassxc.database not set"))
	}
	name := c.Keepassxc.Command
	args := []string{"show"}
	if c.getKeepassxcVersion().Compare(keepassxcNeedShowProtectedArgVersion) >= 0 {
		args = append(args, "--show-protected")
	}
	args = append(args, c.Keepassxc.Args...)
	args = append(args, c.Keepassxc.Database, entry)
	output, err := c.runKeepassxcCLICommand(name, args)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w", name, chezmoi.ShellQuoteArgs(args), err))
	}
	data, err := parseKeyPassXCOutput(output)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w", name, chezmoi.ShellQuoteArgs(args), err))
	}
	if c.Keepassxc.cache == nil {
		c.Keepassxc.cache = make(map[string]map[string]string)
	}
	c.Keepassxc.cache[entry] = data
	return data
}

func (c *Config) keepassxcAttributeFunc(entry, attribute string) string {
	key := keepassxcAttributeCacheKey{
		entry:     entry,
		attribute: attribute,
	}
	if data, ok := c.Keepassxc.attributeCache[key]; ok {
		return data
	}
	if c.Keepassxc.Database == "" {
		panic(errors.New("keepassxc.database not set"))
	}
	name := c.Keepassxc.Command
	args := []string{"show", "--attributes", attribute, "--quiet"}
	if c.getKeepassxcVersion().Compare(keepassxcNeedShowProtectedArgVersion) >= 0 {
		args = append(args, "--show-protected")
	}
	args = append(args, c.Keepassxc.Args...)
	args = append(args, c.Keepassxc.Database, entry)
	output, err := c.runKeepassxcCLICommand(name, args)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w", name, chezmoi.ShellQuoteArgs(args), err))
	}
	outputStr := strings.TrimSpace(string(output))
	if c.Keepassxc.attributeCache == nil {
		c.Keepassxc.attributeCache = make(map[keepassxcAttributeCacheKey]string)
	}
	c.Keepassxc.attributeCache[key] = outputStr
	return outputStr
}

func (c *Config) runKeepassxcCLICommand(name string, args []string) ([]byte, error) {
	if c.Keepassxc.password == "" {
		password, err := readPassword(fmt.Sprintf("Insert password to unlock %s: ", c.Keepassxc.Database))
		fmt.Println()
		if err != nil {
			return nil, err
		}
		c.Keepassxc.password = string(password)
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewBufferString(c.Keepassxc.password + "\n")
	cmd.Stderr = c.stderr
	return c.baseSystem.IdempotentCmdOutput(cmd)
}

func parseKeyPassXCOutput(output []byte) (map[string]string, error) {
	data := make(map[string]string)
	s := bufio.NewScanner(bytes.NewReader(output))
	for i := 0; s.Scan(); i++ {
		if i == 0 {
			continue
		}
		match := keepassxcPairRegexp.FindStringSubmatch(s.Text())
		if match == nil {
			return nil, fmt.Errorf("%s: parse error", s.Text())
		}
		data[match[1]] = match[2]
	}
	return data, s.Err()
}

func readPassword(prompt string) (pw []byte, err error) {
	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		fmt.Print(prompt)
		pw, err = terminal.ReadPassword(fd)
		fmt.Println()
		return
	}

	var b [1]byte
	for {
		n, err := os.Stdin.Read(b[:])
		// terminal.ReadPassword discards any '\r', so do the same.
		if n > 0 && b[0] != '\r' {
			if b[0] == '\n' {
				return pw, nil
			}
			pw = append(pw, b[0])
			// Limit size, so that a wrong input won't fill up the memory.
			if len(pw) > 1024 {
				err = errors.New("password too long")
			}
		}
		if err != nil {
			// terminal.ReadPassword accepts EOF-terminated passwords
			// if non-empty, so do the same.
			if errors.Is(err, io.EOF) && len(pw) > 0 {
				err = nil
			}
			return pw, err
		}
	}
}
