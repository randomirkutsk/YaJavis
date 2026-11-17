package main

// Главный цикл: микрофон → детектор слова → разговор → динамик. Алиса имеет приоритет:
// позвали её — Джарвис сразу замолкает и закрывает разговор.

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WakeDetector interface {
	// Feed — кадр 16 кГц; true — услышано слово.
	Feed(pcm []int16) bool
	Reset()
	Ready() bool
}

type Engine struct {
	micLevel levelMeter // голос в микрофоне за разговор (до усиления) — в журнал: видно, что модель слышит тихо
	store    *Store
	proxy    *Proxy
	mic      *Mic
	player   *Player
	leds     *Leds
	tools    *Tools
	journal  *Journal
	wake     WakeDetector
	stt      *STT        // быстрая расшифровка для команд (соединение держится заранее)
	cmds     *CmdSpotter // «стоп», «громче», «тише» штатным распознавателем Станции
	music    *Music      // музыка с YouTube
	alice    *AliceMusic // что играет у Алисы (Яндекс Музыка)
	train    *Trainer    // записи для дообучения слова
	bg       *BgRecorder // длинная запись фона на сервер
	memory   *MemoryStore
	telegram *TelegramBot

	dataDir string

	mu          sync.Mutex
	live        Session
	busy        bool
	holding     bool    // имя ещё проверяется: разговора с моделью нет, звук копится в hold
	hold        []int16 // звук с момента слова — уйдёт модели, если это окажется вопрос к ней
	aborted     bool    // «Стоп» или «Алиса», пока разговор с моделью ещё не открыт
	ring        []int16 // последние 3 с микрофона: само слово и начало фразы — в разговор, для проверки и дообучения
	lastWake    time.Time
	lastLoud    time.Time // когда микрофон последний раз слышал что-то громче тишины
	Status      string
	opCancel    context.CancelFunc
	pendingWake bool
	notice      func(context.Context, string)
	musicDone   time.Time // модель включила музыку (или управила ей) — договорит «Включаю» и разговор закроется
}

// EndAfterMusic — музыка включилась по просьбе модели: больше не слушаем, разговор закончится,
// как только модель договорит.
func (e *Engine) EndAfterMusic() {
	e.mu.Lock()
	e.musicDone = time.Now()
	e.mu.Unlock()
}

func (e *Engine) Run(ctx context.Context) {
	go e.watchAlice(ctx)
	_, ch := e.mic.Subscribe(256)
	for pcm := range ch {
		e.mu.Lock()
		live := e.live
		// последние 3 с микрофона. Буфер с запасом до 6 с и сдвигается на месте раз в ~3 с:
		// раньше каждые 10 мс копировались все 3 с в новую память — ~10 МБ мусора в секунду,
		// и сборщик мусора работал без передышки
		if len(e.ring)+len(pcm) > cap(e.ring) {
			if cap(e.ring) < 6*MicRate {
				nr := make([]int16, 0, 6*MicRate)
				e.ring = append(nr, e.ring[max(0, len(e.ring)-3*MicRate):]...)
			} else {
				e.ring = e.ring[:copy(e.ring, e.ring[max(0, len(e.ring)-3*MicRate):])]
			}
		}
		e.ring = append(e.ring, pcm...)
		busy := e.busy
		if loud(pcm) {
			e.lastLoud = time.Now()
		}
		if e.holding && live == nil && len(e.hold) < 20*MicRate {
			e.hold = append(e.hold, pcm...)
		}
		e.mu.Unlock()
		if busy || live != nil {
			e.leds.Hear(peakOf(pcm) / 9000)
			e.micLevel.add(pcm, MicRate)
		}
		raw := pcm
		if db := e.store.Flags().MicBoostDB; db > 0 && (busy || live != nil) {
			pcm = boostPCM(pcm, db) // только для того, что уходит модели; слово активации слушает как есть
		}
		if busy && e.stt != nil {
			e.stt.Feed(pcm)
		}
		e.bg.Feed(raw, busy || live != nil)
		if busy || e.music.Playing() {
			e.cmds.Feed(raw)
		} else {
			e.cmds.Idle()
		}
		wakeHit := false
		if e.wake != nil && e.store.Flags().WakeEnabled {
			wakeHit = e.wake.Feed(raw) // поток признаков непрерывный, в том числе при проверке имени
			if !busy && live == nil && !wakeHit && e.train != nil {
				var p float32
				if k, ok := e.wake.(*KWS); ok {
					p = k.LastProb
				}
				e.train.Observe(p, e.ringTail(), time.Since(e.lastLoud) < 300*time.Millisecond)
			}
		}
		if live != nil {
			live.Feed(pcm)
			// «Джарвис» во время разговора — как у Алисы: сразу замолкает и слушает
			if wakeHit && time.Since(e.lastWake) > 2500*time.Millisecond {
				e.interrupt(live)
			}
			continue
		}
		if busy || e.wake == nil || !e.store.Flags().WakeEnabled {
			if busy && wakeHit {
				e.mu.Lock()
				if e.opCancel != nil {
					e.pendingWake = true
					e.opCancel()
				}
				e.mu.Unlock()
			}
			continue
		}
		if wakeHit {
			e.Trigger("слово")
		}
	}
}

// InDialog — идёт беседа (человек сказал уже не одну фразу), а не разовая команда.
func (e *Engine) InDialog() bool {
	e.mu.Lock()
	live := e.live
	e.mu.Unlock()
	return live != nil && live.UserTurns() >= 2
}

// ringTail — последние 3 с микрофона (сам буфер бывает длиннее).
func (e *Engine) ringTail() []int16 {
	return e.ring[max(0, len(e.ring)-3*MicRate):]
}

