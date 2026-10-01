package di

import (
	"fmt"

	adminclient "smart-wardrobe-be/contracts/admin/v1/client"
	grpcx "smart-wardrobe-be/contracts/grpcx"
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

	// 024 T178: the subscription client is no longer built here — see the note on
	// AIContracts. Only the audit-log contract is dialled.
	return &AIContracts{
		AuditLog: adminclient.NewAuditLogClient(accountCC),
	}, cleanup, nil
}
