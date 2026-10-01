package grant

import (
	"context"
	"errors"
	"testing"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	sharedevent "smart-wardrobe-be/shared/application/event"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
)

// 024 T152: the reason this file exists.
//
// `startConsumerLoop` (`rabbitmq.go:120-129`) has exactly three outcomes, and it
// picks between them with `errors.As` against two marker types:
//
//	RetryableError -> Nack(requeue=true)  -> redelivered, up to x-delivery-limit
//	PermanentError -> Nack(requeue=false) -> dead-lettered
//	default        -> Nack(requeue=false) -> dead-lettered
//
// The grant use case returned raw repository errors and bare `RejectionError`
// values, so BOTH landed in `default` and were dead-lettered on the first delivery.
// The queue's `x-delivery-limit: 5`, configured specifically so a transient
// failure survives long enough to be retried, never fired. A two-second database
// blip permanently destroyed a paying customer's AI entitlement.
//
// These tests assert the classification, because "the wrapper is there" is a weaker
// claim than "the broker will requeue this".

// retryableAndPermanentAreDistinguished is the contract the switch in
// rabbitmq.go depends on. If this stops holding, the classification below is
// meaningless regardless of what it returns.
func TestRetryableAndPermanentErrorsAreDistinguishable(t *testing.T) {
	base := errors.New("underlying")

	var retryable sharedevent.RetryableError
	if !errors.As(error(retryableFor(base)), &retryable) {
		t.Error("RetryableError phải nhận diện được bằng errors.As")
	}
	var permanent sharedevent.PermanentError
	if errors.As(error(retryableFor(base)), &permanent) {
		t.Error("RetryableError không được khớp PermanentError - nếu không, transient sẽ bị DLQ")
	}

	if !errors.As(error(permanentFor(base)), &permanent) {
		t.Error("PermanentError phải nhận diện được bằng errors.As")
	}
	if errors.As(error(permanentFor(base)), &retryable) {
		t.Error("PermanentError không được khớp RetryableError - nếu không, poison message sẽ bị requeue vô hạn")
	}

	// A bare error must match NEITHER, which is what sends it to `default`.
	if errors.As(base, &retryable) || errors.As(base, &permanent) {
		t.Error("lỗi trần không được khớp marker nào")
	}

	// Unwrap must survive, so the repository detail is not lost.
	if !errors.Is(error(retryableFor(base)), base) {
		t.Error("RetryableError phải Unwrap để giữ lỗi gốc")
	}
	if !errors.Is(error(permanentFor(base)), base) {
		t.Error("PermanentError phải Unwrap để giữ lỗi gốc")
	}
}

// Each repository failure path must produce a RetryableError, because each one is a
// database call that will plausibly succeed on redelivery.
func TestTransientRepositoryFailuresAreRetryable(t *testing.T) {
	cases := []struct {
		name  string
		build func() (*GrantUseCase, *fakeGrantRepo)
	}{
		{
			name: "read existing grant by source event id",
			build: func() (*GrantUseCase, *fakeGrantRepo) {
				repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), bySourceErr: errors.New("connection reset")}
				return newHandler(repo, &recordingUow{}), repo
			},
		},
		{
			name: "read AI cost policy",
			build: func() (*GrantUseCase, *fakeGrantRepo) {
				repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), lookupErr: errors.New("connection reset")}
				return newHandler(repo, &recordingUow{}), repo
			},
		},
		{
			name: "read current grant for the stale-event check",
			build: func() (*GrantUseCase, *fakeGrantRepo) {
				repo := &fakeGrantRepo{
					policy:      strictPolicyWithAllOperations(),
					currentErr:  errors.New("connection reset"),
					bySourceErr: nil,
				}
				// The stale check runs after the policy lookup, so the policy has to
				// resolve first or the test would stop at the wrong line.
				repo.lookupErr = nil
				return newHandler(repo, &recordingUow{}), repo
			},
		},
		{
			name: "write the grant",
			build: func() (*GrantUseCase, *fakeGrantRepo) {
				repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), createErr: errors.New("deadlock detected")}
				return newHandler(repo, &recordingUow{}), repo
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc, _ := tc.build()
			err := uc.HandlePlanChanged(context.Background(), validPayload())

			var retryable sharedevent.RetryableError
			if !errors.As(err, &retryable) {
				t.Fatalf("lỗi tạm thời phải là RetryableError để broker requeue, thực tế = %#v", err)
			}
			// A transient failure must NOT be mistaken for a rejection: the metrics
			// counters are keyed by reason and an operator alerts on them.
			var rejection *RejectionError
			if errors.As(err, &rejection) {
				t.Error("lỗi tạm thời không được báo thành RejectionError - sẽ tính nhầm vào counter từ chối")
			}
		})
	}
}

// A rejection must be permanent: the same message redelivered would fail the same
// way, so retrying it only delays the dead-letter signal an operator needs.
func TestRejectionsArePermanent(t *testing.T) {
	repo := &fakeGrantRepo{policy: nil}
	uc := newHandler(repo, &recordingUow{})

	err := uc.HandlePlanChanged(context.Background(), validPayload())

	var permanent sharedevent.PermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("từ chối phải là PermanentError, thực tế = %#v", err)
	}
	var retryable sharedevent.RetryableError
	if errors.As(err, &retryable) {
		t.Error("từ chối không được requeue - sẽ xoay vòng vô hạn trên cùng một message")
	}

	// The reason must still be reachable: it is what the metric and the log carry.
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatal("mã lý do phải vẫn truy xuất được qua PermanentError")
	}
	if rejection.Reason != ReasonAICostPolicyNotFound {
		t.Errorf("mã lý do = %q, cần %q", rejection.Reason, ReasonAICostPolicyNotFound)
	}
}

// A payload that fails validation cannot be fixed by redelivery either.
func TestParseFailuresArePermanent(t *testing.T) {
	uc := newHandler(&fakeGrantRepo{}, &recordingUow{})

	payload := validPayload()
	payload.EventVersion = 0 // unsupported version

	err := uc.HandlePlanChanged(context.Background(), payload)

	var permanent sharedevent.PermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("payload hỏng phải bị từ chối vĩnh viễn, thực tế = %#v", err)
	}
}

// A duplicate is not a failure at all. The outbox guarantees at-least-once
// delivery, so this path must return nil and must not touch either counter - if it
// errored, ordinary redelivery would look like a system fault.
func TestDuplicateEventIsAcknowledgedNotFailed(t *testing.T) {
	repo := &fakeGrantRepo{
		policy:        strictPolicyWithAllOperations(),
		existingGrant: &aiexecutor_entities.AIPolicyGrant{},
	}
	uc := newHandler(repo, &recordingUow{})

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("sự kiện trùng lặp phải được xác nhận, không phải báo lỗi: %v", err)
	}
}

func retryableFor(err error) error { return sharedevent.RetryableError{Err: err} }

func permanentFor(err error) error { return sharedevent.PermanentError{Err: err} }

var _ shared_repos.IUnitOfWork = (*recordingUow)(nil)
