package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MemoryItem struct {
	ID        string    `json:"id"`
	Fact      string    `json:"fact"`
	CreatedAt time.Time `json:"created_at"`
}

type MemoryStore struct {
	path  string
	mu    sync.RWMutex
	items []MemoryItem
}

func NewMemoryStore(path string) *MemoryStore {
	m := &MemoryStore{path: path}
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		log.Printf("память: ошибка загрузки %s: %v", path, err)
	}
	return m
}

func (m *MemoryStore) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var items []MemoryItem
	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}
	m.items = items
	return nil
}

func (m *MemoryStore) saveLocked() error {
	if m.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(m.items, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(m.path)
	_ = os.MkdirAll(dir, 0o755)
	tmp := fmt.Sprintf("%s.%d.tmp", m.path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

func (m *MemoryStore) Add(fact string) (*MemoryItem, error) {
	fact = strings.TrimSpace(fact)
	if fact == "" {
		return nil, errors.New("факт не может быть пустым")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// Проверяем на дубликаты без учета регистра
	for i := range m.items {
		if strings.EqualFold(m.items[i].Fact, fact) {
			return &m.items[i], nil
		}
	}

	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	id := hex.EncodeToString(buf)

	item := MemoryItem{
		ID:        id,
		Fact:      fact,
		CreatedAt: time.Now(),
	}
	m.items = append(m.items, item)
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return &item, nil
}

func (m *MemoryStore) Delete(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		if m.items[i].ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			_ = m.saveLocked()
			return true
		}
	}
	return false
}

func (m *MemoryStore) DeleteByQuery(query string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []MemoryItem
	deleted := 0
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Fact), q) {
			deleted++
		} else {
			kept = append(kept, it)
		}
	}
	if deleted > 0 {
		m.items = kept
		_ = m.saveLocked()
	}
	return deleted
}

func (m *MemoryStore) List() []MemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MemoryItem, len(m.items))
	copy(out, m.items)
	return out
}

func (m *MemoryStore) PromptText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Долговременная память о хозяине и его семье:\n")
	for _, it := range m.items {
		sb.WriteString("- ")
		sb.WriteString(it.Fact)
		sb.WriteString("\n")
	}
	sb.WriteString("Естественно учитывай эти факты при ответах. Если хозяин говорит «запомни, что...», вызови инструмент remember. Если просит забыть — вызови инструмент forget.")
	return sb.String()
}
