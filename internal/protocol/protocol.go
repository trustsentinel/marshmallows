// Package protocol defines the JSON control/data frame exchanged over the
// agent<->broker Noise session. A single encrypted session per agent carries
// registration plus any number of multiplexed shell sessions; terminal bytes
// ride in Data (JSON-encoded as base64).
package protocol

type Msg struct {
	Type    string `json:"t"`
	Session string `json:"s,omitempty"`   // shell session id (multiplexing key)
	Device  string `json:"d,omitempty"`   // device id (register)
	Name    string `json:"n,omitempty"`   // human name (register)
	OS      string `json:"os,omitempty"`  // device OS (register)
	Token   string `json:"tok,omitempty"` // enrollment token (register)
	Data    []byte `json:"data,omitempty"`
	Message string `json:"msg,omitempty"`
}

const (
	TypeRegister   = "register"   // agent -> broker
	TypeRegistered = "registered" // broker -> agent
	TypeError      = "error"      // broker -> agent
	TypeOpen       = "open"       // broker -> agent: start a shell for Session
	TypeInput      = "in"         // broker -> agent: keystrokes for Session
	TypeOutput     = "out"        // agent -> broker: shell output for Session
	TypeResize     = "resize"     // broker -> agent: Data = "cols rows"
	TypeClose      = "close"      // either way: end Session
)
