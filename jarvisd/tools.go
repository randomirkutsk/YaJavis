package main

// Инструменты для мозга Джарвиса — Яндекс Умный дом. Модель получает список устройств с их
// возможностями (вкл/выкл, яркость, цвет, температура света, режимы, пауза, температура кондиционера)
// и сценарии с голосовыми фразами, а сама выбирает id. Простые команды колонка делает без модели (fastcmd.go).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const iotBase = "https://api.iot.yandex.net/v1.0/"

type Tools struct {
	HomeAssistant *HomeAssistant
	Xiaomi        *Xiaomi
	SmartLife     *SmartLife
	Memory        *MemoryStore
	Telegram      *TelegramBot
	Room          func() string                                  // комната колонки из настроек
	Use           func(ch string) bool                           // галочки «через что управлять домом» (настройки колонки)
	Music         func(ctx context.Context, query string) string // включить музыку с YouTube (nil — нельзя)
	// MusicCtl — управлять тем, что играет с YouTube: пауза, продолжить, перемотка, следующее, стоп, что играет
	MusicCtl func(ctx context.Context, command string) string
	// MusicQueue — добавить в очередь YouTube (если ничего не играет — включить)
	MusicQueue func(ctx context.Context, query string) string
	token      func() string
	now        func() time.Time
	client     *http.Client

	mu        sync.Mutex
	catalog   *iotCatalog
	fetchedAt time.Time
}

type iotCap struct {
	Type       string `json:"type"`
	Parameters struct {
		Instance   string `json:"instance"`
		ColorModel string `json:"color_model"`
		Range      *struct {
			Min, Max, Precision float64
		} `json:"range"`
		Modes []struct {
			Value string `json:"value"`
		} `json:"modes"`
		TemperatureK *struct{ Min, Max int } `json:"temperature_k"`
		ColorScene   *struct {
			Scenes []struct {
				ID string `json:"id"`
			} `json:"scenes"`
		} `json:"color_scene"`
	} `json:"parameters"`
	State *struct {
		Instance string `json:"instance"`
		Value    any    `json:"value"`
	} `json:"state"`
}

type capList = []iotCap

type iotDevice struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Room         string  `json:"room"`
	Capabilities capList `json:"capabilities"`
}

