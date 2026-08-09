package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	disbot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/thomas-vilte/dave-go/session"

	"loqttsbot"
)

const ttsUsage = "Uso: `!tts [opciones] texto`\n" +
	"Opciones: `-v <voz>`, `-l <idioma>`, `-s <estilo>`, `-r <palabras/min>`, " +
	"`-p <hz>`, `-a <0-100>`\nOtros: `!tts stop` corta la reproducción.\n" +
	"`!loqtts --list-voices`, `!loqtts --list-languages`, `!loqtts --version`"

type bot struct {
	cfg    loqttsbot.Config
	client *disbot.Client
	synth  *loqttsbot.Synthesizer
	player *loqttsbot.Player
}

func main() {
	cfg, err := loqttsbot.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	client, err := disgo.New(cfg.Token,
		disbot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds | cache.FlagVoiceStates),
		),
		disbot.WithGatewayConfigOpts(gateway.WithIntents(
			gateway.IntentGuilds |
				gateway.IntentGuildVoiceStates |
				gateway.IntentGuildMessages |
				gateway.IntentMessageContent,
		)),
		disbot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(session.CreateFunc()),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	synth, err := loqttsbot.NewSynthesizer(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	b := &bot{
		cfg:    cfg,
		client: client,
		synth:  synth,
		player: loqttsbot.NewPlayer(client.VoiceManager),
	}

	client.EventManager.AddEventListeners(disbot.NewListenerFunc(b.onMessageCreate))

	if err := client.OpenGateway(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("loqtts bot online. Ctrl-C to quit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client.Close(ctx)
}

func (b *bot) onMessageCreate(e *events.MessageCreate) {
	m := e.Message
	if m.Author.Bot || m.GuildID == nil {
		return
	}
	content := m.Content
	switch {
	case strings.HasPrefix(content, "!tts"):
		b.handleTTS(e, strings.TrimSpace(strings.TrimPrefix(content, "!tts")))
	case strings.HasPrefix(content, "!loqtts"):
		b.handleLoqtts(e, strings.TrimSpace(strings.TrimPrefix(content, "!loqtts")))
	}
}

func (b *bot) send(channelID snowflake.ID, msg string) {
	b.client.Rest.CreateMessage(channelID, discord.MessageCreate{Content: msg})
}

func (b *bot) handleTTS(e *events.MessageCreate, args string) {
	m := e.Message
	if args == "" {
		b.send(m.ChannelID, ttsUsage)
		return
	}
	if args == "stop" {
		b.player.Stop(*m.GuildID)
		b.send(m.ChannelID, "Reproducción detenida.")
		return
	}

	parsed, err := loqttsbot.ParseTTS(strings.Fields(args))
	if err != nil {
		b.send(m.ChannelID, "Error: "+err.Error())
		return
	}

	vs, ok := b.client.Caches.VoiceState(*m.GuildID, m.Author.ID)
	if !ok || vs.ChannelID == nil {
		b.send(m.ChannelID, "Únete a un canal de voz primero.")
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
			b.send(m.ChannelID, "Error al sintetizar: "+err.Error())
			return
		}
		b.player.Enqueue(loqttsbot.SpeakJob{
			GuildID:   *m.GuildID,
			ChannelID: *vs.ChannelID,
			Wav:       wav,
		})
	}()
}

func (b *bot) handleLoqtts(e *events.MessageCreate, args string) {
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
		b.send(e.Message.ChannelID,
			"Uso: `!loqtts --list-voices | --list-languages | --version`")
		return
	}
	if err != nil {
		b.send(e.Message.ChannelID, "Error: "+err.Error())
		return
	}
	out = strings.TrimSpace(out)
	if out == "" {
		out = "(sin resultado)"
	}
	if len(out) > 1900 {
		out = out[:1900] + "…"
	}
	b.send(e.Message.ChannelID, "```"+out+"```")
}
