package loqttsbot

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
)

const (
	// Discord voice connections run at 48 kHz, mono, 20 ms frames
	// (960 samples per frame).
	voiceSampleRate = 48000
	voiceFrameSize  = 960
	idleTimeout     = 4 * time.Second
)

// SpeakJob is one requested utterance for a guild.
type SpeakJob struct {
	GuildID   string
	ChannelID string // voice channel to play in
	Wav       []byte // synthesized WAV in memory
}

// Player manages one playback queue and voice connection per guild.
type Player struct {
	mu      sync.Mutex
	session *discordgo.Session
	guilds  map[string]*guildPlayer
}

// guildPlayer owns the voice connection and queue of a single guild.
type guildPlayer struct {
	session *discordgo.Session
	jobs    chan SpeakJob
	stop    chan struct{}
	vc      *discordgo.VoiceConnection
	enc     *opus.Encoder
}

// NewPlayer creates a Player for the given Discord session.
func NewPlayer(s *discordgo.Session) *Player {
	return &Player{session: s, guilds: make(map[string]*guildPlayer)}
}

// Enqueue adds an utterance to a guild's playback queue.
func (p *Player) Enqueue(job SpeakJob) {
	p.mu.Lock()
	gp, ok := p.guilds[job.GuildID]
	if !ok {
		gp = &guildPlayer{
			session: p.session,
			jobs:    make(chan SpeakJob, 32),
			stop:    make(chan struct{}, 1),
		}
		p.guilds[job.GuildID] = gp
		go gp.run()
	}
	p.mu.Unlock()
	gp.jobs <- job
}

// Stop interrupts the current playback of a guild and clears its queue.
func (p *Player) Stop(guildID string) {
	p.mu.Lock()
	gp, ok := p.guilds[guildID]
	p.mu.Unlock()
	if !ok {
		return
	}
	select {
	case gp.stop <- struct{}{}:
	default:
	}
}

// run consumes the queue sequentially, keeping the voice connection open
// while messages arrive and leaving it after a short idle period.
func (gp *guildPlayer) run() {
	enc, err := opus.NewEncoder(voiceSampleRate, 1, opus.AppVoIP)
	if err != nil {
		return
	}
	gp.enc = enc

	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	for {
		select {
		case job := <-gp.jobs:
			idle.Reset(idleTimeout)
			gp.play(job)
		case <-idle.C:
			gp.leave()
		case <-gp.stop:
			gp.drain()
			gp.leave()
		}
	}
}

// play connects to the target channel if needed and streams the utterance.
func (gp *guildPlayer) play(job SpeakJob) {
	if gp.vc == nil || gp.vc.ChannelID != job.ChannelID {
		if gp.vc != nil {
			gp.vc.Disconnect()
			gp.vc = nil
		}
		vc, err := gp.session.ChannelVoiceJoin(job.GuildID, job.ChannelID, false, false)
		if err != nil {
			return
		}
		gp.vc = vc
	}
	vc := gp.vc

	w, err := ParseWAV(job.Wav)
	if err != nil {
		return
	}
	// Loquendo is asked for 48 kHz mono; anything else is unexpected.
	if w.SampleRate != voiceSampleRate || w.Channels != 1 {
		return
	}

	vc.Speaking(true)
	frames := len(w.Pcm) / voiceFrameSize
	opusBuf := make([]byte, 4096)
	for i := 0; i < frames; i++ {
		select {
		case <-gp.stop:
			vc.Speaking(false)
			return
		default:
		}
		n, err := gp.enc.Encode(w.Pcm[i*voiceFrameSize:(i+1)*voiceFrameSize], opusBuf)
		if err != nil {
			break
		}
		frame := make([]byte, n)
		copy(frame, opusBuf[:n])
		select {
		case vc.OpusSend <- frame:
		case <-gp.stop:
			vc.Speaking(false)
			return
		}
	}
	vc.Speaking(false)
}

// drain discards the queued utterances (used by !tts stop).
func (gp *guildPlayer) drain() {
	for {
		select {
		case <-gp.jobs:
		default:
			return
		}
	}
}

// leave disconnects from the voice channel if connected.
func (gp *guildPlayer) leave() {
	if gp.vc != nil {
		gp.vc.Disconnect()
		gp.vc = nil
	}
}
