package config

import (
	"testing"
)

// Load/Save go through os.UserConfigDir, which on darwin resolves to
// $HOME/Library/Application Support — redirecting HOME isolates the test
// from the real machine's config.
func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := Save(Config{WorkspaceID: 42}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.WorkspaceID != 42 {
		t.Fatalf("WorkspaceID = %d, want 42", c.WorkspaceID)
	}
}

func TestLoadWithoutFileReturnsZeroValue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.WorkspaceID != 0 {
		t.Fatalf("WorkspaceID = %d, want 0 for a missing config file", c.WorkspaceID)
	}
}

// LoadToken's env-var override must short-circuit before touching the OS
// keychain, so this is testable without real credential storage.
func TestLoadTokenPrefersEnvVar(t *testing.T) {
	t.Setenv(EnvToken, "env-token-value")

	tok, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if tok != "env-token-value" {
		t.Fatalf("token = %q, want the env var value", tok)
	}
}
