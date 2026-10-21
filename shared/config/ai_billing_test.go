package config

import (
	"strings"
	"testing"
)

func TestNormalizeAIBillingSplitsCommaSeparatedEnvValue(t *testing.T) {
	billing := AIBillingConfig{UnbilledCapabilities: []string{"free_text, embedding"}}
	if err := normalizeAIBilling(&billing); err != nil {
		t.Fatalf("normalizeAIBilling: %v", err)
	}
	if len(billing.UnbilledCapabilities) != 2 {
		t.Fatalf("capabilities = %v, want 2 entries", billing.UnbilledCapabilities)
	}
	if !billing.IsUnbilled(CapabilityFreeText) || !billing.IsUnbilled(CapabilityEmbedding) {
		t.Errorf("expected free_text and embedding to be unbilled, got %v", billing.UnbilledCapabilities)
	}
}

func TestNormalizeAIBillingTrimsAndDeduplicates(t *testing.T) {
	billing := AIBillingConfig{UnbilledCapabilities: []string{" Free_Text ", "free_text", "EMBEDDING"}}
	if err := normalizeAIBilling(&billing); err != nil {
		t.Fatalf("normalizeAIBilling: %v", err)
	}
	if len(billing.UnbilledCapabilities) != 2 {
		t.Fatalf("capabilities = %v, want 2 entries", billing.UnbilledCapabilities)
	}
	if billing.UnbilledCapabilities[0] != CapabilityFreeText || billing.UnbilledCapabilities[1] != CapabilityEmbedding {
		t.Errorf("capabilities = %v, want [free_text embedding]", billing.UnbilledCapabilities)
	}
}

func TestNormalizeAIBillingRejectsUnknownCapability(t *testing.T) {
	billing := AIBillingConfig{UnbilledCapabilities: []string{"free_text,not_a_capability"}}
	err := normalizeAIBilling(&billing)
	if err == nil {
		t.Fatal("expected an error for an unknown capability")
	}
	if !strings.Contains(err.Error(), "not_a_capability") {
		t.Errorf("error = %q, want it to name the offending capability", err)
	}
}

func TestNormalizeAIBillingEmptyDefaultsToEverythingBilled(t *testing.T) {
	billing := AIBillingConfig{UnbilledCapabilities: []string{""}}
	if err := normalizeAIBilling(&billing); err != nil {
		t.Fatalf("normalizeAIBilling: %v", err)
	}
	if len(billing.UnbilledCapabilities) != 0 {
		t.Fatalf("capabilities = %v, want empty", billing.UnbilledCapabilities)
	}
	for _, capability := range knownAICapabilities {
		if billing.IsUnbilled(capability) {
			t.Errorf("%s must be billed when the list is empty", capability)
		}
	}
}

func TestIsUnbilledCapabilityHonoursConfiguredList(t *testing.T) {
	cfg := AIServiceConfig{Billing: AIBillingConfig{UnbilledCapabilities: []string{CapabilityEmbedding}}}
	if !cfg.IsUnbilledCapability(CapabilityEmbedding) {
		t.Error("embedding should be unbilled")
	}
	if cfg.IsUnbilledCapability(CapabilityVision) {
		t.Error("vision should be billed when it is not in the list")
	}
}

func TestLoadConfigDecodesAIBillingThroughEnvAlias(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")
	t.Setenv("AI_UNBILLED_CAPABILITIES", "free_text,embedding")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !cfg.AI.IsUnbilledCapability(CapabilityFreeText) || !cfg.AI.IsUnbilledCapability(CapabilityEmbedding) {
		t.Errorf("billing = %v, want free_text and embedding unbilled", cfg.AI.Billing.UnbilledCapabilities)
	}
}

func TestLoadConfigRejectsUnknownAIBillingCapability(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")
	t.Setenv("AI_UNBILLED_CAPABILITIES", "free_text,typo_capability")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected an error for an unknown capability")
	}
	if !strings.Contains(err.Error(), "typo_capability") {
		t.Errorf("error = %q, want it to name the offending capability", err)
	}
}
