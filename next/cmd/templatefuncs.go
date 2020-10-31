package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"howett.net/plist"
)

type ioregData struct {
	value map[string]interface{}
}

func (c *Config) includeFunc(filename string) string {
	contents, err := c.fs.ReadFile(path.Join(c.absSourceDir, filename))
	if err != nil {
		panic(err)
	}
	return string(contents)
}

func (c *Config) ioregFunc() map[string]interface{} {
	if runtime.GOOS != "darwin" {
		return nil
	}

	if c.ioregData.value != nil {
		return c.ioregData.value
	}

	cmd := exec.Command("ioreg", "-a", "-l")
	output, err := c.baseSystem.IdempotentCmdOutput(cmd)
	if err != nil {
		panic(fmt.Errorf("ioreg: %w", err))
	}

	var value map[string]interface{}
	if _, err := plist.Unmarshal(output, &value); err != nil {
		panic(fmt.Errorf("ioreg: %w", err))
	}
	c.ioregData.value = value
	return value
}

func (c *Config) joinPathFunc(elem ...string) string {
	return filepath.Join(elem...)
}

func (c *Config) lookPathFunc(file string) string {
	path, err := exec.LookPath(file)
	switch {
	case err == nil:
		return path
	case errors.Is(err, exec.ErrNotFound):
		return ""
	default:
		panic(err)
	}
}

func (c *Config) statFunc(name string) interface{} {
	info, err := c.fs.Stat(name)
	switch {
	case err == nil:
		return map[string]interface{}{
			"name":    info.Name(),
			"size":    info.Size(),
			"mode":    int(info.Mode()),
			"perm":    int(info.Mode() & os.ModePerm),
			"modTime": info.ModTime().Unix(),
			"isDir":   info.IsDir(),
		}
	case os.IsNotExist(err):
		return nil
	default:
		panic(err)
	}
}
