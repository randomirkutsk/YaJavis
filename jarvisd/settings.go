package main

// Настройки — один JSON на колонке (/data/opt/jarvisd/settings.json, права 0600).
// Пишем атомарно: во временный файл, потом rename — выключение питания не оставит полфайла.

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
	_ "time/tzdata" // на колонке нет базы часовых поясов
)

type Profile struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Color        int     `json:"color"`
	Mode         string  `json:"mode"`          // live | classic
	Engine       string  `json:"engine"`        // голосовой движок: openai (GPT-Live) | chatgpt (GPT-Live по подписке) | gemini (Gemini Live) | grok (Grok Voice)
	LiveModel    string  `json:"live_model"`    // модель Gemini Live (для GPT-Live она одна)
	Voice        string  `json:"voice"`         // голос выбранного движка
	BackendModel string  `json:"backend_model"` // модель, которая думает и вызывает инструменты
	Effort       string  `json:"effort"`        // none | low | medium | high
	Priority     bool    `json:"priority"`      // service_tier=priority — быстрее, дороже
	WebSearch    bool    `json:"web_search"`
	IdleS        float64 `json:"idle_s"` // сколько секунд тишины до конца разговора
	Prompt       string  `json:"prompt"` // id характера
	// классика: распознавание → LLM → синтез
	LLMModel  string `json:"llm_model"`
	TTSVoice  string `json:"tts_voice"` // голос ElevenLabs
	TTSModel  string `json:"tts_model"`
	STTEngine string `json:"stt_engine"` // elevenlabs | openai
	STTModel  string `json:"stt_model,omitempty"`

	// у каждого ассистента свои: громкость голоса (0..100), город и часовой пояс хозяина
	Volume   int          `json:"volume"`
	City     string       `json:"city"`
	Timezone string       `json:"timezone"`
	Led      *LedSettings `json:"led,omitempty"` // цвета кольца этого ассистента (общий выключатель — в Settings.Led)
}

type Prompt struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

