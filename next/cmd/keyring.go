package cmd

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

type keyringKey struct {
	service string
	user    string
}

type keyringData struct {
	cache map[keyringKey]string
}

func (c *Config) keyringFunc(service, user string) string {
	key := keyringKey{
		service: service,
		user:    user,
	}
	if password, ok := c.keyring.cache[key]; ok {
		return password
	}
	password, err := keyring.Get(service, user)
	if err != nil {
		panic(fmt.Errorf("%q %q: %w", service, user, err))
	}
	c.keyring.cache[key] = password
	return password
}
