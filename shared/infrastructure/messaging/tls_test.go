package messaging

import (
	"testing"

	"smart-wardrobe-be/shared/config"
)

func tlsTestConfig() config.Config {
	var cfg config.Config
	cfg.RabbitMQ.Host = "localhost"
	cfg.RabbitMQ.Port = 5672
	cfg.RabbitMQ.User = "guest"
	cfg.RabbitMQ.Password = "guest"
	cfg.RabbitMQ.ManagementPort = 15672
	return cfg
}

// TestAmqpTLSOffUsesPlaintext verifies the default self-hosted profile uses
// amqp without TLS (T067, FR-004).
func TestAmqpTLSOffUsesPlaintext(t *testing.T) {
	cfg := tlsTestConfig()
	cfg.RabbitMQ.TLSEnabled = false

	url := amqpURL(&cfg)
	if len(url) < 7 || url[:7] != "amqp://" {
		t.Fatalf("expected amqp:// scheme, got %q", url)
	}
	if amqpTLSConfig(&cfg) != nil {
		t.Fatal("expected nil TLS config when rabbitmq.tls_enabled=false")
	}
}

// TestAmqpTLSOnUsesAmqps verifies managed CloudAMQP uses amqps + TLS (T067).
func TestAmqpTLSOnUsesAmqps(t *testing.T) {
	cfg := tlsTestConfig()
	cfg.RabbitMQ.TLSEnabled = true

	url := amqpURL(&cfg)
	if len(url) < 8 || url[:8] != "amqps://" {
		t.Fatalf("expected amqps:// scheme, got %q", url)
	}
	if amqpTLSConfig(&cfg) == nil {
		t.Fatal("expected non-nil TLS config when rabbitmq.tls_enabled=true")
	}
}

// TestMgmtURLScheme verifies the management/gauge API uses HTTP by default
// and HTTPS when the management TLS toggle is on (T066/T067).
func TestMgmtURLScheme(t *testing.T) {
	if got := mgmtURL("broker", 15672, false); got != "http://broker:15672/api/queues" {
		t.Fatalf("unexpected mgmt url %q", got)
	}
	if got := mgmtURL("broker", 15671, true); got != "https://broker:15671/api/queues" {
		t.Fatalf("unexpected mgmt url %q", got)
	}
}
