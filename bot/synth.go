package loqttsbot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

// SynthOptions controls a single synthesis request.
type SynthOptions struct {
	Voice    string
	Language string
	Style    string
	Rate     int // words per minute, <=0 = voice default
	Pitch    int // hertz, <=0 = voice default
	Volume   int // 0-100, <=0 = voice default
}

// Synthesizer drives the Windows loqtts.exe under Wine. Synthesis is
// serialized with a mutex because Wine refuses concurrent access to the
// same prefix. The generated audio never touches the disk: the custom
// LoqTTS7AudioUser destination emits the WAV over stdout, which we
// capture straight into memory.
type Synthesizer struct {
	mu   sync.Mutex
	cfg  Config
	exe  string // full path to loqtts.exe
	dir  string // working directory (install bin/, where the DLLs live)
}

// NewSynthesizer validates that loqtts.exe and the audio-user module are
// deployed and returns a ready synthesizer.
func NewSynthesizer(cfg Config) (*Synthesizer, error) {
	dir := filepath.Join(cfg.LoquendoInstall, "bin")
	exe := filepath.Join(dir, "loqtts.exe")
	if _, err := os.Stat(exe); err != nil {
		return nil, fmt.Errorf("loqtts.exe not found in %s (build it and copy it there): %w", dir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "LoqTTS7AudioUser.dll")); err != nil {
		return nil, fmt.Errorf("LoqTTS7AudioUser.dll not found in %s (copy it next to loqtts.exe): %w", dir, err)
	}
	return &Synthesizer{cfg: cfg, exe: exe, dir: dir}, nil
}

// run executes loqtts.exe through steam-run + wine64 and returns stdout.
func (s *Synthesizer) run(ctx context.Context, args ...string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.CommandContext(ctx, s.cfg.SteamRun, s.cfg.Wine64, s.exe)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), "WINEPREFIX="+s.cfg.WinePrefix)

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("loqtts: %w: %s", err, errOut.String())
	}
	return out.Bytes(), nil
}

// Speak synthesizes text and returns the WAV bytes in memory.
func (s *Synthesizer) Speak(ctx context.Context, text string, o SynthOptions) ([]byte, error) {
	// Discord voice wants 48 kHz mono; Loquendo can output it directly.
	args := []string{"-o", "-", "--rate-hz", "48000", "--mono"}
	if o.Voice != "" {
		args = append(args, "-v", o.Voice)
	}
	if o.Language != "" {
		args = append(args, "-l", o.Language)
	}
	if o.Style != "" {
		args = append(args, "-s", o.Style)
	}
	if o.Rate > 0 {
		args = append(args, "-r", strconv.Itoa(o.Rate))
	}
	if o.Pitch > 0 {
		args = append(args, "-p", strconv.Itoa(o.Pitch))
	}
	if o.Volume > 0 {
		args = append(args, "-a", strconv.Itoa(o.Volume))
	}
	args = append(args, text)

	wav, err := s.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	if len(wav) == 0 {
		return nil, fmt.Errorf("loqtts produced no audio")
	}
	return wav, nil
}

// ListVoices returns the installed voices as text.
func (s *Synthesizer) ListVoices(ctx context.Context) (string, error) {
	out, err := s.run(ctx, "--list-voices")
	return string(out), err
}

// ListLanguages returns the installed languages as text.
func (s *Synthesizer) ListLanguages(ctx context.Context) (string, error) {
	out, err := s.run(ctx, "--list-languages")
	return string(out), err
}

// Version returns the Loquendo TTS version string.
func (s *Synthesizer) Version(ctx context.Context) (string, error) {
	out, err := s.run(ctx, "--version")
	return string(out), err
}
