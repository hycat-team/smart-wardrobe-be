package sse

import (
	"testing"
	"time"
)

func TestRealtimeMetrics_SubscriberGauge(t *testing.T) {
	before := GetRealtimeMetrics()["realtime_subscribers"]

	m := newTestManager(5, time.Minute)
	ch := m.Subscribe("t-metrics")
	if got := GetRealtimeMetrics()["realtime_subscribers"]; got != before+1 {
		t.Fatalf("subscribers after subscribe = %d, want %d", got, before+1)
	}
	m.Unsubscribe("t-metrics", ch)
	if got := GetRealtimeMetrics()["realtime_subscribers"]; got != before {
		t.Fatalf("subscribers after unsubscribe = %d, want %d", got, before)
	}
}

func TestRealtimeMetrics_ExposesAllKeys(t *testing.T) {
	m := GetRealtimeMetrics()
	for _, k := range []string{
		"realtime_publish_total",
		"realtime_publish_failed",
		"realtime_relay_total",
		"realtime_relay_failed",
		"realtime_subscribers",
	} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing metric key %q", k)
		}
	}
}

func TestRealtimeMetrics_PublishCounters(t *testing.T) {
	before := GetRealtimeMetrics()
	IncRealtimePublishTotal()
	IncRealtimePublishFailed()
	after := GetRealtimeMetrics()
	if after["realtime_publish_total"] != before["realtime_publish_total"]+1 {
		t.Fatal("publish total not incremented")
	}
	if after["realtime_publish_failed"] != before["realtime_publish_failed"]+1 {
		t.Fatal("publish failed not incremented")
	}
}
