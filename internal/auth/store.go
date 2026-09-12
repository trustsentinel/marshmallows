// Package auth is the marshmallows identity service: passwordless WebAuthn
// (security keys / platform authenticators) with an optional TOTP second
// factor, plus single-use enrollment tokens for devices. It replaces the
// legacy Rails auth app and keeps its state in a small JSON-backed store so
// the service stays self-contained (no MySQL/Redis).
package auth

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

var ErrNotFound = errors.New("not found")

// User is a registered operator. It implements webauthn.User.
type User struct {
	ID          []byte                `json:"id"` // WebAuthn user handle
	Username    string                `json:"username"`
	Credentials []webauthn.Credential `json:"credentials"`
	TOTPSecret  string                `json:"totp_secret,omitempty"` // base32; empty = not enrolled
	Created     time.Time             `json:"created"`
}

func (u *User) WebAuthnID() []byte                         { return u.ID }
func (u *User) WebAuthnName() string                       { return u.Username }
func (u *User) WebAuthnDisplayName() string                { return u.Username }
func (u *User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
func (u *User) HasTOTP() bool                              { return u.TOTPSecret != "" }

type token struct {
	Username string    `json:"username,omitempty"` // set for invites
	Expires  time.Time `json:"expires"`
}

type data struct {
	Users       map[string]*User  `json:"users"`        // keyed by username
	Invites     map[string]*token `json:"invites"`      // keyed by token
	AgentTokens map[string]*token `json:"agent_tokens"` // keyed by token
}

// Store is a mutex-guarded, JSON-file-backed persistence layer. It is sized for
// a single mini-cloud's operators and devices, not a multi-tenant directory.
type Store struct {
	mu   sync.Mutex
	path string
	d    data
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, d: data{
		Users:       map[string]*User{},
		Invites:     map[string]*token{},
		AgentTokens: map[string]*token{},
	}}
	if path == "" {
		return s, nil // in-memory only
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.d); err != nil {
		return nil, err
	}
	return s, nil
}

// save writes the store atomically. Caller must hold s.mu.
func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// GetUser returns the user or ErrNotFound.
func (s *Store) GetUser(username string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.d.Users[username]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

// CreateUser makes a new user with a random WebAuthn handle. Fails if taken.
func (s *Store) CreateUser(username string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.d.Users[username]; ok {
		return nil, errors.New("username already registered")
	}
	u := &User{ID: randomBytes(16), Username: username, Created: time.Now().UTC()}
	s.d.Users[username] = u
	return u, s.save()
}

// PutUser persists mutations to an existing user (new credential, TOTP secret,
// updated sign count).
func (s *Store) PutUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.Users[u.Username] = u
	return s.save()
}

// CreateInvite mints a single-use registration invite for username.
func (s *Store) CreateInvite(username string, ttl time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := randomToken()
	s.d.Invites[t] = &token{Username: username, Expires: time.Now().UTC().Add(ttl)}
	return t, s.save()
}

// PeekInvite validates an invite without consuming it (used at ceremony start;
// the invite is only spent once registration actually completes).
func (s *Store) PeekInvite(tok string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.d.Invites[tok]
	if !ok || time.Now().After(inv.Expires) {
		return "", ErrNotFound
	}
	return inv.Username, nil
}

// ConsumeInvite validates and deletes an invite, returning its username.
func (s *Store) ConsumeInvite(tok string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.d.Invites[tok]
	if !ok || time.Now().After(inv.Expires) {
		delete(s.d.Invites, tok)
		_ = s.save()
		return "", ErrNotFound
	}
	delete(s.d.Invites, tok)
	return inv.Username, s.save()
}

// CreateAgentToken mints a single-use device-enrollment token.
func (s *Store) CreateAgentToken(ttl time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := randomToken()
	s.d.AgentTokens[t] = &token{Expires: time.Now().UTC().Add(ttl)}
	return t, s.save()
}

// CheckAgentToken validates and consumes (deletes) an agent token. This is the
// endpoint the broker calls during device registration — single-use by design.
func (s *Store) CheckAgentToken(tok string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.d.AgentTokens[tok]
	if !ok {
		return false
	}
	delete(s.d.AgentTokens, tok)
	_ = s.save()
	return ok && time.Now().Before(at.Expires)
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	return b
}

func randomToken() string {
	const hex = "0123456789abcdef"
	b := randomBytes(32)
	out := make([]byte, 64)
	for i, c := range b {
		out[i*2] = hex[c>>4]
		out[i*2+1] = hex[c&0x0f]
	}
	return string(out)
}