// Trigger — начать разговор (по слову, по кнопке в настройках).
func (e *Engine) Trigger(source string) {
	e.mu.Lock()
	if e.busy || time.Since(e.lastWake) < 1500*time.Millisecond {
		e.mu.Unlock()
		return
	}
	e.busy, e.lastWake, e.aborted = true, time.Now(), false
	e.hold, e.holding = nil, true // пока не решили, открывать ли разговор, — звук копим
	clip := append([]int16(nil), e.ringTail()...)
	e.mu.Unlock()
	byWord := source == "слово"
	var preroll []int16
	sure := !byWord
	var prob float32
	if byWord {
		// звук с 1.2 с до срабатывания: само «Джарвис» и начало быстрой фразы — разговор слышит всё,
		// а по расшифровке имени проверяем, что это не ложное срабатывание
		preroll = clip // сеть использует ~2,5 с контекста: 1,2 с обрезали начало длинного оклика
		switch k := e.wake.(type) {
		case *KWS:
			prob = k.LastProb
			sure = prob >= 0.9
		case *OWW:
			prob = k.LastWakeProb
			sure = prob >= 0.9
		}
		// как у Алисы: услышал слово — сразу кольцо и разговор, без проверки имени расшифровкой.
		// Проверку можно вернуть в настройках колонки («Проверять имя перед ответом»).
		if !e.store.Get().WakeCheck {
			sure = true
		}
	}
	if sure {
		e.leds.Set(LedListening) // сразу, без сети
	}
	if e.wake != nil {
		e.wake.Reset() // хвост этого же слова не должен сработать ещё раз
	}
	e.train.Reset()
	e.bg.Discard()
	if byWord {
		log.Printf("Джарвис: активация (%s, уверенность %.2f)", source, prob)
	} else {
		log.Printf("Джарвис: активация (%s)", source)
	}
	go func() {
		defer func() {
			e.mu.Lock()
			queued := e.pendingWake
			e.pendingWake = false
			e.busy, e.live, e.aborted = false, nil, false
			e.hold, e.holding = nil, false
			e.mu.Unlock()
			if e.wake != nil {
				e.wake.Reset()
			}
			e.leds.Set(LedOff)
			if l := e.micLevel.level(); l.secs > 0 {
				log.Printf("микрофон: за разговор голос %.1f дБ, пик %.1f дБ (усиление для ИИ +%d дБ)", l.rms, l.peak, e.store.Flags().MicBoostDB)
			}
			e.micLevel.reset()
			if queued {
				e.Trigger("слово")
			}
		}()
		var quick <-chan string
		if e.stt != nil && e.stt.Ready() {
			if ch, err := e.stt.Begin(preroll); err == nil {
				quick = ch
			}
		}
		e.converse(byWord, sure, preroll, clip, quick)
		if e.stt != nil {
			e.stt.End()
		}
	}()
}

// Имя в расшифровке. Проверочная расшифровка часто слышит «Джарвис» иначе: «Джарис», «Чарлис», «Жарови»,
// «Жарится», «Дарвис». Раньше такие оклики отбрасывались как ложные — и Джарвис «не откликался».

// nameSure — куски, которые бывают только в самом имени: ищем их в любом месте фразы.
var nameSure = []string{"арвис", "арвес", "джарв", "жарв", "чарв", "дарв", "ярвис", "джерв", "jarv"}

