package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuessDotfilesRepo(t *testing.T) {
	for argStr, expected := range map[string]string{
		"user":                             "https://github.com/user/dotfiles.git",
		"user/dots":                        "https://github.com/user/dots.git",
		"user/dots.git":                    "https://github.com/user/dots.git",
		"https://gitlab.com/user/dots.git": "https://gitlab.com/user/dots.git",
	} {
		assert.Equal(t, expected, guessDotfilesRepo(argStr))
	}
}
