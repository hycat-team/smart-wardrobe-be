package handler

import (
	"errors"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

// This file pins ONE decision: what a chat stream says to the user when it fails
// after it has already started streaming.
//
// The tension it resolves is that the same text has to serve two audiences. The
// log needs the provider or SQL detail; the phone needs a sentence. The previous
// implementation satisfied both by sending the user a hardcoded generic message
// and logging the rest, which is safe but throws away information the user could
// act on — a user whose stream produced nothing was told the same thing as a user
// whose backend crashed.
//
// The two properties that must both hold:
//
//	1. A classified error reaches the user with ITS message. Otherwise the client
//	   cannot distinguish "the AI gave no answer" from "we are broken".
//	2. An unclassified error does NOT leak. Its text may be a SQL fragment or a
//	   provider response, so it must be replaced rather than forwarded. This is the
//	   property that keeps FR-005 phase 9 true, and it is the one that must not be
//	   traded away to satisfy the first.

// TestClassifiedErrorKeepsItsMessage is property 1.
//
// Every case below is an error that can ACTUALLY reach this call site, which is
// narrower than it looks. Only three things do: an error from the AI service's
// stream channel, an error raised while persisting the reply, and the "no reply at
// all" guard. The daily-quota error is deliberately absent — commerce checks it
// BEFORE opening the stream, so it leaves as a plain HTTP error and never becomes
// an SSE event. A test that named it would look like coverage while testing
// nothing.
func TestClassifiedErrorKeepsItsMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			// The real case from the incident. The AI service panicked, its process
			// died, commerce's gRPC client saw the dropped connection and mapped it
			// to a 503 — and the phone was told "Đã xảy ra lỗi hệ thống", which is
			// both vaguer and more alarming than what was actually true.
			name: "upstream unavailable keeps its own text",
			err:  apperror.NewUpstreamUnavailable(errors.New("connection reset")),
			want: "Dịch vụ đang bận, vui lòng thử lại sau ít phút.",
		},
		{
			// The second real case, and the one this change exists for: the stream
			// ended cleanly but produced nothing. `commitCallback` already writes a
			// message written for a human, and it used to be thrown away in favour
			// of the generic sentence.
			name: "empty AI reply keeps its own text",
			err:  apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này."),
			want: "Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.",
		},
		{
			// Reachable when the AI service itself answers ResourceExhausted, e.g. its
			// own per-account attempt cap. From Account, not commerce's quota check.
			name: "too many requests keeps its own text",
			err:  apperror.NewTooManyRequest("Bạn đã dùng hết lượt trò chuyện hôm nay."),
			want: "Bạn đã dùng hết lượt trò chuyện hôm nay.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sseErrorMessage(tc.err); got != tc.want {
				t.Errorf("classified error must reach the client unchanged:\n got = %q\nwant = %q", got, tc.want)
			}
		})
	}
}

// TestUnclassifiedErrorDoesNotLeak is property 2, and it is the one that must not
// regress. It is written as a leak test on purpose: it asserts that detail which
// looks like infrastructure information never reaches the output, rather than
// merely asserting that some generic message came back.
func TestUnclassifiedErrorDoesNotLeak(t *testing.T) {
	leaks := []struct {
		name string
		err  error
	}{
		{"sql detail", errors.New(`pq: duplicate key value violates unique constraint "ux_ledgers_user_period_fallback"`)},
		{"provider detail", errors.New(`gemini API 429: {"error":{"code":429,"status":"RESOURCE_EXHAUSTED"}}`)},
		{"connection detail", errors.New("dial tcp 10.0.1.7:8103: connect: connection refused")},
	}

	for _, tc := range leaks {
		t.Run(tc.name, func(t *testing.T) {
			got := sseErrorMessage(tc.err)

			if got != apperror.GenericInternalMessage {
				t.Fatalf("an unclassified error must be replaced, got = %q", got)
			}
			// Belt and braces: if someone changes `From` so it returns the raw text,
			// the equality assertion above fails too — but only if they change it
			// back. This asserts the property directly so the intent survives.
			for _, secret := range []string{"ux_ledgers_user_period", "gemini", "10.0.1.7", "429"} {
				if strings.Contains(got, secret) {
					t.Errorf("client message must not contain %q, got = %q", secret, got)
				}
			}
		})
	}
}

// `apperror.From(nil)` returns a NIL POINTER, not a populated error, so
// `apperror.From(err).Message` panics if `err` is nil. That trap is recorded here
// because it is the one way this helper could ever take down a request — and it is
// why `sseErrorMessage` is only ever called inside `if err != nil`, and why the
// helper carries no nil guard of its own: a guard for a case the single call site
// already excludes would be noise that reads as if the case were reachable.
func TestApperrorFromReturnsNilForNilInput(t *testing.T) {
	if apperror.From(nil) != nil {
		t.Error("From(nil) must return nil, not a populated error")
	}
}

// An error produced by `Wrap` carries the generic message but keeps the cause
// internal. This is the shape most persistence failures actually have, so it gets
// its own case rather than relying on the table above.
func TestWrappedInternalErrorStaysGeneric(t *testing.T) {
	wrapped := apperror.Wrap(errors.New("connection refused"))
	got := sseErrorMessage(wrapped)

	if strings.Contains(got, "connection refused") {
		t.Errorf("the cause must stay server-side, got = %q", got)
	}
	if got != apperror.GenericInternalMessage {
		t.Errorf("a wrapped error with no explicit text must stay generic, got = %q", got)
	}
}
