package config

import (
	"fmt"
	"strings"
)

// AI capability names. They match the provider blocks in configs/config*.yaml
// and the environment prefixes in .env*, so the billing list and the provider
// configuration use one vocabulary.
const (
	CapabilityVision             = "vision"
	CapabilityEmbedding          = "embedding"
	CapabilityChatText           = "chat_text"
	CapabilityRecommendationText = "recommendation_text"
	CapabilityFreeText           = "free_text"
)

var knownAICapabilities = []string{
	CapabilityVision,
	CapabilityEmbedding,
	CapabilityChatText,
	CapabilityRecommendationText,
	CapabilityFreeText,
}

// AIBillingConfig classifies which provider keys the team is NOT charged for.
//
// The list lives in configuration rather than in code because whether a key is
// billed is an operational fact, not an application decision: rotating a free
// key or moving a capability onto a paid account must not require a release.
//
// The default is deliberately "everything billed": a capability that is absent
// is treated as billed, so a missing entry can only overstate cost, never hide
// it. `unbilled_capabilities` is fed by AI_UNBILLED_CAPABILITIES through the
// YAML alias `${AI_UNBILLED_CAPABILITIES}`, the same pattern used by every
// other env-backed setting.
type AIBillingConfig struct {
	UnbilledCapabilities []string `mapstructure:"unbilled_capabilities"`
}

// IsUnbilled reports whether the capability's provider key is not billed.
func (c AIBillingConfig) IsUnbilled(capability string) bool {
	for _, unbilled := range c.UnbilledCapabilities {
		if unbilled == capability {
			return true
		}
	}
	return false
}

// IsUnbilledCapability is the AIServiceConfig-level convenience the AI
// infrastructure calls after a provider wins, so it does not need to know how
// the configuration is stored.
func (c AIServiceConfig) IsUnbilledCapability(capability string) bool {
	return c.Billing.IsUnbilled(capability)
}

// normalizeAIBilling accepts both the comma-separated string produced by
// `${AI_UNBILLED_CAPABILITIES}` and a YAML list, trims and deduplicates the
// names, and refuses unknown capabilities instead of silently treating a typo
// as billed.
func normalizeAIBilling(billing *AIBillingConfig) error {
	normalized := make([]string, 0, len(billing.UnbilledCapabilities))
	seen := make(map[string]struct{}, len(billing.UnbilledCapabilities))

	for _, raw := range billing.UnbilledCapabilities {
		for _, part := range strings.Split(raw, ",") {
			name := strings.ToLower(strings.TrimSpace(part))
			if name == "" {
				continue
			}
			if !isKnownAICapability(name) {
				return fmt.Errorf(
					"ai.billing.unbilled_capabilities: unknown capability %q (allowed: %s)",
					name,
					strings.Join(knownAICapabilities, ", "),
				)
			}
			if _, duplicated := seen[name]; duplicated {
				continue
			}
			seen[name] = struct{}{}
			normalized = append(normalized, name)
		}
	}

	billing.UnbilledCapabilities = normalized
	return nil
}

func isKnownAICapability(name string) bool {
	for _, known := range knownAICapabilities {
		if known == name {
			return true
		}
	}
	return false
}
