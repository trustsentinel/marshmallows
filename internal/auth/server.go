package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Config configures the auth server. RPID/RPOrigins are the WebAuthn relying
// party settings; they must match the browser origin the dashboard is served
// from (e.g. RPID "localhost", origin "http://localhost:8080").
type Config struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
	CORSOrigins   []string
	CookieSecure  bool
	InviteTTL     time.Duration
	AgentTokenTTL time.Duration
}

type Server struct {
	cfg   Config
	store *Store
	wa    *webauthn.WebAuthn
	sess  *sessions
}

func NewServer(cfg Config, store *Store) (*Server, error) {
	if cfg.RPDisplayName == "" {
		cfg.RPDisplayName = "marshmallows"
	}
	if cfg.InviteTTL == 0 {
		cfg.InviteTTL = 24 * time.Hour
	}
	if cfg.AgentTokenTTL == 0 {
		cfg.AgentTokenTTL = 60 * time.Minute
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, store: store, wa: wa, sess: newSessions(cfg.CookieSecure)}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Operator WebAuthn enrollment + login (the new frontend calls these).
	mux.HandleFunc("POST /api/register/begin", s.registerBegin)
	mux.HandleFunc("POST /api/register/finish", s.registerFinish)
	mux.HandleFunc("POST /api/login/begin", s.loginBegin)
	mux.HandleFunc("POST /api/login/finish", s.loginFinish)
	// Optional TOTP second factor.
	mux.HandleFunc("POST /api/totp/verify", s.totpVerify)   // login step-up
	mux.HandleFunc("POST /api/totp/enroll", s.totpEnroll)   // start enrollment
	mux.HandleFunc("POST /api/totp/confirm", s.totpConfirm) // finish enrollment
	// Admin: mint invites and device-enrollment tokens.
	mux.HandleFunc("POST /api/invites", s.createInvite)
	mux.HandleFunc("POST /api/agent-tokens", s.createAgentToken)
	// Session introspection / teardown for the SPA route guard.
	mux.HandleFunc("GET /api/session", s.sessionInfo)
	mux.HandleFunc("POST /api/logout", s.logout)
	// Broker-facing contract (preserved): single-use device token validation.
	mux.HandleFunc("POST /agent_registration/check", s.agentCheck)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	return s.cors(mux)
}

// ---- WebAuthn registration ----

