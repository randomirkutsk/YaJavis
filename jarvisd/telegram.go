package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TelegramBot struct {
	store       *Store
	proxy       *Proxy
	engine      *Engine
	memory      *MemoryStore
	mu          sync.Mutex
	running     bool
	cancel      context.CancelFunc
	botUsername string
	lastErr     string
}

func NewTelegramBot(store *Store, proxy *Proxy, engine *Engine, memory *MemoryStore) *TelegramBot {
	return &TelegramBot{
		store:  store,
		proxy:  proxy,
		engine: engine,
		memory: memory,
	}
}

func (b *TelegramBot) Configured() bool {
	s := b.store.Get()
	return s.TelegramEnabled && strings.TrimSpace(s.TelegramBotToken) != "" && strings.TrimSpace(s.TelegramChatID) != ""
}

func (b *TelegramBot) Running() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

func (b *TelegramBot) Status() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.store.Get()
	if strings.TrimSpace(s.TelegramBotToken) == "" {
		return "не настроен"
	}
	if !s.TelegramEnabled {
		return "выключен"
	}
	if b.lastErr != "" {
		return "ошибка: " + b.lastErr
	}
	if b.running {
		if b.botUsername != "" {
			return "работает @" + b.botUsername
		}
		return "работает"
	}
	return "остановлен"
}

func (b *TelegramBot) BotName() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.botUsername
}

func (b *TelegramBot) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.store.Get()
	token := strings.TrimSpace(s.TelegramBotToken)
	if !s.TelegramEnabled || token == "" {
		return
	}
	if b.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.running = true
	b.lastErr = ""
	go b.poll(ctx, token)
}

func (b *TelegramBot) Stop() {
	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	b.running = false
	b.mu.Unlock()
}

func (b *TelegramBot) Restart() {
	b.Stop()
	b.Start()
}

func (b *TelegramBot) SendToOwner(text string) error {
	s := b.store.Get()
	token := strings.TrimSpace(s.TelegramBotToken)
	chatID := strings.TrimSpace(s.TelegramChatID)
	if token == "" || chatID == "" {
		return errors.New("Telegram не настроен (нужны токен и chat ID)")
	}
	return b.SendMessage(token, chatID, text)
}

func (b *TelegramBot) SendMessage(token, chatID, text string) error {
	client := b.proxy.ClientFor("telegram", 20*time.Second)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка запроса к Telegram: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	var res struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("ответ Telegram: %s", string(body))
	}
	if !res.OK {
		return errors.New(res.Description)
	}
	return nil
}

func (b *TelegramBot) Test(token, chatID string) error {
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	if token == "" || chatID == "" {
		return errors.New("укажите токен бота и chat ID")
	}
	msg := "👋 Привет! Это Джарвис с вашей Яндекс Станции.\nСвязь успешно проверена и работает!"
	return b.SendMessage(token, chatID, msg)
}

type tgUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      *struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat *struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

