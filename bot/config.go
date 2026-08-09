package loqttsbot

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds the bot configuration, read from environment variables.
type Config struct {
	Token  string // Discord bot token (DISCORD_TOKEN)
	Prefix string // command prefix (default "!tts")

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
// Discord token.
func LoadConfig() (Config, error) {
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
		Prefix:  envDefault("BOT_PREFIX", "!tts"),
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
