package main

// Веб-настройки на самой колонке (http://<ip колонки>:8080).
// Защита:
//   - пароль (bcrypt), задаётся при установке или при первом входе; сессия — подписанная cookie
//     (HttpOnly, SameSite=Strict), 30 дней;
//   - перебор пароля: после 5 ошибок с адреса — пауза, растущая до 15 минут;
//   - только домашняя сеть (по умолчанию): запросы не из локальных адресов отклоняются;
//   - изменения только JSON-запросами с заголовком X-Jarvis (форма с чужого сайта так не сможет);
//   - ключи и ссылка VLESS обратно в браузер не отдаются — только «задан, …1234»;
//   - строгий CSP, запрет встраивания во фреймы.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed ui
var uiFiles embed.FS

type Web struct {
	store   *Store
	proxy   *Proxy
	engine  *Engine
	journal *Journal
	started time.Time
	dataDir string
	uiDir   string // только для разработки: отдавать интерфейс с диска, а не из бинарника

	failMu sync.Mutex
	fails  map[string]*failInfo

	chatLogin ChatGPTLogin
	updater   *Updater
	stats     *Stats

	mu      sync.Mutex
	resetAt time.Time // пароль сброшен кнопками — новый можно задать в течение 30 минут
}

type failInfo struct {
	n     int
	until time.Time
}

const cookieName = "jarvis_session"

func (w *Web) Serve(addr string) error {
	w.fails = map[string]*failInfo{}
	mux := http.NewServeMux()
	_ = mime.AddExtensionType(".woff2", "font/woff2") // на колонке нет /etc/mime.types
	ui, _ := fs.Sub(uiFiles, "ui")
	if w.uiDir != "" {
		ui = os.DirFS(w.uiDir)
	}
	mux.Handle("GET /", http.FileServerFS(ui))
	mux.HandleFunc("GET /api/state", w.auth(w.state))
	mux.HandleFunc("POST /api/login", w.login)
	mux.HandleFunc("POST /api/logout", w.logout)
	mux.HandleFunc("POST /api/setup", w.setup)
	mux.HandleFunc("PUT /api/settings", w.auth(w.putSettings))
	mux.HandleFunc("PUT /api/eleven-keys", w.auth(w.putElevenKeys))
	mux.HandleFunc("PUT /api/gemini-keys", w.auth(w.putGeminiKeys))
	mux.HandleFunc("POST /api/xiaomi/login", w.auth(w.xiaomiLogin))
	mux.HandleFunc("GET /api/xiaomi/qr", w.auth(w.xiaomiQR))
	mux.HandleFunc("POST /api/xiaomi/refresh", w.auth(w.xiaomiRefresh))
	mux.HandleFunc("DELETE /api/xiaomi", w.auth(w.xiaomiLogout))
	mux.HandleFunc("POST /api/smartlife/login", w.auth(w.smartLifeLogin))
	mux.HandleFunc("GET /api/smartlife/qr", w.auth(w.smartLifeQR))
	mux.HandleFunc("POST /api/smartlife/refresh", w.auth(w.smartLifeRefresh))
	mux.HandleFunc("DELETE /api/smartlife", w.auth(w.smartLifeLogout))
	mux.HandleFunc("PUT /api/profile", w.auth(w.putProfile))
	mux.HandleFunc("DELETE /api/profile", w.auth(w.deleteProfile))
	mux.HandleFunc("POST /api/active", w.auth(w.setActive))
	mux.HandleFunc("POST /api/profiles/order", w.auth(w.orderProfiles))
	mux.HandleFunc("PUT /api/prompt", w.auth(w.putPrompt))
	mux.HandleFunc("DELETE /api/prompt", w.auth(w.deletePrompt))
	mux.HandleFunc("PUT /api/keys", w.auth(w.putKeys))
	mux.HandleFunc("POST /api/chatgpt/login", w.auth(w.chatgptLogin))
	mux.HandleFunc("DELETE /api/chatgpt", w.auth(w.chatgptLogout))
	mux.HandleFunc("POST /api/chatgpt/newchat", w.auth(w.chatgptNewChat))
	mux.HandleFunc("PUT /api/vless", w.auth(w.putVless))
	mux.HandleFunc("PUT /api/youtube", w.auth(w.putYouTube))
	mux.HandleFunc("GET /api/elevenlabs/voices", w.auth(w.elevenVoices))
	mux.HandleFunc("GET /api/home-assistant", w.auth(w.getHomeAssistant))
	mux.HandleFunc("POST /api/home-assistant/rooms/discover", w.auth(w.discoverHARooms))
	mux.HandleFunc("PUT /api/home-assistant", w.auth(w.putHomeAssistant))
	mux.HandleFunc("DELETE /api/home-assistant", w.auth(w.deleteHomeAssistant))
	mux.HandleFunc("GET /api/telegram", w.auth(w.getTelegram))
	mux.HandleFunc("PUT /api/telegram", w.auth(w.putTelegram))
	mux.HandleFunc("POST /api/telegram/test", w.auth(w.testTelegram))
	mux.HandleFunc("DELETE /api/telegram", w.auth(w.deleteTelegram))
	mux.HandleFunc("GET /api/memory", w.auth(w.getMemory))
	mux.HandleFunc("POST /api/memory", w.auth(w.addMemory))
	mux.HandleFunc("DELETE /api/memory", w.auth(w.deleteMemory))
	mux.HandleFunc("POST /api/youtube/login", w.auth(w.youtubeLogin))
	mux.HandleFunc("DELETE /api/youtube/login", w.auth(w.youtubeLogout))
	mux.HandleFunc("GET /api/music", w.auth(w.getMusic))
	mux.HandleFunc("GET /api/stats", w.auth(w.getStats))
	mux.HandleFunc("GET /api/logs", w.auth(w.getLogs))
	mux.HandleFunc("POST /api/music", w.auth(w.postMusic))
	mux.HandleFunc("POST /api/leds/preview", w.auth(w.previewLeds))
	mux.HandleFunc("POST /api/speaker/volume", w.auth(w.setSpeakerVolume))
	mux.HandleFunc("GET /api/bgrec", w.auth(w.getBgRec))
	mux.HandleFunc("PUT /api/bgrec", w.auth(w.putBgRec))
	mux.HandleFunc("POST /api/bgrec/start", w.auth(w.startBgRec))
	mux.HandleFunc("POST /api/bgrec/stop", w.auth(w.stopBgRec))
	mux.HandleFunc("PUT /api/gdrive", w.auth(w.putGDrive))
	mux.HandleFunc("POST /api/gdrive/login", w.auth(w.gdriveLogin))
	mux.HandleFunc("DELETE /api/gdrive/login", w.auth(w.gdriveLogout))
	mux.HandleFunc("GET /api/train", w.auth(w.getTrain))
	mux.HandleFunc("GET /api/train/clip", w.auth(w.getTrainClip))
	mux.HandleFunc("POST /api/train/move", w.auth(w.moveTrainClip))
	mux.HandleFunc("DELETE /api/train", w.auth(w.clearTrain))
	mux.HandleFunc("GET /api/train/download", w.auth(w.downloadTrain))
	mux.HandleFunc("PUT /api/train/model", w.auth(w.putWakeModel))
	mux.HandleFunc("POST /api/train/model/restore", w.auth(w.restoreWakeModel))
	mux.HandleFunc("GET /api/update", w.auth(w.getUpdate))
	mux.HandleFunc("PUT /api/update", w.auth(w.putUpdateSettings))
	mux.HandleFunc("POST /api/update/check", w.auth(w.checkUpdate))
	mux.HandleFunc("POST /api/update/install", w.auth(w.installUpdate))
	mux.HandleFunc("PUT /api/update/upload", w.auth(w.uploadUpdate))
	mux.HandleFunc("DELETE /api/youtube", w.auth(w.deleteYouTube))
	mux.HandleFunc("POST /api/test/net", w.auth(w.testNet))
	mux.HandleFunc("POST /api/talk", w.auth(w.talk))
	mux.HandleFunc("POST /api/say", w.auth(w.say))
	mux.HandleFunc("POST /api/stop", w.auth(w.stop))
	mux.HandleFunc("POST /api/preview", w.auth(w.preview))
	mux.HandleFunc("POST /api/voices", w.auth(w.createVoice))
	mux.HandleFunc("DELETE /api/voices", w.auth(w.deleteVoice))
	mux.HandleFunc("PUT /api/password", w.auth(w.changePassword))
	mux.HandleFunc("GET /api/journal", w.auth(w.getJournal))
	srv := &http.Server{Addr: addr, Handler: w.guard(mux), ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	log.Printf("веб-настройки: http://<ip колонки>%s", addr)
	return srv.ListenAndServe()
}

// guard — общие правила для всех запросов.
func (w *Web) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		h := rw.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https://i.ytimg.com https://avatars.yandex.net; style-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		if w.store.Get().LANOnly && !localAddr(r.RemoteAddr) {
			http.Error(rw, "Доступ только из домашней сети", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Jarvis") != "1" {
			http.Error(rw, "нужен заголовок X-Jarvis", http.StatusForbidden)
			return
		}
		limit, long := int64(256<<10), false
		switch r.URL.Path {
		case "/api/update/upload": // новая версия jarvisd (~30 МБ)
			limit, long = 80<<20, true
		case "/api/train/model":
			limit = 4 << 20
		case "/api/youtube":
			limit = 1 << 20
		case "/api/music", "/api/update/install", "/api/train/download":
			long = true
		}
		if long { // загрузка, скачивание или поиск на YouTube дольше обычных 30/60 с
			rc := http.NewResponseController(rw)
			_ = rc.SetReadDeadline(time.Now().Add(15 * time.Minute))
			_ = rc.SetWriteDeadline(time.Now().Add(15 * time.Minute))
		}
		r.Body = http.MaxBytesReader(rw, r.Body, limit)
		next.ServeHTTP(rw, r)
	})
}

