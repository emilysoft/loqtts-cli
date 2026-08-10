package main

import "testing"

func TestParseCommand(t *testing.T) {
	cases := []struct {
		content string
		prefix  string
		name    string
		args    string
	}{
		{"!tts hola mundo", "!", "tts", "hola mundo"},
		{"!tts", "!", "tts", ""},
		{"!ttsmp3 hola", "!", "ttsmp3", "hola"},
		{">tts hola", ">", "tts", "hola"},
		{">tts", ">", "tts", ""},
		{">ttsmp3 hola", ">", "ttsmp3", "hola"},
		{">ttsmp3", ">", "ttsmp3", ""},
		{">loqtts --version", ">", "loqtts", "--version"},
		{">hola", ">", "", ""},
		{">>tts", ">", "", ""},
		{"hola", ">", "", ""},
	}
	for _, c := range cases {
		name, args := parseCommand(c.content, c.prefix)
		if name != c.name || args != c.args {
			t.Errorf("parseCommand(%q, %q) = (%q, %q), want (%q, %q)", c.content, c.prefix, name, args, c.name, c.args)
		}
	}
}
