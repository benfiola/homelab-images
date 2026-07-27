package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionCookieName = "azerothcore_web_session"

type session struct {
	username  string
	createdAt time.Time
}

// sessionStore is a simple in-memory session store. Sessions are lost on
// process restart - acceptable for a small self-hosted admin tool, and
// avoids needing a separate session backend (Redis, DB-backed sessions,
// etc.) for something this size.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]session)}
}

func (s *sessionStore) create(username string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	s.mu.Lock()
	s.sessions[token] = session{username: username, createdAt: time.Now()}
	s.mu.Unlock()

	return token, nil
}

func (s *sessionStore) get(token string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	return sess, ok
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

type ctxKey string

const accountCtxKey ctxKey = "account"

func accountFromContext(ctx context.Context) *Account {
	a, _ := ctx.Value(accountCtxKey).(*Account)
	return a
}
