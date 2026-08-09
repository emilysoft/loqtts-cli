package loqttsbot

import (
	"strings"
	"testing"
)

func TestParseTTS(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		wantText string
		wantOpts SynthOptions
		wantErr  bool
	}{
		{"plain text", "hola mundo", "hola mundo", SynthOptions{}, false},
		{"empty", "", "", SynthOptions{}, true},
		{"flag and text", "-v Carmen hola", "hola", SynthOptions{Voice: "Carmen"}, false},
		{"long flags", "--voice Carmen --rate 180 --pitch 200 texto largo", "texto largo",
			SynthOptions{Voice: "Carmen", Rate: 180, Pitch: 200}, false},
		{"language and style", "-l SpanishEs -s estilo x", "x",
			SynthOptions{Language: "SpanishEs", Style: "estilo"}, false},
		{"volume", "-a 80 saludo", "saludo", SynthOptions{Volume: 80}, false},
		{"dash in text", "esto es un - texto", "esto es un - texto", SynthOptions{}, false},
		{"missing value", "-v", "", SynthOptions{}, true},
		{"invalid number", "-r abc x", "", SynthOptions{}, true},
		{"unknown flag", "--bogus x", "", SynthOptions{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTTS(strings.Fields(tt.args))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Text != tt.wantText {
				t.Errorf("text = %q, want %q", got.Text, tt.wantText)
			}
			if got.Opts != tt.wantOpts {
				t.Errorf("opts = %+v, want %+v", got.Opts, tt.wantOpts)
			}
		})
	}
}