type iotCatalog struct {
	Rooms []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rooms"`
	Devices []iotDevice `json:"devices"`
	Groups  []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"groups"`
	Scenarios []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
		Triggers []struct {
			Trigger struct {
				Type  string `json:"type"`
				Value any    `json:"value"`
			} `json:"trigger"`
		} `json:"triggers"`
	} `json:"scenarios"`
}

func NewTools(token func() string, client *http.Client) *Tools {
	return &Tools{token: token, client: client, now: time.Now}
}

func (t *Tools) iot(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ { // Яндекс иногда не принимает новое соединение
		var rd io.Reader
		if payload != nil {
			rd = bytes.NewReader(payload)
		}
		req, e := http.NewRequestWithContext(ctx, method, iotBase+path, rd)
		if e != nil {
			return e
		}
		req.Header.Set("Authorization", "Bearer "+t.token())
		req.Header.Set("Content-Type", "application/json")
		resp, err = t.client.Do(req)
		if err == nil || ctx.Err() != nil || method != http.MethodGet {
			break
		}
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Яндекс ответил %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Catalog — устройства и сценарии (кэш 30 с). Обновляется заранее, на слово «Джарвис».
// hasCatalog — список устройств Яндекса уже загружался (Context загружает его до того, как модели отдают инструменты).
func (t *Tools) hasCatalog() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.catalog != nil && len(t.catalog.Devices)+len(t.catalog.Scenarios) > 0
}

func (t *Tools) Catalog(ctx context.Context) (*iotCatalog, error) {
	if t.token() == "" {
		return nil, fmt.Errorf("умный дом не подключён")
	}
	t.mu.Lock()
	if t.catalog != nil && time.Since(t.fetchedAt) < 30*time.Second {
		c := t.catalog
		t.mu.Unlock()
		return c, nil
	}
	t.mu.Unlock()
	var c iotCatalog
	if err := t.iot(ctx, http.MethodGet, "user/info", nil, &c); err != nil {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.catalog != nil {
			return t.catalog, nil
		}
		return nil, err
	}
	t.mu.Lock()
	t.catalog, t.fetchedAt = &c, time.Now()
	t.mu.Unlock()
	return &c, nil
}

func (t *Tools) Prewarm() {
	if t.token() == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	_, _ = t.Catalog(ctx)
}

func (d *iotDevice) cap(typ, instance string) *iotCap {
	for i := range d.Capabilities {
		c := &d.Capabilities[i]
		if c.Type == "devices.capabilities."+typ && (instance == "" || c.Parameters.Instance == instance) {
			return c
		}
	}
	return nil
}

// describe — возможности устройства коротко для модели.
func (d *iotDevice) describe() string {
	var parts []string
	for _, c := range d.Capabilities {
		p := c.Parameters
		switch strings.TrimPrefix(c.Type, "devices.capabilities.") {
		case "on_off":
			s := "вкл/выкл"
			if c.State != nil {
				if v, ok := c.State.Value.(bool); ok {
					s += map[bool]string{true: " (сейчас вкл)", false: " (сейчас выкл)"}[v]
				}
			}
			parts = append(parts, s)
		case "range":
			if p.Range != nil {
				parts = append(parts, fmt.Sprintf("%s %g..%g", p.Instance, p.Range.Min, p.Range.Max))
			}
		case "color_setting":
			if p.ColorModel != "" {
				parts = append(parts, "цвет")
			}
			if p.TemperatureK != nil && p.TemperatureK.Max > p.TemperatureK.Min {
				parts = append(parts, fmt.Sprintf("белый %d..%dK", p.TemperatureK.Min, p.TemperatureK.Max))
			}
			if p.ColorScene != nil && len(p.ColorScene.Scenes) > 0 {
				var ids []string
				for _, s := range p.ColorScene.Scenes {
					ids = append(ids, s.ID)
				}
				parts = append(parts, "сцены "+strings.Join(ids, ","))
			}
		case "mode":
			var ms []string
			for _, m := range p.Modes {
				ms = append(ms, m.Value)
			}
			parts = append(parts, fmt.Sprintf("режим %s [%s]", p.Instance, strings.Join(ms, ",")))
		case "toggle":
			parts = append(parts, "переключатель "+p.Instance)
		}
	}
	return strings.Join(parts, "; ")
}

func scenarioPhrases(s struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
	Triggers []struct {
		Trigger struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"trigger"`
	} `json:"triggers"`
}) []string {
	var out []string
	for _, tr := range s.Triggers {
		if tr.Trigger.Type == "scenario.trigger.voice" {
			if v, ok := tr.Trigger.Value.(string); ok && v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// Context — текст для инструкций бэкенда: время, устройства с возможностями, сценарии с фразами.
func (t *Tools) Context() string {
	var b strings.Builder
	now := t.now()
	days := []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	b.WriteString("\n\nСейчас " + now.Format("02.01.2006 15:04") + ", " + days[now.Weekday()] + ".")
	if t.HomeAssistant != nil && t.use("homeassistant") && t.HomeAssistant.alive() {
		b.WriteString(t.HomeAssistant.context())
	}
	if t.use("xiaomi") {
		b.WriteString(t.Xiaomi.context())
	}
	if t.use("smartlife") {
		b.WriteString(t.SmartLife.context())
	}
	room := t.room()
	if !t.use("yandex") {
		if room != "" {
			b.WriteString(roomLine(room))
		}
		return b.String()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	c, err := t.Catalog(ctx)
	if err != nil || c == nil {
		if err != nil && time.Since(catalogErrAt) > 10*time.Minute {
			catalogErrAt = time.Now()
			log.Printf("умный дом: список устройств Яндекса не загрузился: %v", err)
		}
		if t.use("alice") {
			b.WriteString("\n\nСписок устройств Яндекса сейчас недоступен: команды умного дома (свет, шторы, розетки, сценарии) передавай Алисе инструментом alice_command.")
		}
		if room != "" {
			b.WriteString(roomLine(room))
		}
		return b.String()
	}
	rooms := map[string]string{}
	for _, r := range c.Rooms {
		rooms[r.ID] = r.Name
	}
	speakerRoom := ""
	b.WriteString("\n\nУмный дом хозяина (в инструмент smart_home передавай id):")
	var lines []string
	for _, d := range c.Devices {
		if strings.HasPrefix(d.Type, "devices.types.smart_speaker") {
			speakerRoom = rooms[d.Room]
			continue
		}
		if strings.HasPrefix(d.Type, "devices.types.sensor") {
			continue
		}
		lines = append(lines, fmt.Sprintf("\n- %s «%s» (%s, %s): %s", d.ID, d.Name, strings.TrimPrefix(d.Type, "devices.types."), orDash(rooms[d.Room]), d.describe()))
	}
	sort.Strings(lines)
	b.WriteString(strings.Join(lines, ""))
	for _, g := range c.Groups {
		fmt.Fprintf(&b, "\n- группа %s «%s»", g.ID, g.Name)
	}
	if len(c.Scenarios) > 0 {
		b.WriteString("\nСценарии (запуск — scenario_id; если хозяин говорит фразу сценария — запусти его):")
		for _, s := range c.Scenarios {
			fmt.Fprintf(&b, "\n- %s «%s»", s.ID, s.Name)
			if ph := scenarioPhrases(s); len(ph) > 0 {
				b.WriteString(" — фразы: " + strings.Join(ph, " / "))
			}
		}
	}
	if room != "" {
		speakerRoom = room
	}
	if speakerRoom == "" {
		speakerRoom = "Зал"
	}
	b.WriteString(roomLine(speakerRoom))
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func (t *Tools) Definitions() []any {
	defs := []any{}
	if t.HomeAssistant.ready() && t.use("homeassistant") && t.HomeAssistant.alive() {
		defs = append(defs, haDefinition())
	}
	if t.Xiaomi.ready() && len(t.Xiaomi.Devices()) > 0 && t.use("xiaomi") {
		defs = append(defs, xiaomiDefinition())
	}
	if t.SmartLife.ready() && len(t.SmartLife.Devices()) > 0 && t.use("smartlife") {
		defs = append(defs, smartLifeDefinition())
	}
	if t.Music != nil {
		who := "Обычную музыку по названию включает Алиса, но если просят включить что-то на колонке тебя («что-нибудь интересное», «чтобы болтало на фоне», «с ютуба») — " +
			"вызывай сразу, запрос придумай сам."
		if !t.use("yandexmusic") {
			who = "Яндекс Музыка выключена в настройках: любую музыку, песню или радио включай только этим инструментом, сразу, запрос придумай сам, если не назвали."
		}
		defs = append(defs, map[string]any{"type": "function", "name": "play_music",
			"description": "Найти по всему YouTube и включить звук видео на колонке: любой ролик, кавер, ремикс, концерт, подкаст или песню. " + who +
				" Не обещай «сейчас поищу» без вызова.",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "что искать: исполнитель и название"},
				"queue": map[string]any{"type": "boolean", "description": "true — не включать сразу, а добавить в очередь после того, что играет («добавь в очередь», «потом включи»)"}},
				"required": []any{"query"}}})
	}
	if t.MusicCtl != nil {
		defs = append(defs, map[string]any{"type": "function", "name": "music_control",
			"description": "Управлять тем, что сейчас играет с YouTube: пауза, продолжить, перемотка вперёд/назад, с начала, следующее видео, похожее (радио), выключить, что играет.",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "команда по-русски, например «пауза», «продолжи», «перемотай вперёд на 2 минуты», «назад на 30 секунд», «сначала», «следующее», «включи похожее», «выключи музыку», «что играет»"}},
				"required": []any{"command"}}})
	}
	if t.use("alice") {
		defs = append(defs, aliceDefinition())
	}
	if t.Memory != nil {
		defs = append(defs, map[string]any{"type": "function", "name": "remember",
			"description": "Сохранить факт или предпочтение о хозяине в долговременную память (имя, предпочтения, привычки, семья, питомцы). Вызывай, когда человек говорит «запомни, что...», «меня зовут...», «я люблю...».",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{
				"fact": map[string]any{"type": "string", "description": "краткий факт о хозяине от третьего лица, например «Хозяина зовут Александр», «Любимый чай — эрл грей»"}},
				"required": []any{"fact"}}})
		defs = append(defs, map[string]any{"type": "function", "name": "forget",
			"description": "Удалить факт из долговременной памяти. Вызывай, когда человек просит забыть («забудь, что...», «удали из памяти»).",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "ключевое слово или тема для удаления, например «чай»"}},
				"required": []any{"query"}}})
	}
	if t.Telegram != nil && t.Telegram.Configured() {
		defs = append(defs, map[string]any{"type": "function", "name": "send_telegram",
			"description": "Отправить сообщение или напоминание хозяину в Telegram. Вызывай, когда просят отправить что-то в телеграм, скинуть в телегу или напомнить в телеграме.",
			"parameters": map[string]any{"type": "object", "properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "текст сообщения для отправки"}},
				"required": []any{"message"}}})
	}
	if t.token() == "" || !t.use("yandex") || !t.hasCatalog() {
		return defs // список устройств Яндекса не загрузился — без него модель зовёт инструмент вслепую; пусть передаёт Алисе
	}
	return append(defs, t.homeDefinition())
}

