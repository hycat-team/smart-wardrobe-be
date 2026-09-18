package di

import (
	"fmt"

	adminclient "smart-wardrobe-be/contracts/admin/v1/client"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	subscriptionclient "smart-wardrobe-be/contracts/subscription/v1/client"
	sharedconfig "smart-wardrobe-be/shared/config"
)

// NewAIContracts dials Account once and builds all remote contracts AI
// consumes. The connection is shared for the process lifetime; the returned
// cleanup closes it.
func NewAIContracts(cfg *sharedconfig.Config) (*AIContracts, func(), error) {
	provider := grpcx.NewTokenProvider(cfg)
	accountCC, err := grpcx.DialService(cfg.Ingress.AccountBaseURL, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("dial account service: %w", err)
	}

	cleanup := func() {
		_ = accountCC.Close()
	}

	return &AIContracts{
		UserSubscription: subscriptionclient.NewSubscriptionClient(accountCC),
		AuditLog:         adminclient.NewAuditLogClient(accountCC),
	}, cleanup, nil
}
