package loqttsbot

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os/exec"
	"testing"
)

// buildTestWAV builds a valid in-memory 16-bit PCM WAV with a sine wave.
func buildTestWAV(seconds, rate int) []byte {
	n := seconds * rate
	pcm := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		s := int16(0.5 * math.MaxInt16 * math.Sin(2*math.Pi*440*float64(i)/float64(rate)))
		binary.LittleEndian.PutUint16(pcm[i*2:i*2+2], uint16(s))
	}

	dataSize := len(pcm)
	buf := make([]byte, 44+dataSize)
	binary.LittleEndian.PutUint32(buf[0:4], 0x46464952)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	binary.LittleEndian.PutUint32(buf[8:12], 0x45564157)
	binary.LittleEndian.PutUint32(buf[12:16], 0x20746d66)
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1)
	binary.LittleEndian.PutUint16(buf[22:24], 1)
	binary.LittleEndian.PutUint32(buf[24:28], uint32(rate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(rate*2))
	binary.LittleEndian.PutUint16(buf[32:34], 2)
	binary.LittleEndian.PutUint16(buf[34:36], 16)
	binary.LittleEndian.PutUint32(buf[36:40], 0x61746164)
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	copy(buf[44:], pcm)
	return buf
}

func TestEncodeMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	out, err := EncodeMP3(context.Background(), buildTestWAV(1, 48000))
	if err != nil {
		t.Fatalf("EncodeMP3: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("EncodeMP3 returned empty output")
	}
	// Valid MPEG audio: an ID3 tag or an MPEG sync frame (0xFFEx).
	if !bytes.HasPrefix(out, []byte("ID3")) && out[0] != 0xFF && out[1]&0xE0 != 0xE0 {
		t.Fatalf("output does not look like MP3: first bytes % x", out[:min(8, len(out))])
	}
}

func TestEncodeMP3Invalid(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	if _, err := EncodeMP3(context.Background(), []byte("not a wav")); err == nil {
		t.Fatal("expected error for invalid WAV input")
	}
}