// nameLike — слово похоже на «Джарвис» (для первых слов фразы, где имя и должно стоять).
func nameLike(w string) bool {
	w = strings.NewReplacer("ž", "zh", "š", "sh").Replace(strings.ToLower(w))
	for _, name := range []string{"jarvis", "jervis", "zharvis", "charvis", "djarvis", "dzharvis", "zarvis"} {
		if w == name {
			return true
		}
	}
	if (strings.HasPrefix(w, "jar") || strings.HasPrefix(w, "djar") || strings.HasPrefix(w, "zhar")) && editDistance(w, "jarvis") <= 2 {
		return true
	}
	for _, s := range nameSure {
		if strings.Contains(w, s) {
			return true
		}
	}
	for _, p := range []string{"джар", "жар", "чарлис", "джер", "жерв"} {
		if strings.HasPrefix(w, p) {
			return true
		}
	}
	n := len([]rune(w))
	return n >= 5 && n <= 9 && editDistance(w, "джарвис") <= 2
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// namePos — номер слова-имени во фразе или -1. Точные куски имени — где угодно, похожие слова — только
// в первых двух («эй, Джарис, …»): в середине фразы «жарко» и «Чарли» именем не считаем.
func namePos(words []string) int {
	for i, w := range words {
		for _, s := range nameSure {
			if strings.Contains(w, s) {
				return i
			}
		}
	}
	for i, w := range words {
		if i < 2 && nameLike(w) {
			return i
		}
	}
	return -1
}

// hasName — в расшифровке есть «Джарвис» (в любом написании распознавания)
func hasName(text string) bool { return namePos(strings.Fields(normText(text))) >= 0 }

// canonName — та же фраза, но имя записано как «джарвис»: дальше её разбирают быстрые команды,
// а они знают только правильное написание («Чарлис, выключи лампу» → «джарвис выключи лампу»).
func canonName(text string) string {
	words := strings.Fields(normText(text))
	if i := namePos(words); i >= 0 {
		words[i] = "джарвис"
	}
	return strings.Join(words, " ")
}

// commandAfterLostName — детектор слова уверен, а расшифровка вместо имени услышала другое слово
// («Сейчас отключи свет»): если остаток фразы — чистая команда или «стоп», считаем, что звали.
func commandAfterLostName(text string) (string, bool) {
	words := strings.Fields(normText(text))
	if len(words) < 2 {
		return "", false
	}
	rest := strings.Join(words[1:], " ")
	if fastCmdRe.MatchString(rest) || colorRe.MatchString(rest) || stopRe.MatchString(rest) {
		return "джарвис " + rest, true
	}
	return "", false
}

// saveFalse — кусок звука ложного срабатывания: на нём модель слова дообучится.
func (e *Engine) saveFalse(clip []int16) {
	if e.dataDir == "" || len(clip) == 0 {
		return
	}
	if e.train == nil && e.store != nil {
		e.train = &Trainer{dataDir: e.dataDir, store: e.store}
	}
	e.train.Save("false", clip, "")
}

// langNames — язык хозяина для инструкций моделям.
var langNames = map[string]string{"ru": "русском", "en": "английском", "uk": "украинском"}

func (e *Engine) instructions(s Settings, p Profile) string {
	text := s.PromptText(p.Prompt)
	if n := langNames[s.Language]; n != "" {
		text += "\n\nХозяин всегда говорит на " + n + " языке. Понимай его речь как " + n +
			", даже если она звучит неразборчиво или похоже на другой язык, и отвечай только на " + n + "."
	}
	if city := s.City; city != "" {
		text += "\n\nХозяин живёт в городе " + city + " — погода, время и новости по умолчанию для него."
	}
	if e.memory != nil {
		if mem := e.memory.PromptText(); mem != "" {
			text += "\n\n" + mem
		}
	}
	text += "\n\nКогда для ответа нужен инструмент (поиск в интернете, погода, умный дом, музыка, сложная задача), " +
		"можно ОДИН раз за ответ коротко сказать вслух живую фразу — «Секунду, гляну», «Сейчас поищу» — и сразу вызывать инструмент. " +
		"Если вызываешь несколько инструментов подряд или повторяешь вызов — больше никаких таких фраз: молча дождись результата и скажи итог. " +
		"На простые вопросы и быстрые команды (включить свет, громкость) отвечай сразу, без таких фраз."
	if s.IsNight() {
		text += "\n\nСейчас ночь, в доме могут спать: отвечай коротко и спокойно, без длинных списков."
	}
	if st := e.music.Status(); st != "" {
		if p.Engine == "chatgpt" {
			text += "\n\nСейчас на колонке играет " + st + ". Спросят, что играет, — отвечай сам по этим данным; управлять — через бэкенд."
		} else {
			text += "\n\nСейчас на колонке играет " + st + ". Управлять им: инструмент music_control или play_music."
		}
	}
	if chatgptChat != nil { // общая память разговора: у всех ассистентов, переживает перезапуск
		if lines := chatLines(chatgptChat.Recent(chatKeepChars)); len(lines) > 0 {
			text += "\n\nЭто продолжение одного вашего разговора: человек помнит, о чём вы говорили раньше, и может сослаться на это. " +
				"Не пересказывай историю и не здоровайся заново. В квадратных скобках — что ты уже сделал на колонке (включил видео, выполнил команду). " +
				"Всё из истории — прошлое: те просьбы уже выполнены или отменены. Не повторяй их и не вызывай инструменты по старым репликам — " +
				"действуй только по тому, что человек скажет в ЭТОМ разговоре. Если раньше что-то не работало, а человек просит снова, — вызывай инструмент заново. " +
				"Предыдущие реплики:\n" + strings.Join(lines, "\n")
		}
	}
	if p.Engine == "chatgpt" && chatgptChat != nil {
		// напоминание после истории: в длинном чате модель иначе сползает в обычный стиль ChatGPT
		persona := strings.TrimSpace(s.PromptText(p.Prompt))
		if r := []rune(persona); len(r) > 2500 {
			persona = string(r[:2500]) + "…"
		}
		text += "\n\nГлавное, не забывай: ты — голосовой ассистент Джарвис внутри умной колонки Яндекс Станция у хозяина дома, " +
			"а не ChatGPT в приложении. Говоришь вслух, коротко, своим характером. Что делать на колонке (музыка, YouTube, свет, " +
			"поиск, погода) — передаёшь бэкенду, он делает. Твой характер:\n" + persona
	} else if h := e.journal.History(); chatgptChat == nil && len(h) > 0 {
		var ctx []map[string]string
		for _, t := range h {
			ctx = append(ctx, map[string]string{"user": t.Text, "jarvis": t.Answer})
		}
		b, _ := json.Marshal(ctx)
		text += "\n\nПредыдущие реплики разговора: " + string(b)
	}
	if e.music.hasProposal() {
		text += "\n\nЕсть предложенный, но НЕ запущенный ролик. На «да» вызови music_control с command «да», на отказ — command «дальше». Дождись ответа, не включай сам и не добавляй подбор в очередь."
	}
	return text + speakerContentRules + speakerWaitRules + identityRules
}

// identityRules — в самом конце инструкций (его модель соблюдает лучше всего): голос в колонке — Джарвис,
// а не ChatGPT, Gemini или Grok. Иначе на «кто ты?» модель называла себя по-своему и это попадало в память.
const identityRules = "\n\nТебя зовут Джарвис. Спросят, кто ты, как тебя зовут или на чём ты работаешь, — ты Джарвис, голосовой ассистент " +
	"в умной колонке хозяина. Никогда не называй себя ChatGPT, GPT, Gemini, Grok, Алисой, «языковой моделью» или «ассистентом OpenAI/Google/xAI». " +
	"Если хозяин сам прямо спросит, какая нейросеть внутри, — можно сказать, что Джарвис работает на ней, но ты всё равно Джарвис."

// newSession — открыть разговор с моделью тем движком, который выбран у ассистента.
func (e *Engine) newSession(s Settings, p Profile) Session {
	if p.Engine == "eleven_realtime" {
		return NewElevenRealtimeSession(context.Background(), s.ElevenLabsKey, p, e.instructions(s, p), e.tools, e.proxy.LiveClientFor("elevenlabs"), e.journal)
	}
	if isClassic(p) {
		classicDefaults(&p)
		return NewClassicSession(context.Background(), s.ElevenLabsKey, p, e.classicBrain(s, p), e.proxy.LiveClientFor("elevenlabs"), e.journal)
	}
	if p.Engine == "gemini" {
		return NewGeminiSession(context.Background(), s.GeminiKey, p, e.instructions(s, p), e.tools, e.proxy.LiveClientFor("gemini"), e.journal)
	}
	if p.Engine == "grok" {
		return NewGrokSession(context.Background(), s.GrokKey, p, e.instructions(s, p), e.tools, e.proxy.LiveClientFor("grok"), e.journal)
	}
	if p.Engine == "chatgpt" && chatgptLimited() && s.GeminiKey != "" {
		g := p // лимит подписки кончился — тот же характер, но голос Gemini
		g.Engine, g.Voice, g.LiveModel = "gemini", geminiVoice(""), ""
		log.Printf("Джарвис: лимит ChatGPT ещё не обновился — отвечаю через Gemini")
		return NewGeminiSession(context.Background(), s.GeminiKey, g, e.instructions(s, p), e.tools, e.proxy.LiveClientFor("gemini"), e.journal)
	}
	if p.Engine == "chatgpt" {
		return NewChatGPTSession(context.Background(), e.chatgptAuth, p, e.instructions(s, p), e.proxy.LiveClientFor("chatgpt"),
			e.brain(s, p), e.journal)
	}
	voicePrompt := ""
	for _, v := range s.LiveVoices {
		if v.ID == p.Voice {
			voicePrompt = v.Prompt
		}
	}
	return NewLiveSession(context.Background(), s.OpenAIKey, p, e.instructions(s, p), voicePrompt, e.tools,
		e.proxy.LiveClientFor("openai"), e.journal)
}

// chatgptAuth — действующий вход в ChatGPT (продлевается сам).
func (e *Engine) chatgptAuth(ctx context.Context) (ChatGPTAuth, error) {
	return ChatGPTToken(ctx, e.store, e.proxy.ClientFor("chatgpt", 20*time.Second))
}

// brain — кто думает и управляет домом, когда голосовая модель передаёт задачу: модель OpenAI по ключу,
// иначе Gemini, иначе модель подписки ChatGPT.
func (e *Engine) brain(s Settings, p Profile) Brain {
	base := backendRules + e.tools.Context()
	return func(ctx context.Context, task string) (string, error) {
		instr := base
		if st := e.music.Status(); st != "" {
			instr += "\n\nСейчас на колонке играет " + st + "."
		}
		if s.OpenAIKey != "" {
			return askOpenAI(ctx, e.proxy.ClientFor("openai", 60*time.Second), s.OpenAIKey, p, instr, task, e.tools)
		}
		// подписка ChatGPT — раньше Gemini: у неё есть умный дом, музыка и поиск, а у текстового Gemini только поиск
		if s.ChatGPT.SignedIn() || s.GeminiKey == "" {
			a, err := e.chatgptAuth(ctx)
			if err == nil {
				var answer string
				if answer, err = askCodex(ctx, e.proxy.ClientFor("chatgpt", 60*time.Second), a, p, instr, task, e.tools); err == nil {
					return answer, nil
				}
			}
			if s.GeminiKey == "" || ctx.Err() != nil {
				return "", err
			}
			log.Printf("мозг: подписка ChatGPT не ответила (%v) — спрашиваю Gemini", err)
		}
		pref := p.BackendModel
		if pref == "" {
			pref = p.LLMModel
		}
		return askGemini(ctx, e.proxy.ClientFor("gemini", 60*time.Second), s.GeminiKey, pref, instr, task, p.WebSearch)
	}
}

// keyMissing — чего не хватает, чтобы говорить выбранным движком ("" — всё на месте).
func keyMissing(s Settings, engine string) string {
	if engine == "eleven_realtime" {
		if s.ElevenLabsKey == "" {
			return "нужен ключ ElevenLabs в «Подключениях»"
		}
		return ""
	}
	if engine == "elevenlabs" {
		if s.ElevenLabsKey == "" {
			return "нет ключа ElevenLabs — добавьте его в «Подключениях»"
		}
		if s.OpenAIKey == "" {
			return "для текстовой модели классического режима нужен ключ OpenAI в «Подключениях»"
		}
		return ""
	}
	if engine == "grok" {
		if s.GrokKey == "" {
			return "нет ключа xAI (Grok) — добавьте его в «Подключениях»"
		}
		return ""
	}
	if engine == "chatgpt" {
		if !s.ChatGPT.SignedIn() {
			return "не выполнен вход в ChatGPT — войдите в «Подключениях»"
		}
		return ""
	}
	if engine == "gemini" {
		if s.GeminiKey == "" {
			return "нет ключа Gemini — добавьте его в «Подключениях»"
		}
		return ""
	}
	if s.OpenAIKey == "" {
		return "нет ключа OpenAI — добавьте его в «Подключениях»"
	}
	return ""
}

// converse — один разговор от слова (или кнопки) до тишины.
//
// Если детектор слова не уверен (ниже 0.97), разговор с моделью сразу не открывается: сначала дешёвая
// расшифровка проверяет, что в начале фразы есть имя. Ложное срабатывание тогда не стоит ничего, а «стоп»,
// команды Алисе и умный дом выполняются вообще без модели. Звук с момента слова копится и уходит модели целиком,
// когда выясняется, что это вопрос к ней.
func (e *Engine) converse(byWord, sure bool, preroll, clip []int16, quick <-chan string) {
	s := e.store.Get()
	p := s.ActiveProfile()
	if msg := keyMissing(s, p.Engine); msg != "" {
		e.fail(msg)
		return
	}
	go e.tools.Prewarm()

	var sess Session
	var out <-chan []int16
	var interruptions <-chan struct{}
	defer func() {
		if sess != nil {
			sess.Close()
		}
	}()
	idle := p.IdleS
	if idle < 2 {
		idle = 8
	}
	speaking := false
	var checked int64           // реплика, которую уже проверили на «стоп» и быструю команду
	verified := !byWord || sure // уверенное акустическое имя не обязано второй раз правильно написаться облаком
	var held [][]int16
	start := time.Now()
	readyAt := start
	var lastVoice time.Time // когда модель последний раз говорила вслух
	heard := false          // быстрая расшифровка что-то услышала
	e.mu.Lock()
	e.musicDone = time.Time{}
	e.mu.Unlock()

	// open — открыть разговор с моделью и отдать ему весь звук с момента слова
	open := func() bool {
		if verified && byWord {
			e.leds.Set(LedThinking) // имя подтвердилось, соединяюсь с моделью — бегущий огонёк
		}
		sess = e.newSession(s, p)
		e.mu.Lock()
		sess.Feed(boostPCM(append(append([]int16(nil), preroll...), e.hold...), e.store.Flags().MicBoostDB))
		e.hold, e.holding = nil, false
		e.live = sess
		e.mu.Unlock()
		if err := sess.Ready(12 * time.Second); err != nil {
			log.Printf("разговор не открылся: %v", err)
			sess.FinishTurn("разговор не открылся: " + shortErr(err))
			return false
		}
		if err := e.player.StartVoice(sess); err != nil {
			sess.FinishTurn("динамик: " + err.Error())
			return false
		}
		out, readyAt = sess.Audio(), time.Now()
		if t, ok := sess.(interface{ Touch() }); ok {
			t.Touch()
		}
		if x, ok := sess.(interface{ Interruptions() <-chan struct{} }); ok {
			interruptions = x.Interruptions()
		}
		if verified {
			e.leds.Set(LedListening)
		}
		return true
	}
	// note — реплика, выполненная без модели: в журнал разговора (если он открыт) или отдельной записью
	note := func(route, text, answer string) {
		if sess == nil {
			e.journal.note(route, text, answer)
			return
		}
		sess.Annotate(route, answer)
		if sess.turnText() == "" {
			sess.SetText(text)
		}
		sess.FinishTurn("")
	}
	quiet := func() {
		if sess != nil {
			sess.Mute()
		}
		e.player.Stop(true)
	}
	closeVoice := func() {
		if sess != nil {
			sess.FinishTurn("")
			sess.Close() // отменяет параллельные вызовы инструментов этой голосовой сессии
			e.mu.Lock()
			if e.live == sess {
				e.live = nil
			}
			e.mu.Unlock()
			sess = nil
		}
	}
	reject := func(why string) {
		log.Printf("Джарвис: ложное срабатывание (%s) — молчу", why)
		if sess != nil {
			quiet()
			sess.Annotate("ложное срабатывание", "")
			sess.FinishTurn("")
		}
		e.saveFalse(clip)
	}
	// handle — законченная фраза человека (из быстрой расшифровки или от модели).
	// true — разговор закончен (стоп, команда выполнена, ложное срабатывание).
	savedPos := false
	handle := func(text, from string) bool {
		cmd := canonName(text)
		if byWord && !savedPos && hasName(text) { // имя подтвердилось расшифровкой — настоящий оклик
			savedPos = true
			e.train.Save("pos", clip, "")
		}
		if !verified {
			if !hasName(text) {
				// детектор уверен (≥0.9) — значит, звали, просто распознаватель съел имя: отвечаем
				if !sure {
					reject("в начале нет «Джарвис»: «" + text + "»")
					return true
				}
				log.Printf("Джарвис: имя в расшифровке потерялось («%s»), но детектор уверен — отвечаю", text)
				if fixed, ok := commandAfterLostName(text); ok {
					cmd = fixed
				}
			}
			verified = true
			e.leds.Set(LedListening)
			for _, h := range held {
				_ = e.player.Write(h)
			}
			held = nil
		}
		if isStop(cmd) {
			quiet()
			if isStopAll(cmd) {
				log.Printf("Джарвис: «%s» — стоп (%s)", text, from)
				e.music.Stop()
			} else {
				log.Printf("Джарвис: «%s» — разговор закрыт, музыка не тронута (%s)", text, from)
			}
			note("стоп", text, "")
			return true
		}
		if e.music.hasProposal() && proposalReply(cmd) != "" {
			quiet()
			closeVoice()
			ctx, finish := e.beginMedia()
			defer finish()
			answer, _ := e.music.respondProposal(ctx, cmd)
			note("выбор ролика", text, answer)
			e.speakSelection(ctx, answer)
			return true
		}
		if q, ok := savedResumeQuery(cmd); ok && e.music != nil && (q != "" || !e.music.Active()) {
			quiet()
			closeVoice()
			ctx, finish := e.beginMedia()
			defer finish()
			answer := e.mediaWithChatterFor(ctx, "resume", func() string { return e.music.resumeSaved(ctx, q) })
			note("продолжение ролика", text, answer)
			if ctx.Err() == nil {
				e.sayMusicStatusContext(ctx, answer)
			}
			return true
		}
		if q, ok := youtubeRequest(cmd); ok && e.music != nil {
			quiet()
			closeVoice()
			ctx, finish := e.beginMedia()
			defer finish()
			// «Включаю» — сразу и коротко, пока ищется видео; вслух подробности только при неудаче
			action := "search"
			if recommendationRequest(q) {
				action = "recommendations"
			}
			answer := e.mediaWithChatterFor(ctx, action, func() string { return e.playMusicContext(ctx, q) })
			note("музыка", text, answer)
			if strings.HasPrefix(answer, "Не получилось") && ctx.Err() == nil {
				e.sayMusicStatusContext(ctx, answer)
			}
			if e.music.hasProposal() && ctx.Err() == nil {
				e.speakSelection(ctx, answer)
			}
			return true
		}
		if q, ok := queueRequest(cmd); ok && e.music != nil {
			quiet()
			closeVoice()
			ctx, finish := e.beginMedia()
			defer finish()
			answer := "Добавил в очередь."
			if t, err := e.music.Enqueue(ctx, q); err != nil {
				answer = "Не получилось: " + shortErr(err)
			} else if t.Title != "" {
				answer = "Добавил: " + t.Title + "."
			}
			note("музыка", text, answer)
			if ctx.Err() == nil {
				e.sayMusicStatusContext(ctx, answer)
			}
			return true
		}
		if e.music.Active() {
			if action, _, _, ok := musicCommand(text); ok {
				quiet()
				closeVoice()
				ctx, finish := e.beginMedia()
				defer finish()
				work := func() string { answer, _ := e.controlMusicContext(ctx, text); return answer }
				var answer string
				if action == "next" {
					answer = e.mediaWithChatterFor(ctx, "related", work)
				} else {
					answer = work()
				}
				note("музыка", text, answer)
				if action != "stop" && ctx.Err() == nil {
					if e.music.hasProposal() {
						e.speakSelection(ctx, answer)
					} else {
						e.sayMusicStatusContext(ctx, answer)
					}
				}
				return true
			}
		}
		// быстрый путь для пылесоса в Home Assistant — только если HA включён галочкой и отвечает;
		// иначе фразу разбирает модель (пылесос может быть в Xiaomi Home или Smart Life)
		if h := e.tools.HomeAssistant; e.tools.use("homeassistant") && h.vacuumIntent(cmd) && h.alive() {
			quiet()
			closeVoice()
			ctx, finish := e.beginMedia()
			defer finish()
			answer := e.mediaWithChatterFor(ctx, "home", func() string { return h.vacuumCommand(ctx, cmd) })
			note("Home Assistant", text, answer)
			if ctx.Err() == nil {
				e.sayMusicStatusContext(ctx, answer)
			}
			return true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		t0 := time.Now()
		// таймеры, будильники, напоминания, музыка, громкость — умеет только штатная Алиса; команды умного дома
		// отдаём ей, если так выбрано в настройках
		if sl := e.tools.SmartLife; sl.ready() && e.store.Get().UseHome("smartlife") {
			if ok, answer := sl.Fast(ctx, cmd); ok {
				quiet()
				log.Printf("Джарвис: «%s» → Smart Life (%.2f с, %s)", text, time.Since(t0).Seconds(), from)
				note("Smart Life", text, answer)
				if e.player.Start(MicRate) == nil {
					_ = e.player.Write(earcon(answer == "Готово."))
					e.player.Finish(2 * time.Second)
				}
				return true
			}
		}
		alice, toAlice := AliceSkillFor(cmd, e.store.Get().UseHome("yandexmusic"))
		if !toAlice && e.store.Get().UseHome("alice") {
			toAlice, alice = e.tools.HomeCommand(ctx, cmd)
		}
		if toAlice {
			quiet()
			err := AliceText(alice, false) // Алиса выполнит и ответит сама, как обычно
			log.Printf("Джарвис: «%s» → Алисе (%s, %.2f с)", alice, from, time.Since(t0).Seconds())
			if err == nil {
				note("умный дом → Алиса", text, "передал Алисе: «"+alice+"»")
				return true
			}
			log.Printf("Алиса: %v — выполняю сам", err)
		}
		if ok, answer := e.tools.Fast(ctx, cmd); ok && e.store.Get().UseHome("yandex") {
			quiet()
			log.Printf("Джарвис: быстрая команда «%s» → %s (Яндекс %.2f с, %s)", text, answer, time.Since(t0).Seconds(), from)
			note("умный дом (быстро)", text, answer)
			if e.player.Start(MicRate) == nil {
				_ = e.player.Write(earcon(answer == "Готово."))
				e.player.Finish(2 * time.Second)
			}
			return true
		}
		return false
	}

	if byWord && !sure && quick != nil {
		log.Printf("Джарвис: сначала проверяю имя по расшифровке — разговор с моделью пока не открываю")
	} else if !open() { // кнопка, уверенное срабатывание или быстрой расшифровки нет — открываем сразу
		return
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-interruptions:
			e.player.Stop(true)
			speaking, held = false, nil
			e.leds.Set(LedListening)
		case text := <-quick:
			if strings.TrimSpace(text) == "" {
				continue
			}
			heard = true
			if handle(text, "быстрая расшифровка") {
				return
			}
			if sess == nil && !open() { // имя есть, и это не команда — вопрос к модели
				return
			}
		case pcm, ok := <-out:
			if !ok {
				e.player.Stop(true)
				sess.FinishTurn("")
				return
			}
			if sess.Muted() {
				continue
			}
			if !verified { // пока не убедились, что звали, — копим ответ, вслух не говорим
				if len(held) < 400 {
					held = append(held, pcm)
				}
				continue
			}
			if !e.player.Playing() && loud(pcm) { // после «Джарвис»-перебивания плеер был остановлен
				_ = e.player.StartVoice(sess)
			}
			_ = e.player.Write(pcm)
			if loud(pcm) {
				lastVoice = time.Now()
			}
			e.leds.Level(peakOf(pcm) / 9000)
			if loud(pcm) != speaking {
				speaking = !speaking
				e.leds.Set(map[bool]LedMode{true: LedSpeaking, false: LedListening}[speaking])
			}
		case <-tick.C:
			e.mu.Lock()
			stopped := e.aborted || (sess != nil && e.live != sess) // остановили (Алиса, кнопка)
			musicDone := e.musicDone
			e.mu.Unlock()
			// музыка включена — дослушиваем «Включаю» и закрываем разговор, микрофон больше не слушает
			if sess != nil && !musicDone.IsZero() {
				since := time.Since(musicDone)
				saidIt := lastVoice.After(musicDone) && time.Since(lastVoice) > 900*time.Millisecond
				if saidIt || (since > 3*time.Second && lastVoice.Before(musicDone)) || since > 10*time.Second {
					log.Printf("Джарвис: музыка включена — разговор окончен")
					sess.Mute() // дальше модель не слушаем и не озвучиваем
					e.player.Finish(3 * time.Second)
					sess.FinishTurn("")
					return
				}
			}
			if stopped {
				e.player.Stop(true)
				if sess != nil {
					sess.FinishTurn("")
				}
				return
			}
			if sess == nil { // ждём первую фразу: имя проверит быстрая расшифровка
				waited := time.Since(start)
				if waited >= time.Second && !e.stt.Speaking() {
					log.Printf("Джарвис: быстрая расшифровка задержалась — проверяю оклик голосовым движком")
					if !open() {
						return
					}
					continue
				}
				if (waited > 4500*time.Millisecond && !e.stt.Speaking()) || waited > 15*time.Second {
					reject("никто ничего не сказал")
					return
				}
				continue
			}
			// фраза человека закончилась (расшифровка не растёт 0.35 с): «стоп» или команда умного дома — сами, без модели
			if !verified && time.Since(readyAt) > 4500*time.Millisecond {
				reject("никто ничего не сказал")
				return
			}
			// разговор открыт сразу по слову, а человек так ничего и не сказал — слово сработало зря
			if byWord && checked == 0 && !heard && sess.UserTurns() == 0 && time.Since(readyAt) > 5*time.Second &&
				(e.stt == nil || !e.stt.Speaking()) && !lastVoice.After(readyAt) {
				reject("никто ничего не сказал")
				return
			}
			if id, text, since := sess.Utterance(); id != 0 && id != checked && since >= 350*time.Millisecond {
				checked = id
				if handle(text, "модель") {
					return
				}
			}
			if sess.Idle(idle) {
				log.Printf("разговор: тишина %.0f с — окончен", idle)
				e.player.Finish(3 * time.Second)
				sess.FinishTurn("")
				return
			}
		}
	}
}

func peakOf(pcm []int16) float64 {
	p := 0
	for _, v := range pcm {
		if v < 0 {
			v = -v
		}
		if int(v) > p {
			p = int(v)
		}
	}
	return float64(p)
}

func loud(pcm []int16) bool {
	for _, v := range pcm {
		if v > 600 || v < -600 {
			return true
		}
	}
	return false
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}

func (e *Engine) fail(msg string) {
	t := e.journal.New("Джарвис", "")
	e.journal.Update(t, func(t *Turn) { t.Error = msg })
	log.Print(msg)
}

// RoomActive — в комнате недавно что-то звучало (речь, шаги, телевизор), и Джарвис сейчас свободен.
// Только тогда есть смысл держать наготове соединение с моделью: в тихой комнате звать некому.
func (e *Engine) RoomActive() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return !e.busy && e.live == nil && time.Since(e.lastLoud) < 20*time.Second
}

