package loqttsbot

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// EncodeMP3 transcodes 16-bit PCM WAV bytes to an MP3 stream in memory via
// ffmpeg (read from stdin, written to stdout). The bitrate is 128 kbps, which
// is a good balance for speech.
func EncodeMP3(ctx context.Context, wav []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "error",
		"-f", "wav", "-i", "-",
		"-f", "mp3", "-b:a", "128k", "-",
	)
	cmd.Stdin = bytes.NewReader(wav)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, msg)
	}
	return out.Bytes(), nil
}
