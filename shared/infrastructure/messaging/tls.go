package messaging

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"smart-wardrobe-be/shared/config"
)

// amqpURL builds the broker URL (T065/FR-004): `amqps` when TLS is enabled
// (managed CloudAMQP), plain `amqp` for self-hosted compatibility. An empty
// vhost selects the default "/" vhost.
func amqpURL(cfg *config.Config) string {
	scheme := "amqp"
	if cfg.RabbitMQ.TLSEnabled {
		scheme = "amqps"
	}
	vhost := strings.TrimSpace(cfg.RabbitMQ.Vhost)
	if vhost == "" || vhost == "/" {
		vhost = "/"
	} else {
		vhost = "/" + url.PathEscape(strings.TrimPrefix(vhost, "/"))
	}
	return fmt.Sprintf("%s://%s:%s@%s:%d%s",
		scheme,
		cfg.RabbitMQ.User,
		cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		vhost,
	)
}

// amqpTLSConfig returns the TLS client config for `amqps`, or nil for
// plaintext `amqp` (T065).
func amqpTLSConfig(cfg *config.Config) *tls.Config {
	if !cfg.RabbitMQ.TLSEnabled {
		return nil
	}
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// mgmtURL builds the RabbitMQ Management API URL (T066): HTTPS when the
// management TLS toggle is on, HTTP otherwise.
func mgmtURL(host string, mgmtPort int, useTLS bool) string {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/api/queues", scheme, host, mgmtPort)
}
