package grpcx

import (
	"context"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeTokenProvider struct {
	enabled bool
	token   string
	err     error
	audence string
}

func (f *fakeTokenProvider) Enabled() bool { return f.enabled }

func (f *fakeTokenProvider) Token(_ context.Context, audience string) (string, error) {
	f.audence = audience
	return f.token, f.err
}

// TestNoopProviderAttachesNoToken verifies self-hosted runs never send an
// infrastructure token (FR-008/FR-025).
func TestNoopProviderAttachesNoToken(t *testing.T) {
	auth := &clientAuth{provider: noopTokenProvider{}, audience: "https://account.run.app"}

	var seen metadata.MD
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		seen, _ = metadata.FromOutgoingContext(ctx)
		return nil
	}

	if err := auth.Unary(context.Background(), "/svc/Method", nil, nil, nil, invoker); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := seen.Get(MetadataAuthorization); len(got) != 0 {
		t.Fatalf("expected no authorization metadata, got %v", got)
	}
}

// TestProviderAttachesBearerToken verifies the infrastructure ID token is
// attached as `authorization: Bearer <token>` for the target audience (T049).
func TestProviderAttachesBearerToken(t *testing.T) {
	provider := &fakeTokenProvider{enabled: true, token: "id-token-value"}
	auth := &clientAuth{provider: provider, audience: "https://account.run.app"}

	var seen metadata.MD
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		seen, _ = metadata.FromOutgoingContext(ctx)
		return nil
	}

	if err := auth.Unary(context.Background(), "/svc/Method", nil, nil, nil, invoker); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := seen.Get(MetadataAuthorization); len(got) != 1 || got[0] != "Bearer id-token-value" {
		t.Fatalf("expected bearer id token, got %v", got)
	}
	if provider.audence != "https://account.run.app" {
		t.Fatalf("expected audience passed to provider, got %q", provider.audence)
	}
}

// TestProviderErrorFailsFast verifies a minting failure aborts the call rather
// than sending a request Cloud Run IAM would reject (SC-022).
func TestProviderErrorFailsFast(t *testing.T) {
	provider := &fakeTokenProvider{enabled: true, err: context.DeadlineExceeded}
	auth := &clientAuth{provider: provider, audience: "https://account.run.app"}

	called := false
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		called = true
		return nil
	}

	if err := auth.Unary(context.Background(), "/svc/Method", nil, nil, nil, invoker); err == nil {
		t.Fatal("expected minting error to abort the call")
	}
	if called {
		t.Fatal("expected invoker not to run when minting fails")
	}
}

// TestContextWithAppJWTUsesSeparateKey verifies the app JWT travels under its
// own key so it never collides with the infrastructure token (FR-034).
func TestContextWithAppJWTUsesSeparateKey(t *testing.T) {
	ctx := ContextWithAppJWT(context.Background(), "closy-jwt-value")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	if got := md.Get(MetadataAppJWT); len(got) != 1 || got[0] != "closy-jwt-value" {
		t.Fatalf("expected app jwt metadata, got %v", got)
	}
	if got := md.Get(MetadataAuthorization); len(got) != 0 {
		t.Fatalf("expected authorization key untouched, got %v", got)
	}
}

// TestAudienceForNormalizesURL ensures the audience matches Cloud Run's
// expectation (scheme + host, no trailing slash).
func TestAudienceForNormalizesURL(t *testing.T) {
	cases := map[string]string{
		"https://account-abc.a.run.app":      "https://account-abc.a.run.app",
		"https://account-abc.a.run.app/":     "https://account-abc.a.run.app",
		"https://account-abc.a.run.app/grpc": "https://account-abc.a.run.app",
		"account:50051":                      "account:50051",
	}
	for in, want := range cases {
		if got := audienceFor(in); got != want {
			t.Fatalf("audienceFor(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestTokenExpiryReadsJWTExp verifies cache refresh scheduling is derived from
// the token's own exp claim (signed test token payload).
func TestTokenExpiryReadsJWTExp(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	payload := `{"exp":` + strconv.FormatInt(exp, 10) + `,"sub":"x"}`
	token := "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".sig"

	got := tokenExpiry(token)
	if got.Unix() != exp {
		t.Fatalf("tokenExpiry = %v, want unix %d", got.Unix(), exp)
	}

	if fallback := tokenExpiry("not-a-jwt"); time.Until(fallback) <= 0 {
		t.Fatal("expected fallback expiry in the future")
	}
}
