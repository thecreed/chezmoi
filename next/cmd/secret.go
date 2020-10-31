package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/twpayne/chezmoi/next/internal/chezmoi"
)

type secretConfig struct {
	Command   string
	cache     map[string]string
	jsonCache map[string]interface{}
}

func (c *Config) secretFunc(args ...string) string {
	key := strings.Join(args, "\x00")
	if value, ok := c.Secret.cache[key]; ok {
		return value
	}
	name := c.Secret.Command
	cmd := exec.Command(name, args...)
	cmd.Stdin = c.stdin
	cmd.Stderr = c.stderr
	output, err := c.baseSystem.IdempotentCmdOutput(cmd)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w\n%s", name, chezmoi.ShellQuoteArgs(args), err, output))
	}
	value := string(bytes.TrimSpace(output))
	if c.Secret.cache == nil {
		c.Secret.cache = make(map[string]string)
	}
	c.Secret.cache[key] = value
	return value
}

func (c *Config) secretJSONFunc(args ...string) interface{} {
	key := strings.Join(args, "\x00")
	if value, ok := c.Secret.jsonCache[key]; ok {
		return value
	}
	name := c.Secret.Command
	cmd := exec.Command(name, args...)
	cmd.Stdin = c.stdin
	cmd.Stderr = c.stderr
	output, err := c.baseSystem.IdempotentCmdOutput(cmd)
	if err != nil {
		panic(fmt.Errorf("%s %s: %w\n%s", name, chezmoi.ShellQuoteArgs(args), err, output))
	}
	var value interface{}
	if err := json.Unmarshal(output, &value); err != nil {
		panic(fmt.Errorf("%s %s: %w\n%s", name, chezmoi.ShellQuoteArgs(args), err, output))
	}
	if c.Secret.jsonCache == nil {
		c.Secret.jsonCache = make(map[string]interface{})
	}
	c.Secret.jsonCache[key] = value
	return value
}
