// Command mm-auth is the marshmallows identity service: WebAuthn enrollment and
// login (with an optional TOTP second factor) plus single-use device tokens.
// It replaces the legacy Rails auth app.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/trustsentinel/marshmallows/internal/auth"
)

func main() {
	addr := flag.String("addr", envOr("MM_AUTH_ADDR", ":8090"), "listen address")
	store := flag.String("store", envOr("MM_AUTH_STORE", "data/auth.json"), "JSON store path (empty = in-memory)")
	rpID := flag.String("rp-id", envOr("MM_AUTH_RP_ID", "localhost"), "WebAuthn RP ID (the site's registrable domain)")
	origins := flag.String("rp-origins", envOr("MM_AUTH_RP_ORIGINS", "http://localhost:8080"), "comma-separated allowed browser origins")
	secure := flag.Bool("cookie-secure", os.Getenv("MM_AUTH_COOKIE_SECURE") == "1", "set Secure on session cookies (enable behind HTTPS)")
	flag.Parse()

	st, err := auth.OpenStore(*store)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	originList := splitCSV(*origins)
	srv, err := auth.NewServer(auth.Config{
		RPID:          *rpID,
		RPDisplayName: "marshmallows",
		RPOrigins:     originList,
		CORSOrigins:   originList,
		CookieSecure:  *secure,
	}, st)
	if err != nil {
		log.Fatalf("new server: %v", err)
	}

	log.Printf("mm-auth listening on %s (rp-id=%s origins=%v store=%q)", *addr, *rpID, originList, *store)
	hs := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(hs.ListenAndServe())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
