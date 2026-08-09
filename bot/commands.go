package loqttsbot

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsedTTS is the result of parsing a "!tts ..." command.
type ParsedTTS struct {
	Opts SynthOptions
	Text string
}

// ParseTTS parses the arguments after "!tts". Flags take their value as the
// next argument; the first token that is not a flag starts the text (the
// remaining tokens are joined with spaces).
func ParseTTS(args []string) (*ParsedTTS, error) {
	var opts SynthOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("option %s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch a {
		case "-v", "--voice":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			opts.Voice = v
		case "-l", "--language":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			opts.Language = v
		case "-s", "--style":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			opts.Style = v
		case "-r", "--rate":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid rate %q", v)
			}
			opts.Rate = n
		case "-p", "--pitch":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid pitch %q", v)
			}
			opts.Pitch = n
		case "-a", "--volume":
			v, err := next(a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid volume %q", v)
			}
			opts.Volume = n
		default:
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return nil, fmt.Errorf("unknown option %q", a)
			}
			return &ParsedTTS{Opts: opts, Text: strings.Join(args[i:], " ")}, nil
		}
	}
	return nil, fmt.Errorf("no text to speak")
}