// playMusic — включить с YouTube; ответ — что включилось (для переписки и для модели).
func (e *Engine) playMusic(query string) string {
	return e.playMusicContext(context.Background(), query)
}

func (e *Engine) playMusicContext(parent context.Context, query string) string {
	if answer, ok := e.resumeSavedRequest(parent, query); ok {
		return answer
	}
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	if answer, ok := e.music.respondProposal(ctx, query); ok {
		return answer
	}
	if recommendationRequest(query) {
		return e.music.propose(ctx, query)
	}
	t, err := e.music.Play(ctx, query)
	if err != nil {
		log.Printf("музыка: «%s»: %v", query, err)
		return "Не получилось включить: " + shortErr(err)
	}
	if recommendationRequest(query) {
		e.music.mu.Lock()
		searched := e.music.personalSearch
		e.music.mu.Unlock()
		if searched {
			return "Нашёл в YouTube с твоим входом: " + t.Title + " — " + t.Author
		}
		return "Включил из твоих рекомендаций YouTube: " + t.Title + " — " + t.Author
	}
	return "Включил с YouTube: " + t.Title + " — " + t.Author
}

// Listening — только что позвали и слушают команду (первые секунды после «Джарвис»): тогда музыку
// приглушаем, чтобы команду было слышно. Дальше — только пока Джарвис сам говорит.
func (e *Engine) Listening() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return (e.busy || e.live != nil) && time.Since(e.lastWake) < 6*time.Second
}

