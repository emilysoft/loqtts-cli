package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"loqttsbot"
)

const ttsUsage = "Uso: `!tts [opciones] texto`\n" +
	"Opciones: `-v <voz>`, `-l <idioma>`, `-s <estilo>`, `-r <palabras/min>`, " +
	"`-p <hz>`, `-a <0-100>`\nOtros: `!tts stop` corta la reproducción.\n" +
	"`!loqtts --list-voices`, `!loqtts --list-languages`, `!loqtts --version`"

type bot struct {
	cfg    loqttsbot.Config
	synth  *loqttsbot.Synthesizer
	player *loqttsbot.Player
}

func main() {
	cfg, err := loqttsbot.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsMessageContent

	synth, err := loqttsbot.NewSynthesizer(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	b := &bot{cfg: cfg, synth: synth, player: loqttsbot.NewPlayer(dg)}
	dg.AddHandler(b.onMessageCreate)

	if err := dg.Open(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("loqtts bot online. Ctrl-C to quit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}

func (b *bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.GuildID == "" {
		return
	}
	content := m.Content
	switch {
	case strings.HasPrefix(content, "!tts"):
		b.handleTTS(s, m, strings.TrimSpace(strings.TrimPrefix(content, "!tts")))
	case strings.HasPrefix(content, "!loqtts"):
		b.handleLoqtts(s, m, strings.TrimSpace(strings.TrimPrefix(content, "!loqtts")))
	}
}

func (b *bot) handleTTS(s *discordgo.Session, m *discordgo.MessageCreate, args string) {
	if args == "" {
		s.ChannelMessageSend(m.ChannelID, ttsUsage)
		return
	}
	if args == "stop" {
		b.player.Stop(m.GuildID)
		s.ChannelMessageSend(m.ChannelID, "Reproducción detenida.")
		return
	}

	parsed, err := loqttsbot.ParseTTS(strings.Fields(args))
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
		return
	}

	vs, err := s.State.VoiceState(m.GuildID, m.Author.ID)
	if err != nil || vs == nil || vs.ChannelID == "" {
		s.ChannelMessageSend(m.ChannelID, "Únete a un canal de voz primero.")
		return
	}

	// Fill defaults from config when not given per-command.
	opts := parsed.Opts
	if opts.Voice == "" {
		opts.Voice = b.cfg.Voice
	}
	if opts.Language == "" {
		opts.Language = b.cfg.Language
	}
	if opts.Style == "" {
		opts.Style = b.cfg.Style
	}
	if opts.Rate <= 0 {
		opts.Rate = b.cfg.Rate
	}
	if opts.Pitch <= 0 {
		opts.Pitch = b.cfg.Pitch
	}
	if opts.Volume <= 0 {
		opts.Volume = b.cfg.Volume
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		wav, err := b.synth.Speak(ctx, parsed.Text, opts)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error al sintetizar: "+err.Error())
			return
		}
		b.player.Enqueue(loqttsbot.SpeakJob{
			GuildID:   m.GuildID,
			ChannelID: vs.ChannelID,
			Wav:       wav,
		})
	}()
}

func (b *bot) handleLoqtts(s *discordgo.Session, m *discordgo.MessageCreate, args string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var out string
	var err error
	switch args {
	case "--list-voices":
		out, err = b.synth.ListVoices(ctx)
	case "--list-languages":
		out, err = b.synth.ListLanguages(ctx)
	case "--version":
		out, err = b.synth.Version(ctx)
	default:
		s.ChannelMessageSend(m.ChannelID,
			"Uso: `!loqtts --list-voices | --list-languages | --version`")
		return
	}
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
		return
	}
	out = strings.TrimSpace(out)
	if out == "" {
		out = "(sin resultado)"
	}
	if len(out) > 1900 {
		out = out[:1900] + "…"
	}
	s.ChannelMessageSend(m.ChannelID, "```"+out+"```")
}
