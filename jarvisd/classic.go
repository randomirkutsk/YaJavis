package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ClassicSession: ElevenLabs Scribe -> text model with local tools -> ElevenLabs TTS.
type ClassicSession struct {
	dialog
	ctx        context.Context
	cancel     context.CancelFunc
	key        string
	prof       Profile
	brain      Brain
	client     *http.Client
	listen     bool
	ws         *websocket.Conn
	writeMu    sync.Mutex
	pendingMu  sync.Mutex
	pending    []int16
	out        chan []int16
	started    chan error
	interrupts chan struct{}
	turnMu     sync.Mutex
	turnCancel context.CancelFunc
	generation uint64
	closing    bool
	history    []map[string]string
	jobs       sync.WaitGroup
}

func NewClassicSession(parent context.Context, key string, p Profile, brain Brain, client *http.Client, journal *Journal) *ClassicSession {
	return newClassicSession(parent, key, p, brain, client, journal, true)
}
func NewClassicSpeaker(parent context.Context, key string, p Profile, client *http.Client, journal *Journal) *ClassicSession {
	return newClassicSession(parent, key, p, nil, client, journal, false)
}
func newClassicSession(parent context.Context, key string, p Profile, brain Brain, client *http.Client, journal *Journal, listen bool) *ClassicSession {
	ctx, cancel := context.WithCancel(parent)
	classicDefaults(&p)
	s := &ClassicSession{ctx: ctx, cancel: cancel, key: key, prof: p, brain: brain, client: client, listen: listen, out: make(chan []int16, 8), started: make(chan error, 1), interrupts: make(chan struct{}, 1)}
	s.dialog.journal, s.dialog.route, s.dialog.model, s.dialog.lastActivity = journal, "ElevenLabs классический", p.LLMModel, time.Now()
	go s.run()
	return s
}
func (s *ClassicSession) Audio() <-chan []int16          { return s.out }
func (s *ClassicSession) Rate() int                      { return elevenRate }
func (s *ClassicSession) Interruptions() <-chan struct{} { return s.interrupts }
func (s *ClassicSession) OutBoost() float64              { return 0.35 }
func (s *ClassicSession) Ready(timeout time.Duration) error {
	select {
	case e := <-s.started:
		return e
	case <-time.After(timeout):
		return errors.New("ElevenLabs долго не отвечает")
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}
func (s *ClassicSession) Feed(pcm []int16) {
	if !s.listen {
		return
	}
	s.pendingMu.Lock()
	s.pending = queueMic(s.pending, pcm)
	s.pendingMu.Unlock()
}
func (s *ClassicSession) send(v any) error {
	b, _ := json.Marshal(v)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.ws.Write(s.ctx, websocket.MessageText, b)
}
func (s *ClassicSession) run() {
	defer func() {
		s.cancel()
		s.turnMu.Lock()
		s.closing = true
		if s.turnCancel != nil {
			s.turnCancel()
		}
		s.turnMu.Unlock()
		s.jobs.Wait()
		close(s.out)
	}()
	if s.key == "" || s.prof.TTSVoice == "" {
		s.started <- errors.New("нужны ключ и голос ElevenLabs")
		return
	}
	if !s.listen {
		s.started <- nil
		<-s.ctx.Done()
		return
	}
	q := url.Values{"model_id": {"scribe_v2_realtime"}, "audio_format": {"pcm_16000"}, "language_code": {"ru"}, "commit_strategy": {"vad"}, "vad_silence_threshold_secs": {"0.5"}, "vad_threshold": {"0.4"}, "keyterms": {"Джарвис", "апвоут"}}
	if s.prof.STTModel == "scribe_v3_live" {
		q.Set("model_id", "scribe_v3_live")
		q.Set("commit_strategy", "turn_prediction")
	}
	ctx, cancel := context.WithTimeout(s.ctx, 12*time.Second)
	ws, resp, err := websocket.Dial(ctx, "wss://api.elevenlabs.io/v1/speech-to-text/realtime?"+q.Encode(), &websocket.DialOptions{HTTPClient: s.client, HTTPHeader: http.Header{"xi-api-key": {s.key}}})
	cancel()
	if err != nil {
		if resp != nil {
			err = fmt.Errorf("ElevenLabs Scribe ответил %d", resp.StatusCode)
		}
		s.started <- err
		return
	}
	s.ws = ws
	defer ws.CloseNow()
	ws.SetReadLimit(2 << 20)
	ctx, cancel = context.WithTimeout(s.ctx, 10*time.Second)
	_, raw, err := ws.Read(ctx)
	cancel()
	var first struct {
		Type  string `json:"message_type"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(raw, &first)
	if err != nil || first.Type != "session_started" {
		s.started <- errors.New("ElevenLabs Scribe не открыл распознавание: " + first.Type)
		return
	}
	log.Printf("ElevenLabs: классический режим готов, Scribe → %s → %s", s.prof.LLMModel, s.prof.TTSModel)
	s.started <- nil
	s.jobs.Add(1)
	go func() { defer s.jobs.Done(); s.feeder() }()
	for {
		_, raw, err := ws.Read(s.ctx)
		if err != nil {
			return
		}
		var ev struct {
			Type  string `json:"message_type"`
			Text  string `json:"text"`
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &ev) != nil {
			continue
		}
		if os.Getenv("JARVIS_SCRIBE_DEBUG") != "" {
			log.Printf("Scribe event: %.800s", raw)
		}
		switch ev.Type {
		case "partial_transcript":
			if strings.TrimSpace(ev.Text) != "" {
				s.turnMu.Lock()
				active := s.turnCancel != nil
				s.turnMu.Unlock()
				if active {
					s.Mute()
					select {
					case s.interrupts <- struct{}{}:
					default:
					}
				}
			}
		case "committed_transcript":
			if text := strings.TrimSpace(ev.Text); text != "" {
				s.Mute()
				s.FinishTurn("")
				s.onUser(text)
				s.dialog.mu.Lock()
				s.dialog.muted = false
				s.dialog.mu.Unlock()
				s.respond(text)
			}
		case "error", "auth_error", "quota_exceeded", "rate_limited", "input_error", "transcriber_error":
			if ev.Type == "quota_exceeded" || quotaText(ev.Error) {
				elevenQuota(s.key, "Scribe: "+ev.Type)
			}
			s.FinishTurn("ElevenLabs Scribe: " + ev.Type)
			log.Printf("ElevenLabs Scribe: %s", ev.Type)
			return
		}
	}
}
func (s *ClassicSession) feeder() {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
		}
		s.pendingMu.Lock()
		pcm := s.pending
		s.pending = nil
		s.pendingMu.Unlock()
		if len(pcm) == 0 {
			pcm = make([]int16, 1600)
		}
		b := make([]byte, 2*len(pcm))
		for i, v := range pcm {
			x := max(-32767, min(32767, float64(v)*5.6))
			binary.LittleEndian.PutUint16(b[2*i:], uint16(int16(x)))
		}
		if err := s.send(map[string]any{"message_type": "input_audio_chunk", "audio_base_64": base64.StdEncoding.EncodeToString(b), "sample_rate": 16000}); err != nil {
			s.cancel()
			return
		}
	}
}
func (s *ClassicSession) Mute() {
	s.dialog.Mute()
	s.turnMu.Lock()
	defer s.turnMu.Unlock()
	s.generation++
	if s.turnCancel != nil {
		s.turnCancel()
		s.turnCancel = nil
	}
	for {
		select {
		case _, ok := <-s.out:
			if !ok {
				return
			}
		default:
			return
		}
	}
}
func (s *ClassicSession) newJob() (context.Context, uint64) {
	s.turnMu.Lock()
	defer s.turnMu.Unlock()
	if s.closing || s.ctx.Err() != nil {
		return s.ctx, 0
	}
	if s.turnCancel != nil {
		s.turnCancel()
	}
	s.generation++
	ctx, cancel := context.WithTimeout(s.ctx, 90*time.Second)
	s.turnCancel = cancel
	s.jobs.Add(1)
	return ctx, s.generation
}
func (s *ClassicSession) finishJob(gen uint64) {
	s.turnMu.Lock()
	defer s.turnMu.Unlock()
	if gen == s.generation {
		if s.turnCancel != nil {
			s.turnCancel()
		}
		s.turnCancel = nil
	}
}
func (s *ClassicSession) respond(text string) {
	ctx, gen := s.newJob()
	if gen == 0 {
		return
	}
	s.working(1)
	go func() {
		defer s.jobs.Done()
		defer s.working(-1)
		defer s.finishJob(gen)
		select {
		case <-ctx.Done():
			return
		case <-time.After(450 * time.Millisecond):
		}
		if s.brain == nil {
			return
		}
		s.turnMu.Lock()
		history := append([]map[string]string(nil), s.history...)
		s.turnMu.Unlock()
		task := text
		if len(history) > 0 {
			b, _ := json.Marshal(history)
			task = "Предыдущие реплики текущего разговора: " + string(b) + "\nТекущая просьба: " + text
		}
		answer, err := s.brain(ctx, task)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.FinishTurn("Текстовая модель: " + shortErr(err))
			return
		}
		s.onBot(answer)
		s.turnMu.Lock()
		s.history = append(s.history, map[string]string{"user": text, "assistant": answer})
		if len(s.history) > 8 {
			s.history = s.history[len(s.history)-8:]
		}
		s.turnMu.Unlock()
		if err = s.speak(ctx, gen, answer); err != nil && ctx.Err() == nil {
			s.FinishTurn("ElevenLabs: " + shortErr(err))
		}
	}()
}
func (s *ClassicSession) speak(ctx context.Context, gen uint64, text string) error {
	playhead := time.Now()
	return elevenSpeak(ctx, s.client, s.key, s.prof, text, func(pcm []int16) error {
		now := time.Now()
		if playhead.Before(now) {
			playhead = now
		}
		if ahead := playhead.Sub(now) - 200*time.Millisecond; ahead > 0 {
			select {
			case <-time.After(ahead):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		s.turnMu.Lock()
		if gen != s.generation {
			s.turnMu.Unlock()
			return context.Canceled
		}
		select {
		case s.out <- pcm:
		default:
		}
		s.turnMu.Unlock()
		playhead = playhead.Add(time.Duration(len(pcm)) * time.Second / elevenRate)
		s.heardVoice()
		return ctx.Err()
	})
}
func (s *ClassicSession) Say(text string) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	ctx, gen := s.newJob()
	if gen == 0 {
		return context.Canceled
	}
	s.working(1)
	go func() {
		defer s.jobs.Done()
		defer s.working(-1)
		defer s.finishJob(gen)
		if err := s.speak(ctx, gen, text); err != nil && ctx.Err() == nil {
			s.FinishTurn(shortErr(err))
		}
	}()
	return nil
}
func (s *ClassicSession) Close() { s.cancel() }

func (e *Engine) classicBrain(cfg Settings, p Profile) Brain {
	p.BackendModel = p.LLMModel
	if p.BackendModel == "" {
		p.BackendModel = cfg.ActiveProfile().BackendModel
	}
	instructions := e.instructions(cfg, p) + "\n" + backendRules + e.tools.Context()
	return func(ctx context.Context, text string) (string, error) {
		if cfg.OpenAIKey != "" {
			return askOpenAI(ctx, e.proxy.Client(60*time.Second), cfg.OpenAIKey, p, instructions, text, e.tools)
		}
		if cfg.GeminiKey != "" {
			pref := p.LLMModel
			if pref == "" {
				pref = p.BackendModel
			}
			return askGemini(ctx, e.proxy.ClientFor("gemini", 60*time.Second), cfg.GeminiKey, pref, instructions, text, p.WebSearch)
		}
		return "", errors.New("нет ключа OpenAI или Gemini")
	}
}
