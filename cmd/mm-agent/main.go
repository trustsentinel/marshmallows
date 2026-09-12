// Command mm-agent runs on a device: it pins the broker's static key, enrolls
// with a one-time token, and serves brokered PTY shells over Noise. No inbound
// port is opened on the device.
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/trustsentinel/marshmallows/internal/agent"
	"github.com/trustsentinel/marshmallows/internal/secure"
)

func main() {
	brokerWS := flag.String("broker", envOr("MM_AGENT_BROKER", "ws://localhost:8081"), "broker websocket base URL")
	brokerKey := flag.String("broker-key", os.Getenv("MM_AGENT_BROKER_KEY"), "broker static public key (base64) to pin — required")
	identity := flag.String("identity", envOr("MM_AGENT_IDENTITY", "data/agent.key"), "agent identity file (persistent)")
	cloud := flag.String("cloud", envOr("MM_AGENT_CLOUD", "valc31"), "mini-cloud id")
	device := flag.String("device", os.Getenv("MM_AGENT_DEVICE"), "device id (default: <cloud>.<random>)")
	name := flag.String("name", envOr("MM_AGENT_NAME", "device"), "human-friendly name")
	osName := flag.String("os", os.Getenv("MM_AGENT_OS"), "device OS label (default: runtime GOOS)")
	token := flag.String("token", os.Getenv("MM_AGENT_TOKEN"), "one-time enrollment token (prompted if empty)")
	shellPath := flag.String("shell", envOr("MM_AGENT_SHELL", "/bin/sh"), "login shell")
	flag.Parse()

	if *brokerKey == "" {
		log.Fatal("-broker-key is required (the broker prints it at startup)")
	}
	bkey, err := secure.DecodePublic(*brokerKey)
	if err != nil {
		log.Fatalf("bad -broker-key: %v", err)
	}
	kp, err := secure.LoadOrCreateIdentity(*identity)
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	dev := *device
	if dev == "" {
		dev = *cloud + "." + shortID()
	}
	tok := *token
	if tok == "" {
		tok = prompt("Enter enrollment token: ")
	}
	osLabel := *osName
	if osLabel == "" {
		osLabel = runtime.GOOS
	}

	log.Printf("agent %s -> %s", dev, *brokerWS)
	err = agent.Run(agent.Config{
		BrokerWS: *brokerWS, BrokerKey: bkey, Identity: kp,
		DeviceID: dev, Name: *name, OS: osLabel, Token: tok, ShellPath: *shellPath,
	})
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
}

func prompt(msg string) string {
	fmt.Fprint(os.Stderr, msg)
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

func shortID() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
