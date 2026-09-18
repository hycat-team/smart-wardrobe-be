package grpcx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"smart-wardrobe-be/shared/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Metadata keys for the two independent auth layers on gRPC calls (FR-034).
const (
	// MetadataAuthorization carries the infrastructure-layer Google ID token.
	// Cloud Run IAM reads this key for gRPC requests, so it MUST stay
	// "authorization" (T049).
	MetadataAuthorization = "authorization"
	// MetadataAppJWT carries the application-layer Closy JWT. It is a custom
	// key because "authorization" is consumed by Cloud Run IAM for gRPC; both
	// client (T049) and server (T039/T073a) MUST use this constant.
	MetadataAppJWT = "x-closy-jwt"
)

// metadataIdentityEndpoint is the GCE/Cloud Run metadata server endpoint that
// mints an ID token for the instance service account. Using it keeps the
// dependency surface at stdlib (no Google SDK), which suits FR-025/FR-033.
const metadataIdentityEndpoint = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity"

const (
	metadataFlavorHeader = "Metadata-Flavor"
	metadataFlavorValue  = "Google"
	refreshAhead         = 5 * time.Minute
	fallbackTokenTTL     = 45 * time.Minute
)

// TokenProvider supplies the infrastructure ID token for a target audience.
// It is disabled (Enabled() == false) outside managed environments so local
// and VPS runs attach no infrastructure token.
type TokenProvider interface {
	Enabled() bool
	Token(ctx context.Context, audience string) (string, error)
}

type noopTokenProvider struct{}

// NewTokenProvider selects the provider for the running environment: a
// metadata-server provider on managed platforms (Cloud Run), otherwise a no-op
// so self-hosted runs never depend on GCP (FR-008/FR-025).
func NewTokenProvider(cfg *config.Config) TokenProvider {
	if cfg == nil || !config.IsManagedEnv(cfg.Server.Env) {
		return noopTokenProvider{}
	}
	return newMetadataTokenProvider()
}

func (noopTokenProvider) Enabled() bool { return false }

func (noopTokenProvider) Token(context.Context, string) (string, error) { return "", nil }

type cachedIDToken struct {
	value     string
	expiresAt time.Time
}

type metadataTokenProvider struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]cachedIDToken
}

func newMetadataTokenProvider() *metadataTokenProvider {
	return &metadataTokenProvider{
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]cachedIDToken),
	}
}

func (p *metadataTokenProvider) Enabled() bool { return true }

// Token returns a cached ID token for the audience, minting a new one when the
// cached token is missing or close to expiry.
func (p *metadataTokenProvider) Token(ctx context.Context, audience string) (string, error) {
	if audience == "" {
		return "", fmt.Errorf("grpcx: audience is required to mint an ID token")
	}

	p.mu.Lock()
	if c, ok := p.cache[audience]; ok && time.Until(c.expiresAt) > refreshAhead {
		p.mu.Unlock()
		return c.value, nil
	}
	p.mu.Unlock()

	token, expiresAt, err := p.mint(ctx, audience)
	if err != nil {
		return "", err
	}

	p.mu.Lock()
	p.cache[audience] = cachedIDToken{value: token, expiresAt: expiresAt}
	p.mu.Unlock()

	return token, nil
}

func (p *metadataTokenProvider) mint(ctx context.Context, audience string) (string, time.Time, error) {
	endpoint := metadataIdentityEndpoint + "?audience=" + url.QueryEscape(audience)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("grpcx: build metadata request: %w", err)
	}
	req.Header.Set(metadataFlavorHeader, metadataFlavorValue)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("grpcx: call metadata server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("grpcx: metadata server returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("grpcx: read metadata response: %w", err)
	}

	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", time.Time{}, fmt.Errorf("grpcx: metadata server returned an empty token")
	}
	return token, tokenExpiry(token), nil
}

// tokenExpiry reads the JWT `exp` claim so the cache refreshes before expiry,
// falling back to a conservative TTL when the claim cannot be parsed.
func tokenExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Now().Add(fallbackTokenTTL)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Now().Add(fallbackTokenTTL)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Now().Add(fallbackTokenTTL)
	}
	return time.Unix(claims.Exp, 0)
}

// audienceFor normalizes a service base URL into the ID-token audience
// expected by Cloud Run IAM (scheme + host, no trailing slash or path).
func audienceFor(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(rawURL, "/")
	}
	return u.Scheme + "://" + u.Host
}

type clientAuth struct {
	provider TokenProvider
	audience string
}

func (c *clientAuth) Unary(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx, err := c.withToken(ctx)
	if err != nil {
		return err
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func (c *clientAuth) Stream(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx, err := c.withToken(ctx)
	if err != nil {
		return nil, err
	}
	return streamer(ctx, desc, cc, method, opts...)
}

// withToken attaches the infrastructure ID token when the provider is enabled.
// A minting failure fails fast rather than sending a request Cloud Run IAM
// would reject with 403 (SC-022).
func (c *clientAuth) withToken(ctx context.Context) (context.Context, error) {
	if c.provider == nil || !c.provider.Enabled() {
		return ctx, nil
	}
	token, err := c.provider.Token(ctx, c.audience)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return ctx, nil
	}
	return metadata.AppendToOutgoingContext(ctx, MetadataAuthorization, "Bearer "+token), nil
}
