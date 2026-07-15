// Package config handles lazytoggl's two persistence needs: the API token
// (secret, lives in the OS keychain) and the cached workspace id (not
// secret, lives in a small JSON file). Keeping the workspace id cached lets
// normal launches skip the /me call entirely (see docs/ARCHITECTURE.md).
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "lazytoggl"
	keyringUser    = "api_token"

	// EnvToken overrides the keychain when set, for headless/dev use.
	EnvToken = "TOGGL_API_TOKEN"
)

type Config struct {
	WorkspaceID int64 `json:"workspace_id"`
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "lazytoggl"), nil
}

func filePath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the cached config, returning a zero Config if none exists yet.
func Load() (Config, error) {
	p, err := filePath()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes the config file with 0600 permissions, creating its directory
// if needed.
func Save(c Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	p, err := filePath()
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0600)
}

// LoadToken returns the API token from $TOGGL_API_TOKEN if set, otherwise
// from the OS keychain. An empty string with a nil error means no token is
// configured yet — the caller should run onboarding.
func LoadToken() (string, error) {
	if t := os.Getenv(EnvToken); t != "" {
		return t, nil
	}
	t, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return t, nil
}

// SaveToken stores the token in the OS keychain.
func SaveToken(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}
