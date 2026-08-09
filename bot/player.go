package loqttsbot

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/hraban/opus"
)

const (
	// Discord voice connections run at 48 kHz, mono, 20 ms frames
	// (960 samples per frame).
	voiceSampleRate = 48000
	voiceFrameSize  = 960
	// voiceFrameDur is the real-time duration of one Opus frame.
	voiceFrameDur = 20 * time.Millisecond
	idleTimeout   = 4 * time.Second
)

// SpeakJob is one requested utterance for a guild.
type SpeakJob struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID // voice channel to play in
	Wav       []byte       // synthesized WAV in memory
}

// Player manages one playback queue and voice connection per guild.
type Player struct {
	mu     sync.Mutex
	voice  voice.Manager
	guilds map[snowflake.ID]*guildPlayer
}

// guildPlayer owns the voice connection and queue of a single guild.
type guildPlayer struct {
	voice voice.Manager
	jobs  chan SpeakJob
	stop  chan struct{}
	conn  voice.Conn
	enc   *opus.Encoder
}

// NewPlayer creates a Player using the given voice manager.
func NewPlayer(vm voice.Manager) *Player {
	return &Player{voice: vm, guilds: make(map[snowflake.ID]*guildPlayer)}
}

// Enqueue adds an utterance to a guild's playback queue.
func (p *Player) Enqueue(job SpeakJob) {
	p.mu.Lock()
	gp, ok := p.guilds[job.GuildID]
	if !ok {
		gp = &guildPlayer{
			voice: p.voice,
			jobs:  make(chan SpeakJob, 32),
			stop:  make(chan struct{}, 1),
		}
		p.guilds[job.GuildID] = gp
		go gp.run()
	}
	p.mu.Unlock()
	gp.jobs <- job
}

// Stop interrupts the current playback of a guild and clears its queue.
func (p *Player) Stop(guildID snowflake.ID) {
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
	if gp.conn == nil || gp.conn.ChannelID() == nil || *gp.conn.ChannelID() != job.ChannelID {
		if gp.conn != nil {
			gp.conn.Close(context.Background())
			gp.conn = nil
		}
		conn := gp.voice.CreateConn(job.GuildID)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := conn.Open(ctx, job.ChannelID, false, false); err != nil {
			return
		}
		// Drain incoming UDP packets so the socket buffer never fills up.
		go func() {
			for {
				if _, err := conn.UDP().ReadPacket(); err != nil {
					return
				}
			}
		}()
		gp.conn = conn
	}
	conn := gp.conn

	w, err := ParseWAV(job.Wav)
	if err != nil {
		return
	}
	// Loquendo is asked for 48 kHz mono; anything else is unexpected.
	if w.SampleRate != voiceSampleRate || w.Channels != 1 {
		return
	}

	ctx := context.Background()
	conn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone)

	frames := len(w.Pcm) / voiceFrameSize
	opusBuf := make([]byte, 4096)
	ticker := time.NewTicker(voiceFrameDur)
	defer ticker.Stop()
	for i := 0; i < frames; i++ {
		select {
		case <-gp.stop:
			conn.SetSpeaking(ctx, voice.SpeakingFlagNone)
			return
		default:
		}
		n, err := gp.enc.Encode(w.Pcm[i*voiceFrameSize:(i+1)*voiceFrameSize], opusBuf)
		if err != nil {
			break
		}
		frame := make([]byte, n)
		copy(frame, opusBuf[:n])
		if _, err := conn.UDP().Write(frame); err != nil {
			break
		}
		select {
		case <-ticker.C:
		case <-gp.stop:
			conn.SetSpeaking(ctx, voice.SpeakingFlagNone)
			return
		}
	}
	conn.SetSpeaking(ctx, voice.SpeakingFlagNone)
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
	if gp.conn != nil {
		gp.conn.Close(context.Background())
		gp.conn = nil
	}
}
