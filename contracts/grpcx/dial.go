package grpcx

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// DialService opens one shared gRPC connection to a service base URL.
// https targets use system-root TLS (Cloud Run); http or bare host:port
// targets use plaintext (local dev). The provider attaches the infrastructure
// ID token on managed platforms (T049); outside managed environments the
// provider is a no-op and no token is sent.
func DialService(rawURL string, provider TokenProvider) (*grpc.ClientConn, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("service base URL must not be empty")
	}
	target := rawURL
	useTLS := true
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		target = u.Host
		useTLS = u.Scheme != "http"
	}

	auth := &clientAuth{provider: provider, audience: audienceFor(rawURL)}
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(auth.Unary),
		grpc.WithChainStreamInterceptor(auth.Stream),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	return grpc.NewClient(target, opts...)
}

// ContextWithAppJWT attaches the application-layer Closy JWT to outgoing gRPC
// metadata under MetadataAppJWT. It is a separate key from the infrastructure
// ID token that DialService attaches, so both layers coexist (FR-034).
func ContextWithAppJWT(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, MetadataAppJWT, token)
}
