package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/config"
)

const validAccountKey = "abcdef.0123456789abcdef0123456789abcdef.gh01"

func TestResolveUsesFlagOverEnvAndConfig(t *testing.T) {
	setConfigHome(t)
	t.Setenv(EnvKeyName, "0123456789abcdef0123456789abcdef")

	if err := (&config.Config{AccountKey: "fedcba.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.zz99"}).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, source, err := Resolve(validAccountKey)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if source != "--account-key flag" {
		t.Fatalf("source = %q, want %q", source, "--account-key flag")
	}

	if got.Raw != validAccountKey {
		t.Fatalf("Raw = %q, want %q", got.Raw, validAccountKey)
	}
}

func TestResolveUsesEnvWhenFlagEmpty(t *testing.T) {
	setConfigHome(t)
	t.Setenv(EnvKeyName, validAccountKey)

	got, source, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if source != EnvKeyName+" env var" {
		t.Fatalf("source = %q, want %q", source, EnvKeyName+" env var")
	}

	if got.Raw != validAccountKey {
		t.Fatalf("Raw = %q, want %q", got.Raw, validAccountKey)
	}
}

func TestResolveUsesConfigWhenFlagAndEnvEmpty(t *testing.T) {
	setConfigHome(t)

	if err := (&config.Config{AccountKey: validAccountKey}).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, source, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if source != "~/.config/healthexport/config.yaml" {
		t.Fatalf("source = %q, want %q", source, "~/.config/healthexport/config.yaml")
	}

	if got.Raw != validAccountKey {
		t.Fatalf("Raw = %q, want %q", got.Raw, validAccountKey)
	}
}

func TestResolveReturnsErrNoAccountKeyWhenAllSourcesEmpty(t *testing.T) {
	setConfigHome(t)

	_, source, err := Resolve("")
	if !errors.Is(err, ErrNoAccountKey) {
		t.Fatalf("Resolve() error = %v, want ErrNoAccountKey", err)
	}

	if source != "" {
		t.Fatalf("source = %q, want empty", source)
	}
}

func TestResolveReturnsFlagSourceContextOnInvalidKey(t *testing.T) {
	setConfigHome(t)

	_, _, err := Resolve("bad")
	assertInvalidKeySourceError(t, err, "--account-key flag")
}

func TestResolveReturnsEnvSourceContextOnInvalidKey(t *testing.T) {
	setConfigHome(t)
	t.Setenv(EnvKeyName, "bad")

	_, _, err := Resolve("")
	assertInvalidKeySourceError(t, err, EnvKeyName+" env var")
}

func TestResolveReturnsConfigSourceContextOnInvalidKey(t *testing.T) {
	setConfigHome(t)

	if err := (&config.Config{AccountKey: "bad"}).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, _, err := Resolve("")
	assertInvalidKeySourceError(t, err, "~/.config/healthexport/config.yaml")
}

func TestResolveFlagTakesPriorityOverEnv(t *testing.T) {
	setConfigHome(t)
	t.Setenv(EnvKeyName, "bad")

	got, source, err := Resolve(validAccountKey)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if source != "--account-key flag" {
		t.Fatalf("source = %q, want %q", source, "--account-key flag")
	}

	if got.Raw != validAccountKey {
		t.Fatalf("Raw = %q, want %q", got.Raw, validAccountKey)
	}
}

func assertInvalidKeySourceError(t *testing.T, err error, source string) {
	t.Helper()

	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("error = %v, want ErrInvalidKeyFormat", err)
	}

	if err == nil || !strings.Contains(err.Error(), source) {
		t.Fatalf("error = %v, want source %q in message", err, source)
	}
}

func setConfigHome(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)
}
