package main

// VPN по подключениям: у каждого облака своя галочка «через VPN». Например, в России YouTube без VPN
// не работает, а Яндекс через зарубежный VPN, наоборот, может начать сбоить.

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/xtls/xray-core/core"
)

type netService struct {
	ID, Name string
	Def      bool // по умолчанию через VPN
}

// Яндекс и Home Assistant — свои/домашние, их VPN не касается (Home Assistant вообще в домашней сети).
var netServices = []netService{
	{"chatgpt", "Подписка ChatGPT", true},
	{"openai", "OpenAI", true},
	{"gemini", "Google Gemini", true},
	{"grok", "xAI Grok", true},
	{"elevenlabs", "ElevenLabs", true},
	{"telegram", "Telegram", true},
	{"youtube", "YouTube", false},
	{"xiaomi", "Xiaomi Home", false},
	{"smartlife", "Smart Life", false},
	{"gdrive", "Google Диск", false},
	{"yandex", "Яндекс", false},
}

// UseVPN — ходит ли подключение через VPN (если VPN вообще задан).
func (s Settings) UseVPN(svc string) bool {
	if v, ok := s.VPN[svc]; ok {
		return v
	}
	if svc == "yandex" {
		return s.ProxyYandex
	}
	for _, n := range netServices {
		if n.ID == svc {
			return n.Def
		}
	}
	return true // неизвестное облако — как раньше, через VPN
}

// VPNState — для панели: галочка каждого подключения.
func (s Settings) VPNState() map[string]bool {
	out := map[string]bool{}
	for _, n := range netServices {
		out[n.ID] = s.UseVPN(n.ID)
	}
	return out
}

func knownService(svc string) bool {
	for _, n := range netServices {
		if n.ID == svc {
			return true
		}
	}
	return false
}

// engineService — к какому подключению относится движок ассистента.
func engineService(engine string) string {
	switch engine {
	case "gemini", "grok", "chatgpt":
		return engine
	case "elevenlabs", "eleven_realtime":
		return "elevenlabs"
	}
	return "openai"
}

var directDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

// viaVPN — пойдёт ли это подключение через туннель прямо сейчас.
func (p *Proxy) viaVPN(svc string) bool {
	if svc != "" && p.use != nil && !p.use(svc) {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.inst != nil
}

// DialFor — соединение для подключения svc: через VPN или напрямую, по его галочке.
func (p *Proxy) DialFor(svc string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !p.viaVPN(svc) {
			return directDialer.DialContext(ctx, network, addr)
		}
		return p.DialContext(ctx, network, addr)
	}
}

// ClientFor — HTTP-клиент подключения svc (timeout 0 — без общего ограничения).
func (p *Proxy) ClientFor(svc string, timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: serviceTransport{p, svc}}
}

type servicePool struct {
	inst *core.Instance
	t    *http.Transport
}

type serviceTransport struct {
	p   *Proxy
	svc string
}

func (t serviceTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.p.servicePool(t.svc).RoundTrip(r)
}

// Route selection happens for every request, including clients kept by a service.
// A route change retires the old pool; its in-flight requests can still finish.
func (p *Proxy) servicePool(svc string) *http.Transport {
	use := p.use == nil || p.use(svc)
	p.poolMu.Lock()
	defer p.poolMu.Unlock()
	var inst *core.Instance
	if use {
		p.mu.RLock()
		inst = p.inst
		p.mu.RUnlock()
	}
	if old, ok := p.transports[svc]; ok {
		if old.inst == inst {
			return old.t
		}
		old.t.CloseIdleConnections()
	}
	t := p.TransportFor(svc)
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if inst == nil {
			return directDialer.DialContext(ctx, network, addr)
		}
		return dialInstance(ctx, inst, network, addr)
	}
	if p.transports == nil {
		p.transports = make(map[string]servicePool)
	}
	p.transports[svc] = servicePool{inst, t}
	return t
}

func (p *Proxy) TransportFor(svc string) *http.Transport {
	t := &http.Transport{
		DialContext: p.DialFor(svc), ForceAttemptHTTP2: true, MaxIdleConns: 8,
		IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second,
	}
	if svc == "youtube" {
		// googlevideo часто отдают кэш-серверы провайдера (GGC): на части из них большое приветствие TLS
		// с постквантовым ключом (1,8 КБ, два пакета) зависает до таймаута. Обычные X25519/P-256 — один пакет.
		t.TLSClientConfig = &tls.Config{CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256}}
	}
	return t
}

// LiveClientFor — клиент для открытия разговора: готовое соединение берём, только если оно через тот же путь.
func (p *Proxy) LiveClientFor(svc string) *http.Client {
	if p.use == nil || p.use(svc) { // без туннеля LiveClient сам ходит напрямую и тоже держит готовое соединение
		return p.LiveClient()
	}
	return &http.Client{Transport: &http.Transport{
		DialContext: directDialer.DialContext,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			raw, err := directDialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			c := tls.Client(raw, &tls.Config{ServerName: host, NextProtos: []string{"http/1.1"}, MinVersion: tls.VersionTLS12})
			if err := c.HandshakeContext(ctx); err != nil {
				raw.Close()
				return nil, err
			}
			return c, nil
		},
		MaxIdleConns: 2, IdleConnTimeout: 30 * time.Second,
	}}
}
