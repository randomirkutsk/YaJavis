// jarvisd — голосовой ассистент «Джарвис» целиком на Яндекс Станции: без своего сервера.
// Один файл: слово активации, GPT-Live, умный дом, VLESS-туннель и веб-настройки.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/thesyncim/gopus"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) > 2 && os.Args[1] == "add-eleven-options" {
		if err := addElevenOptions(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "probe-eleven-new" {
		if err := probeElevenNew(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-eleven-models" {
		if err := probeElevenModels(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-classic" {
		if err := probeClassic(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "configure-classic" {
		if err := configureClassic(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-elevenlabs" {
		if err := elevenCLI(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-gemini-interruption" {
		if err := probeGeminiInterruption(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "configure-home-assistant" || os.Args[1] == "probe-home-assistant") {
		if err := homeAssistantCLI(os.Args[1] == "configure-home-assistant"); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-gemini-models" {
		path := "/data/opt/jarvisd/settings.json"
		if len(os.Args) > 2 {
			path = os.Args[2]
		}
		probeGeminiModels(path)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-gemini-config" {
		path := "/data/opt/jarvisd/settings.json"
		if len(os.Args) > 2 {
			path = os.Args[2]
		}
		store, err := OpenStore(path)
		if err != nil {
			log.Fatal(err)
		}
		s := store.Get()
		p := s.ActiveProfile()
		proxy := &Proxy{}
		if err := proxy.Apply(s.Vless); err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		probeTools := NewTools(func() string { return "" }, DirectClient)
		probeTools.HomeAssistant = &HomeAssistant{config: func() HASettings { return s.HomeAssistant }, client: DirectClient}
		sess := NewGeminiSession(ctx, s.GeminiKey, p, russianSessionRules, probeTools, proxy.LiveClient(), &Journal{})
		defer sess.Close()
		if err := sess.Ready(12 * time.Second); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Gemini: русская конфигурация принята, звук не включался")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-recommendations" {
		probeRecommendations()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "patch-launch" {
		if len(os.Args) != 6 {
			log.Fatal("usage: jarvisd patch-launch <launch.json> <out.json> <vqe_tap.so> <mic.sock>")
		}
		if err := patchLaunch(os.Args[2], os.Args[3], os.Args[4], os.Args[5]); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "volume" { // проверка: jarvisd volume 3 — поставить громкость Алисы шагами
		var lvl int
		if _, err := fmt.Sscan(os.Args[2], &lvl); err != nil {
			log.Fatal("jarvisd volume 0..10")
		}
		show := func(what string) {
			v, stamp := aliceVolumeLog()
			fmt.Printf("%s %.1f (журнал Алисы %.1f от %s, Master %d%%)\n", what, currentVolume(), v, stamp, masterPercent())
		}
		show("было")
		if err := setAliceVolume(lvl); err != nil {
			log.Fatal(err)
		}
		for i := 0; i < 150; i++ { // шагает фоновый обработчик — ждём, пока дойдёт
			time.Sleep(100 * time.Millisecond)
			volMu.Lock()
			busy := volBusy
			volMu.Unlock()
			if !busy {
				break
			}
		}
		show("стало")
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "alice-text" { // проверка: jarvisd alice-text "сколько времени"
		if err := AliceText(os.Args[2], false); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "ask" { // проверка на колонке: jarvisd ask "вопрос" — текстовый ответ модели
		store, err := OpenStore("/data/opt/jarvisd/settings.json")
		if err != nil {
			log.Fatal(err)
		}
		st := store.Get()
		proxy := &Proxy{}
		if err := proxy.Apply(st.Vless); err != nil {
			log.Fatal(err)
		}
		tools := NewTools(func() string { return st.YandexToken }, DirectClient)
		tools.now = func() time.Time { return st.Now() }
		t0 := time.Now()
		p := st.ActiveProfile()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		answer, err := askOpenAI(ctx, proxy.Client(60*time.Second), st.OpenAIKey, p, st.PromptText(p.Prompt)+textRules+"\n\n"+backendRules+tools.Context(), os.Args[2], tools)
		fmt.Printf("%.1f с | %s | ошибка: %v\n", time.Since(t0).Seconds(), answer, err)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-live" { // замер: открытие GPT-Live с нуля и с готового соединения
		store, err := OpenStore("/data/opt/jarvisd/settings.json")
		if err != nil {
			log.Fatal(err)
		}
		st := store.Get()
		proxy := &Proxy{}
		if err := proxy.Apply(st.Vless); err != nil {
			log.Fatal(err)
		}
		open := func(label string) {
			t0 := time.Now()
			sess := NewLiveSession(context.Background(), st.OpenAIKey, st.ActiveProfile(), "Молчи.", "", NewTools(func() string { return "" }, DirectClient), proxy.LiveClient(), &Journal{})
			err := sess.Ready(12 * time.Second)
			fmt.Printf("%s: %.2f с, ошибка: %v\n", label, time.Since(t0).Seconds(), err)
			sess.Close()
		}
		open("с нуля")
		for _, wait := range []time.Duration{time.Second, 4 * time.Second, 8 * time.Second} {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			td := time.Now()
			c, err := proxy.dialTLS(ctx, "api.openai.com:443")
			cancel()
			fmt.Printf("само соединение (туннель + TLS): %.2f с\n", time.Since(td).Seconds())
			if err != nil {
				log.Fatal(err)
			}
			proxy.warm = map[string]*warmConn{"api.openai.com:443": {c: c, at: time.Now()}}
			time.Sleep(wait)
			open(fmt.Sprintf("готовое соединение, ждало %v", wait))
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-chatgpt" { // пробный звонок GPT-Live по подписке: открывается ли и звучит ли
		store, err := OpenStore("/data/opt/jarvisd/settings.json")
		if err != nil {
			log.Fatal(err)
		}
		st := store.Get()
		proxy := &Proxy{}
		if err := proxy.Apply(st.Vless); err != nil {
			log.Fatal(err)
		}
		// токен не продлеваем: этим занимается работающий jarvisd, иначе вход у него устареет
		auth := func(context.Context) (ChatGPTAuth, error) {
			if !st.ChatGPT.SignedIn() || time.Until(time.Unix(st.ChatGPT.Expires, 0)) < time.Minute {
				return st.ChatGPT, errors.New("нет действующего входа в ChatGPT")
			}
			return st.ChatGPT, nil
		}
		p := st.ActiveProfile()
		t0 := time.Now()
		sess := NewChatGPTSession(context.Background(), auth, p, "Ты Джарвис.", proxy.LiveClient(),
			func(context.Context, string) (string, error) { return "Всё работает.", nil }, &Journal{})
		err = sess.Ready(25 * time.Second)
		fmt.Printf("звонок: %.2f с, ошибка: %v\n", time.Since(t0).Seconds(), err)
		if err == nil {
			_ = sess.Say("Скажи по-русски: проверка связи, всё работает.")
			got, loudN := 0, 0
			deadline := time.After(12 * time.Second)
		loop:
			for {
				select {
				case pcm, ok := <-sess.Audio():
					if !ok {
						break loop
					}
					got += len(pcm)
					if loud(pcm) {
						loudN++
					}
				case <-deadline:
					break loop
				}
			}
			fmt.Printf("голос: %.1f с звука, громких кусков %d\n", float64(got)/MicRate, loudN)
		}
		sess.Close()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-chatgpt-who" { // спросить голос подписки «кто ты?» текстом
		runChatGPTWhoProbe(os.Args[2:], "/data/opt/jarvisd")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-chatgpt-mic" { // N звонков ChatGPT по подписке с записанной фразой
		runChatGPTMicProbe(os.Args[2:], "/data/opt/jarvisd")
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "probe-music" { // полный трек и перемотка, без звука в динамике
		debug.SetGCPercent(50)
		debug.SetMemoryLimit(40 << 20)
		client := &http.Client{Timeout: 60 * time.Second, Transport: DirectClient.Transport}
		m := NewMusic(func() *http.Client { return client }, func() float64 { return 1 })
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		mem := func() {
			if b, err := os.ReadFile("/proc/self/status"); err == nil {
				for _, l := range strings.Split(string(b), "\n") {
					if strings.HasPrefix(l, "VmHWM") || strings.HasPrefix(l, "VmRSS") {
						fmt.Println(l)
					}
				}
			}
		}
		t0 := time.Now()
		tracks, err := m.search(ctx, strings.Join(os.Args[2:], " "))
		if err != nil || len(tracks) == 0 {
			log.Fatalf("поиск: %v", err)
		}
		// как Play: видео без Opus пропускаем, берём следующее
		var st *ytStream
		var info Track
		for len(tracks) > 0 {
			st, info, err = m.audioURL(ctx, tracks[0].ID)
			if err == nil {
				break
			}
			fmt.Printf("«%s»: %v — следующее\n", tracks[0].Title, err)
			tracks = tracks[1:]
		}
		if err != nil || len(tracks) == 0 {
			log.Fatalf("ссылка: %v", err)
		}
		fmt.Printf("нашлось «%s» — %s (%s) за %.2f с, клиент %s, %.0f с, ждать рекламу %.1f с\n", tracks[0].Title, tracks[0].Author, info.Author,
			time.Since(t0).Seconds(), st.client, st.dur, time.Until(st.ready).Seconds())
		mem()
		src, w, err := m.openAudio(ctx, tracks[0].ID, st, 0)
		if err != nil {
			log.Fatal(err)
		}
		defer src.Close()
		dec, _ := gopus.NewDecoder(gopus.DefaultDecoderConfig(musicRate, 1))
		pcm := make([]int16, musicRate/8)
		samples := 0
		t1 := time.Now()
		var cpu time.Duration
		for {
			f, _, err := w.next()
			if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			if err != nil {
				log.Fatalf("после %d отсчётов: %v", samples, err)
			}
			tc := time.Now()
			n, err := dec.DecodeInt16(f, pcm)
			if err != nil {
				log.Fatalf("декодирование: %v", err)
			}
			cpu += time.Since(tc)
			samples += n
		}
		sec := float64(samples) / musicRate
		if sec+2 < st.dur {
			log.Fatalf("трек оборвался: %.1f с из %.1f с", sec, st.dur)
		}
		fmt.Printf("%.0f с звука (весь трек): скачано и разобрано за %.2f с, декодер %.1f%% одного ядра\n", sec, time.Since(t1).Seconds(), 100*cpu.Seconds()/max(sec, 1))
		for _, target := range []float64{min(120, st.dur/2), 15} {
			tSeek := time.Now()
			src2, w2, err := m.openAudio(ctx, tracks[0].ID, st, target)
			if err != nil {
				log.Fatal(err)
			}
			frames := &musicFrames{src: src2, w: w2, target: int64(target * 1000)}
			for {
				f, ms, err := frames.next()
				if err != nil {
					log.Fatalf("перемотка: %v", err)
				}
				if _, err := dec.DecodeInt16(f, pcm); err != nil {
					log.Fatal(err)
				}
				if ms >= frames.target {
					if ms-frames.target > 100 {
						log.Fatalf("перемотка промахнулась: %d вместо %d мс", ms, frames.target)
					}
					fmt.Printf("перемотка на %.2f с → %.2f с за %.2f с\n", target, float64(ms)/1000, time.Since(tSeek).Seconds())
					break
				}
			}
			_ = src2.Close()
		}
		mem()
		return
	}
	if len(os.Args) > 3 && os.Args[1] == "bench-oww" { // замер: сколько процессора ест новое слово активации
		feat, _ := os.ReadFile(os.Args[2])
		head, _ := os.ReadFile(os.Args[3])
		o, err := LoadOWW(feat, head, func() int { return 5 })
		if err != nil {
			log.Fatal(err)
		}
		if tvPath := os.Getenv("OWW_TESTVEC"); tvPath != "" { // сверка с обучением на Python
			var tv struct {
				PCM   []int     `json:"pcm"`
				Probs []float64 `json:"probs"`
			}
			raw, _ := os.ReadFile(tvPath)
			_ = json.Unmarshal(raw, &tv)
			var got []float64
			for i := 0; i+owwStep <= len(tv.PCM); i += owwStep {
				chunk := make([]int16, owwStep)
				for j := range chunk {
					chunk[j] = int16(tv.PCM[i+j])
				}
				before, last := len(o.ring), o.LastProb
				o.Feed(chunk)
				if len(o.ring) == o.seq && (before == o.seq || o.LastProb != last) {
					got = append(got, float64(o.LastProb))
				}
			}
			worst := 0.0
			n := min(len(got), len(tv.Probs))
			for i := 0; i < n; i++ {
				worst = max(worst, math.Abs(got[len(got)-n+i]-tv.Probs[len(tv.Probs)-n+i]))
			}
			fmt.Printf("сверка: Go %d, Python %d, наибольшее расхождение %.5f"+"\n", len(got), len(tv.Probs), worst)
			o, _ = LoadOWW(feat, head, func() int { return 5 })
		}
		va, vb := make([]float32, 300), make([]float32, 300)
		for i := range va {
			va[i], vb[i] = float32(i%7)-3, float32(i%5)*0.5-1
		}
		for _, n := range []int{16, 48, 72, 288, 300} {
			fmt.Printf("dot %d: NEON %v, Go %v\n", n, dot(va[:n], vb[:n]), dotGo(va[:n], vb[:n]))
		}
		pcm := make([]int16, MicRate*30)
		seed := uint32(1)
		for i := range pcm {
			seed = seed*1664525 + 1013904223
			pcm[i] = int16(int32(seed>>16)%3000 - 1500)
		}
		if len(os.Args) > 4 {
			f, _ := os.Create(os.Args[4])
			_ = pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		t0 := time.Now()
		for i := 0; i+160 <= len(pcm); i += 160 {
			o.Feed(pcm[i : i+160])
		}
		el := time.Since(t0)
		fmt.Printf("30 с звука за %v: %.1f%% одного ядра\n", el, 100*el.Seconds()/30)
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "set-key" { // ключ со стандартного ввода — чтобы он не попадал в историю команд
		raw, _ := io.ReadAll(io.LimitReader(os.Stdin, 4096))
		key := strings.TrimSpace(string(raw))
		store, err := OpenStore("/data/opt/jarvisd/settings.json")
		if err != nil {
			log.Fatal(err)
		}
		err = store.Update(func(s *Settings) error {
			switch os.Args[2] {
			case "grok":
				s.GrokKey = key
			case "gemini":
				s.GeminiKey = key
			case "openai":
				s.OpenAIKey = key
			default:
				return errors.New("неизвестный ключ")
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ключ %s сохранён (%d символов)\n", os.Args[2], len(key))
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "probe-grok" { // проверка Grok: модели и пробный разговор
		store, err := OpenStore("/data/opt/jarvisd/settings.json")
		if err != nil {
			log.Fatal(err)
		}
		st := store.Get()
		proxy := &Proxy{}
		if err := proxy.Apply(st.Vless); err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		ids, err := grokModels(ctx, proxy.Client(20*time.Second), st.GrokKey)
		cancel()
		fmt.Printf("модели: %v, ошибка: %v\n", ids, err)
		p := st.ActiveProfile()
		if len(os.Args) > 2 {
			p.Voice = os.Args[2]
		}
		t0 := time.Now()
		sess := NewGrokSession(context.Background(), st.GrokKey, p, "Ты Джарвис.", NewTools(func() string { return "" }, DirectClient), proxy.LiveClient(), &Journal{})
		err = sess.Ready(15 * time.Second)
		fmt.Printf("сессия: %.2f с, ошибка: %v\n", time.Since(t0).Seconds(), err)
		if err == nil {
			_ = sess.Say("Привет, это проверка связи.")
			got, loudN := 0, 0
			deadline := time.After(12 * time.Second)
		gloop:
			for {
				select {
				case pcm, ok := <-sess.Audio():
					if !ok {
						break gloop
					}
					got += len(pcm)
					if loud(pcm) {
						loudN++
					}
				case <-deadline:
					break gloop
				}
			}
			fmt.Printf("голос: %.1f с, громких кусков %d\n", float64(got)/grokRate, loudN)
		}
		sess.Close()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}
	dataDir := flag.String("data", "/data/opt/jarvisd", "папка настроек")
	micSock := flag.String("mic", "/tmp/jarvisd/mic.sock", "сокет микрофона (vqe_tap.so)")
	listen := flag.String("listen", ":8080", "адрес веб-настроек")
	uiDir := flag.String("ui", "", "для разработки: папка интерфейса на диске")
	flag.Parse()

	// памяти у колонки мало (Лайт — 238 МБ на всё): держим кучу Go компактной
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(48 << 20)
	logTZ.out = os.Stderr
	log.SetFlags(0)
	log.SetOutput(logTZ) // время в журнале — по часовому поясу из настроек колонки
	log.Printf("jarvisd %s", version)
	go ensureStationIntegrations(*dataDir)

	store, err := OpenStore(filepath.Join(*dataDir, "settings.json"))
	if err != nil {
		log.Fatalf("настройки: %v", err)
	}
	proxy := &Proxy{}
	proxy.use = func(svc string) bool { s := store.Get(); return s.UseVPN(svc) }
	logTZ.set(store.Get().Timezone)
	if err := proxy.Apply(store.Get().Vless); err != nil {
		log.Printf("VLESS: %v (работаю напрямую)", err)
	}
	journal := &Journal{}
	chatgptChat = LoadChatThread(filepath.Join(*dataDir, "chatgpt-chat.json"))
	loadGeminiSpent(filepath.Join(*dataDir, "gemini-spent.json"))
	geminiKeyList = func() []string { return store.Get().GeminiKeys }
	journal.OnRemember = func(t Turn) {
		switch {
		case t.Model != "": // разговор с моделью — в общую память, какой бы ассистент ни отвечал
			chatgptChat.Add(t.Text, t.Answer)
		case t.Answer != "" && memoryWorthy(t.Answer): // выполнено без модели (умный дом, Алиса, музыка) — модель тоже должна знать
			chatgptChat.Add(t.Text, "["+t.Answer+"]")
		}
	}
	tools := NewTools(func() string { return store.Get().YandexToken }, &http.Client{Timeout: 10 * time.Second, Transport: proxy.TransportFor("yandex")})
	tools.HomeAssistant = &HomeAssistant{config: func() HASettings { return store.Get().HomeAssistant }, client: &http.Client{Timeout: 15 * time.Second}}
	tools.now = func() time.Time { s := store.Get(); return s.Now() }
	tools.Xiaomi = &Xiaomi{store: store, dataDir: *dataDir, client: func() *http.Client { return proxy.ClientFor("xiaomi", 0) }}
	tuyaLAN := &TuyaLAN{}
	tools.SmartLife = &SmartLife{store: store, dataDir: *dataDir, lan: tuyaLAN, client: func() *http.Client { return proxy.ClientFor("smartlife", 30*time.Second) }}
	tools.Use = func(ch string) bool { s := store.Get(); return s.UseHome(ch) }
	tools.Room = func() string { return store.Get().SpeakerRoom }
	go tuyaLAN.Listen(func() bool { return store.Get().Tuya.SignedIn() })
	mic := NewMic(*micSock)
	mic.GainDB = func() int { return store.Flags().MicGainDB }
	go mic.Run()
	go WatchAliceMic(mic) // Midi/Mini: Алиса стартует раньше S98jarvisd — без перезапуска микрофона нет
	player := NewPlayer()
	player.Volume = func() int { s := store.Get(); return s.VoiceVolume() }
	player.Trim = func() float64 { return voiceTrim(speakerLevel()) }
	speechLang = func() string { return store.Get().Language }
	leds := NewLeds()
	leds.Source = func() LedSettings { s := store.Get(); return s.LedFor() }
	{
		s := store.Get()
		leds.Configure(s.LedFor())
	}
	memory := NewMemoryStore(filepath.Join(*dataDir, "memory.json"))
	tools.Memory = memory
	engine := &Engine{store: store, proxy: proxy, mic: mic, player: player, leds: leds, tools: tools,
		journal: journal, wake: LoadWake(*dataDir, store), dataDir: *dataDir, train: &Trainer{dataDir: *dataDir, store: store},
		bg: &BgRecorder{store: store, client: func() *http.Client { return proxy.ClientFor("gdrive", 0) },
			drive: &GDrive{store: store, client: func() *http.Client { return proxy.ClientFor("gdrive", 0) }}},
		memory: memory}
	telegram := NewTelegramBot(store, proxy, engine, memory)
	tools.Telegram = telegram
	engine.telegram = telegram
	go telegram.Start()
	engine.stt = NewSTT(func() string { return store.Get().OpenAIKey }, func() *http.Client { return proxy.ClientFor("openai", 0) })
	engine.cmds = NewCmdSpotter(*dataDir, engine.onCommand)
	voiceTestHook = engine.VoiceTest
	engine.train.cloud = engine.bg.drive // старые записи для обучения — на тот же Google Диск, что и фон
	// музыка с YouTube: «Джарвис, включи с ютуба …»
	musicClient := &http.Client{Timeout: 60 * time.Second, Transport: proxy.TransportFor("youtube")}
	{
		engine.music = NewMusic(func() *http.Client { return musicClient }, func() float64 {
			v := 0.8                                  // громкость музыки — это громкость самой колонки (ставится на главной или голосом Алисе)
			if engine.Talking() || player.Playing() { // идёт разговор — музыка тихо фоном, как у Алисы
				v *= 0.2
			}
			return v
		})
		engine.music.Radio = func() bool { return !store.Get().RadioOff }
		engine.music.progress = openPlaybackHistory(filepath.Join(*dataDir, "youtube-progress.json"))
		go engine.music.watchProgress(context.Background())
		engine.music.OnRelatedEnd = engine.offerNextAfterEnd
		engine.music.CookieFile = filepath.Join(*dataDir, "youtube_cookies.txt")
		engine.music.Account = NewYTAccount(store, func() *http.Client { return proxy.ClientFor("youtube", 25*time.Second) })
		// «Джарвис, включи …» первой фразой — команда: «Включаю», и разговор закрывается.
		// Попросили посреди беседы — музыка тихо играет фоном, а разговор продолжается.
		tools.Music = func(ctx context.Context, query string) string {
			inDialog := engine.InDialog()
			a := engine.playMusicContext(ctx, query)
			if engine.music.hasProposal() {
				return a + " Ролик НЕ включён. Коротко опиши предложение и дождись «да» или отказа; ответ передай в music_control. Не добавляй ничего в очередь."
			}
			if !strings.HasPrefix(a, "Не получилось") && engine.music.Playing() {
				if inDialog {
					a += ". Уже играет тихо фоном, разговор продолжается: коротко и по-живому скажи, что включил, и продолжай беседу как обычно."
				} else {
					a += ". Уже включается — скажи вслух только «Включаю» и больше ничего."
					engine.EndAfterMusic()
				}
			}
			return a
		}
		go func() {
			time.Sleep(45 * time.Second) // после запуска колонка и так занята — разбираем плеер чуть позже
			engine.music.Warm(context.Background())
		}()
		tools.MusicQueue = func(ctx context.Context, query string) string {
			t, err := engine.music.Enqueue(ctx, query)
			if err != nil {
				return "Не получилось: " + shortErr(err)
			}
			if !engine.InDialog() {
				engine.EndAfterMusic()
			}
			return "Добавлено в очередь: «" + t.Title + "». Скажи об этом одной короткой фразой."
		}
		tools.MusicCtl = func(ctx context.Context, command string) string {
			if answer, ok := engine.resumeSavedRequest(ctx, command); ok {
				return answer
			}
			if !engine.music.Active() && !engine.music.hasProposal() {
				return "Сейчас с YouTube ничего не играет."
			}
			if answer, ok := engine.controlMusicContext(ctx, command); ok {
				if engine.music.hasProposal() {
					return answer + " Это только предложение, ролик НЕ запущен. Коротко объясни, что предлагаешь, и дождись подтверждения."
				}
				if !engine.InDialog() && !engine.music.hasProposal() {
					engine.EndAfterMusic()
				}
				return answer + " Скажи об этом одним-двумя словами."
			}
			return "Не понял команду для музыки: «" + command + "»."
		}
	}
	elevenQuotaHook = func(key, why string) { restElevenKey(store, proxy.ClientFor("elevenlabs", 20*time.Second), key, why) }
	go watchElevenKeys(store)
	go memGuard(func() bool { return !engine.Talking() && !engine.music.Playing() })
	go WatchNight(store, leds)
	go WarmChatGPTLoop(func() bool { s := store.Get(); return s.ActiveProfile().Engine == "chatgpt" }) // звонок по подписке — на готовом соединении
	yandexClient := &http.Client{Timeout: 10 * time.Second, Transport: proxy.TransportFor("yandex")}
	engine.alice = NewAliceMusic(func() *http.Client { return yandexClient })
	engine.alice.OnStart = engine.aliceStarted
	engine.music.mu.Lock()
	engine.music.OnPlay = engine.youtubeStarted
	engine.music.mu.Unlock()
	go engine.alice.Run(context.Background())
	go engine.Run(context.Background())
	// пока в комнате кто-то есть, держим наготове соединение с OpenAI: разговор открывается быстрее
	proxy.KeepWarm("api.openai.com:443", func() bool {
		s := store.Get()
		return s.OpenAIKey != "" && s.WakeEnabled && engineService(s.ActiveProfile().Engine) == "openai" &&
			(s.UseVPN("openai") || s.Vless == "") && engine.RoomActive()
	})

	// разговор открывается быстрее: готовое TLS-соединение с облаком активного движка, пока в комнате кто-то есть
	for _, h := range []struct{ svc, addr string }{{"elevenlabs", "api.elevenlabs.io:443"}, {"gemini", geminiHost + ":443"},
		{"grok", "api.x.ai:443"}, {"chatgpt", "chatgpt.com:443"}} {
		proxy.KeepWarm(h.addr, func() bool {
			s := store.Get()
			return s.WakeEnabled && engineService(s.ActiveProfile().Engine) == h.svc && (s.UseVPN(h.svc) || s.Vless == "") && engine.RoomActive()
		})
	}
	// Eleven Realtime: пропуск на сессию берём заранее — после слова остаётся только подключиться
	go func() {
		for {
			time.Sleep(20 * time.Second)
			s := store.Get()
			p := s.ActiveProfile()
			if p.Engine != "eleven_realtime" || !s.WakeEnabled || s.ElevenLabsKey == "" || !engine.RoomActive() || engine.Talking() {
				continue
			}
			PrewarmEleven(s.ElevenLabsKey, p, engine.instructions(s, p), engine.tools, proxy.LiveClientFor("elevenlabs"))
		}
	}()
	updaterClient := &http.Client{Timeout: 60 * time.Second, Transport: DirectClient.Transport}
	updater := &Updater{store: store, client: func() *http.Client { return updaterClient },
		idle: func() bool { return !engine.Talking() && !engine.music.Playing() }}
	if _, err := os.Stat(updateNext); err == nil { // jarvisd.sh без поддержки обновлений не подменил файл
		log.Printf("обновление: %s не установлено старым jarvisd.sh — удаляю", updateNext)
		_ = os.Remove(updateNext)
	}
	_ = os.Remove(updateNext + ".part")
	go updater.Run()
	go RunCounter(store, func() *http.Client { return updaterClient }, *dataDir)
	web := &Web{store: store, proxy: proxy, engine: engine, journal: journal, started: time.Now(), dataDir: *dataDir, uiDir: *uiDir,
		updater: updater, stats: &Stats{dataDir: *dataDir}}
	go web.stats.Run()
	go WatchButtons(web.resetPassword)
	if err := web.Serve(*listen); err != nil {
		log.Printf("веб-настройки: %v", err)
		os.Exit(1)
	}
}
