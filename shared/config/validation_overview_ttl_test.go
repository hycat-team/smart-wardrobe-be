package config

import "testing"

// 024 T146 (FR-074): the overview cache TTL has a CEILING, not just a value.
//
// This TTL is how long a freshly granted plan can remain invisible to the customer,
// because entitlement decisions read this cache. Nothing else bounds it, so without
// a check a deployment could set it to an hour and silently break a customer-facing
// guarantee: the customer pays Premium and is refused by the AI service for the
// duration.
func TestOverviewTTLMustNotExceedTheFRO74Ceiling(t *testing.T) {
	cfg := &Config{}
	cfg.Cache.OverviewTTLSeconds = MaxOverviewTTLSeconds + 1

	err := validateConfig(cfg)
	if err == nil {
		t.Fatalf("TTL %d phai bi tu choi khi vuot tran FR-074 (%d)", cfg.Cache.OverviewTTLSeconds, MaxOverviewTTLSeconds)
	}
}

func TestOverviewTTLAcceptsExactlyTheCeiling(t *testing.T) {
	cfg := &Config{}
	cfg.Cache.OverviewTTLSeconds = MaxOverviewTTLSeconds

	// A valid config is not expected to pass the WHOLE validator, which checks many
	// unrelated fields; this asserts only that the TTL check is not what rejects it.
	if err := validateConfig(cfg); err != nil && err.Error() == "cache overview_ttl_seconds must not exceed 60 seconds (FR-074): a longer cache would keep a freshly granted plan invisible to the customer" {
		t.Fatal("dung o tran FR-074 khong duoc bi tu choi")
	}
}

func TestOverviewTTLZeroMeansUseTheDefault(t *testing.T) {
	cfg := &Config{}
	cfg.Cache.OverviewTTLSeconds = 0

	if err := validateConfig(cfg); err != nil && err.Error() == "cache overview_ttl_seconds must not exceed 60 seconds (FR-074): a longer cache would keep a freshly granted plan invisible to the customer" {
		t.Fatal("gia tri 0 nghia la dung mac dinh, khong duoc coi la vuot tran")
	}
}

func TestOverviewTTLMustNotBeNegative(t *testing.T) {
	cfg := &Config{}
	cfg.Cache.OverviewTTLSeconds = -1

	if err := validateConfig(cfg); err == nil {
		t.Fatal("TTL am phai bi tu choi")
	}
}