type CustomVoice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type Settings struct {
	HomeAssistant HASettings `json:"home_assistant"`
	PasswordHash  string     `json:"password_hash"`
	SessionKey    string     `json:"session_key"`

	OpenAIKey     string      `json:"openai_key"`
	GeminiKey     string      `json:"gemini_key"`
	GeminiKeys    []string    `json:"gemini_keys,omitempty"` // все ключи Google; GeminiKey — первый. Кончились лимиты у одного — берётся следующий
	GrokKey       string      `json:"grok_key"`              // xAI: голосовой движок Grok Voice
	ChatGPT       ChatGPTAuth `json:"chatgpt"`               // вход в подписку ChatGPT (голос без оплаты API)
	Xiaomi        XiaomiAuth  `json:"xiaomi"`                // вход в Xiaomi Home (Mi Home) по QR-коду
	Tuya          TuyaAuth    `json:"tuya"`                  // вход в Smart Life (Tuya) по QR-коду
	ElevenLabsKey string      `json:"elevenlabs_key"`
	YandexToken   string      `json:"yandex_token"`

	ElevenKeys []ElevenKey `json:"eleven_keys,omitempty"` // все ключи ElevenLabs; ElevenLabsKey — тот, что сейчас с квотой

	Vless       string          `json:"vless"`
	ProxyYandex bool            `json:"proxy_yandex"`  // обычно Яндекс — напрямую (старая настройка, теперь VPN["yandex"])
	VPN         map[string]bool `json:"vpn,omitempty"` // галочка «через VPN» у подключения (netroute.go)

	Profiles   []Profile     `json:"profiles"`
	Active     string        `json:"active"`
	Prompts    []Prompt      `json:"prompts"`
	LiveVoices []CustomVoice `json:"live_voices"`

	WakeEnabled     bool    `json:"wake_enabled"`
	WakeCheck       bool    `json:"wake_check"`       // перед ответом проверять имя расшифровкой (медленнее; без неё — сразу, как Алиса)
	WakeSensitivity int     `json:"wake_sensitivity"` // 1..10
	MicBoostDB      int     `json:"mic_boost_db"`     // доп. усиление микрофона для ИИ, 0..18 дБ (на Лайт 2 голос после обработки тише)
	MicGainDB       int     `json:"mic_gain_db"`      // общее усиление микрофона для всего (слово, ИИ, обучение), 0..30 дБ: на Мини 2 обработка Алисы отдаёт голос тихо
	FollowUpS       float64 `json:"follow_up_s"`
	City            string  `json:"city"`
	Timezone        string  `json:"timezone"`     // например Europe/Moscow: часы колонки в UTC
	PlaceMoved      bool    `json:"place_moved"`  // город и пояс перенесены из ассистентов в настройки колонки
	SpeakerRoom     string  `json:"speaker_room"` // где стоит колонка: «свет», «кондиционер» без комнаты — здесь
	Volume          int     `json:"volume"`       // громкость ответов Джарвиса, 0..100 (с шкалой 2: 50 — прежние 100)
	VolScale        int     `json:"vol_scale"`    // 2 — громкость голоса в новой шкале: 100% вдвое громче прежнего максимума
	LANOnly         bool    `json:"lan_only"`
	HomeVia         string  `json:"home_via"` // кто выполняет короткие команды умного дома: alice | jarvis (старое, теперь HomeUse)
	// через что Джарвис управляет домом: alice, yandex, smartlife, xiaomi, homeassistant (нет записи — можно)
	HomeUse map[string]bool `json:"home_use,omitempty"`

	Led LedSettings `json:"led"`

	DevMode bool `json:"dev_mode"` // режим разработчика: страница «Обучение» и запись фона

	Language string `json:"language"` // на каком языке говорит хозяин: ru | en | uk | "" — определять самому

	RadioOff bool          `json:"radio_off"` // после очереди не играть похожее, а замолчать
	Night    NightSettings `json:"night"`

	// длинная запись фона: файлы уходят HTTP PUT (WebDAV, kws/bg_receiver.py)
	BgURL  string `json:"bg_url"`
	BgUser string `json:"bg_user"`
	BgPass string `json:"bg_pass"`
	BgTo   string `json:"bg_to"` // gdrive | server

	// Google Диск для фона: свой OAuth-клиент Google (тип «TVs and Limited Input devices»)
	GDriveClient  string `json:"gdrive_client"`
	GDriveSecret  string `json:"gdrive_secret"`
	GDriveRefresh string `json:"gdrive_refresh"`
	GDriveFolder  string `json:"gdrive_folder"`

	TrainMode    bool `json:"train_mode"`     // режим обучения: копить оклики, «почти оклики» и фон для дообучения слова
	TrainLimitMB int  `json:"train_limit_mb"` // сколько места на колонке можно занять записями

	YouTubeRefresh string `json:"youtube_refresh"` // вход в YouTube кодом, как на телевизоре (refresh token)
	YouTubeName    string `json:"youtube_name"`

	UpdateRepo  string `json:"update_repo"`  // GitHub owner/name, откуда брать обновления
	UpdateCheck bool   `json:"update_check"` // проверять обновления раз в 6 часов
	UpdateAuto  bool   `json:"update_auto"`  // ставить найденное обновление само (ночью, когда никто не говорит)
	NoStats     bool   `json:"no_stats"`     // не отмечаться в счётчике установок на GitHub
	AutoOnce    int    `json:"auto_once"`    // 1 — автообновление уже включено по умолчанию (дальше решает человек)
}

// LedSettings — цвета подсветки (#rrggbb) и яркость 0..100.
type LedSettings struct {
	Enabled    bool   `json:"enabled"`
	Brightness int    `json:"brightness"`
	Listening  string `json:"listening"`
	Thinking   string `json:"thinking"`
	Speaking   string `json:"speaking"`
}

// NightSettings — ночной режим: по расписанию голос тише, кольцо тусклее, музыка не громче, ответы короче.
type NightSettings struct {
	Enabled bool   `json:"enabled"`
	From    string `json:"from"`    // "23:00"
	To      string `json:"to"`      // "08:00"
	Voice   int    `json:"voice"`   // голос Джарвиса не громче, 0..100
	Bright  int    `json:"bright"`  // яркость кольца не больше, 0..100
	Speaker int    `json:"speaker"` // громкость колонки не больше, 0..10
}

var defaultNight = NightSettings{From: "23:00", To: "08:00", Voice: 20, Bright: 10, Speaker: 3}

// fill — пустые поля ночного режима значениями по умолчанию.
func (n *NightSettings) fill() {
	if _, ok := hhmm(n.From); !ok {
		n.From = defaultNight.From
	}
	if _, ok := hhmm(n.To); !ok {
		n.To = defaultNight.To
	}
	n.Voice = max(5, min(100, n.Voice))
	n.Bright = max(0, min(100, n.Bright))
	n.Speaker = max(1, min(10, n.Speaker))
}

