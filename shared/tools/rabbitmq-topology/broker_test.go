package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// TestFetchQueuesReadsTypeFromManagementAPI pins the reason this tool uses the
// HTTP API: the queue type is only available here, never from AMQP.
func TestFetchQueuesReadsTypeFromManagementAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/queues" {
			t.Errorf("path = %q, want /api/queues", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "closy" || pass != "secret" {
			t.Errorf("basic auth = %q/%q", user, pass)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"subscription_plan_changed_queue","vhost":"/","type":"classic","messages":0},
			{"name":"audit_log_queue","vhost":"/","type":"quorum","messages":3}
		]`))
	}))
	defer srv.Close()

	s := settingsForTest(srv)
	types, messages, err := s.FetchQueues(context.Background())
	if err != nil {
		t.Fatalf("FetchQueues: %v", err)
	}
	if types["subscription_plan_changed_queue"] != "classic" {
		t.Errorf("type = %q, want classic", types["subscription_plan_changed_queue"])
	}
	if messages["audit_log_queue"] != 3 {
		t.Errorf("messages = %d, want 3", messages["audit_log_queue"])
	}
}

// TestFetchQueuesFiltersByVhost matters on CloudAMQP, where the API lists every
// vhost the user can see. Without the filter a same-named queue in another
// vhost would corrupt the comparison.
func TestFetchQueuesFiltersByVhost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"name":"shared_name","vhost":"/","type":"classic","messages":1},
			{"name":"shared_name","vhost":"other","type":"quorum","messages":9}
		]`))
	}))
	defer srv.Close()

	s := settingsForTest(srv)
	types, _, err := s.FetchQueues(context.Background())
	if err != nil {
		t.Fatalf("FetchQueues: %v", err)
	}
	if types["shared_name"] != "classic" {
		t.Errorf("type = %q, want classic (vhost \"/\" only)", types["shared_name"])
	}
}

// TestFetchQueuesRejects403 keeps a mistyped RABBITMQ_VHOST from reading as
// "every queue is missing", which would otherwise look like a clean deploy.
func TestFetchQueuesRejects403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"not_found","reason":"no access to this vhost"}`))
	}))
	defer srv.Close()

	s := settingsForTest(srv)
	if _, _, err := s.FetchQueues(context.Background()); err == nil {
		t.Fatal("expected an error for a 403, got nil")
	}
}

func TestAMQPURLEncodesVhost(t *testing.T) {
	tests := []struct {
		name    string
		vhost   string
		tls     bool
		wantURL string
	}{
		{"empty vhost is root", "", false, "amqp://u:p@host:5672/"},
		{"explicit root vhost", "/", false, "amqp://u:p@host:5672/"},
		{"named vhost is escaped", "prod/team", false, "amqp://u:p@host:5672/prod%2Fteam"},
		{"tls switches scheme", "", true, "amqps://u:p@host:5671/"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Settings{Host: "host", Port: 5672, User: "u", Password: "p", Vhost: tc.vhost, TLSEnabled: tc.tls}
			if tc.tls {
				s.Port = 5671
			}
			if got := s.amqpURL(); got != tc.wantURL {
				t.Errorf("amqpURL() = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

func TestMgmtURLFollowsTLSToggle(t *testing.T) {
	if got := (Settings{Host: "h", MgmtPort: 15672}).mgmtBaseURL(); got != "http://h:15672" {
		t.Errorf("mgmtBaseURL() = %q, want http://h:15672", got)
	}
	if got := (Settings{Host: "h", MgmtPort: 15672, MgmtTLS: true}).mgmtBaseURL(); got != "https://h:15672" {
		t.Errorf("mgmtBaseURL() = %q, want https://h:15672", got)
	}
}

// settingsForTest points Settings at the httptest server, deriving the port from
// its URL so TLS stays off.
func settingsForTest(srv *httptest.Server) Settings {
	return Settings{
		Host:         "127.0.0.1",
		MgmtUser:     "closy",
		MgmtPassword: "secret",
		MgmtPort:     portOf(srv),
	}
}

func portOf(srv *httptest.Server) int {
	u, err := url.Parse(srv.URL)
	if err != nil {
		panic(err)
	}
	n, err := strconv.Atoi(u.Port())
	if err != nil {
		panic(err)
	}
	return n
}
