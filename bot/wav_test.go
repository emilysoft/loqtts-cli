package loqttsbot

import (
	"encoding/binary"
	"testing"
)

// buildWAV assembles a 16-bit PCM RIFF/WAVE file in memory.
func buildWAV(sampleRate, channels int, pcm []int16) []byte {
	block := channels * 2
	dataSize := len(pcm) * 2
	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(buf[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(sampleRate*block))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(block))
	binary.LittleEndian.PutUint16(buf[34:36], 16)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(buf[44+i*2:44+i*2+2], uint16(s))
	}
	return buf
}

func TestParseWAV(t *testing.T) {
	pcm := []int16{0, 1, 2, 3, 4, 5, 6, 7}
	wav := buildWAV(48000, 1, pcm)

	got, err := ParseWAV(wav)
	if err != nil {
		t.Fatalf("ParseWAV: %v", err)
	}
	if got.SampleRate != 48000 || got.Channels != 1 {
		t.Errorf("header = %d Hz/%d ch, want 48000/1", got.SampleRate, got.Channels)
	}
	if len(got.Pcm) != len(pcm) {
		t.Fatalf("samples = %d, want %d", len(got.Pcm), len(pcm))
	}
	for i := range pcm {
		if got.Pcm[i] != pcm[i] {
			t.Errorf("pcm[%d] = %d, want %d", i, got.Pcm[i], pcm[i])
		}
	}
}

func TestParseWAVRejectsGarbage(t *testing.T) {
	if _, err := ParseWAV([]byte("not a wav file at all")); err == nil {
		t.Error("expected error for non-WAV data")
	}
	if _, err := ParseWAV(buildWAV(48000, 2, []int16{0, 0})); err != nil {
		t.Errorf("unexpected error for stereo: %v", err)
	}
}
