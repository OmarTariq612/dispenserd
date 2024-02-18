package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const (
	Port = 8080
)

type Config struct {
	AuthKey string `json:"auth_key"`

	// TODO: can we handle this better ?
	// Duration and StrDuration can be out of sync for a bit of time
	StrDuration string        `json:"duration"`
	Duration    time.Duration `json:"-"`
}

func LoadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	// TODO: support yaml and toml formats
	decoder := json.NewDecoder(bytes.NewBuffer(content))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&config); err != nil {
		return nil, err
	}
	// validate auth key
	if config.AuthKey == "" {
		return nil, fmt.Errorf("AuthKey is required")
	}
	// validate duration
	config.Duration, err = time.ParseDuration(config.StrDuration)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Complete() error {
	if c.AuthKey == "" {
		return fmt.Errorf("AuthKey is required")
	}
	if c.StrDuration == "" {
		c.StrDuration = "30s"
		var err error
		if c.Duration, err = time.ParseDuration(c.StrDuration); err != nil {
			panic(err)
		}
	}
	return nil
}

func (c *Config) SaveConfig(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	// TODO: support yaml and toml formats
	return json.NewEncoder(file).Encode(c)
}