func localAddr(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func (w *Web) sign(exp int64) string {
	m := hmac.New(sha256.New, []byte(w.store.Get().SessionKey))
	fmt.Fprintf(m, "%d", exp)
	return fmt.Sprintf("%d.%s", exp, hex.EncodeToString(m.Sum(nil)))
}

func (w *Web) loggedIn(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	exp, _, ok := strings.Cut(c.Value, ".")
	n, err := strconv.ParseInt(exp, 10, 64)
	if !ok || err != nil || time.Now().Unix() > n {
		return false
	}
	return hmac.Equal([]byte(c.Value), []byte(w.sign(n)))
}

func (w *Web) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if w.store.Get().PasswordHash == "" {
			writeJSON(rw, 428, map[string]any{"setup": true, "error": "Задайте пароль для настроек"})
			return
		}
		if !w.loggedIn(r) {
			writeJSON(rw, 401, map[string]any{"error": "Нужно войти"})
			return
		}
		h(rw, r)
	}
}

func writeJSON(rw http.ResponseWriter, code int, v any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

func bad(rw http.ResponseWriter, msg string) { writeJSON(rw, 400, map[string]any{"error": msg}) }

func (w *Web) setCookie(rw http.ResponseWriter) {
	exp := time.Now().Add(30 * 24 * time.Hour).Unix()
	http.SetCookie(rw, &http.Cookie{Name: cookieName, Value: w.sign(exp), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: 30 * 24 * 3600})
}

func clientIP(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func (w *Web) login(rw http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	w.failMu.Lock()
	f := w.fails[ip]
	if f != nil && time.Now().Before(f.until) {
		wait := time.Until(f.until).Round(time.Second)
		w.failMu.Unlock()
		writeJSON(rw, 429, map[string]any{"error": fmt.Sprintf("Слишком много попыток. Подождите %v", wait)})
		return
	}
	w.failMu.Unlock()
	var req struct{ Password string }
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	hash := w.store.Get().PasswordHash
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		w.failMu.Lock()
		if f == nil {
			f = &failInfo{}
			w.fails[ip] = f
		}
		f.n++
		if f.n >= 5 {
			f.until = time.Now().Add(min(15*time.Minute, time.Duration(f.n-4)*30*time.Second))
		}
		w.failMu.Unlock()
		time.Sleep(700 * time.Millisecond)
		writeJSON(rw, 401, map[string]any{"error": "Неверный пароль"})
		return
	}
	w.failMu.Lock()
	delete(w.fails, ip)
	w.failMu.Unlock()
	w.setCookie(rw)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) logout(rw http.ResponseWriter, r *http.Request) {
	http.SetCookie(rw, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func checkPassword(p string) error {
	if len([]rune(p)) < 6 {
		return errors.New("пароль — минимум 6 символов")
	}
	return nil
}

// setup — первый пароль. Только пока пароля нет и только в первые 30 минут после запуска
// (чтобы чужой в сети не успел задать свой, если установщик пароль не поставил).
func (w *Web) setup(rw http.ResponseWriter, r *http.Request) {
	if w.store.Get().PasswordHash != "" {
		writeJSON(rw, 409, map[string]any{"error": "Пароль уже задан"})
		return
	}
	w.mu.Lock()
	resetAt := w.resetAt
	w.mu.Unlock()
	if time.Since(w.started) > 30*time.Minute && time.Since(resetAt) > 30*time.Minute {
		writeJSON(rw, 403, map[string]any{"error": "Для безопасности первый пароль можно задать только в первые 30 минут после включения колонки. Перезагрузите её."})
		return
	}
	var req struct{ Password string }
	if readJSON(r, &req) != nil || checkPassword(req.Password) != nil {
		bad(rw, "пароль — минимум 6 символов")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	_ = w.store.Update(func(s *Settings) error {
		if s.PasswordHash != "" {
			return errors.New("уже задан")
		}
		s.PasswordHash = string(hash)
		return nil
	})
	w.setCookie(rw)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) changePassword(rw http.ResponseWriter, r *http.Request) {
	var req struct{ Old, New string }
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(w.store.Get().PasswordHash), []byte(req.Old)) != nil {
		writeJSON(rw, 403, map[string]any{"error": "Текущий пароль не тот"})
		return
	}
	if err := checkPassword(req.New); err != nil {
		bad(rw, err.Error())
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.New), 10)
	_ = w.store.Update(func(s *Settings) error {
		s.PasswordHash = string(hash)
		s.SessionKey = randHex(32) // все старые входы — недействительны
		return nil
	})
	w.setCookie(rw)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "задан"
	}
	return "задан, …" + s[len(s)-4:]
}

func (w *Web) state(rw http.ResponseWriter, r *http.Request) {
	s := w.store.Get()
	if s.LiveVoices == nil {
		s.LiveVoices = []CustomVoice{}
	}
	if s.Prompts == nil {
		s.Prompts = []Prompt{}
	}
	link, perr := w.proxy.Status()
	vless := map[string]any{"set": s.Vless != "", "error": perr}
	if link != nil {
		vless["name"], vless["host"], vless["port"] = link.Name, link.Host, link.Port
		vless["network"], vless["security"], vless["sni"] = link.Network, link.Security, link.SNI
	}
	_, noModel := w.engine.wake.(noWake)
	wakeModel := !noModel
	night := s.Night
	night.fill()
	writeJSON(rw, 200, map[string]any{
		"version":  version,
		"profiles": s.Profiles, "active": s.Active, "prompts": s.Prompts, "live_voices": s.LiveVoices,
		"voice_groups": LiveVoiceGroups, "backend_models": BackendModels, "efforts": Efforts, "model_efforts": modelEffortOptions(),
		"keys":        map[string]string{"openai": mask(s.OpenAIKey), "gemini": mask(s.GeminiKey), "grok": mask(s.GrokKey), "elevenlabs": mask(s.ElevenLabsKey), "yandex": mask(s.YandexToken)},
		"eleven_keys": s.ElevenKeysState(), "gemini_keys": GeminiKeysState(s.GeminiKeys), "vpn": s.VPNState(),
		"gemini_models": GeminiModels, "gemini_model_info": GeminiModelInfo, "gemini_voice_groups": GeminiVoiceGroups, "chatgpt_voice_groups": ChatGPTVoiceGroups, "grok_voice_groups": GrokVoiceGroups,
		"chatgpt": map[string]any{"signed_in": s.ChatGPT.SignedIn(), "plan": s.ChatGPT.Plan, "email": s.ChatGPT.Email,
			"login": w.chatLogin.State(), "chat": chatgptChat.State(), "limited_until": chatgptLimitedUntil.Load() / 1e9},
		"vless": vless, "youtube": w.youtubeState(), "home_assistant": w.haState(), "xiaomi": w.xiaomiState(), "smartlife": w.smartLifeState(), "rooms": w.knownRooms(), "update_available": w.updater != nil && w.updater.Available(),
		"speaker": map[string]any{"model": stationModel().Name, "model_id": stationModel().ID, "mic": w.engine.mic.Online(), "leds": w.engine.leds.Available(), "wake_model": wakeModel,
			"talking": w.engine.Talking(), "uptime_s": int(time.Since(w.started).Seconds()), "volume": speakerVolume()},
		"settings": map[string]any{"wake_enabled": s.WakeEnabled, "wake_sensitivity": s.WakeSensitivity, "mic_boost_db": s.MicBoostDB, "mic_gain_db": s.MicGainDB, "wake_check": s.WakeCheck,
			"follow_up_s": s.FollowUpS, "city": s.City, "timezone": s.Timezone, "volume": s.Volume, "lan_only": s.LANOnly, "home_via": s.HomeVia, "home_use": s.HomeUseState(), "speaker_room": s.SpeakerRoom,
			"led": s.Led, "train_mode": s.TrainMode, "train_limit_mb": s.TrainLimitMB, "dev_mode": s.DevMode, "language": s.Language,
			"night": night, "night_now": s.IsNight(), "radio": !s.RadioOff},
		"security": securityChecks(),
		"telegram": w.telegramState(),
		"memory":   w.memoryState(),
	})
}

// securityChecks — что на колонке открыто наружу (подсказки на странице «Безопасность»).
func securityChecks() []map[string]any {
	listening := func(port int) bool {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond)
		if err == nil {
			c.Close()
			return true
		}
		return false
	}
	return []map[string]any{
		{"id": "adb", "title": "ADB по сети (порт 5555)", "open": listening(5555),
			"hint": "Через ADB любой в вашей Wi-Fi сети получает полный доступ к колонке без пароля. Нужен только для установки и обновления — потом его можно закрыть в программе установки."},
		{"id": "ssh", "title": "SSH (порт 2222)", "open": listening(2222),
			"hint": "Удалённый вход в консоль колонки. Если не пользуетесь — лучше закрыть; если пользуетесь — только по ключу."},
	}
}

