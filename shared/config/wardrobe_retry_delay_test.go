package config

import (
	"strings"
	"testing"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
)

// The analyze retry delay is a broker queue TTL (x-message-ttl cannot change in
// place), so a config value the broker will not honour must fail at startup
// instead of silently retrying at a different pace than the operator expects.
func TestValidateWardrobeRetryDelaysAcceptsTheBrokerTTLs(t *testing.T) {
	cfg := &Config{Wardrobe: WardrobeProcessing{
		RetryDelay1Seconds: eventconstants.FashionAnalyzeRetryDelay1Millis / 1000,
		RetryDelay2Seconds: eventconstants.FashionAnalyzeRetryDelay2Millis / 1000,
		RetryDelay3Seconds: eventconstants.FashionAnalyzeRetryDelay3Millis / 1000,
	}}

	if err := validateWardrobeRetryDelays(cfg); err != nil {
		t.Fatalf("the queue TTLs must validate, got: %v", err)
	}
}

func TestValidateWardrobeRetryDelaysRejectsAValueTheBrokerCannotHonour(t *testing.T) {
	cfg := &Config{Wardrobe: WardrobeProcessing{
		RetryDelay1Seconds: 120,
		RetryDelay2Seconds: eventconstants.FashionAnalyzeRetryDelay2Millis / 1000,
		RetryDelay3Seconds: eventconstants.FashionAnalyzeRetryDelay3Millis / 1000,
	}}

	err := validateWardrobeRetryDelays(cfg)
	if err == nil {
		t.Fatal("a retry delay that differs from the queue TTL must be rejected")
	}
	if want := "wardrobe.retry_delay_1_seconds = 120"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to name %q", err, want)
	}
}