// Talking — идёт ли разговор (или он открывается).
func (e *Engine) Talking() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.busy || e.live != nil
}

// interrupt — «Джарвис» посреди ответа: замолчать сразу, слушать новую фразу.
func (e *Engine) interrupt(sess Session) {
	log.Printf("Джарвис: перебили словом — замолкаю")
	sess.Mute()
	e.player.Stop(true)
	e.leds.Set(LedListening)
}

// Stop — прервать разговор и музыку (Алиса, кнопка «Стоп» в настройках, «стоп»).
func (e *Engine) Stop(reason string) {
	e.music.Stop()
	e.EndChat(reason)
}

// EndChat — закончить только разговор («хватит», «закрой чат»): музыка, если играет, играет дальше.
func (e *Engine) EndChat(reason string) {
	e.mu.Lock()
	if e.opCancel != nil {
		e.opCancel()
	}
	sess := e.live
	e.live = nil
	// «остановлено» — только для того, что идёт сейчас; раньше флаг оставался навсегда, и любая следующая
	// озвучка (прослушать голос, ответ из переписки) обрывалась через 0,1 с
	if e.busy || sess != nil {
		e.aborted = true
	}
	if sess == nil && e.busy {
		e.aborted = true // разговор ещё не открыт (идёт проверка имени) — просто не открывать
		log.Printf("Джарвис: остановлен до начала разговора (%s)", reason)
	}
	e.mu.Unlock()
	if sess != nil {
		log.Printf("Джарвис: разговор прерван (%s)", reason)
		e.player.Stop(true)
		sess.Close()
	}
}

