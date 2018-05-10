package config

import (
	"errors"

	"github.com/IngCr3at1on/glas"
	"github.com/JustAnotherOrganization/l5424"
)

type (
	// Config is a Glas with REST configuration.
	Config struct {
		// Logger is the Logger interface outlined in the current stdlogger proposals.
		Logger l5424.Logger
		// Address is the address to host the REST interface at.
		Address string
		// Glas is a pointer to a glas.Glas instance.
		Glas *glas.Glas
	}
)

// Validate a configuration, error if a required value is missing and set
// defaults (if a value is not provided) when not required.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}
	if c.Glas == nil {
		return errors.New("config.Glas cannot be nil")
	}

	// Disable logging if no logger is provided.
	if c.Logger == nil {
		c.Logger = &l5424.NoOpLogger{}
	}

	return nil
}
