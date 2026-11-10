package playlist

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type Item struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	MediaURL  string  `json:"media_url"`
	SubURL    string  `json:"sub_url"`
	Format    string  `json:"format"`
	Duration  float64 `json:"duration"`
	IsCurrent bool    `json:"is_current"`
}

type Manager struct {
	mu          sync.RWMutex
	items       []*Item
	currentIndex int
	autoAdvance bool
}

func NewManager(autoAdvance bool) *Manager {
	return &Manager{
		items:        make([]*Item, 0),
		currentIndex: -1,
		autoAdvance:  autoAdvance,
	}
}

func (m *Manager) Add(title, mediaURL, subURL, format string, duration float64) *Item {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &Item{
		ID:        randomID(8),
		Title:     title,
		MediaURL:  mediaURL,
		SubURL:    subURL,
		Format:    format,
		Duration:  duration,
		IsCurrent: false,
	}

	m.items = append(m.items, item)
	if m.currentIndex == -1 {
		m.currentIndex = 0
		item.IsCurrent = true
	}
	return item
}

func (m *Manager) List() []*Item {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyList := make([]*Item, len(m.items))
	for i, item := range m.items {
		itemCopy := *item
		itemCopy.IsCurrent = (i == m.currentIndex)
		copyList[i] = &itemCopy
	}
	return copyList
}

func (m *Manager) Current() *Item {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentIndex >= 0 && m.currentIndex < len(m.items) {
		return m.items[m.currentIndex]
	}
	return nil
}

func (m *Manager) Next() *Item {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.items) == 0 {
		return nil
	}

	nextIdx := m.currentIndex + 1
	if nextIdx < len(m.items) {
		m.currentIndex = nextIdx
		return m.items[m.currentIndex]
	}
	return nil
}

func (m *Manager) Remove(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, item := range m.items {
		if item.ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			if m.currentIndex >= len(m.items) {
				m.currentIndex = len(m.items) - 1
			}
			return true
		}
	}
	return false
}

func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make([]*Item, 0)
	m.currentIndex = -1
}

func (m *Manager) SetAutoAdvance(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoAdvance = enabled
}

func (m *Manager) IsAutoAdvance() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.autoAdvance
}

func randomID(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
