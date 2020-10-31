package cmd

import "path/filepath"

// An osPath is a native OS path.
type osPath string

func (p *osPath) AbsSlash() (string, error) {
	abs, err := filepath.Abs(string(*p))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(abs), nil
}

func (p *osPath) Set(s string) error {
	*p = osPath(s)
	return nil
}

func (p *osPath) String() string {
	return string(*p)
}

func (p *osPath) Type() string {
	return "path"
}