func (w *Web) getJournal(rw http.ResponseWriter, r *http.Request) {
	phase := map[LedMode]string{LedListening: "listening", LedThinking: "thinking", LedSpeaking: "speaking"}[w.engine.leds.Phase()]
	if !w.engine.Talking() {
		phase = ""
	}
	writeJSON(rw, 200, map[string]any{"items": w.journal.Items(50), "mic": w.engine.mic.Online(),
		"talking": w.engine.Talking(), "phase": phase, "sys": readSys(w.dataDir), "volume": speakerVolume(), "version": version})
}

func (w *Web) putSettings(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		WakeEnabled     *bool           `json:"wake_enabled"`
		WakeCheck       *bool           `json:"wake_check"`
		VPN             map[string]bool `json:"vpn"`
		WakeSensitivity *int            `json:"wake_sensitivity"`
		MicBoostDB      *int            `json:"mic_boost_db"`
		MicGainDB       *int            `json:"mic_gain_db"`
		FollowUpS       *float64        `json:"follow_up_s"`
		City            *string         `json:"city"`
		Timezone        *string         `json:"timezone"`
		Volume          *int            `json:"volume"`
		LANOnly         *bool           `json:"lan_only"`
		HomeVia         *string         `json:"home_via"`
		HomeUse         map[string]bool `json:"home_use"`
		SpeakerRoom     *string         `json:"speaker_room"`
		DevMode         *bool           `json:"dev_mode"`
		Language        *string         `json:"language"`
		Led             *LedSettings    `json:"led"`
		TrainMode       *bool           `json:"train_mode"`
		TrainLimitMB    *int            `json:"train_limit_mb"`
		Night           *NightSettings  `json:"night"`
		Radio           *bool           `json:"radio"`
	}
	if readJSON(r, &req) != nil {
		bad(rw, "не понял настройки")
		return
	}
	err := w.store.Update(func(s *Settings) error {
		if req.WakeEnabled != nil {
			s.WakeEnabled = *req.WakeEnabled
		}
		if req.WakeCheck != nil {
			s.WakeCheck = *req.WakeCheck
		}
		for svc, on := range req.VPN {
			if !knownService(svc) {
				return errors.New("неизвестное подключение: " + svc)
			}
			if s.VPN == nil {
				s.VPN = map[string]bool{}
			}
			s.VPN[svc] = on
		}
		if req.WakeSensitivity != nil {
			s.WakeSensitivity = max(1, min(10, *req.WakeSensitivity))
		}
		if req.MicBoostDB != nil {
			s.MicBoostDB = max(0, min(18, *req.MicBoostDB))
		}
		if req.MicGainDB != nil {
			s.MicGainDB = max(0, min(30, *req.MicGainDB))
		}
		if req.FollowUpS != nil {
			s.FollowUpS = max(0, min(30, *req.FollowUpS))
		}
		if req.City != nil {
			s.City = strings.TrimSpace(*req.City)
		}
		if req.Timezone != nil {
			if _, err := time.LoadLocation(*req.Timezone); err != nil {
				return errors.New("неизвестный часовой пояс")
			}
			s.Timezone = *req.Timezone
		}
		if req.Volume != nil {
			s.Volume = max(0, min(100, *req.Volume))
		}
		if req.HomeVia != nil && (*req.HomeVia == "alice" || *req.HomeVia == "jarvis") {
			s.HomeVia = *req.HomeVia
		}
		if req.SpeakerRoom != nil {
			s.SpeakerRoom = strings.TrimSpace(*req.SpeakerRoom)
		}
		for ch, on := range req.HomeUse {
			known := false
			for _, c := range HomeChannels {
				known = known || c == ch
			}
			if !known {
				return errors.New("неизвестный способ управления домом: " + ch)
			}
			if s.HomeUse == nil {
				s.HomeUse = map[string]bool{}
			}
			s.HomeUse[ch] = on
		}
		if req.LANOnly != nil {
			s.LANOnly = *req.LANOnly
		}
		if req.DevMode != nil {
			s.DevMode = *req.DevMode
		}
		if req.Language != nil {
			if _, ok := langNames[*req.Language]; !ok && *req.Language != "" {
				return errors.New("неизвестный язык")
			}
			s.Language = *req.Language
		}
		if req.Led != nil { // общий выключатель (цвета — у ассистентов; старые клиенты присылают и их)
			s.Led.Enabled = req.Led.Enabled
		}
		if req.TrainMode != nil {
			s.TrainMode = *req.TrainMode
		}
		if req.TrainLimitMB != nil {
			s.TrainLimitMB = max(1, min(20, *req.TrainLimitMB))
		}
		if req.Night != nil {
			n := *req.Night
			if _, ok := hhmm(n.From); !ok && n.From != "" {
				return errors.New("время начала ночи — в виде 23:00")
			}
			if _, ok := hhmm(n.To); !ok && n.To != "" {
				return errors.New("время конца ночи — в виде 08:00")
			}
			n.fill()
			s.Night = n
		}
		if req.Radio != nil {
			s.RadioOff = !*req.Radio
		}
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	if req.Led != nil || req.Night != nil {
		s := w.store.Get()
		w.engine.leds.Configure(s.LedFor())
	}
	if req.Night != nil {
		go nightCap(w.store) // включили посреди ночи — громкость колонки сразу в пределах
	}
	if req.Timezone != nil {
		logTZ.set(*req.Timezone)
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// previewLeds — зажечь кольцо: примерить цвет (mode + color, 3 с) или «поиграть» (seconds до 30, mode=off — погасить).
func (w *Web) previewLeds(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode       string
		Color      string
		Brightness int
		Seconds    float64
	}
	_ = readJSON(r, &req)
	if !w.engine.leds.Available() {
		bad(rw, "у этой колонки нет подсветки, которой может управлять Джарвис")
		return
	}
	if req.Mode == "off" {
		w.engine.leds.set(LedOff, true)
		writeJSON(rw, 200, map[string]any{"ok": true})
		return
	}
	mode := map[string]LedMode{"listening": LedListening, "thinking": LedThinking, "speaking": LedSpeaking}[req.Mode]
	if mode == LedOff {
		bad(rw, "неизвестный режим")
		return
	}
	s := w.store.Get()
	l := s.LedFor()
	color := map[LedMode]string{LedListening: l.Listening, LedThinking: l.Thinking, LedSpeaking: l.Speaking}[mode]
	if hexColorRe.MatchString(req.Color) {
		color = req.Color
	}
	bright := l.Brightness
	if req.Brightness > 0 {
		bright = min(100, req.Brightness)
	}
	dur := 3 * time.Second
	if req.Seconds > 0 {
		dur = time.Duration(min(30, req.Seconds) * float64(time.Second))
	}
	w.engine.leds.Show(mode, color, bright, dur)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) putProfile(rw http.ResponseWriter, r *http.Request) {
	var p Profile
	if readJSON(r, &p) != nil {
		bad(rw, "не понял профиль")
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len([]rune(p.Name)) > 40 {
		bad(rw, "имя — от 1 до 40 символов")
		return
	}
	if p.Mode != "live" && p.Mode != "classic" {
		p.Mode = "live"
	}
	p.Effort = normalizedModelEffort(p.BackendModel, p.Effort)
	if isClassic(p) {
		classicDefaults(&p)
	} else if p.Engine == "eleven_realtime" {
		p.Mode = "live"
		p.LiveModel = elevenRealtimeModel(p.LiveModel)
	} else if p.Engine == "gemini" {
		p.Voice, p.LiveModel = geminiVoice(p.Voice), geminiModel(p.LiveModel)
	} else if p.Engine == "chatgpt" {
		p.Voice, p.LiveModel = chatgptVoice(p.Voice), ""
	} else if p.Engine == "grok" {
		p.Voice, p.LiveModel = grokVoice(p.Voice), ""
	} else {
		p.Engine, p.LiveModel = "openai", ""
	}
	if p.IdleS <= 0 {
		p.IdleS = 8
	}
	p.IdleS = max(2, min(60, p.IdleS))
	p.City = strings.TrimSpace(p.City)
	if p.Timezone != "" {
		if _, err := time.LoadLocation(p.Timezone); err != nil {
			bad(rw, "неизвестный часовой пояс")
			return
		}
	}
	if p.Volume != 0 {
		p.Volume = max(10, min(100, p.Volume))
	}
	if p.Led != nil {
		p.Led.Brightness = max(5, min(100, p.Led.Brightness))
		for _, c := range []*string{&p.Led.Listening, &p.Led.Thinking, &p.Led.Speaking} {
			if !hexColorRe.MatchString(*c) {
				bad(rw, "цвет подсветки — в виде #rrggbb")
				return
			}
		}
	}
	glob := w.store.Get()
	p.fillDefaults(&glob)
	err := w.store.Update(func(s *Settings) error {
		for i := range s.Profiles {
			if s.Profiles[i].ID == p.ID {
				s.Profiles[i] = p
				return nil
			}
		}
		for _, x := range s.Profiles {
			if strings.EqualFold(x.Name, p.Name) {
				return errors.New("ассистент с таким именем уже есть")
			}
		}
		p.ID = "p" + randHex(4)
		s.Profiles = append(s.Profiles, p)
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true, "id": p.ID})
}

func (w *Web) deleteProfile(rw http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	err := w.store.Update(func(s *Settings) error {
		if s.Active == id {
			return errors.New("активного ассистента удалить нельзя — сначала включите другого")
		}
		out := s.Profiles[:0]
		for _, p := range s.Profiles {
			if p.ID != id {
				out = append(out, p)
			}
		}
		s.Profiles = out
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) setActive(rw http.ResponseWriter, r *http.Request) {
	var req struct{ ID string }
	_ = readJSON(r, &req)
	err := w.store.Update(func(s *Settings) error {
		for _, p := range s.Profiles {
			if p.ID == req.ID {
				s.Active = p.ID
				return nil
			}
		}
		return errors.New("нет такого ассистента")
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) orderProfiles(rw http.ResponseWriter, r *http.Request) {
	var req struct{ IDs []string }
	_ = readJSON(r, &req)
	_ = w.store.Update(func(s *Settings) error {
		pos := map[string]int{}
		for i, id := range req.IDs {
			pos[id] = i + 1
		}
		out := make([]Profile, len(s.Profiles))
		copy(out, s.Profiles)
		for i := 1; i < len(out); i++ {
			for j := i; j > 0 && rank(pos, out[j].ID) < rank(pos, out[j-1].ID); j-- {
				out[j], out[j-1] = out[j-1], out[j]
			}
		}
		s.Profiles = out
		return nil
	})
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func rank(pos map[string]int, id string) int {
	if p, ok := pos[id]; ok {
		return p
	}
	return 1 << 20
}

func (w *Web) putPrompt(rw http.ResponseWriter, r *http.Request) {
	var p Prompt
	if readJSON(r, &p) != nil || strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Text) == "" {
		bad(rw, "нужны название и текст")
		return
	}
	if len(p.Text) > 20000 {
		bad(rw, "слишком длинный текст")
		return
	}
	_ = w.store.Update(func(s *Settings) error {
		for i := range s.Prompts {
			if s.Prompts[i].ID == p.ID {
				s.Prompts[i] = p
				return nil
			}
		}
		p.ID = "c" + randHex(4)
		s.Prompts = append(s.Prompts, p)
		return nil
	})
	writeJSON(rw, 200, map[string]any{"ok": true, "id": p.ID})
}

func (w *Web) deletePrompt(rw http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	err := w.store.Update(func(s *Settings) error {
		for _, p := range s.Profiles {
			if p.Prompt == id {
				return fmt.Errorf("этот характер использует ассистент «%s»", p.Name)
			}
		}
		out := s.Prompts[:0]
		for _, p := range s.Prompts {
			if p.ID != id {
				out = append(out, p)
			}
		}
		s.Prompts = out
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) putKeys(rw http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	_ = w.store.Update(func(s *Settings) error {
		for k, v := range req {
			v = strings.TrimSpace(v)
			switch k {
			case "openai":
				s.OpenAIKey = v
			case "gemini":
				if v == "" { // ключ добавляется к списку; пустой — убрать все
					s.GeminiKeys = nil
				} else if !slices.Contains(s.GeminiKeys, v) {
					s.GeminiKeys = append(s.GeminiKeys, v)
				}
				s.GeminiKey = ""
				if len(s.GeminiKeys) > 0 {
					s.GeminiKey = s.GeminiKeys[0]
				}
			case "grok":
				s.GrokKey = v
			case "elevenlabs": // ключ добавляется к списку; пустой — убрать все
				if v == "" {
					s.ElevenKeys = nil
				} else {
					s.addElevenKey(v)
				}
				s.pickEleven()
			case "yandex":
				was := s.HomeConnected("yandex")
				s.YandexToken = v
				s.homeJustConnected("yandex", was)
			}
		}
		return nil
	})
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// putElevenKeys — убрать ключ ElevenLabs из списка или вернуть отложенный в работу (по номеру).
// putGeminiKeys — убрать один ключ Google из списка.
func (w *Web) putGeminiKeys(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Remove int `json:"remove"`
	}
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	err := w.store.Update(func(s *Settings) error {
		if req.Remove < 0 || req.Remove >= len(s.GeminiKeys) {
			return errors.New("такого ключа нет")
		}
		s.GeminiKeys = append(s.GeminiKeys[:req.Remove:req.Remove], s.GeminiKeys[req.Remove+1:]...)
		s.GeminiKey = ""
		if len(s.GeminiKeys) > 0 {
			s.GeminiKey = s.GeminiKeys[0]
		}
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) putElevenKeys(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Remove *int `json:"remove"`
		Reset  *int `json:"reset"`
	}
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	err := w.store.Update(func(s *Settings) error {
		pick := func(i *int) (int, error) {
			if *i < 0 || *i >= len(s.ElevenKeys) {
				return 0, errors.New("такого ключа нет")
			}
			return *i, nil
		}
		if req.Remove != nil {
			i, err := pick(req.Remove)
			if err != nil {
				return err
			}
			s.ElevenKeys = append(s.ElevenKeys[:i], s.ElevenKeys[i+1:]...)
		}
		if req.Reset != nil {
			i, err := pick(req.Reset)
			if err != nil {
				return err
			}
			s.ElevenKeys[i].Until, s.ElevenKeys[i].Why = 0, ""
		}
		s.pickEleven()
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) putVless(rw http.ResponseWriter, r *http.Request) {
	var req struct{ Link string }
	if readJSON(r, &req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	link := strings.TrimSpace(req.Link)
	if link != "" {
		if _, err := ParseVless(link); err != nil {
			bad(rw, err.Error())
			return
		}
	}
	if err := w.proxy.Apply(link); err != nil {
		bad(rw, err.Error())
		return
	}
	_ = w.store.Update(func(s *Settings) error { s.Vless = link; return nil })
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// youtubeState — загружен ли вход в YouTube (сами куки в браузер не отдаются).
func (w *Web) youtubeState() map[string]any {
	if w.engine.music == nil || w.engine.music.CookieFile == "" {
		return map[string]any{"set": false}
	}
	_, err := os.Stat(w.engine.music.CookieFile)
	out := map[string]any{"set": err == nil}
	if w.engine.music.Account != nil {
		out["account"] = w.engine.music.Account.State()
	}
	return out
}

// putYouTube — куки аккаунта YouTube (cookies.txt): проверяем, что вход принят, и кладём на колонку.
func (w *Web) putYouTube(rw http.ResponseWriter, r *http.Request) {
	var req struct{ Cookies string }
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req) != nil {
		bad(rw, "пустой запрос")
		return
	}
	if w.engine.music == nil || w.engine.music.CookieFile == "" {
		bad(rw, "музыка с YouTube выключена")
		return
	}
	c := parseYTCookies(req.Cookies)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := w.engine.music.CheckLogin(ctx, c); err != nil {
		bad(rw, err.Error())
		return
	}
	if err := os.WriteFile(w.engine.music.CookieFile, []byte(req.Cookies), 0o600); err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) deleteYouTube(rw http.ResponseWriter, r *http.Request) {
	if w.engine.music != nil && w.engine.music.CookieFile != "" {
		_ = os.Remove(w.engine.music.CookieFile)
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// testNet — проверить, что OpenAI (и ElevenLabs) доступны через текущий туннель.
func (w *Web) testNet(rw http.ResponseWriter, r *http.Request) {
	s := w.store.Get()
	check := func(svc, url, key, header string) map[string]any {
		client := w.proxy.ClientFor(svc, 12*time.Second)
		t0 := time.Now()
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
		if key != "" {
			req.Header.Set(header, key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return map[string]any{"ok": false, "error": shortErr(err)}
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
		ok := resp.StatusCode >= 200 && resp.StatusCode < 300 // YouTube отвечает 204 — это и есть «работает»
		res := map[string]any{"ok": ok, "code": resp.StatusCode, "ms": time.Since(t0).Milliseconds()}
		switch {
		case resp.StatusCode == 401:
			res["error"] = "ключ не подошёл"
		case resp.StatusCode == 403:
			res["error"] = "доступ запрещён — похоже, сервис видит российский/белорусский адрес, нужен VLESS"
		case !ok:
			res["error"] = fmt.Sprintf("ответ %d", resp.StatusCode)
		}
		return res
	}
	out := map[string]any{}
	bearer := ""
	if s.OpenAIKey != "" {
		bearer = "Bearer " + s.OpenAIKey
	}
	out["openai"] = check("openai", "https://api.openai.com/v1/models", bearer, "Authorization")
	if s.GeminiKey != "" {
		out["gemini"] = check("gemini", "https://"+geminiHost+"/v1beta/models?pageSize=1", s.GeminiKey, "x-goog-api-key")
	}
	if s.GrokKey != "" {
		out["grok"] = check("grok", "https://api.x.ai/v1/models", "Bearer "+s.GrokKey, "Authorization")
	}
	if s.ElevenLabsKey != "" {
		out["elevenlabs"] = check("elevenlabs", "https://api.elevenlabs.io/v2/voices?page_size=1", s.ElevenLabsKey, "xi-api-key")
	}
	if s.YouTubeRefresh != "" || w.engine.music != nil {
		out["youtube"] = check("youtube", "https://www.youtube.com/generate_204", "", "")
	}
	writeJSON(rw, 200, out)
}

func (w *Web) talk(rw http.ResponseWriter, r *http.Request) {
	w.engine.Trigger("кнопка в настройках")
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) stop(rw http.ResponseWriter, r *http.Request) {
	w.engine.Stop("кнопка в настройках")
	writeJSON(rw, 200, map[string]any{"ok": true})
}

const previewPhrase = "Привет! Я Джарвис. Так звучит мой голос."

// voiceFile — где лежит готовая пробная фраза голоса (PCM s16le, 16 кГц). Генерируется один раз,
// дальше играет с диска: бесплатно и мгновенно. Имя зависит от фразы и описания голоса.
func (w *Web) voiceFile(voice, prompt string) string {
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			return r
		}
		return -1
	}, voice)
	sum := sha256.Sum256([]byte(previewPhrase + "\n" + prompt))
	return filepath.Join(w.dataDir, "voices", name+"-"+hex.EncodeToString(sum[:4])+".s16")
}

// playPCM — отдать плееру весь кусок (очередь вмещает ~50 с) и дать доиграть в фоне.
func (w *Web) playPCM(pcm []int16, rate int, boost float64) error {
	if err := w.engine.player.Start(rate); err != nil {
		return err
	}
	w.engine.player.SetBoost(boost)
	for i := 0; i < len(pcm); i += 1600 {
		_ = w.engine.player.Write(pcm[i:min(i+1600, len(pcm))])
	}
	go w.engine.player.Finish(30 * time.Second)
	return nil
}

func saveVoice(path string, pcm []int16) {
	end := len(pcm) // срезать хвостовую тишину, оставив 0.3 с
	for end > 0 && pcm[end-1] < 600 && pcm[end-1] > -600 {
		end--
	}
	pcm = pcm[:min(len(pcm), end+MicRate*3/10)]
	if len(pcm) > MicRate*20 {
		pcm = pcm[:MicRate*20]
	}
	b := make([]byte, 2*len(pcm))
	for i, s := range pcm {
		b[2*i], b[2*i+1] = byte(s), byte(s>>8)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// chatgptLogin — начать вход в подписку ChatGPT: панель показывает код и ссылку.
func (w *Web) chatgptLogin(rw http.ResponseWriter, r *http.Request) {
	if err := StartChatGPTLogin(&w.chatLogin, w.store, w.proxy.ClientFor("chatgpt", 30*time.Second)); err != nil {
		bad(rw, shortErr(err))
		return
	}
	writeJSON(rw, 200, w.chatLogin.State())
}

// chatgptNewChat — начать новый чат с ChatGPT: прошлые реплики забыть.
func (w *Web) chatgptNewChat(rw http.ResponseWriter, r *http.Request) {
	chatgptChat.Reset()
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) chatgptLogout(rw http.ResponseWriter, r *http.Request) {
	_ = w.store.Update(func(s *Settings) error { s.ChatGPT = ChatGPTAuth{}; return nil })
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// preview — колонка произносит пробную фразу выбранным голосом. Первый раз — короткая Live-сессия
// (около цента), результат сохраняется на колонке; дальше та же фраза играет с диска.
func (w *Web) preview(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Voice, Engine string
		TTSModel      string `json:"tts_model"`
	}
	_ = readJSON(r, &req)
	s := w.store.Get()
	gem := req.Engine == "gemini"
	rate := MicRate
	if gem {
		req.Voice, rate = geminiVoice(req.Voice), geminiRate
	}
	if req.Engine == "chatgpt" {
		req.Voice = chatgptVoice(req.Voice)
	}
	if req.Engine == "grok" {
		req.Voice, rate = grokVoice(req.Voice), grokRate
	}
	if req.Engine == "elevenlabs" {
		rate = elevenRate
	}
	if req.Engine == "eleven_realtime" {
		rate = elevenRate
	}
	if w.engine.Talking() {
		bad(rw, "Джарвис сейчас разговаривает")
		return
	}
	vp := ""
	for _, v := range s.LiveVoices {
		if v.ID == req.Voice {
			vp = v.Prompt
		}
	}
	path := w.voiceFile(req.Voice, vp)
	if gem {
		path = w.voiceFile("gemini-"+req.Voice, "24k")
	}
	if req.Engine == "chatgpt" {
		path = w.voiceFile("chatgpt-"+req.Voice, "")
	}
	if req.Engine == "grok" {
		path = w.voiceFile("grok-"+req.Voice, "24k")
	}
	if req.Engine == "elevenlabs" {
		path = w.voiceFile("elevenlabs-"+req.Voice, "24k/"+req.TTSModel)
	}
	if req.Engine == "eleven_realtime" {
		path = w.voiceFile("eleven-realtime-"+req.Voice, "24k")
	}
	if raw, err := os.ReadFile(path); err == nil && len(raw) > MicRate { // хотя бы полсекунды
		pcm := make([]int16, len(raw)/2)
		for i := range pcm {
			pcm[i] = int16(uint16(raw[2*i]) | uint16(raw[2*i+1])<<8)
		}
		boost := 1.0
		if req.Engine == "elevenlabs" {
			boost = (&ClassicSession{}).OutBoost()
		}
		if req.Engine == "chatgpt" { // проба записана до усиления — играем с тем же усилением, что и разговор
			boost = (&ChatGPTSession{}).OutBoost()
		}
		if err := w.playPCM(pcm, rate, boost); err != nil {
			bad(rw, err.Error())
			return
		}
		writeJSON(rw, 200, map[string]any{"ok": true, "cached": true})
		return
	}
	if msg := keyMissing(s, req.Engine); msg != "" {
		bad(rw, msg)
		return
	}
	p := s.ActiveProfile()
	p.Voice, p.WebSearch, p.Engine = req.Voice, false, req.Engine
	if req.Engine == "eleven_realtime" {
		p.TTSVoice = req.Voice
	}
	p.Mode = "live"
	if req.Engine == "elevenlabs" {
		p.Mode, p.TTSVoice = "classic", req.Voice
		p.TTSModel = req.TTSModel
	}
	e := w.engine
	e.mu.Lock()
	if e.busy || e.live != nil {
		e.mu.Unlock()
		bad(rw, "Джарвис сейчас занят")
		return
	}
	e.busy, e.aborted = true, false // пока звучит проба, «Стоп» её обрывает
	e.mu.Unlock()
	rec, err := e.sayAloud(s, p, vp, previewPhrase, 25*time.Second)
	e.mu.Lock()
	e.busy, e.aborted = false, false
	e.mu.Unlock()
	e.leds.Set(LedOff)
	if err != nil {
		bad(rw, shortErr(err))
		return
	}
	saveVoice(path, rec)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// say — написать Джарвису текстом из переписки. Ответ придёт в журнал; speak — прочитать его вслух на колонке.
func (w *Web) say(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Text  string
		Speak bool
	}
	if readJSON(r, &req) != nil || strings.TrimSpace(req.Text) == "" {
		bad(rw, "пустое сообщение")
		return
	}
	if len([]rune(req.Text)) > 600 {
		bad(rw, "слишком длинно — до 600 знаков")
		return
	}
	go w.engine.TextTurn(req.Text, req.Speak)
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// createVoice — свой голос GPT-Live по описанию (OpenAI prompt-to-voice). Создание бесплатное.
func (w *Web) createVoice(rw http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Prompt string }
	if readJSON(r, &req) != nil || strings.TrimSpace(req.Name) == "" || len(strings.TrimSpace(req.Prompt)) < 10 {
		bad(rw, "нужны название и описание голоса (хотя бы пара фраз)")
		return
	}
	s := w.store.Get()
	body, _ := json.Marshal(map[string]any{"type": "prompt", "name": req.Name, "prompt": req.Prompt, "model": "auto"})
	httpReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://api.openai.com/v1/audio/voices", strings.NewReader(string(body)))
	httpReq.Header.Set("Authorization", "Bearer "+s.OpenAIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := w.proxy.ClientFor("openai", 60*time.Second).Do(httpReq)
	if err != nil {
		bad(rw, shortErr(err))
		return
	}
	defer resp.Body.Close()
	var out struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 200 || out.ID == "" {
		msg := fmt.Sprintf("OpenAI ответил %d", resp.StatusCode)
		if out.Error != nil {
			msg += ": " + out.Error.Message
		}
		bad(rw, msg)
		return
	}
	_ = w.store.Update(func(s *Settings) error {
		s.LiveVoices = append(s.LiveVoices, CustomVoice{ID: out.ID, Name: req.Name, Prompt: req.Prompt})
		return nil
	})
	writeJSON(rw, 200, map[string]any{"ok": true, "id": out.ID})
}

func (w *Web) deleteVoice(rw http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	_ = w.store.Update(func(s *Settings) error {
		out := s.LiveVoices[:0]
		for _, v := range s.LiveVoices {
			if v.ID != id {
				out = append(out, v)
			}
		}
		s.LiveVoices = out
		return nil
	})
	if files, _ := filepath.Glob(filepath.Join(w.dataDir, "voices", "*.s16")); len(files) > 0 {
		prefix := filepath.Base(w.voiceFile(id, ""))
		prefix = prefix[:strings.LastIndex(prefix, "-")+1]
		for _, f := range files {
			if strings.HasPrefix(filepath.Base(f), prefix) {
				_ = os.Remove(f)
			}
		}
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

// knownRooms — комнаты из Smart Life и Xiaomi Home (для выбора, где стоит колонка).
func (w *Web) knownRooms() []string {
	seen := map[string]bool{}
	var out []string
	add := func(r string) {
		r = strings.TrimSpace(r)
		if r != "" && !seen[strings.ToLower(r)] {
			seen[strings.ToLower(r)] = true
			out = append(out, r)
		}
	}
	if t := w.engine.tools.SmartLife; t != nil {
		for _, d := range t.Devices() {
			add(d.Room)
		}
	}
	if x := w.engine.tools.Xiaomi; x != nil {
		for _, d := range x.Devices() {
			add(d.Room)
		}
	}
	return out
}