// watchAlice — следим за журналом maind: штатный споттер «Алиса» = Джарвис уступает.
func (e *Engine) watchAlice(ctx context.Context) {
	const path = "/data/quasar/daemons_logs/maind.log"
	var f *os.File
	var ino uint64
	var rd *bufio.Reader
	for ctx.Err() == nil {
		time.Sleep(150 * time.Millisecond)
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		cur := inode(st)
		if f == nil || cur != ino {
			if f != nil {
				f.Close()
			}
			if f, err = os.Open(path); err != nil {
				f = nil
				continue
			}
			if ino == 0 {
				_, _ = f.Seek(0, io.SeekEnd) // старые активации не переигрываем
			}
			ino = cur
			rd = bufio.NewReaderSize(f, 8192)
		}
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				if len(line) > 0 {
					_, _ = f.Seek(-int64(len(line)), io.SeekCurrent)
					rd.Reset(f)
				}
				break
			}
			if strings.Contains(line, "onPhraseSpotted") && strings.Contains(line, "alice_capability") {
				e.Stop("позвали Алису")
			}
			if strings.Contains(line, "Toggle play/pause") { // кнопка «пауза» на колонке (короткое нажатие)
				e.playPauseButton()
			}
			if e.alice != nil {
				e.alice.onMain(line)
			}
		}
	}
}

