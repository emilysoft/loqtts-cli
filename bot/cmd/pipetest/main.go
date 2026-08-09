package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"loqttsbot"
)

// pipetest exercises the synthesis -> WAV-in-memory -> Opus pipeline
// without needing Discord: it speaks the given text with Loquendo, parses
// the resulting WAV held in a memory buffer and encodes it to Opus frames.
//
// Usage:
//   go run ./cmd/pipetest "hola mundo" [-v Voz] [-r 180]
func main() {
	loqttsbot.LoadDotEnv()
	cfg := loqttsbot.LoadSynthConfig()

	synth, err := loqttsbot.NewSynthesizer(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pipetest \"text\" [-v Voz] [-r 180]")
		os.Exit(2)
	}

	var text string
	opts := loqttsbot.SynthOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-v", "--voice":
			if i+1 < len(args) {
				opts.Voice = args[i+1]
				i++
			}
		case "-r", "--rate":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.Rate)
				i++
			}
		case "-p", "--pitch":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.Pitch)
				i++
			}
		default:
			text = args[i]
		}
	}
	if text == "" {
		fmt.Fprintln(os.Stderr, "no text")
		os.Exit(2)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	wav, err := synth.Speak(ctx, text, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "synthesis failed:", err)
		os.Exit(1)
	}
	fmt.Printf("synthesized in %s: %d bytes in memory buffer\n", time.Since(start).Round(time.Millisecond), len(wav))

	w, err := loqttsbot.ParseWAV(wav)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wav parse failed:", err)
		os.Exit(1)
	}
	fmt.Printf("wav: %d Hz, %d channel(s), %d samples (%.2f s)\n",
		w.SampleRate, w.Channels, len(w.Pcm), float64(len(w.Pcm))/float64(w.SampleRate))

	frames, err := loqttsbot.EncodeOpus(w)
	if err != nil {
		fmt.Fprintln(os.Stderr, "opus encode failed:", err)
		os.Exit(1)
	}
	ms := len(frames) * 20
	fmt.Printf("opus: %d frames of 20 ms (%.2f s), ready for discordgo OpusSend\n", len(frames), float64(ms)/1000)
}
