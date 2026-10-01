package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

// Settings is the broker connection config, read from the RABBITMQ_* environment
// variables that .env.production (VPS) and .env.aws (CloudAMQP) already define,
// so one code path serves both environments.
type Settings struct {
	Host          string
	Port          int
	User          string
	Password      string
	Vhost         string
	TLSEnabled    bool
	MgmtPort      int
	MgmtTLS       bool
	MgmtUser      string
	MgmtPassword  string
	SkipTLSVerify bool
}

// mgmtQueue is the subset of a RabbitMQ Management API queue object this tool
// needs. QueueType is what amqp091-go's QueueInspect cannot report — declare-ok
// carries no type field — so the type comparison has to come from here.
type mgmtQueue struct {
	Name     string `json:"name"`
	Vhost    string `json:"vhost"`
	Type     string `json:"type"`
	Messages int    `json:"messages"`
}

// mgmtConnection is one entry of /api/connections, used to report connection
// headroom before draining: the CloudAMQP shared plan caps total connections
// and this tool needs one of its own.
type mgmtConnection struct {
	Name string `json:"name"`
}

func (s Settings) amqpURL() string {
	scheme := "amqp"
	if s.TLSEnabled {
		scheme = "amqps"
	}
	vhost := strings.TrimSpace(s.Vhost)
	if vhost == "" || vhost == "/" {
		vhost = "/"
	} else {
		vhost = "/" + url.PathEscape(strings.TrimPrefix(vhost, "/"))
	}
	return fmt.Sprintf("%s://%s:%s@%s:%d%s",
		scheme, url.QueryEscape(s.User), url.QueryEscape(s.Password), s.Host, s.Port, vhost)
}

func (s Settings) tlsConfig() *tls.Config {
	if !s.TLSEnabled {
		return nil
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: s.SkipTLSVerify}
}

// mgmtBaseURL targets /api/queues across all vhosts rather than a single
// vhost-scoped path. A vhost name may contain characters that need escaping,
// and this endpoint avoids that entirely; the tool filters by vhost client-side.
func (s Settings) mgmtBaseURL() string {
	scheme := "http"
	if s.MgmtTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, s.Host, s.MgmtPort)
}

// Dial opens the AMQP connection used for drain, delete, redeclare and replay.
func (s Settings) Dial() (*amqp091.Connection, error) {
	return amqp091.DialConfig(s.amqpURL(), amqp091.Config{
		TLSClientConfig: s.tlsConfig(),
		Heartbeat:       10 * time.Second,
	})
}

// FetchQueues returns the broker-reported type and depth of every queue in the
// configured vhost. A vhost the user cannot see (CloudAMQP returns 403 for a
// mistyped RABBITMQ_VHOST) surfaces as an explicit error rather than as an empty
// result, which would otherwise read as "everything is MISSING".
func (s Settings) FetchQueues(ctx context.Context) (types map[string]string, messages map[string]int, err error) {
	base, err := url.Parse(s.mgmtBaseURL() + "/api/queues")
	if err != nil {
		return nil, nil, fmt.Errorf("build management url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req.SetBasicAuth(s.MgmtUser, s.MgmtPassword)

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: s.SkipTLSVerify},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("management api %s: %w", base, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read management response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("management api %s returned %d: %s",
			base, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var queues []mgmtQueue
	if err := json.Unmarshal(body, &queues); err != nil {
		return nil, nil, fmt.Errorf("parse management response: %w", err)
	}

	types = make(map[string]string)
	messages = make(map[string]int)
	wantVhost := normalizeVhost(s.Vhost)
	for _, q := range queues {
		if normalizeVhost(q.Vhost) != wantVhost {
			continue
		}
		types[q.Name] = q.Type
		messages[q.Name] = q.Messages
	}
	return types, messages, nil
}

// ConnectionCount reports open connections on the instance, so a migrate run
// can show how much headroom is left against the plan's connection cap.
func (s Settings) ConnectionCount(ctx context.Context) (int, error) {
	base, err := url.Parse(s.mgmtBaseURL() + "/api/connections")
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(s.MgmtUser, s.MgmtPassword)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("management api returned %d", resp.StatusCode)
	}
	var conns []mgmtConnection
	if err := json.NewDecoder(resp.Body).Decode(&conns); err != nil {
		return 0, err
	}
	return len(conns), nil
}

func normalizeVhost(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "/"
	}
	return v
}
