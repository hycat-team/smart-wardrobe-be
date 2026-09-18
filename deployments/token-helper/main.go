// Command token-helper runs on the VPS and mints/caches Google ID tokens
// for the Nginx → Cloud Run hop (FR-033/FR-039).
//
// Nginx calls GET /token?audience=<run.app URL> via auth_request; the helper
// answers 200 with header X-Serverless-Authorization: Bearer <id token>.
// Service account is least-privilege (roles/run.invoker on the 3 services);
// key file MUST have mode 0600.
package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultPort      = "8082"
	refreshAhead     = 5 * time.Minute
	tokenHeader      = "X-Serverless-Authorization"
	audienceParam    = "audience"
	keyFileEnv       = "TOKEN_HELPER_SA_KEY_FILE"
	defaultKeyFile   = "/run/secrets/token-helper-sa.json"
	cacheControlSkip = "no-store"
)

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

// mintIDToken is a package variable so tests can stub the GCP call.
var mintIDToken = mintGoogleIDToken

// serviceAccountKey is the subset of a Google service-account JSON key needed
// to mint an ID token via the JWT-bearer flow.
type serviceAccountKey struct {
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
	TokenURI     string `json:"token_uri"`
}

const defaultTokenURI = "https://oauth2.googleapis.com/token"

// mintGoogleIDToken exchanges the service-account key for a Google-signed ID
// token whose audience is the Cloud Run service URL (FR-033/FR-039).
//
// Implemented with the standard library only: it signs a JWT assertion
// (RS256) and exchanges it at the token endpoint
// (urn:ietf:params:oauth:grant-type:jwt-bearer) with `target_audience`, which
// returns an `id_token`. This avoids pulling google.golang.org/api into the
// workspace (which would force broad transitive version bumps).
func mintGoogleIDToken(key []byte, audience string) (string, time.Time, error) {
	if len(key) == 0 {
		return "", time.Time{}, errors.New("service account key is empty")
	}
	if strings.TrimSpace(audience) == "" {
		return "", time.Time{}, errors.New("audience is required")
	}

	var sa serviceAccountKey
	if err := json.Unmarshal(key, &sa); err != nil {
		return "", time.Time{}, fmt.Errorf("parse service account key: %w", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return "", time.Time{}, errors.New("service account key is missing client_email/private_key")
	}
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = defaultTokenURI
	}

	now := time.Now()
	expiresAt := now.Add(time.Hour)
	assertion, err := signJWTAssertion(sa, tokenURI, audience, now, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("call token endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		IDToken   string `json:"id_token"`
		ExpiresIn int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", time.Time{}, fmt.Errorf("parse token response: %w", err)
	}
	if payload.IDToken == "" {
		return "", time.Time{}, errors.New("token endpoint returned no id_token")
	}
	if payload.ExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return payload.IDToken, expiresAt, nil
}

// signJWTAssertion signs the JWT-bearer assertion for a service-account key.
func signJWTAssertion(sa serviceAccountKey, tokenURI, targetAudience string, iat, exp time.Time) (string, error) {
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return "", errors.New("invalid private_key PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		if pkcs1, err1 := x509.ParsePKCS1PrivateKey(block.Bytes); err1 == nil {
			parsed = pkcs1
		} else {
			return "", fmt.Errorf("parse private key: %w", err)
		}
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("private key is not RSA")
	}

	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	if sa.PrivateKeyID != "" {
		header["kid"] = sa.PrivateKeyID
	}
	claims := map[string]any{
		"iss": sa.ClientEmail,
		"sub": sa.ClientEmail,
		"aud": tokenURI,
		"iat": iat.Unix(),
		"exp": exp.Unix(),
	}
	if targetAudience != "" {
		claims["target_audience"] = targetAudience
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(headerJSON) + "." + enc.EncodeToString(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign assertion: %w", err)
	}
	return signingInput + "." + enc.EncodeToString(signature), nil
}

// newServeMux builds the HTTP surface (kept separate for tests).
func newServeMux(m *minter) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		audience := r.URL.Query().Get(audienceParam)
		if audience == "" {
			http.Error(w, "missing audience", http.StatusBadRequest)
			return
		}
		value, _, err := m.tokenFor(audience)
		if err != nil {
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
	return mux
}

// listenAddr resolves the bind address. It listens on ALL interfaces so the
// Nginx container can reach it by service name (it is only exposed on the
// internal compose network, never published).
func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	if !strings.Contains(port, ":") {
		port = ":" + port
	}
	return port
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

	addr := listenAddr()
	fmt.Printf("token-helper listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, newServeMux(newMinter(key))))
}
