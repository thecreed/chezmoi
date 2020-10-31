// +build !linux

package cmd

import "github.com/twpayne/chezmoi/next/internal/chezmoi"

func getKernelInfo(system chezmoi.System) (map[string]string, error) {
	return nil, nil
}

func getOSRelease(system chezmoi.System) (map[string]string, error) {
	return nil, nil
}
