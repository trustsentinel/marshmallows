// Package agent is the device-side of marshmallows: it dials the broker over a
// mutually-authenticated Noise session (pinning the broker's static key),
// registers with a one-time enrollment token, then serves PTY shells on demand —
// every session multiplexed over the one control channel. The device needs no
// open inbound port.
package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/trustsentinel/marshmallows/internal/protocol"
	"github.com/trustsentinel/marshmallows/internal/secure"
	"github.com/trustsentinel/marshmallows/internal/shell"
	"github.com/trustsentinel/marshmallows/internal/transport"
)

type Config struct {
	BrokerWS  string // ws://host:port
	BrokerKey []byte // broker static public key (pinned by IK)
	Identity  secure.Keypair
	DeviceID  string
	Name      string
	OS        string
	Token     string
	ShellPath string
}

type Agent struct {
	cfg    Config
	sess   *secure.Session
	wmu    sync.Mutex // serialize writes to the control session
	mu     sync.Mutex
	shells map[string]*shell.Session
}

// Run connects, registers, and serves until the connection drops (returning the
// error). Callers can loop with backoff to reconnect.
func Run(cfg Config) error {
	c, _, err := websocket.DefaultDialer.Dial(cfg.BrokerWS+"/agent", nil)
	if err != nil {
		return fmt.Errorf("dial broker: %w", err)
	}
	conn := transport.NewWSConn(c)
	sess, err := secure.Handshake(conn, secure.Config{Static: cfg.Identity, Initiator: true, PeerStatic: cfg.BrokerKey})
	if err != nil {
		return fmt.Errorf("noise handshake: %w", err)
	}
	a := &Agent{cfg: cfg, sess: sess, shells: map[string]*shell.Session{}}

	if err := a.write(protocol.Msg{
		Type: protocol.TypeRegister, Device: cfg.DeviceID, Name: cfg.Name, OS: cfg.OS, Token: cfg.Token,
	}); err != nil {
		return err
	}
	raw, err := sess.Read()
	if err != nil {
		return err
	}
	var m protocol.Msg
	_ = json.Unmarshal(raw, &m)
	switch m.Type {
	case protocol.TypeError:
		return fmt.Errorf("broker rejected registration: %s", m.Message)
	case protocol.TypeRegistered:
		log.Printf("registered with broker as %s", cfg.DeviceID)
	default:
		return fmt.Errorf("unexpected reply: %q", m.Type)
	}

	for {
		raw, err := sess.Read()
		if err != nil {
			a.closeAll()
			return err
		}
		var m protocol.Msg
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		switch m.Type {
		case protocol.TypeOpen:
			a.openShell(m.Session)
		case protocol.TypeInput:
			a.input(m.Session, m.Data)
		case protocol.TypeClose:
			a.closeShell(m.Session)
		}
	}
}

func (a *Agent) write(m protocol.Msg) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	a.wmu.Lock()
	defer a.wmu.Unlock()
	return a.sess.Write(b)
}

func (a *Agent) openShell(session string) {
	sh, err := shell.Start(a.cfg.ShellPath)
	if err != nil {
		_ = a.write(protocol.Msg{Type: protocol.TypeClose, Session: session})
		return
	}
	a.mu.Lock()
	a.shells[session] = sh
	a.mu.Unlock()
	log.Printf("shell opened: session %s", session)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, rerr := sh.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				if werr := a.write(protocol.Msg{Type: protocol.TypeOutput, Session: session, Data: data}); werr != nil {
					break
				}
			}
			if rerr != nil {
				break
			}
		}
		a.closeShell(session)
	}()
}

func (a *Agent) input(session string, data []byte) {
	a.mu.Lock()
	sh := a.shells[session]
	a.mu.Unlock()
	if sh != nil {
		_, _ = sh.Write(data)
	}
}

func (a *Agent) closeShell(session string) {
	a.mu.Lock()
	sh := a.shells[session]
	delete(a.shells, session)
	a.mu.Unlock()
	if sh != nil {
		_ = sh.Close()
		_ = a.write(protocol.Msg{Type: protocol.TypeClose, Session: session})
	}
}

func (a *Agent) closeAll() {
	a.mu.Lock()
	shells := a.shells
	a.shells = map[string]*shell.Session{}
	a.mu.Unlock()
	for _, sh := range shells {
		_ = sh.Close()
	}
}
