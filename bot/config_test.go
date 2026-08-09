package loqttsbot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	content := "# commento\n\nDISCORD_TOKEN=abc123\nTTS_RATE=180\nexport TTS_PITCH=200\nMAL = sin valor\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// os.Setenv can't be undone per-key reliably, so scope with a prefix.
	t.Setenv("BOT_PREFIX", "set")
	t.Setenv("DISCORD_TOKEN", "ya-existente")

	dotEnvCandidates = []string{".env"}
	defer func() { dotEnvCandidates = []string{"bot/.env", ".env"} }()

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	// Already-set vars must not be overridden.
	if got := os.Getenv("DISCORD_TOKEN"); got != "ya-existente" {
		t.Errorf("DISCORD_TOKEN = %q, want existing value", got)
	}
	// New vars are picked up.
	if got := os.Getenv("TTS_RATE"); got != "180" {
		t.Errorf("TTS_RATE = %q, want 180", got)
	}
	if got := os.Getenv("TTS_PITCH"); got != "200" {
		t.Errorf("TTS_PITCH = %q, want 200", got)
	}
}

func TestLoadDotEnvMissing(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	dotEnvCandidates = []string{".env"}
	defer func() { dotEnvCandidates = []string{"bot/.env", ".env"} }()

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv with no .env file should be a no-op, got: %v", err)
	}
}

func TestLoadDotEnvMalformed(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SIN_IGUAL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dotEnvCandidates = []string{".env"}
	defer func() { dotEnvCandidates = []string{"bot/.env", ".env"} }()

	if err := LoadDotEnv(); err == nil {
		t.Fatal("LoadDotEnv should error on a malformed line")
	}
}

func TestLoadConfigRequiresToken(t *testing.T) {
	// Ignore any real bot/.env in the working directory.
	dotEnvCandidates = []string{"does-not-exist"}
	defer func() { dotEnvCandidates = []string{"bot/.env", ".env"} }()

	t.Setenv("DISCORD_TOKEN", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig should fail without DISCORD_TOKEN")
	}
}