func hhmm(v string) (int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(v, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// IsNight — сейчас ночь по расписанию (интервал может переходить через полночь).
func (s *Settings) IsNight() bool {
	if !s.Night.Enabled {
		return false
	}
	from, ok1 := hhmm(s.Night.From)
	to, ok2 := hhmm(s.Night.To)
	if !ok1 || !ok2 || from == to {
		return false
	}
	t := s.Now()
	now := t.Hour()*60 + t.Minute()
	if from < to {
		return now >= from && now < to
	}
	return now >= from || now < to
}

var defaultLed = LedSettings{Enabled: true, Brightness: 45, Listening: "#ffffff", Thinking: "#00b4ff", Speaking: "#5a2dff"}

const defaultPrompt = `Ты — Джарвис (J.A.R.V.I.S.), личный голосовой ассистент и дворецкий в умной колонке. Говоришь по-русски, лаконично, точно и с лёгкой иронией. Обращаешься к хозяину на «вы» или «сэр». Отвечай устно и по делу, без спецсимволов и markdown.`
const broPrompt = `Ты — Джарвис, но общаешься с хозяином как близкий кореш: «братишка», «бро», на «ты», с юмором и подколами, можешь крепко выразиться к месту. Говоришь коротко, живо, по-пацански, но по делу и честно. Если чего-то не знаешь — так и говоришь.`

func defaultSettings() *Settings {
	return &Settings{
		SessionKey: randHex(32),
		Profiles: []Profile{{
			ID: "p1", Name: "Джарвис", Color: 1, Mode: "live", Engine: "gemini",
			LiveModel: "gemini-3.8-live", Voice: "Charon", BackendModel: "gemini-3.8-flash",
			Effort: "low", Priority: true, WebSearch: true, IdleS: 8, Prompt: "jarvis",
			LLMModel: "gemini-3.8-flash", TTSModel: "eleven_flash_v2_5", STTEngine: "elevenlabs",
		}},
		Active:          "p1",
		Prompts:         []Prompt{{ID: "jarvis", Name: "Джарвис", Text: defaultPrompt}, {ID: "bro", Name: "Братишка", Text: broPrompt}},
		WakeEnabled:     true,
		WakeSensitivity: 5,
		FollowUpS:       6,
		Volume:          40,
		Timezone:        "UTC",
		HomeVia:         "alice",
		LANOnly:         true,
		Led:             defaultLed,
		TrainLimitMB:    5,
		BgTo:            "gdrive",
		Language:        "ru",
		UpdateCheck:     true,
		UpdateAuto:      true,
		AutoOnce:        1,
		Night:           defaultNight,
	}
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type Store struct {
	mu   sync.RWMutex
	path string
	s    *Settings
}

func OpenStore(path string) (*Store, error) {
	st := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		st.s = defaultSettings()
		st.s.VolScale = 2 // новая установка — сразу в новой шкале громкости
		if err := st.save(); err != nil {
			return nil, err
		}
		return st, st.applySeed()
	}
	if err != nil {
		return nil, err
	}
	s := defaultSettings()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.SessionKey == "" {
		s.SessionKey = randHex(32)
	}
	dirty := false
	if !s.PlaceMoved { // город и пояс снова общие — у колонки; берём те, что были у активного ассистента
		for _, p := range s.Profiles {
			if p.ID == s.Active {
				if p.Timezone != "" {
					s.Timezone = p.Timezone
				}
				if p.City != "" {
					s.City = p.City
				}
			}
		}
		s.PlaceMoved, dirty = true, true
	}
	if len(s.ElevenKeys) == 0 && s.ElevenLabsKey != "" { // был один ключ — он первый в списке
		s.ElevenKeys, dirty = []ElevenKey{{Key: s.ElevenLabsKey}}, true
	}
	s.pickEleven()
	if len(s.GeminiKeys) == 0 && s.GeminiKey != "" { // был один ключ — он первый в списке
		s.GeminiKeys = []string{s.GeminiKey}
	}
	for i := range s.Profiles { // громкость раздаётся ассистентам; город и пояс — копии общих
		s.Profiles[i].fillDefaults(s)
	}
	st.s = s
	if s.VolScale < 2 { // новая шкала громкости голоса: прежние 100% стали 50%, звучит так же, а выше — запас ×2
		s.Volume = max(5, s.Volume/2)
		for i := range s.Profiles {
			s.Profiles[i].Volume = max(5, s.Profiles[i].Volume/2)
		}
		s.Night.Voice = max(5, s.Night.Voice/2)
		s.VolScale, dirty = 2, true
	}
	if s.AutoOnce < 1 || !bytes.Contains(data, []byte(`"auto_once"`)) { // автообновление теперь по умолчанию: один раз включаем и тем, кто ставил раньше
		s.UpdateCheck, s.UpdateAuto, s.AutoOnce, dirty = true, true, 1, true
	}
	if dirty {
		if err := st.save(); err != nil {
			return nil, err
		}
	}
	return st, st.applySeed()
}

// applySeed — начальные настройки от программы установки (seed.json рядом с settings.json).
// Применяются один раз и файл удаляется: пароль (уже хэш), ключ OpenAI, ссылка VLESS.
func (st *Store) applySeed() error {
	path := filepath.Join(filepath.Dir(st.path), "seed.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	defer os.Remove(path)
	var seed struct {
		PasswordHash string   `json:"password_hash"`
		OpenAIKey    string   `json:"openai_key"`
		Vless        string   `json:"vless"`
		Timezone     string   `json:"timezone"`
		YandexToken  string   `json:"yandex_token"`
		ElevenLabs   string   `json:"elevenlabs_key"`
		City         string   `json:"city"`
		Prompts      []Prompt `json:"prompts"`       // перенос характеров (по имени)
		ActivePrompt string   `json:"active_prompt"` // имя характера для активного ассистента
		Profile      *Profile `json:"profile"`       // перенос активного ассистента
	}
	if json.Unmarshal(data, &seed) != nil {
		return nil
	}
	return st.Update(func(s *Settings) error {
		if seed.PasswordHash != "" {
			s.PasswordHash = seed.PasswordHash
			s.SessionKey = randHex(32)
		}
		if seed.OpenAIKey != "" {
			s.OpenAIKey = seed.OpenAIKey
		}
		if seed.Vless != "" {
			s.Vless = seed.Vless
		}
		if seed.Timezone != "" {
			s.Timezone = seed.Timezone
		}
		if seed.YandexToken != "" {
			s.YandexToken = seed.YandexToken
		}
		if seed.ElevenLabs != "" {
			s.addElevenKey(seed.ElevenLabs)
		}
		if seed.City != "" {
			s.City = seed.City
		}
		for _, np := range seed.Prompts {
			found := false
			for i := range s.Prompts {
				if s.Prompts[i].Name == np.Name {
					s.Prompts[i].Text, found = np.Text, true
				}
			}
			if !found && np.Name != "" && np.Text != "" {
				s.Prompts = append(s.Prompts, Prompt{ID: "c" + randHex(4), Name: np.Name, Text: np.Text})
			}
		}
		if seed.Profile != nil {
			p := *seed.Profile
			for i := range s.Profiles {
				if s.Profiles[i].ID == s.Active {
					p.ID, p.Color = s.Profiles[i].ID, max(1, p.Color)
					if p.Prompt == "" {
						p.Prompt = s.Profiles[i].Prompt
					}
					s.Profiles[i] = p
				}
			}
		}
		if seed.ActivePrompt != "" {
			for _, c := range s.Prompts {
				if c.Name == seed.ActivePrompt {
					for i := range s.Profiles {
						if s.Profiles[i].ID == s.Active {
							s.Profiles[i].Prompt = c.ID
						}
					}
				}
			}
		}
		return nil
	})
}

// Get — копия настроек (чтение без гонок).
func (st *Store) Get() Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	c := *st.s
	c.Profiles = append([]Profile(nil), st.s.Profiles...)
	for i := range c.Profiles {
		if led := st.s.Profiles[i].Led; led != nil {
			value := *led
			c.Profiles[i].Led = &value
		}
	}
	c.Prompts = append([]Prompt(nil), st.s.Prompts...)
	c.LiveVoices = append([]CustomVoice(nil), st.s.LiveVoices...)
	// карты и срезы — тоже свои: панель меняет их под замком, а копию читают без него
	c.VPN, c.HomeUse = maps.Clone(st.s.VPN), maps.Clone(st.s.HomeUse)
	c.GeminiKeys, c.ElevenKeys = slices.Clone(st.s.GeminiKeys), slices.Clone(st.s.ElevenKeys)
	c.HomeAssistant.Rooms = slices.Clone(st.s.HomeAssistant.Rooms)
	for i := range c.HomeAssistant.Rooms {
		c.HomeAssistant.Rooms[i].Aliases = slices.Clone(st.s.HomeAssistant.Rooms[i].Aliases)
	}
	return c
}

// WakeFlags — то, что нужно 100 раз в секунду (каждый кадр микрофона): без копии всех настроек,
// иначе Get со списками профилей и голосов мусорил сотнями килобайт в секунду.
type WakeFlags struct {
	WakeEnabled     bool
	WakeSensitivity int
	TrainMode       bool
	MicBoostDB      int
	MicGainDB       int
}

func (st *Store) Flags() WakeFlags {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return WakeFlags{st.s.WakeEnabled, st.s.WakeSensitivity, st.s.TrainMode, st.s.MicBoostDB, st.s.MicGainDB}
}

// Update — изменить и сохранить.
func (st *Store) Update(fn func(s *Settings) error) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	backup, _ := json.Marshal(st.s)
	if err := fn(st.s); err != nil {
		_ = json.Unmarshal(backup, st.s)
		return err
	}
	return st.save()
}

func (st *Store) save() error {
	data, err := json.MarshalIndent(st.s, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(st.path), 0o700); err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}

// Now — время в часовом поясе хозяина (настройки колонки).
func (s *Settings) Now() time.Time {
	tz := s.Timezone
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return time.Now().In(loc)
	}
	return time.Now()
}

