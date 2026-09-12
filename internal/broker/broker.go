// Package broker is the marshmallows coordinator. Agents dial in over a
// mutually-authenticated Noise session (enrollment token verified with the
// identity service) and stay attached; browsers open a terminal to a device and
// the broker bridges their keystrokes/output to that agent's shell, multiplexing
// every session over the agent's single Noise control channel.
package broker

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/trustsentinel/marshmallows/internal/protocol"
	"github.com/trustsentinel/marshmallows/internal/registry"
	"github.com/trustsentinel/marshmallows/internal/secure"
	"github.com/trustsentinel/marshmallows/internal/transport"
)

type Config struct {
	Static      secure.Keypair // broker's long-lived Noise identity
	Cloud       string         // mini-cloud id (e.g. "valc31")
	AuthURL     string         // mm-auth base URL; "" = dev (accept any token)
	WebDir      string         // optional static dashboard to serve
	CORSOrigins []string
}

// agentConn is a registered agent's control session. Writes are serialized
// because multiple browser sessions fan input into the one Noise channel.
type agentConn struct {
	device string
	sess   *secure.Session
	mu     sync.Mutex
}

func (a *agentConn) send(m protocol.Msg) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sess.Write(b)
}

type sessionPipe struct {
	browser *websocket.Conn
	agent   *agentConn
}

type Broker struct {
	cfg      Config
	reg      *registry.Registry
	up       websocket.Upgrader
	mu       sync.Mutex
	agents   map[string]*agentConn   // deviceID -> control session
	sessions map[string]*sessionPipe // sessionID -> bridge
}

func New(cfg Config) *Broker {
	return &Broker{
		cfg:      cfg,
		reg:      registry.New(cfg.Cloud),
		up:       websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
		agents:   map[string]*agentConn{},
		sessions: map[string]*sessionPipe{},
	}
}

func (b *Broker) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices.json", b.devices)
	mux.HandleFunc("GET /agent", b.agentWS)
	mux.HandleFunc("GET /connect/{device}", b.connectWS)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	if b.cfg.WebDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(b.cfg.WebDir)))
	}
	return b.cors(mux)
}

func (b *Broker) devices(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b.reg.Graph())
}

// agentWS: an agent attaches over Noise, registers (token-checked), then holds
// the control session; the broker drives shells over it.
func (b *Broker) agentWS(w http.ResponseWriter, r *http.Request) {
	c, err := b.up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := transport.NewWSConn(c)
	sess, err := secure.Handshake(conn, secure.Config{Static: b.cfg.Static, Initiator: false})
	if err != nil {
		log.Printf("agent handshake failed: %v", err)
		_ = conn.Close()
		return
	}
	raw, err := sess.Read()
	if err != nil {
		_ = sess.Close()
		return
	}
	var reg protocol.Msg
	if json.Unmarshal(raw, &reg) != nil || reg.Type != protocol.TypeRegister || reg.Device == "" {
		_ = sess.Close()
		return
	}
	if !b.checkAgentToken(reg.Token) {
		_ = writeMsg(sess, protocol.Msg{Type: protocol.TypeError, Message: "invalid or expired enrollment token"})
		_ = sess.Close()
		return
	}

	ac := &agentConn{device: reg.Device, sess: sess}
	b.mu.Lock()
	if old := b.agents[reg.Device]; old != nil {
		_ = old.sess.Close() // replace a stale attachment
	}
	b.agents[reg.Device] = ac
	b.mu.Unlock()
	b.reg.Add(&registry.Device{
		ID: reg.Device, Name: reg.Name, OS: reg.OS, Addr: r.RemoteAddr,
		PubKey: secure.EncodePublic(sess.PeerStatic), LastSeen: time.Now(),
	})
	_ = ac.send(protocol.Msg{Type: protocol.TypeRegistered})
	log.Printf("agent registered: %s (%s)", reg.Device, reg.Name)

	for {
		raw, err := sess.Read()
		if err != nil {
			break
		}
		var m protocol.Msg
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		switch m.Type {
		case protocol.TypeOutput:
			b.mu.Lock()
			sp := b.sessions[m.Session]
			b.mu.Unlock()
			if sp != nil {
				_ = sp.browser.WriteMessage(websocket.BinaryMessage, m.Data)
			}
		case protocol.TypeClose:
			b.closeSession(m.Session)
		}
	}

	b.mu.Lock()
	if b.agents[reg.Device] == ac {
		delete(b.agents, reg.Device)
	}
	b.mu.Unlock()
	b.reg.Remove(reg.Device)
	log.Printf("agent disconnected: %s", reg.Device)
}

// connectWS: a browser opens a terminal to a device. The broker allocates a
// session, asks the agent to open a shell, and pipes bytes both ways.
func (b *Broker) connectWS(w http.ResponseWriter, r *http.Request) {
	device := r.PathValue("device")
	if !b.checkUser(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.mu.Lock()
	ac := b.agents[device]
	b.mu.Unlock()
	if ac == nil {
		http.Error(w, "device not connected", http.StatusNotFound)
		return
	}
	c, err := b.up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	session := newID()
	b.mu.Lock()
	b.sessions[session] = &sessionPipe{browser: c, agent: ac}
	b.mu.Unlock()

	if err := ac.send(protocol.Msg{Type: protocol.TypeOpen, Session: session}); err != nil {
		b.closeSession(session)
		return
	}
	log.Printf("session %s: browser -> %s", session, device)

	for {
		mt, data, err := c.ReadMessage()
		if err != nil {
			break
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		if err := ac.send(protocol.Msg{Type: protocol.TypeInput, Session: session, Data: data}); err != nil {
			break
		}
	}
	_ = ac.send(protocol.Msg{Type: protocol.TypeClose, Session: session})
	b.closeSession(session)
}

func (b *Broker) closeSession(session string) {
	b.mu.Lock()
	sp := b.sessions[session]
	delete(b.sessions, session)
	b.mu.Unlock()
	if sp != nil && sp.browser != nil {
		_ = sp.browser.Close()
	}
}

// checkAgentToken validates an enrollment token with mm-auth (single-use). In
// dev (no AuthURL) any token is accepted.
func (b *Broker) checkAgentToken(token string) bool {
	if b.cfg.AuthURL == "" {
		return true
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	resp, err := http.Post(b.cfg.AuthURL+"/agent_registration/check", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("auth check error: %v", err)
		return false
	}
	defer resp.Body.Close()
	var out struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Status == "ok"
}

// checkUser gates browser terminal access. Real deployments put mm-auth and the
// broker behind one origin (reverse proxy) so the session cookie is shared and
// this can verify it; on localhost/dev it is permissive.
func (b *Broker) checkUser(r *http.Request) bool {
	if b.cfg.AuthURL == "" {
		return true
	}
	req, err := http.NewRequest(http.MethodGet, b.cfg.AuthURL+"/api/session", nil)
	if err != nil {
		return false
	}
	if c, err := r.Cookie("mm_session"); err == nil {
		req.AddCookie(c) // works when served same-origin behind a proxy
	} else {
		return true // no shared cookie available (separate origin) — defer to network/proxy
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (b *Broker) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range b.cfg.CORSOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && allowed[o] {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func writeMsg(s *secure.Session, m protocol.Msg) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.Write(b)
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
