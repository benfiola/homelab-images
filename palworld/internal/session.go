package internal

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionCookieName = "palworld_mgmt_session"

// sessionStore is an in-memory session store. Sessions are lost on process
// restart - fine for a small self-hosted tool with a single shared
// credential, no separate backend needed.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]time.Time)}
}

func (s *sessionStore) create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	s.mu.Lock()
	s.sessions[token] = time.Now()
	s.mu.Unlock()

	return token, nil
}

func (s *sessionStore) valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[token]
	return ok
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}
