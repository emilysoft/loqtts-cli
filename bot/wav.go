package loqttsbot

import (
	"encoding/binary"
	"fmt"

	"github.com/hraban/opus"
)

// WavData is the decoded content of a PCM WAV file.
type WavData struct {
	SampleRate int
	Channels   int
	Pcm        []int16 // interleaved 16-bit little-endian samples
}

// ParseWAV extracts the samples of a 16-bit PCM RIFF/WAVE file held in
// memory. Only the "fmt " and "data" chunks are interpreted; any other
// chunks are skipped. The RIFF format needs no external dependencies.
func ParseWAV(data []byte) (*WavData, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a RIFF/WAVE file")
	}

	var channels uint16
	var sampleRate uint32
	var bits uint16
	dataOffset, dataSize := -1, 0

	pos := 12
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		switch id {
		case "fmt ":
			if pos+24 > len(data) {
				return nil, fmt.Errorf("truncated fmt chunk")
			}
			if format := binary.LittleEndian.Uint16(data[pos+8 : pos+10]); format != 1 {
				return nil, fmt.Errorf("unsupported audio format %d (only PCM)", format)
			}
			channels = binary.LittleEndian.Uint16(data[pos+10 : pos+12])
			sampleRate = binary.LittleEndian.Uint32(data[pos+12 : pos+16])
			bits = binary.LittleEndian.Uint16(data[pos+22 : pos+24])
		case "data":
			dataOffset = pos + 8
			dataSize = size
		}
		pos += 8 + size + (size & 1) // skip pad byte when size is odd
		if dataOffset >= 0 {
			break
		}
	}

	if dataOffset < 0 {
		return nil, fmt.Errorf("no data chunk found")
	}
	if bits != 16 {
		return nil, fmt.Errorf("unsupported bit depth %d (only 16-bit PCM)", bits)
	}

	end := dataOffset + dataSize
	if end > len(data) {
		end = len(data)
	}
	raw := data[dataOffset:end]
	pcm := make([]int16, len(raw)/2)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(raw[i*2 : i*2+2]))
	}

	return &WavData{
		SampleRate: int(sampleRate),
		Channels:   int(channels),
		Pcm:        pcm,
	}, nil
}

// EncodeOpus encodes the WAV samples into 20 ms Opus frames (48 kHz mono,
// the format Discord expects). The encoder is created once per call and
// destroyed through the finalizer of the opus package.
func EncodeOpus(w *WavData) ([][]byte, error) {
	if w.SampleRate != voiceSampleRate {
		return nil, fmt.Errorf("unexpected sample rate %d (want %d)", w.SampleRate, voiceSampleRate)
	}
	if w.Channels != 1 {
		return nil, fmt.Errorf("unexpected channel count %d (want 1)", w.Channels)
	}

	enc, err := opus.NewEncoder(voiceSampleRate, 1, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("opus encoder: %w", err)
	}

	frames := len(w.Pcm) / voiceFrameSize
	buf := make([]byte, 4096)
	out := make([][]byte, 0, frames)
	for i := 0; i < frames; i++ {
		n, err := enc.Encode(w.Pcm[i*voiceFrameSize:(i+1)*voiceFrameSize], buf)
		if err != nil {
			return nil, fmt.Errorf("opus encode frame %d: %w", i, err)
		}
		frame := make([]byte, n)
		copy(frame, buf[:n])
		out = append(out, frame)
	}
	return out, nil
}
