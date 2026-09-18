package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func stubMint(t *testing.T, fn func([]byte, string) (string, time.Time, error)) {
	t.Helper()
	orig := mintIDToken
	mintIDToken = fn
	t.Cleanup(func() { mintIDToken = orig })
}

func TestTokenForCachesUntilRefreshAhead(t *testing.T) {
	var calls int32
	stubMint(t, func(_ []byte, aud string) (string, time.Time, error) {
		atomic.AddInt32(&calls, 1)
		return "tok-" + aud, time.Now().Add(time.Hour), nil
	})

	m := newMinter([]byte("key"))
	v1, _, err := m.tokenFor("aud")
	if err != nil || v1 != "tok-aud" {
		t.Fatalf("first token = %q, %v", v1, err)
	}
	v2, _, err := m.tokenFor("aud")
	if err != nil || v2 != "tok-aud" {
		t.Fatalf("cached token = %q, %v", v2, err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("mint calls = %d, want 1 (cache hit)", got)
	}
}

func TestTokenForRefreshesNearExpiry(t *testing.T) {
	var calls int32
	stubMint(t, func(_ []byte, _ string) (string, time.Time, error) {
		atomic.AddInt32(&calls, 1)
		return "tok", time.Now().Add(time.Minute), nil // inside refreshAhead
	})

	m := newMinter([]byte("key"))
	for i := 0; i < 2; i++ {
		if _, _, err := m.tokenFor("aud"); err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("mint calls = %d, want 2 (refresh ahead)", got)
	}
}

func TestTokenForCachesPerAudience(t *testing.T) {
	var calls int32
	stubMint(t, func(_ []byte, aud string) (string, time.Time, error) {
		atomic.AddInt32(&calls, 1)
		return "tok-" + aud, time.Now().Add(time.Hour), nil
	})

	m := newMinter([]byte("key"))
	_, _, _ = m.tokenFor("a")
	_, _, _ = m.tokenFor("b")
	_, _, _ = m.tokenFor("a")
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("mint calls = %d, want 2 (one per audience)", got)
	}
}

func TestTokenHandlerInjectsHeader(t *testing.T) {
	stubMint(t, func(_ []byte, aud string) (string, time.Time, error) {
		return "tok-" + aud, time.Now().Add(time.Hour), nil
	})
	srv := httptest.NewServer(newServeMux(newMinter([]byte("key"))))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/token?audience=aud1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get(tokenHeader); got != "Bearer tok-aud1" {
		t.Fatalf("%s = %q, want %q", tokenHeader, got, "Bearer tok-aud1")
	}
}

func TestTokenHandlerMissingAudience(t *testing.T) {
	srv := httptest.NewServer(newServeMux(newMinter([]byte("key"))))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/token")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestTokenHandlerMintError(t *testing.T) {
	stubMint(t, func(_ []byte, _ string) (string, time.Time, error) {
		return "", time.Time{}, errors.New("boom")
	})
	srv := httptest.NewServer(newServeMux(newMinter([]byte("key"))))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/token?audience=aud1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(newServeMux(newMinter([]byte("key"))))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMintGoogleIDTokenRejectsEmptyInputs(t *testing.T) {
	if _, _, err := mintGoogleIDToken(nil, "aud"); err == nil {
		t.Fatal("empty key must error")
	}
	if _, _, err := mintGoogleIDToken([]byte("key"), "  "); err == nil {
		t.Fatal("empty audience must error")
	}
}

func TestListenAddr(t *testing.T) {
	t.Setenv("PORT", "")
	if got := listenAddr(); got != ":"+defaultPort {
		t.Fatalf("listenAddr() = %q, want %q", got, ":"+defaultPort)
	}
	t.Setenv("PORT", "9000")
	if got := listenAddr(); got != ":9000" {
		t.Fatalf("listenAddr() = %q, want :9000", got)
	}
	t.Setenv("PORT", "127.0.0.1:9001")
	if got := listenAddr(); got != "127.0.0.1:9001" {
		t.Fatalf("listenAddr() = %q, want 127.0.0.1:9001", got)
	}
}
