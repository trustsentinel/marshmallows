// Command mm-broker is the marshmallows coordinator: agents attach over Noise
// (enrollment verified with mm-auth) and browsers open device terminals through
// it. Its static public key (logged at startup) is what agents pin.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/trustsentinel/marshmallows/internal/broker"
	"github.com/trustsentinel/marshmallows/internal/secure"
)

func main() {
	addr := flag.String("addr", envOr("MM_BROKER_ADDR", ":8081"), "listen address")
	cloud := flag.String("cloud", envOr("MM_BROKER_CLOUD", "valc31"), "mini-cloud id")
	authURL := flag.String("auth-url", os.Getenv("MM_BROKER_AUTH_URL"), "mm-auth base URL (empty = dev, accept any token)")
	webdir := flag.String("webdir", os.Getenv("MM_BROKER_WEBDIR"), "static dashboard directory to serve (optional)")
	identity := flag.String("identity", envOr("MM_BROKER_IDENTITY", "data/broker.key"), "broker Noise identity file")
	origins := flag.String("cors-origins", envOr("MM_BROKER_CORS", "http://localhost:8080"), "comma-separated allowed browser origins")
	flag.Parse()

	kp, err := secure.LoadOrCreateIdentity(*identity)
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	log.Printf("broker static key (agents pin this): %s", secure.EncodePublic(kp.Public))

	b := broker.New(broker.Config{
		Static: kp, Cloud: *cloud, AuthURL: *authURL, WebDir: *webdir, CORSOrigins: splitCSV(*origins),
	})
	hs := &http.Server{Addr: *addr, Handler: b.Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("mm-broker listening on %s (cloud=%s, auth=%q)", *addr, *cloud, *authURL)
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
