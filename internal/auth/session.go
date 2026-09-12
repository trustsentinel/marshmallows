package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const cookieName = "mm_session"

// session holds per-browser state: the logged-in user once authenticated, plus
// the transient data for an in-flight WebAuthn ceremony or TOTP enrollment.
type session struct {
	Username      string
	Authenticated bool
	Ceremony      *webauthn.SessionData // pending register/login challenge
	PendingInvite string                // invite to consume on successful register
	PendingTOTP   string                // TOTP secret being enrolled, pre-confirmation
	Expires       time.Time
}

type sessions struct {
	mu     sync.Mutex
	m      map[string]*session
	secure bool
}

func newSessions(secure bool) *sessions {
	return &sessions{m: map[string]*session{}, secure: secure}
}

func (s *sessions) get(r *http.Request) (string, *session) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[c.Value]
	if !ok || time.Now().After(sess.Expires) {
		return "", nil
	}
	return c.Value, sess
}

func (s *sessions) getOrCreate(w http.ResponseWriter, r *http.Request) *session {
	if _, sess := s.get(r); sess != nil {
		return sess
	}
	id := randomToken()
	sess := &session{Expires: time.Now().Add(12 * time.Hour)}
	s.mu.Lock()
	s.m[id] = sess
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: id, Path: "/", HttpOnly: true,
		Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600,
	})
	return sess
}

func (s *sessions) destroy(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		delete(s.m, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
}