func (t *Tools) homeDefinition() any {
	return map[string]any{"type": "function", "name": "smart_home",
		"description": "Яндекс Умный дом: включить/выключить, яркость, цвет, температура белого, режимы (пылесос, кондиционер), пауза, температура кондиционера; или запустить сценарий. Указывай только то, что просили.",
		"parameters": map[string]any{"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"device_ids":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "id устройств или групп"},
				"on":            map[string]any{"type": "boolean", "description": "true — включить (пылесос: начать уборку), false — выключить (пылесос: на базу)"},
				"brightness":    map[string]any{"type": "integer", "description": "яркость 1–100"},
				"color":         map[string]any{"type": "string", "description": "цвет #RRGGBB (красный #FF0000 и т. п.)"},
				"temperature_k": map[string]any{"type": "integer", "description": "температура белого света в кельвинах: тёплый 2700, нейтральный 4500, холодный 6500"},
				"modes":         map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "режимы {instance: value}, например {\"cleanup_mode\": \"wet_cleaning\", \"work_speed\": \"turbo\"} или {\"thermostat\": \"cool\"}"},
				"toggles":       map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "boolean"}, "description": "переключатели {instance: true/false}, например {\"pause\": true}"},
				"ranges":        map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "number"}, "description": "числовые параметры {instance: value}, например {\"temperature\": 22}"},
				"scenario_id":   map[string]any{"type": "string", "description": "id сценария, если просили его запустить (тогда остальное не нужно)"},
			}}}
}

