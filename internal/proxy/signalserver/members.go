package signalserver

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Members struct {
	sync.RWMutex

	ws map[string]*websocket.Conn
}

func (m *Members) Get(id string) (*websocket.Conn, bool) {
	m.RLock()
	member, ok := m.ws[id]
	m.RUnlock()
	return member, ok
}

func (m *Members) Set(id string, member *websocket.Conn) {
	if _, exists := m.Get(id); exists {
		return
	}
	m.Lock()
	m.ws[id] = member
	m.Unlock()
}

func (m *Members) Delete(id string) {
	m.Lock()
	delete(m.ws, id)
	m.Unlock()
}

func (m *Members) Count() int {
	var n int
	m.Range(func(member *websocket.Conn) bool {
		if member != nil {
			n++
		}
		return true
	})
	return n
}

func (m *Members) Range(f func(*websocket.Conn) bool) {
	m.RLock()
	defer m.RUnlock()
	for _, member := range m.ws {
		if next := f(member); !next {
			return
		}
	}
}