// Кнопку «пауза» на колонке видит и Алиса: она ставит на паузу или продолжает СВОЮ музыку. Если играет
// YouTube Джарвиса — пауза ему; если Алиса при этом продолжила свой старый трек, сразу ставим её на паузу.
var buttonAt atomic.Int64 // когда кнопкой управляли музыкой Джарвиса (UnixNano)

func (e *Engine) playPauseButton() {
	m, a := e.music, e.alice
	switch {
	case m.Playing():
		buttonAt.Store(time.Now().UnixNano())
		m.Pause()
		ytButtonPaused.Store(true)
		log.Printf("кнопка: пауза YouTube")
	case !m.PausedSince().IsZero() && !a.Playing() && (ytButtonPaused.Load() || m.PausedSince().After(a.PausedSince())):
		// YouTube остановили кнопкой — следующее нажатие продолжает его, даже если потом мы гасили
		// «продолжить» Алисы (её пауза тогда новее нашей, но это не выбор человека)
		// на паузе оба — продолжаем то, что остановили последним
		buttonAt.Store(time.Now().UnixNano())
		if err := m.Resume(); err != nil {
			log.Printf("кнопка: продолжить YouTube: %v", err)
			return
		}
		ytButtonPaused.Store(false)
		log.Printf("кнопка: продолжаю YouTube")
	}
}

