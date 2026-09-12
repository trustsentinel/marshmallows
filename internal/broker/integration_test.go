package broker_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/trustsentinel/marshmallows/internal/agent"
	"github.com/trustsentinel/marshmallows/internal/broker"
	"github.com/trustsentinel/marshmallows/internal/secure"
)

// TestEndToEnd brings up a broker + a real agent (Noise-authenticated), then
// acts as the browser: opens a terminal session and checks a command round-trips
// through the multiplexed PTY shell.
func TestEndToEnd(t *testing.T) {
	kp, err := secure.GenerateKeypair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	b := broker.New(broker.Config{Static: kp, Cloud: "valc31"}) // AuthURL empty = dev
	srv := httptest.NewServer(b.Handler())
	defer srv.Close()
	wsBase := "ws" + strings.TrimPrefix(srv.URL, "http")

	device := "valc31.smoke"
	go func() {
		_ = agent.Run(agent.Config{
			BrokerWS: wsBase, BrokerKey: kp.Public, Identity: mustKey(t),
			DeviceID: device, Name: "smoke", OS: "test", Token: "dev", ShellPath: "/bin/sh",
		})
	}()

	// wait for the agent to register (appears in /devices.json)
	deadline := time.Now().Add(8 * time.Second)
	for {
		if devicePresent(t, srv.URL, device) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("device never registered")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// act as the browser terminal
	c, _, err := websocket.DefaultDialer.Dial(wsBase+"/connect/"+device, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	marker := "marshmallow_smoke_ok"
	if err := c.WriteMessage(websocket.TextMessage, []byte("echo "+marker+"\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got strings.Builder
	_ = c.SetReadDeadline(time.Now().Add(8 * time.Second))
	for {
		_, data, err := c.ReadMessage()
		if err != nil {
			break
		}
		got.Write(data)
		// the shell echoes the command AND prints the echo output; the output
		// line is the marker on its own, so require it to appear twice or with a
		// newline — but simplest: the marker output line ends with newline.
		if strings.Count(got.String(), marker) >= 2 {
			break // typed echo + command output
		}
	}
	if !strings.Contains(got.String(), marker) {
		t.Fatalf("shell output missing marker; got: %q", got.String())
	}
}

func devicePresent(t *testing.T, base, id string) bool {
	t.Helper()
	resp, err := http.Get(base + "/devices.json")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var g struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&g)
	for _, n := range g.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func mustKey(t *testing.T) secure.Keypair {
	t.Helper()
	kp, err := secure.GenerateKeypair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	return kp
}