// fillDefaults — пустые громкость, город и пояс ассистента берутся из общих настроек.
func (p *Profile) fillDefaults(s *Settings) {
	if isClassic(*p) {
		classicDefaults(p)
	}
	if p.Volume <= 0 {
		p.Volume = s.Volume
		if p.Volume <= 0 {
			p.Volume = 40
		}
	}
	p.City, p.Timezone = s.City, s.Timezone // город и пояс — общие, в настройках колонки
	if p.Led == nil {
		l := s.Led
		p.Led = &l
	}
}

// LedFor — подсветка активного ассистента с общим выключателем.
func (s *Settings) LedFor() LedSettings {
	l := s.Led
	if p := s.ActiveProfile(); p.Led != nil {
		l.Brightness, l.Listening, l.Thinking, l.Speaking = p.Led.Brightness, p.Led.Listening, p.Led.Thinking, p.Led.Speaking
	}
	if s.IsNight() {
		l.Brightness = min(l.Brightness, s.Night.Bright)
	}
	return l
}

// VoiceVolume — громкость голоса активного ассистента.
func (s *Settings) VoiceVolume() int {
	v := s.Volume
	if pv := s.ActiveProfile().Volume; pv > 0 {
		v = pv
	}
	if s.IsNight() {
		v = min(v, s.Night.Voice)
	}
	return v
}