// ytButtonPaused — YouTube стоит на паузе после кнопки колонки (сбрасывается, когда Алису попросили о её музыке).
var ytButtonPaused atomic.Bool

// aliceStarted — у Алисы зазвучала музыка. После нажатия кнопки это её «продолжить» поверх YouTube — гасим;
// иначе её попросили включить — значит, YouTube на паузу.
func (e *Engine) aliceStarted() {
	if time.Since(time.Unix(0, buttonAt.Load())) < 4*time.Second {
		if err := AliceText("пауза", true); err == nil {
			log.Printf("кнопка: Алиса тоже включила свою музыку — поставил её на паузу")
		}
		return
	}
	ytButtonPaused.Store(false) // человек сам выбрал музыку Алисы
	if e.music.Playing() {
		e.music.Pause()
		log.Printf("музыка: Алиса включила свою — YouTube на паузу")
	}
}

// youtubeStarted — YouTube Джарвиса зазвучал: музыку Алисы — на паузу, чтобы не играли одновременно.
var lastFactTrack atomic.Value // id видео, о котором уже записали в чат

func (e *Engine) youtubeStarted() {
	e.chatgptFact()
	if !e.alice.Playing() {
		return
	}
	if err := AliceText("пауза", true); err == nil {
		log.Printf("музыка: у Алисы играла своя — поставил её на паузу")
	}
}

// chatgptFact — включилось новое видео: записать в общий чат ChatGPT, чтобы он знал, что играет.
func (e *Engine) chatgptFact() {
	if chatgptChat == nil {
		return
	}
	e.music.mu.Lock()
	id, title, author := e.music.cur.ID, e.music.cur.Title, e.music.cur.Author
	e.music.mu.Unlock()
	if id == "" || title == "" || lastFactTrack.Load() == id {
		return
	}
	lastFactTrack.Store(id)
	chatgptChat.AddFact("включил на YouTube: «" + title + "» — " + author)
}
