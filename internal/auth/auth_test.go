package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func newTestServer(t *testing.T) (*Server, *Store) {
	t.Helper()
	st, err := OpenStore(filepath.Join(t.TempDir(), "auth.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	srv, err := NewServer(Config{RPID: "localhost", RPOrigins: []string{"http://localhost:8080"}}, st)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv, st
}

func TestAgentTokenSingleUse(t *testing.T) {
	_, st := newTestServer(t)
	tok, err := st.CreateAgentToken(time.Minute)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !st.CheckAgentToken(tok) {
		t.Fatal("first check should succeed")
	}
	if st.CheckAgentToken(tok) {
		t.Fatal("second check must fail (single-use)")
	}
	if st.CheckAgentToken("nope") {
		t.Fatal("unknown token must fail")
	}
}

func TestExpiredAgentToken(t *testing.T) {
	_, st := newTestServer(t)
	tok, _ := st.CreateAgentToken(-time.Second) // already expired
	if st.CheckAgentToken(tok) {
		t.Fatal("expired token must fail")
	}
}

func TestInviteLifecycle(t *testing.T) {
	_, st := newTestServer(t)
	tok, err := st.CreateInvite("alice", time.Hour)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if u, err := st.PeekInvite(tok); err != nil || u != "alice" {
		t.Fatalf("peek = %q,%v; want alice,nil", u, err)
	}
	if u, err := st.ConsumeInvite(tok); err != nil || u != "alice" {
		t.Fatalf("consume = %q,%v; want alice,nil", u, err)
	}
	if _, err := st.PeekInvite(tok); err == nil {
		t.Fatal("invite must be gone after consume")
	}
}

func TestStorePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	st1, _ := OpenStore(path)
	if _, err := st1.CreateUser("bob"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	st2, err := OpenStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := st2.GetUser("bob"); err != nil {
		t.Fatalf("bob should persist: %v", err)
	}
}

func TestTOTPRoundTrip(t *testing.T) {
	key, err := generateTOTP("alice")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if !validateTOTP(code, key.Secret()) {
		t.Fatal("valid code should verify")
	}
	if validateTOTP("000000", key.Secret()) {
		t.Fatal("bogus code should not verify (flaky ~1e-6; rerun)")
	}
}

func TestAgentCheckEndpoint(t *testing.T) {
	srv, st := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tok, _ := st.CreateAgentToken(time.Minute)
	post := func(token string) map[string]any {
		body, _ := json.Marshal(map[string]string{"token": token})
		resp, err := http.Post(ts.URL+"/agent_registration/check", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return out
	}
	if got := post(tok)["status"]; got != "ok" {
		t.Fatalf("first check status = %v; want ok", got)
	}
	if got := post(tok)["status"]; got != "error" {
		t.Fatalf("second check status = %v; want error (consumed)", got)
	}
}

func TestSessionGuardRejectsAnonymous(t *testing.T) {
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/session")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", resp.StatusCode)
	}
}
