package melody

import (
	"sync"
)

type envelope struct {
	t   int
	msg []byte
}

type hub struct {
	sessions   map[string]*Session
	mutex      sync.RWMutex
	queue      chan *envelope
	register   chan *Session
	unregister chan *Session
	exit       chan *envelope
	open       bool
}

func newHub() *hub {
	return &hub{
		sessions:   make(map[string]*Session),
		queue:      make(chan *envelope),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		exit:       make(chan *envelope),
		open:       true,
	}
}

func (h *hub) run() {
loop:
	for {
		select {
		case s := <-h.register:
			if h.open {
				h.mutex.Lock()
				h.sessions[s.UUID] = s
				h.mutex.Unlock()
			}
		case s := <-h.unregister:
			h.mutex.Lock()
			delete(h.sessions, s.UUID)
			h.mutex.Unlock()
		case m := <-h.queue:
			h.mutex.RLock()
			for _, s := range h.sessions {
				s.writeMessage(m)
			}
			h.mutex.RUnlock()
		case m := <-h.exit:
			h.open = false
			h.mutex.Lock()
			for _, s := range h.sessions {
				s.writeMessage(m)
				s.Close()
			}
			h.sessions = make(map[string]*Session)
			h.mutex.Unlock()
			break loop
		}
	}
}

func (h *hub) closed() bool {
	return !h.open
}

func (h *hub) len() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.sessions)
}
