package main

// Написать Джарвису текстом (поле в переписке на странице настроек). Та же логика, что у голоса:
// «стоп», команды Алисе и умный дом выполняются сразу, остальное — вопрос к модели. Ответ появляется
// в переписке и, если включено, звучит на колонке.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const responsesURL = "https://api.openai.com/v1/responses"

// askOpenAI — текстовый вопрос модели (Responses API) с теми же инструментами, что у голоса.
func askOpenAI(ctx context.Context, client *http.Client, key string, p Profile, instructions, text string, tools *Tools) (string, error) {
	model := p.BackendModel
	if model == "" {
		model = BackendModels[0]
	}
	effort := normalizedModelEffort(model, p.Effort)
	defs := append([]any{}, tools.Definitions()...)
	if p.WebSearch {
		defs = append(defs, map[string]any{"type": "web_search"})
	}
	body := map[string]any{"model": model, "instructions": instructions, "tools": defs, "tool_choice": "auto",
		"reasoning": map[string]any{"effort": effort},
		"input":     []any{map[string]any{"role": "user", "content": text}}}
	for step := 0; step < 5; step++ {
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		var out struct {
			ID     string `json:"id"`
			Output []struct {
				Type      string `json:"type"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				CallID    string `json:"call_id"`
				Content   []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &out)
		if resp.StatusCode != 200 {
			msg := fmt.Sprintf("OpenAI ответил %d", resp.StatusCode)
			if out.Error != nil {
				msg += ": " + out.Error.Message
			}
			return "", errors.New(msg)
		}
		var answer strings.Builder
		var results []any
		for _, item := range out.Output {
			switch item.Type {
			case "message":
				for _, c := range item.Content {
					if c.Type == "output_text" {
						answer.WriteString(c.Text)
					}
				}
			case "function_call":
				tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
				result := tools.Call(tctx, item.Name, item.Arguments)
				cancel()
				log.Printf("текст: %s(%s) → %s", item.Name, item.Arguments, result)
				results = append(results, map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": result})
			}
		}
		if len(results) == 0 {
			if s := strings.TrimSpace(answer.String()); s != "" {
				return s, nil
			}
			return "", errors.New("модель ничего не ответила")
		}
		body = map[string]any{"model": model, "instructions": instructions, "tools": defs, "previous_response_id": out.ID,
			"reasoning": map[string]any{"effort": effort}, "input": results}
	}
	return "", errors.New("модель не закончила за 5 шагов")
}

const textRules = "\n\nТебе написали текстом в переписке настроек. Ответ покажут там же и прочитают вслух на колонке, " +
	"поэтому отвечай коротко, по-русски, без списков и разметки."

// TextTurn — реплика, присланная текстом. speak — прочитать ответ вслух на колонке.
func (e *Engine) TextTurn(text string, speak bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if isStop(text) {
		if isStopAll(text) {
			e.Stop("написали «стоп»")
		} else {
			e.EndChat("написали «" + text + "»")
		}
		e.journal.note("стоп", text, "")
		return
	}
	if answer, ok := e.music.respondProposal(context.Background(), text); ok {
		e.journal.note("выбор ролика", text, answer)
		return
	}
	if answer, ok := e.resumeSavedRequest(context.Background(), text); ok {
		e.journal.note("продолжение ролика", text, answer)
		return
	}
	if q, ok := youtubeRequest(text); ok && e.music != nil {
		e.journal.note("музыка", text, e.playMusic(q))
		return
	}
	if answer, ok := e.controlMusic(text); ok {
		e.journal.note("музыка", text, answer)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if h := e.tools.HomeAssistant; h.vacuumIntent(text) {
		e.journal.note("Home Assistant", text, h.vacuumCommand(ctx, text))
		return
	}
	if sl := e.tools.SmartLife; sl.ready() && e.store.Get().UseHome("smartlife") {
		if ok, answer := sl.Fast(ctx, text); ok {
			e.journal.note("Smart Life", text, answer)
			return
		}
	}
	alice, toAlice := AliceSkillFor(text, e.store.Get().UseHome("yandexmusic"))
	if !toAlice && e.store.Get().UseHome("alice") {
		toAlice, alice = e.tools.HomeCommand(ctx, text)
	}
	if toAlice {
		if err := AliceText(alice, false); err == nil {
			log.Printf("текст: «%s» → Алисе", alice)
			e.journal.note("умный дом → Алиса", text, "передал Алисе: «"+alice+"»")
			return
		}
	}
	if ok, answer := e.tools.Fast(ctx, text); ok && e.store.Get().UseHome("yandex") {
		log.Printf("текст: быстрая команда «%s» → %s", text, answer)
		e.journal.note("умный дом (быстро)", text, answer)
		return
	}
	s := e.store.Get()
	p := s.ActiveProfile()
	t := e.journal.New("текст", p.BackendModel)
	e.journal.Update(t, func(t *Turn) { t.Text, t.Live = text, true })
	t0 := time.Now()
	var answer string
	var err error
	instr := e.instructions(s, p) + textRules + "\n\n" + backendRules + e.tools.Context()
	switch {
	case s.GeminiKey != "" && (p.Engine == "gemini" || s.OpenAIKey == ""):
		pref := p.LLMModel
		if pref == "" {
			pref = p.BackendModel
		}
		answer, err = askGemini(ctx, e.proxy.ClientFor("gemini", 60*time.Second), s.GeminiKey, pref, instr, text, p.WebSearch)
	case s.OpenAIKey != "":
		answer, err = askOpenAI(ctx, e.proxy.ClientFor("openai", 60*time.Second), s.OpenAIKey, p, instr, text, e.tools)
	case s.ChatGPT.SignedIn():
		var a ChatGPTAuth
		if a, err = e.chatgptAuth(ctx); err == nil {
			answer, err = askCodex(ctx, e.proxy.ClientFor("chatgpt", 60*time.Second), a, p, instr, text, e.tools)
		}
	default:
		err = errors.New("нет ключа OpenAI или Gemini и не выполнен вход в ChatGPT — это делается в «Подключениях»")
	}
	e.journal.Update(t, func(t *Turn) {
		t.Live = false
		if err != nil {
			t.Error = shortErr(err)
		} else {
			t.Answer = answer
		}
	})
	e.journal.Remember(t)
	if err != nil {
		log.Printf("текст: %v", err)
		return
	}
	log.Printf("текст: ответ за %.1f с", time.Since(t0).Seconds())
	if !speak || keyMissing(s, p.Engine) != "" {
		return
	}
	e.mu.Lock()
	if e.busy || e.live != nil { // идёт голосовой разговор — не перебиваем
		e.mu.Unlock()
		return
	}
	e.busy, e.aborted = true, false
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.busy, e.aborted = false, false
		e.mu.Unlock()
		e.leds.Set(LedOff)
	}()
	voicePrompt := ""
	for _, v := range s.LiveVoices {
		if v.ID == p.Voice {
			voicePrompt = v.Prompt
		}
	}
	if _, err := e.sayAloud(s, p, voicePrompt, answer, 90*time.Second); err != nil {
		log.Printf("текст: озвучка не удалась: %v", err)
	}
}

// sayAloud — произнести фразу голосом ассистента: короткий разговор с моделью, которой велено сказать ровно это.
// Возвращает сыгранный звук (16 кГц): пробную фразу голоса сохраняют и потом играют с диска.
func (e *Engine) sayAloud(s Settings, p Profile, voicePrompt, text string, limit time.Duration) ([]int16, error) {
	return e.sayAloudContext(context.Background(), s, p, voicePrompt, text, limit)
}

func (e *Engine) sayAloudContext(parent context.Context, s Settings, p Profile, voicePrompt, text string, limit time.Duration) ([]int16, error) {
	ctx, cancel := context.WithTimeout(parent, limit)
	defer cancel()
	p.WebSearch = false
	const instr = "Ты озвучиваешь готовый текст. Произнеси ровно то, что тебе дали, ничего не добавляя, и замолчи."
	noTools := NewTools(func() string { return "" }, DirectClient)
	var sess Session
	if p.Engine == "eleven_realtime" {
		sess = NewElevenRealtimeSession(ctx, s.ElevenLabsKey, p, instr, noTools, e.proxy.LiveClientFor("elevenlabs"), &Journal{})
	} else if isClassic(p) {
		sess = NewClassicSpeaker(ctx, s.ElevenLabsKey, p, e.proxy.LiveClientFor("elevenlabs"), &Journal{})
	} else if p.Engine == "gemini" {
		sess = NewGeminiSession(ctx, s.GeminiKey, p, instr, noTools, e.proxy.LiveClientFor("gemini"), &Journal{})
	} else if p.Engine == "grok" {
		sess = NewGrokSession(ctx, s.GrokKey, p, instr, noTools, e.proxy.LiveClientFor("grok"), &Journal{})
	} else if p.Engine == "chatgpt" {
		sess = NewChatGPTSession(ctx, e.chatgptAuth, p, instr, e.proxy.LiveClientFor("chatgpt"),
			func(context.Context, string) (string, error) {
				return "", errors.New("озвучка без бэкенда")
			}, &Journal{})
	} else {
		sess = NewLiveSession(ctx, s.OpenAIKey, p, instr, voicePrompt, noTools, e.proxy.LiveClientFor("openai"), &Journal{})
	}
	defer sess.Close()
	if err := sess.Ready(10 * time.Second); err != nil {
		return nil, err
	}
	if err := sess.Say(text); err != nil {
		return nil, err
	}
	if err := e.player.StartVoice(sess); err != nil {
		return nil, err
	}
	e.leds.Set(LedSpeaking)
	var rec []int16
	heard := false
	quietSince := time.Now()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for time.Since(quietSince) < 1200*time.Millisecond || !heard {
		select {
		case pcm, ok := <-sess.Audio():
			if !ok {
				e.player.Finish(3 * time.Second)
				if !heard {
					return nil, errors.New("голос не прозвучал")
				}
				return rec, nil
			}
			_ = e.player.Write(pcm)
			e.leds.Level(peakOf(pcm) / 9000)
			rec = append(rec, pcm...)
			if loud(pcm) {
				heard, quietSince = true, time.Now()
			}
		case <-tick.C:
			e.mu.Lock()
			stop := e.aborted
			e.mu.Unlock()
			if stop { // «Стоп» из настроек или позвали Алису
				e.player.Stop(true)
				return rec, nil
			}
		case <-ctx.Done():
			e.player.Stop(true)
			return nil, errors.New("голос не успел прозвучать")
		}
	}
	e.player.Finish(3 * time.Second)
	return rec, nil
}

const codexURL = "https://chatgpt.com/backend-api/codex/responses"

// askCodex — тот же вопрос модели, но по подписке ChatGPT (как это делает Codex): ответ приходит потоком,
// история не хранится на сервере, поэтому вызовы инструментов отправляются обратно целиком.
func askCodex(ctx context.Context, client *http.Client, a ChatGPTAuth, p Profile, instructions, text string, tools *Tools) (string, error) {
	model := p.BackendModel
	if model == "" {
		model = BackendModels[0]
	}
	effort := normalizedModelEffort(model, p.Effort)
	defs := append([]any{}, tools.Definitions()...)
	if p.WebSearch {
		defs = append(defs, map[string]any{"type": "web_search"})
	}
	input := []any{map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": text}}}}
	session := newID()
	for step := 0; step < 5; step++ {
		raw, _ := json.Marshal(map[string]any{"model": model, "instructions": instructions, "input": input, "tools": defs,
			"tool_choice": "auto", "parallel_tool_calls": false, "reasoning": map[string]any{"effort": effort},
			"store": false, "stream": true, "include": []any{}})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, codexURL, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+a.Access)
		req.Header.Set("chatgpt-account-id", a.Account)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", "codex_cli_rs")
		req.Header.Set("session_id", session)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		if resp.StatusCode != 200 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return "", fmt.Errorf("ChatGPT ответил %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
		var answer strings.Builder
		var calls []map[string]any
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
		for sc.Scan() {
			line, ok := strings.CutPrefix(sc.Text(), "data: ")
			if !ok {
				continue
			}
			var ev map[string]any
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			switch str(ev["type"]) {
			case "response.output_item.done":
				item, _ := ev["item"].(map[string]any)
				switch str(item["type"]) {
				case "message":
					parts, _ := item["content"].([]any)
					for _, c := range parts {
						if m, ok := c.(map[string]any); ok && str(m["type"]) == "output_text" {
							answer.WriteString(str(m["text"]))
						}
					}
				case "function_call":
					calls = append(calls, item)
				}
			case "response.failed", "error":
				resp.Body.Close()
				return "", fmt.Errorf("ChatGPT: %s", errMessage(ev))
			}
		}
		resp.Body.Close()
		if len(calls) == 0 {
			if s := strings.TrimSpace(answer.String()); s != "" {
				return s, nil
			}
			return "", errors.New("модель ничего не ответила")
		}
		for _, item := range calls {
			tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			result := tools.Call(tctx, str(item["name"]), str(item["arguments"]))
			cancel()
			log.Printf("подписка: %s(%s) → %s", str(item["name"]), str(item["arguments"]), result)
			input = append(input, map[string]any{"type": "function_call", "name": item["name"], "arguments": item["arguments"], "call_id": item["call_id"]},
				map[string]any{"type": "function_call_output", "call_id": item["call_id"], "output": result})
		}
	}
	return "", errors.New("модель не закончила за 5 шагов")
}
