package main

import (
	"net/http"
	"strings"
)

func (w *Web) telegramState() map[string]any {
	s := w.store.Get()
	token := s.TelegramBotToken
	masked := ""
	if token != "" {
		if len(token) > 10 {
			masked = token[:6] + "..." + token[len(token)-4:]
		} else {
			masked = "задан"
		}
	}
	botName := ""
	running := false
	status := "не настроен"
	if w.engine.telegram != nil {
		botName = w.engine.telegram.BotName()
		running = w.engine.telegram.Running()
		status = w.engine.telegram.Status()
	}
	return map[string]any{
		"enabled":  s.TelegramEnabled,
		"set":      token != "",
		"token":    masked,
		"chat_id":  s.TelegramChatID,
		"running":  running,
		"status":   status,
		"bot_name": botName,
	}
}

func (w *Web) memoryState() map[string]any {
	if w.engine.memory == nil {
		return map[string]any{"count": 0, "items": []MemoryItem{}}
	}
	items := w.engine.memory.List()
	return map[string]any{
		"count": len(items),
		"items": items,
	}
}

func (w *Web) getTelegram(rw http.ResponseWriter, r *http.Request) {
	writeJSON(rw, 200, w.telegramState())
}

func (w *Web) putTelegram(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Token   *string `json:"token"`
		ChatID  *string `json:"chat_id"`
		Enabled *bool   `json:"enabled"`
	}
	if readJSON(r, &req) != nil {
		bad(rw, "неверный запрос")
		return
	}
	err := w.store.Update(func(s *Settings) error {
		if req.Token != nil {
			t := cleanTgToken(*req.Token)
			if t != "" {
				s.TelegramBotToken = t
			}
		}
		if req.ChatID != nil {
			s.TelegramChatID = strings.TrimSpace(*req.ChatID)
		}
		if req.Enabled != nil {
			s.TelegramEnabled = *req.Enabled
		}
		return nil
	})
	if err != nil {
		bad(rw, err.Error())
		return
	}
	if w.engine.telegram != nil {
		w.engine.telegram.Restart()
	}
	writeJSON(rw, 200, w.telegramState())
}

func (w *Web) deleteTelegram(rw http.ResponseWriter, r *http.Request) {
	_ = w.store.Update(func(s *Settings) error {
		s.TelegramBotToken = ""
		s.TelegramChatID = ""
		s.TelegramEnabled = false
		return nil
	})
	if w.engine.telegram != nil {
		w.engine.telegram.Stop()
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) testTelegram(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Token  string `json:"token"`
		ChatID string `json:"chat_id"`
	}
	_ = readJSON(r, &req)
	s := w.store.Get()
	token := cleanTgToken(req.Token)
	if token == "" {
		token = cleanTgToken(s.TelegramBotToken)
	}
	chatID := strings.TrimSpace(req.ChatID)
	if chatID == "" {
		chatID = s.TelegramChatID
	}
	if token == "" || chatID == "" {
		bad(rw, "укажите токен бота и chat ID")
		return
	}
	if w.engine.telegram == nil {
		bad(rw, "сервис Telegram не инициализирован")
		return
	}
	if err := w.engine.telegram.Test(token, chatID); err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, map[string]any{"ok": true})
}

func (w *Web) getMemory(rw http.ResponseWriter, r *http.Request) {
	writeJSON(rw, 200, w.memoryState())
}

func (w *Web) addMemory(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		Fact string `json:"fact"`
	}
	if readJSON(r, &req) != nil || strings.TrimSpace(req.Fact) == "" {
		bad(rw, "пустой факт")
		return
	}
	if w.engine.memory == nil {
		bad(rw, "память не инициализирована")
		return
	}
	item, err := w.engine.memory.Add(req.Fact)
	if err != nil {
		bad(rw, err.Error())
		return
	}
	writeJSON(rw, 200, item)
}

func (w *Web) deleteMemory(rw http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if readJSON(r, &req) != nil || strings.TrimSpace(req.ID) == "" {
		bad(rw, "не указан id")
		return
	}
	if w.engine.memory == nil {
		bad(rw, "память не инициализирована")
		return
	}
	ok := w.engine.memory.Delete(req.ID)
	writeJSON(rw, 200, map[string]any{"ok": ok})
}