type homeArgs struct {
	DeviceIDs    []string           `json:"device_ids"`
	On           *bool              `json:"on"`
	Brightness   int                `json:"brightness"`
	Color        string             `json:"color"`
	TemperatureK int                `json:"temperature_k"`
	Modes        map[string]string  `json:"modes"`
	Toggles      map[string]bool    `json:"toggles"`
	Ranges       map[string]float64 `json:"ranges"`
	ScenarioID   string             `json:"scenario_id"`
}

func hexRGB(s string) (int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 16, 32)
	return int(v), err == nil
}

func rgbToHSV(rgb int) map[string]any {
	r, g, b := float64(rgb>>16&255)/255, float64(rgb>>8&255)/255, float64(rgb&255)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	h := 0.0
	switch d := mx - mn; {
	case d == 0:
	case mx == r:
		h = 60 * math.Mod((g-b)/d, 6)
	case mx == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	s := 0.0
	if mx > 0 {
		s = (mx - mn) / mx
	}
	return map[string]any{"h": int(math.Round(h)), "s": int(math.Round(s * 100)), "v": int(math.Round(mx * 100))}
}

// actionsFor — действия для одного устройства: только то, что оно умеет.
func actionsFor(d *iotDevice, a *homeArgs) []any {
	var acts []any
	add := func(typ, inst string, val any) {
		acts = append(acts, map[string]any{"type": "devices.capabilities." + typ, "state": map[string]any{"instance": inst, "value": val}})
	}
	if a.Color != "" {
		if c := d.cap("color_setting", ""); c != nil && c.Parameters.ColorModel != "" {
			if rgb, ok := hexRGB(a.Color); ok {
				if c.Parameters.ColorModel == "hsv" {
					add("color_setting", "hsv", rgbToHSV(rgb))
				} else {
					add("color_setting", "rgb", rgb)
				}
			}
		}
	}
	if a.TemperatureK > 0 {
		if c := d.cap("color_setting", ""); c != nil && c.Parameters.TemperatureK != nil {
			k := max(c.Parameters.TemperatureK.Min, min(c.Parameters.TemperatureK.Max, a.TemperatureK))
			add("color_setting", "temperature_k", k)
		}
	}
	if a.Brightness > 0 && d.cap("range", "brightness") != nil {
		add("range", "brightness", min(100, a.Brightness))
	}
	for inst, v := range a.Ranges {
		if c := d.cap("range", inst); c != nil {
			if c.Parameters.Range != nil {
				v = math.Max(c.Parameters.Range.Min, math.Min(c.Parameters.Range.Max, v))
			}
			add("range", inst, v)
		}
	}
	for inst, v := range a.Modes {
		if c := d.cap("mode", inst); c != nil {
			for _, m := range c.Parameters.Modes {
				if m.Value == v {
					add("mode", inst, v)
				}
			}
		}
	}
	for inst, v := range a.Toggles {
		if d.cap("toggle", inst) != nil {
			add("toggle", inst, v)
		}
	}
	on := a.On
	if on == nil && (a.Color != "" || a.TemperatureK > 0 || a.Brightness > 0) && len(acts) > 0 {
		t := true // попросили цвет или яркость — значит, свет должен гореть
		on = &t
	}
	if on != nil && d.cap("on_off", "") != nil {
		acts = append([]any{map[string]any{"type": "devices.capabilities.on_off", "state": map[string]any{"instance": "on", "value": *on}}}, acts...)
	}
	return acts
}

func (t *Tools) Call(ctx context.Context, name, args string) string {
	if name == "home_assistant" && t.HomeAssistant.ready() && t.use("homeassistant") {
		return t.HomeAssistant.control(ctx, args)
	}
	if name == "xiaomi_home" && t.Xiaomi.ready() {
		return t.Xiaomi.control(ctx, args)
	}
	if name == "smart_life" && t.SmartLife.ready() {
		return t.SmartLife.control(ctx, args)
	}
	if name == "remember" && t.Memory != nil {
		var a struct{ Fact string `json:"fact"` }
		if json.Unmarshal([]byte(args), &a) == nil && strings.TrimSpace(a.Fact) != "" {
			it, err := t.Memory.Add(a.Fact)
			if err != nil {
				return "Не удалось запомнить: " + err.Error()
			}
			return fmt.Sprintf("Запомнил: %s", it.Fact)
		}
		return "Не понял, что именно запомнить."
	}
	if name == "forget" && t.Memory != nil {
		var a struct{ Query string `json:"query"` }
		if json.Unmarshal([]byte(args), &a) == nil && strings.TrimSpace(a.Query) != "" {
			n := t.Memory.DeleteByQuery(a.Query)
			if n > 0 {
				return fmt.Sprintf("Удалил из памяти (записей: %d).", n)
			}
			return "В памяти не нашлось подходящих записей."
		}
		return "Не понял, что именно забыть."
	}
	if name == "send_telegram" && t.Telegram != nil {
		var a struct{ Message string `json:"message"` }
		if json.Unmarshal([]byte(args), &a) == nil && strings.TrimSpace(a.Message) != "" {
			err := t.Telegram.SendToOwner(a.Message)
			if err != nil {
				return "Не удалось отправить в Telegram: " + err.Error()
			}
			return "Сообщение успешно отправлено в Telegram."
		}
		return "Не указан текст сообщения."
	}
	if name == "play_music" && t.Music != nil {
		var a struct {
			Query string
			Queue bool
		}
		if json.Unmarshal([]byte(args), &a) != nil || strings.TrimSpace(a.Query) == "" {
			return "Не понял, что включить."
		}
		if a.Queue && !recommendationRequest(a.Query) && t.MusicQueue != nil {
			return t.MusicQueue(ctx, a.Query)
		}
		return t.Music(ctx, a.Query)
	}
	if name == "music_control" && t.MusicCtl != nil {
		var a struct{ Command string }
		if json.Unmarshal([]byte(args), &a) != nil || strings.TrimSpace(a.Command) == "" {
			return "Не понял команду для музыки."
		}
		return t.MusicCtl(ctx, a.Command)
	}
	if name == "alice_command" && t.use("alice") {
		var a struct{ Command string }
		if json.Unmarshal([]byte(args), &a) != nil || strings.TrimSpace(a.Command) == "" {
			return "Не понял команду для Алисы."
		}
		cmd := strings.TrimSpace(a.Command)
		if err := AliceText(cmd, true); err != nil { // беззвучно: результат озвучит сам Джарвис
			return "Алиса недоступна: " + err.Error()
		}
		log.Printf("инструмент: «%s» → Алисе", cmd)
		return "Передал Алисе: «" + cmd + "». Она выполняет — скажи коротко, что сделано."
	}
	if name != "smart_home" {
		return "Нет такого инструмента."
	}
	var a homeArgs
	if err := json.Unmarshal([]byte(args), &a); err != nil {
		return "Не понял команду."
	}
	return t.Apply(ctx, &a)
}

var catalogErrAt time.Time // когда последний раз писали в журнал, что список устройств не загрузился

func aliceDefinition() any {
	return map[string]any{"type": "function", "name": "alice_command",
		"description": "Передать команду штатной Алисе на этой колонке — она выполнит всё, что подключено к Яндексу: свет, шторы, розетки, " +
			"сценарии, таймеры, будильники, напоминания. Пиши команду коротко, как её сказали бы Алисе: «закрой шторы», «поставь таймер на 10 минут».",
		"parameters": map[string]any{"type": "object", "properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "команда для Алисы, без её имени"}},
			"required": []any{"command"}}}
}

func (t *Tools) Apply(ctx context.Context, a *homeArgs) string {
	if a.ScenarioID != "" {
		var r struct{ Status string }
		if err := t.iot(ctx, http.MethodPost, "scenarios/"+a.ScenarioID+"/actions", nil, &r); err != nil || r.Status != "ok" {
			return "Сценарий не запустился."
		}
		return "Готово."
	}
	if len(a.DeviceIDs) == 0 {
		if t.use("alice") {
			return "Не указано, какие устройства. Если нужного устройства нет в списке — передай команду Алисе инструментом alice_command."
		}
		return "Не указано, какие устройства."
	}
	c, err := t.Catalog(ctx)
	if err != nil {
		return "Не удалось связаться с Яндекс Умным домом."
	}
	byID := map[string]*iotDevice{}
	for i := range c.Devices {
		byID[c.Devices[i].ID] = &c.Devices[i]
	}
	groups := map[string]bool{}
	for _, g := range c.Groups {
		groups[g.ID] = true
	}
	var devices []any
	failed, total, unsupported := 0, 0, 0
	for _, id := range a.DeviceIDs {
		if groups[id] {
			total++
			var acts []any
			if a.On != nil {
				acts = append(acts, map[string]any{"type": "devices.capabilities.on_off", "state": map[string]any{"instance": "on", "value": *a.On}})
			}
			var r struct{ Status string }
			if len(acts) == 0 || t.iot(ctx, http.MethodPost, "groups/"+id+"/actions", map[string]any{"actions": acts}, &r) != nil || r.Status != "ok" {
				failed++
			}
			continue
		}
		d := byID[id]
		if d == nil {
			continue
		}
		acts := actionsFor(d, a)
		if len(acts) == 0 {
			unsupported++
			continue
		}
		total++
		devices = append(devices, map[string]any{"id": id, "actions": acts})
	}
	if len(devices) > 0 {
		var r struct {
			Devices []struct {
				Capabilities []struct {
					State struct {
						ActionResult struct{ Status string } `json:"action_result"`
					} `json:"state"`
				} `json:"capabilities"`
			} `json:"devices"`
		}
		if err := t.iot(ctx, http.MethodPost, "devices/actions", map[string]any{"devices": devices}, &r); err != nil {
			return "Не удалось связаться с Яндекс Умным домом."
		}
		for _, d := range r.Devices {
			for _, cp := range d.Capabilities {
				if cp.State.ActionResult.Status != "DONE" {
					failed++
					break
				}
			}
		}
	}
	t.mu.Lock()
	t.catalog = nil // состояние изменилось
	t.mu.Unlock()
	switch {
	case total == 0 && unsupported > 0:
		return "Эти устройства так не умеют."
	case total == 0:
		return "Не нашёл такие устройства."
	case failed == 0:
		return "Готово."
	case failed < total:
		return fmt.Sprintf("Готово, но %d из %d не ответили.", failed, total)
	default:
		return "Яндекс не подтвердил выполнение."
	}
}

func (t *Tools) use(ch string) bool { return t.Use == nil || t.Use(ch) }

func (t *Tools) room() string {
	if t.Room == nil {
		return ""
	}
	return strings.TrimSpace(t.Room())
}

func roomLine(room string) string {
	return "\nКолонка стоит в комнате «" + room + "». Если хозяин не назвал комнату («выключи свет», «включи кондиционер») — " +
		"имеется в виду эта комната; «весь свет» — все лампы и выключатели света во всём доме."
}
