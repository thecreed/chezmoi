package cmd

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type passConfig struct {
	Command string
	cache   map[string]string
}

func (c *Config) passFunc(id string) string {
	if s, ok := c.Pass.cache[id]; ok {
		return s
	}
	name := c.Pass.Command
	args := []string{"show", id}
	cmd := exec.Command(name, args...)
	cmd.Stdin = c.stdin
	cmd.Stderr = c.stderr
	output, err := c.baseSystem.IdempotentCmdOutput(cmd)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w", name, chezmoi.ShellQuoteArgs(args), err))
	}
	var password string
	if index := bytes.IndexByte(output, '\n'); index != -1 {
		password = string(output[:index])
	} else {
		password = string(output)
	}
	if c.Pass.cache == nil {
		c.Pass.cache = make(map[string]string)
	}
	c.Pass.cache[id] = password
	return password
}
