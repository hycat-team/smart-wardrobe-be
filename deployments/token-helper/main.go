// Command token-helper runs on the VPS and mints/caches Google ID tokens
// for the Nginx → Cloud Run hop (FR-033/FR-039).
//
// Nginx calls GET /token?audience=<run.app URL> via auth_request; the helper
// answers 200 with header X-Serverless-Authorization: Bearer <id token>.
// Service account is least-privilege (roles/run.invoker on the 3 services);
// key file MUST have mode 0600. Full GCP minting arrives in T071 — the cache,
// refresh-ahead and HTTP plumbing below are already final.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultAddr      = "127.0.0.1:8082"
	refreshAhead     = 5 * time.Minute
	tokenHeader      = "X-Serverless-Authorization"
	audienceParam    = "audience"
	keyFileEnv       = "TOKEN_HELPER_SA_KEY_FILE"
	defaultKeyFile   = "/run/secrets/token-helper-sa.json"
	cacheControlSkip = "no-store"
)

// errNotImplemented is returned until T071 wires the real GCP mint.
var errNotImplemented = errors.New("token-helper: GCP ID token mint not implemented (T071)")

type cachedToken struct {
	value     string
	expiresAt time.Time
}

// minter caches one ID token per audience and refreshes ahead of expiry.
type minter struct {
	mu     sync.Mutex
	tokens map[string]cachedToken
	key    []byte
}

func newMinter(key []byte) *minter {
	return &minter{tokens: make(map[string]cachedToken), key: key}
}

// tokenFor returns a cached token when still valid beyond refreshAhead,
// otherwise mints a fresh one.
func (m *minter) tokenFor(audience string) (string, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.tokens[audience]; ok && time.Until(c.expiresAt) > refreshAhead {
		return c.value, c.expiresAt, nil
	}
	value, expiresAt, err := mintIDToken(m.key, audience)
	if err != nil {
		return "", time.Time{}, err
	}
	m.tokens[audience] = cachedToken{value: value, expiresAt: expiresAt}
	return value, expiresAt, nil
}

// mintIDToken exchanges the service account key for a Google-signed ID token.
// Implemented in T071 (Phase 6) using golang.org/x/oauth2/google + idtoken.
func mintIDToken(_ []byte, _ string) (string, time.Time, error) {
	return "", time.Time{}, errNotImplemented
}

func main() {
	keyFile := os.Getenv(keyFileEnv)
	if keyFile == "" {
		keyFile = defaultKeyFile
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("token-helper: cannot read SA key file %s: %v", keyFile, err)
	}
	m := newMinter(key)

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = defaultAddr
	} else if len(addr) > 0 && addr[0] == ':' {
		addr = "127.0.0.1" + addr
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		audience := r.URL.Query().Get(audienceParam)
		if audience == "" {
			http.Error(w, "missing audience", http.StatusBadRequest)
			return
		}
		value, _, err := m.tokenFor(audience)
		if err != nil {
			if errors.Is(err, errNotImplemented) {
				http.Error(w, "token mint not implemented (T071)", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, "cannot mint token", http.StatusBadGateway)
			return
		}
		w.Header().Set(tokenHeader, "Bearer "+value)
		w.Header().Set("Cache-Control", cacheControlSkip)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	fmt.Printf("token-helper listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
