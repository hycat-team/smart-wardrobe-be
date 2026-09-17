package di

import (
	"fmt"

	adminclient "smart-wardrobe-be/contracts/admin/v1/client"
	aiclient "smart-wardrobe-be/contracts/ai/v1/client"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	identityclient "smart-wardrobe-be/contracts/identity/v1/client"
	subscriptionclient "smart-wardrobe-be/contracts/subscription/v1/client"
	sharedconfig "smart-wardrobe-be/shared/config"
)

// NewCommerceContracts dials Account and AI once and builds all remote
// contracts Commerce consumes. Connections are shared for the process
// lifetime; the returned cleanup closes them.
func NewCommerceContracts(cfg *sharedconfig.Config) (*CommerceContracts, func(), error) {
	provider := grpcx.NewTokenProvider(cfg)
	accountCC, err := grpcx.DialService(cfg.Ingress.AccountBaseURL, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("dial account service: %w", err)
	}
	aiCC, err := grpcx.DialService(cfg.Ingress.AIBaseURL, provider)
	if err != nil {
		_ = accountCC.Close()
		return nil, nil, fmt.Errorf("dial ai service: %w", err)
	}

	cleanup := func() {
		_ = accountCC.Close()
		_ = aiCC.Close()
	}

	subscriptionClient := subscriptionclient.NewSubscriptionClient(accountCC)
	return &CommerceContracts{
		UserBrief:             identityclient.NewUserBriefClient(accountCC),
		UserDashboardReader:   identityclient.NewUserDashboardReaderClient(accountCC),
		UserSubscription:      subscriptionClient,
		UserQuota:             subscriptionClient,
		SubscriptionDashboard: subscriptionclient.NewSubscriptionDashboardReaderClient(accountCC),
		AuditLog:              adminclient.NewAuditLogClient(accountCC),
		AIOrchestrator:        aiclient.NewAIOrchestratorClient(aiCC),
		AICostDashboardReader: aiclient.NewAICostDashboardReaderClient(aiCC),
	}, cleanup, nil
}
