package loqttsbot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds the bot configuration, read from environment variables.
type Config struct {
	Token  string // Discord bot token (DISCORD_TOKEN)
	Prefix string // command prefix (default ">")

	// Default synthesis options.
	Voice    string
	Language string
	Style    string
	Rate     int // words per minute, 0 = voice default
	Pitch    int // hertz, 0 = voice default
	Volume   int // 0-100, 0 = voice default

	// Loquendo / Wine environment.
	LoquendoInstall string // LTTS7 install root (contains bin/loqtts.exe)
	WinePrefix      string // WINEPREFIX used by the wine runner
	Wine64          string // path to the wine64 binary
	SteamRun        string // steam-run (FHS) wrapper command
}

func env(key string) string { return os.Getenv(key) }

// dotEnvCandidates are the .env files probed in order, relative to the
// working directory. bot/.env is checked first because the helpers can be
// run from either the repository root or the bot/ folder.
var dotEnvCandidates = []string{"bot/.env", ".env"}

// LoadDotEnv reads KEY=VALUE pairs from the first existing .env file among
// dotEnvCandidates and exports the variables that are not already set in the
// environment. This lets the bot pick up its configuration from bot/.env
// without needing `source .env` (which would not export them anyway).
func LoadDotEnv() error {
	for _, cand := range dotEnvCandidates {
		if _, err := os.Stat(cand); err != nil {
			continue
		}
		return loadDotEnv(cand)
	}
	return nil
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		raw = strings.TrimPrefix(raw, "export ")
		key, val, ok := strings.Cut(raw, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			return fmt.Errorf("%s:%d: empty variable name", path, line)
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string) int {
	v, _ := strconv.Atoi(os.Getenv(key))
	return v
}

// LoadConfig reads the configuration from the environment and requires a
// Discord token. A bot/.env file (if present) is loaded first.
func LoadConfig() (Config, error) {
	LoadDotEnv()
	cfg := LoadSynthConfig()
	if cfg.Token == "" {
		return cfg, errors.New("DISCORD_TOKEN is required")
	}
	return cfg, nil
}

// LoadSynthConfig reads the configuration from the environment without
// requiring a Discord token (useful for tools that only synthesize).
func LoadSynthConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Token:   env("DISCORD_TOKEN"),
		Prefix:  envDefault("BOT_PREFIX", ">"),
		Voice:   envDefault("TTS_VOICE", "Carlos"),
		Language: env("TTS_LANGUAGE"),
		Style:   env("TTS_STYLE"),
		Rate:    envInt("TTS_RATE"),
		Pitch:   envInt("TTS_PITCH"),
		Volume:  envInt("TTS_VOLUME"),

		LoquendoInstall: envDefault("LOQTTS_INSTALL",
			"/home/nit/Games/loquendo/drive_c/Program Files (x86)/Loquendo/LTTS7"),
		WinePrefix: envDefault("WINEPREFIX", "/home/nit/Games/loquendo"),
		Wine64: envDefault("WINE64",
			filepath.Join(home, ".local/share/lutris/runners/wine/wine-ge-8-26-x86_64/bin/wine64")),
		SteamRun: envDefault("STEAM_RUN", "steam-run"),
	}
}