func (s *Settings) ActiveProfile() Profile {
	for _, p := range s.Profiles {
		if p.ID == s.Active {
			return p
		}
	}
	if len(s.Profiles) > 0 {
		return s.Profiles[0]
	}
	return defaultSettings().Profiles[0]
}

func (s *Settings) PromptText(id string) string {
	for _, p := range s.Prompts {
		if p.ID == id {
			return p.Text
		}
	}
	return defaultPrompt
}

// HomeChannels — через что можно управлять домом.
var HomeChannels = []string{"alice", "yandexmusic", "yandex", "smartlife", "xiaomi", "homeassistant"}

// HomeConnected — канал подключён (ключ или вход есть). Без подключения галочка недоступна и выключена.
func (s Settings) HomeConnected(ch string) bool {
	switch ch {
	case "yandex":
		return s.YandexToken != ""
	case "smartlife":
		return s.Tuya.SignedIn()
	case "xiaomi":
		return s.Xiaomi.SignedIn()
	case "homeassistant":
		return s.HomeAssistant.URL != "" && s.HomeAssistant.Token != ""
	}
	return true // Алиса и Яндекс Музыка — всегда есть
}

// homeJustConnected — канал только что подключили: галочка включается сама (прежнее «выключено» забываем).
func (s *Settings) homeJustConnected(ch string, was bool) {
	if !was && s.HomeConnected(ch) {
		delete(s.HomeUse, ch)
	}
}

// UseHome — можно ли управлять домом через этот канал.
func (s Settings) UseHome(ch string) bool {
	if !s.HomeConnected(ch) {
		return false
	}
	if v, ok := s.HomeUse[ch]; ok {
		return v
	}
	if ch == "alice" {
		return s.HomeVia != "jarvis"
	}
	return true
}

func (s Settings) HomeUseState() map[string]bool {
	out := map[string]bool{}
	for _, ch := range HomeChannels {
		out[ch] = s.UseHome(ch)
	}
	return out
}