func (s *Server) registerBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Invite   string `json:"invite_token"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	username, err := s.store.PeekInvite(req.Invite)
	if err != nil {
		httpError(w, http.StatusForbidden, "invalid or expired invite")
		return
	}
	user, err := s.store.GetUser(username)
	if errors.Is(err, ErrNotFound) {
		if user, err = s.store.CreateUser(username); err != nil {
			httpError(w, http.StatusConflict, err.Error())
			return
		}
	} else if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	options, sd, err := s.wa.BeginRegistration(user)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess := s.sess.getOrCreate(w, r)
	sess.Username = username
	sess.Ceremony = sd
	sess.PendingInvite = req.Invite
	sess.Authenticated = false
	writeJSON(w, http.StatusOK, options)
}

func (s *Server) registerFinish(w http.ResponseWriter, r *http.Request) {
	_, sess := s.sess.get(r)
	if sess == nil || sess.Ceremony == nil || sess.Username == "" {
		httpError(w, http.StatusBadRequest, "no registration in progress")
		return
	}
	user, err := s.store.GetUser(sess.Username)
	if err != nil {
		httpError(w, http.StatusBadRequest, "unknown user")
		return
	}
	cred, err := s.wa.FinishRegistration(user, *sess.Ceremony, r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "registration failed: "+err.Error())
		return
	}
	user.Credentials = append(user.Credentials, *cred)
	if err := s.store.PutUser(user); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = s.store.ConsumeInvite(sess.PendingInvite)
	sess.Ceremony = nil
	sess.PendingInvite = ""
	sess.Authenticated = true
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "username": user.Username})
}

// ---- WebAuthn login ----

func (s *Server) loginBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	user, err := s.store.GetUser(req.Username)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unknown user")
		return
	}
	options, sd, err := s.wa.BeginLogin(user)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess := s.sess.getOrCreate(w, r)
	sess.Username = req.Username
	sess.Ceremony = sd
	sess.Authenticated = false
	writeJSON(w, http.StatusOK, options)
}

func (s *Server) loginFinish(w http.ResponseWriter, r *http.Request) {
	_, sess := s.sess.get(r)
	if sess == nil || sess.Ceremony == nil || sess.Username == "" {
		httpError(w, http.StatusBadRequest, "no login in progress")
		return
	}
	user, err := s.store.GetUser(sess.Username)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unknown user")
		return
	}
	cred, err := s.wa.FinishLogin(user, *sess.Ceremony, r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "login failed")
		return
	}
	// Persist the updated signature counter (clone detection).
	for i := range user.Credentials {
		if slices.Equal(user.Credentials[i].ID, cred.ID) {
			user.Credentials[i].Authenticator.SignCount = cred.Authenticator.SignCount
			break
		}
	}
	_ = s.store.PutUser(user)
	sess.Ceremony = nil
	if user.HasTOTP() {
		// Hold the session unauthenticated until the TOTP step-up succeeds.
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "totp_required": true})
		return
	}
	sess.Authenticated = true
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "totp_required": false})
}

// ---- TOTP ----

func (s *Server) totpVerify(w http.ResponseWriter, r *http.Request) {
	_, sess := s.sess.get(r)
	if sess == nil || sess.Username == "" {
		httpError(w, http.StatusBadRequest, "no login in progress")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	user, err := s.store.GetUser(sess.Username)
	if err != nil || !user.HasTOTP() || !validateTOTP(req.Code, user.TOTPSecret) {
		httpError(w, http.StatusUnauthorized, "invalid code")
		return
	}
	sess.Authenticated = true
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) totpEnroll(w http.ResponseWriter, r *http.Request) {
	sess, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	key, err := generateTOTP(user.Username)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess.PendingTOTP = key.Secret()
	writeJSON(w, http.StatusOK, map[string]any{"secret": key.Secret(), "otpauth_url": key.URL()})
}

func (s *Server) totpConfirm(w http.ResponseWriter, r *http.Request) {
	sess, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if sess.PendingTOTP == "" || !validateTOTP(req.Code, sess.PendingTOTP) {
		httpError(w, http.StatusBadRequest, "invalid code")
		return
	}
	user.TOTPSecret = sess.PendingTOTP
	sess.PendingTOTP = ""
	if err := s.store.PutUser(user); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Admin: invites + device tokens ----

func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuth(w, r); !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Username == "" {
		httpError(w, http.StatusBadRequest, "username required")
		return
	}
	tok, err := s.store.CreateInvite(req.Username, s.cfg.InviteTTL)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok})
}

func (s *Server) createAgentToken(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuth(w, r); !ok {
		return
	}
	tok, err := s.store.CreateAgentToken(s.cfg.AgentTokenTTL)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok})
}

// ---- Session + broker contract ----

func (s *Server) sessionInfo(w http.ResponseWriter, r *http.Request) {
	_, sess := s.sess.get(r)
	if sess == nil || !sess.Authenticated {
		httpError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	user, _ := s.store.GetUser(sess.Username)
	totp := user != nil && user.HasTOTP()
	writeJSON(w, http.StatusOK, map[string]any{"username": sess.Username, "totp": totp})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.sess.destroy(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// agentCheck validates (and consumes) a device-enrollment token. The broker
// posts a flat {"token": "..."} body; success is {"status":"ok"}.
func (s *Server) agentCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if s.store.CheckAgentToken(req.Token) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": "Invalid token"})
}

// ---- helpers ----

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (*session, *User, bool) {
	_, sess := s.sess.get(r)
	if sess == nil || !sess.Authenticated {
		httpError(w, http.StatusUnauthorized, "not authenticated")
		return nil, nil, false
	}
	user, err := s.store.GetUser(sess.Username)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unknown user")
		return nil, nil, false
	}
	return sess, user, true
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.CORSOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"status": "error", "message": msg})
}