func (b *TelegramBot) poll(ctx context.Context, token string) {
	log.Printf("Telegram: бот запускается...")
	client := b.proxy.ClientFor("telegram", 35*time.Second)

	// Получаем имя бота через getMe
	{
		meURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
		if resp, err := client.Do(req); err == nil {
			var meRes struct {
				OK     bool `json:"ok"`
				Result struct {
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
				} `json:"result"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&meRes)
			resp.Body.Close()
			if meRes.OK {
				b.mu.Lock()
				b.botUsername = meRes.Result.Username
				b.mu.Unlock()
				log.Printf("Telegram: подключён бот @%s (%s)", meRes.Result.Username, meRes.Result.FirstName)
			}
		}
	}

	offset := 0
	failCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("Telegram: бот остановлен")
			return
		default:
		}

		pollURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=25", token, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			failCount++
			b.mu.Lock()
			b.lastErr = shortErr(err)
			b.mu.Unlock()
			sleep := time.Duration(min(failCount*2, 30)) * time.Second
			time.Sleep(sleep)
			continue
		}

		var updateRes struct {
			OK          bool       `json:"ok"`
			Result      []tgUpdate `json:"result"`
			Description string     `json:"description"`
		}
		decErr := json.NewDecoder(resp.Body).Decode(&updateRes)
		resp.Body.Close()

		if decErr != nil || !updateRes.OK {
			failCount++
			b.mu.Lock()
			if updateRes.Description != "" {
				b.lastErr = updateRes.Description
			} else if decErr != nil {
				b.lastErr = shortErr(decErr)
			}
			b.mu.Unlock()
			time.Sleep(3 * time.Second)
			continue
		}

		failCount = 0
		b.mu.Lock()
		b.lastErr = ""
		b.mu.Unlock()

		for _, up := range updateRes.Result {
			if up.UpdateID >= offset {
				offset = up.UpdateID + 1
			}
			if up.Message == nil || up.Message.From == nil || up.Message.Chat == nil {
				continue
			}
			go b.handleMessage(token, up.Message.Chat.ID, up.Message.From.ID, strings.TrimSpace(up.Message.Text))
		}
	}
}

func (b *TelegramBot) handleMessage(token string, chatID, senderID int64, text string) {
	if text == "" {
		return
	}
	s := b.store.Get()
	targetChatID := strings.TrimSpace(s.TelegramChatID)
	strSender := strconv.FormatInt(senderID, 10)
	strChat := strconv.FormatInt(chatID, 10)

	// Безопасность: доступ только для указанного Chat ID
	if targetChatID != "" && strSender != targetChatID && strChat != targetChatID {
		warn := fmt.Sprintf("⚠️ Доступ ограничен.\nЭтот бот привязан к Яндекс Станции.\n\nВаш Chat ID: <code>%s</code>\nУкажите его в веб-интерфейсе колонки (Подключения → Telegram).", strChat)
		_ = b.SendMessage(token, strChat, warn)
		return
	}

	reply := func(msg string) {
		_ = b.SendMessage(token, strChat, msg)
	}

	// 1. Команды /start и /help
	if text == "/start" || text == "/help" {
		help := "👋 <b>Привет! Я Джарвис</b> с вашей Яндекс Станции.\n\n" +
			"Команды:\n" +
			"• <code>/say &lt;текст&gt;</code> — сказать фразу вслух на колонке\n" +
			"• <code>/vol &lt;0-10&gt;</code> — изменить громкость колонки\n" +
			"• <code>/status</code> — состояние колонки и сервисов\n" +
			"• <code>/remember &lt;факт&gt;</code> — сохранить в память\n" +
			"• <code>/memory</code> — список воспоминаний\n" +
			"• <code>/forget &lt;слово&gt;</code> — удалить из памяти\n\n" +
			"💡 Вы также можете просто писать мне любые вопросы — я отвечу с учётом вашей личной памяти и настроек!"
		reply(help)
		return
	}

	// 2. Команда /status
	if text == "/status" {
		vol := speakerVolume()
		memCount := 0
		if b.memory != nil {
			memCount = len(b.memory.List())
		}
		p := s.ActiveProfile()
		status := fmt.Sprintf("📊 <b>Статус Яндекс Станции</b>\n\n"+
			"• Ассистент: %s (%s)\n"+
			"• Громкость: %d/10\n"+
			"• Записей в памяти: %d\n"+
			"• Микрофон: %v\n"+
			"• Разговор сейчас: %v",
			p.Name, p.Engine, vol, memCount, b.engine.mic.Online(), b.engine.Talking())
		if b.engine.music != nil && b.engine.music.Status() != "" {
			status += "\n• Музыка: " + b.engine.music.Status()
		}
		reply(status)
		return
	}

	// 3. Команда /say <текст>
	if strings.HasPrefix(text, "/say") {
		msg := strings.TrimSpace(strings.TrimPrefix(text, "/say"))
		if msg == "" {
			reply("ℹ️ Укажите текст фразы: <code>/say Обед готов!</code>")
			return
		}
		p := s.ActiveProfile()
		voicePrompt := ""
		for _, v := range s.LiveVoices {
			if v.ID == p.Voice {
				voicePrompt = v.Prompt
			}
		}
		go func() {
			_, err := b.engine.sayAloud(s, p, voicePrompt, msg, 60*time.Second)
			if err != nil {
				log.Printf("Telegram: ошибка озвучки /say: %v", err)
			}
		}()
		reply("🗣 Сказал на колонке: «" + msg + "»")
		return
	}

	// 4. Команда /vol <уровень>
	if strings.HasPrefix(text, "/vol") {
		arg := strings.TrimSpace(strings.TrimPrefix(text, "/vol"))
		lvl, err := strconv.Atoi(arg)
		if err != nil {
			reply("ℹ️ Укажите число от 0 до 10: <code>/vol 5</code>")
			return
		}
		if lvl > 10 {
			lvl = (lvl + 5) / 10
		}
		if lvl < 0 {
			lvl = 0
		}
		if lvl > 10 {
			lvl = 10
		}
		if err := setAliceVolume(lvl); err != nil {
			reply("❌ Ошибка изменения громкости: " + err.Error())
			return
		}
		reply(fmt.Sprintf("🔊 Громкость установлена на %d из 10", lvl))
		return
	}

	// 5. Команда /remember <факт>
	if strings.HasPrefix(text, "/remember") {
		fact := strings.TrimSpace(strings.TrimPrefix(text, "/remember"))
		if fact == "" {
			reply("ℹ️ Укажите, что нужно запомнить: <code>/remember Любимый чай — эрл грей</code>")
			return
		}
		if b.memory == nil {
			reply("❌ Память не инициализирована")
			return
		}
		it, err := b.memory.Add(fact)
		if err != nil {
			reply("❌ Ошибка сохранения: " + err.Error())
			return
		}
		reply("🧠 Запомнил: «" + it.Fact + "»")
		return
	}

	// 6. Команда /memory
	if text == "/memory" {
		if b.memory == nil {
			reply("❌ Память не инициализирована")
			return
		}
		items := b.memory.List()
		if len(items) == 0 {
			reply("🧠 Память пока пуста. Напишите <code>/remember &lt;факт&gt;</code> или скажите колонке «Джарвис, запомни, что...».")
			return
		}
		var sb strings.Builder
		sb.WriteString("🧠 <b>Долговременная память:</b>\n\n")
		for i, it := range items {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, it.Fact))
		}
		reply(sb.String())
		return
	}

	// 7. Команда /forget <запрос>
	if strings.HasPrefix(text, "/forget") {
		q := strings.TrimSpace(strings.TrimPrefix(text, "/forget"))
		if q == "" {
			reply("ℹ️ Укажите ключевое слово: <code>/forget чай</code>")
			return
		}
		if b.memory == nil {
			reply("❌ Память не инициализирована")
			return
		}
		n := b.memory.DeleteByQuery(q)
		if n > 0 {
			reply(fmt.Sprintf("🗑 Удалено из памяти записей: %d", n))
		} else {
			reply("🔍 В памяти не нашлось записей по запросу «" + q + "»")
		}
		return
	}

	// 8. Обычный текстовый вопрос ассистенту через нейросеть
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	p := s.ActiveProfile()
	instr := b.engine.instructions(s, p) + "\n\nТы отвечаешь хозяину в мессенджере Telegram. Отвечай кратко, емко и структурировано."
	var answer string
	var err error

	if s.GeminiKey != "" && (p.Engine == "gemini" || s.OpenAIKey == "") {
		answer, err = askGemini(ctx, b.proxy.ClientFor("gemini", 45*time.Second), s.GeminiKey, instr, text, p.WebSearch)
	} else if s.OpenAIKey != "" {
		answer, err = askOpenAI(ctx, b.proxy.ClientFor("openai", 45*time.Second), s.OpenAIKey, p, instr, text, b.engine.tools)
	} else if s.ChatGPT.SignedIn() {
		var a ChatGPTAuth
		if a, err = b.engine.chatgptAuth(ctx); err == nil {
			answer, err = askCodex(ctx, b.proxy.ClientFor("chatgpt", 45*time.Second), a, p, instr, text, b.engine.tools)
		}
	} else {
		err = errors.New("нет доступных ключей API (Gemini или OpenAI)")
	}

	if err != nil {
		reply("⚠️ Не удалось получить ответ: " + shortErr(err))
		return
	}
	reply(answer)
}
