package campaign

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

// 026 T054 (FR-024) — the campaign CODE cannot be changed.
//
// WHY THIS IS A TEST ABOUT A FIELD THAT DOES NOT EXIST. FR-024 is not "the code is
// validated" — there is no `code` in `CampaignUpdateReq` to validate, and that absence
// is the mechanism. `campaign_claims` and `campaign_sweep_runs` reference the code as
// TEXT with no foreign key (data-model.md §2.1), so a rename would leave every grant
// ever made pointing at a code that no longer exists: a campaign's whole history
// orphaned, and the FR-069 read endpoints answering "no such campaign" for grants that
// demonstrably happened.
//
// A body field that had to be REFUSED would be a second way to ask the question the URL
// already answers, and it would make "ignored" and "rejected" two different behaviours
// for the same mistake. Not having the field makes the answer structural.
//
// THE `UPDATE` STATEMENT IS THE SECOND HALF, and it is not optional either.
// `CampaignRepository.Update` names the five mutable columns one by one rather than
// saving the struct, so `code` cannot be rewritten by a later field added to the
// entity. That is asserted in the repository's own test; what is asserted HERE is the
// DTO and the use case, which is the layer an operator's request actually passes
// through.

// TestUpdateRequestHasNoCodeField is FR-024's primary guarantee, stated as reflection
// rather than as behaviour.
//
// Reflection rather than a compile-time check because the failure this guards against
// is someone ADDING the field back: a test that merely sent a body with `code` would
// still pass on a DTO that has one, because a value equal to the current code changes
// nothing. Naming the field in a reflection assertion is what fails on its addition.
func TestUpdateRequestHasNoCodeField(t *testing.T) {
	reqType := reflect.TypeOf(sub_dto.CampaignUpdateReq{})

	// Go names first, so a rename of the JSON tag alone is caught as well.
	for _, goName := range []string{"Code", "CampaignCode"} {
		if _, found := reqType.FieldByName(goName); found {
			t.Errorf("FR-024: `CampaignUpdateReq` must not carry %s — the code is immutable after "+
				"creation and comes from the URL, never from a body", goName)
		}
	}

	// And no tag anywhere claims `code`, so a client cannot be told the field exists.
	tagClaimed := false
	for i := 0; i < reqType.NumField(); i++ {
		if tag := reqType.Field(i).Tag.Get("json"); tag == "code" {
			tagClaimed = true
		}
	}
	if tagClaimed {
		t.Error("FR-024: no field of `CampaignUpdateReq` may be tagged `json:\"code\"`")
	}
}

// TestUpdateIgnoresACodeSentInTheBody is the behavioural half, and it goes through
// `encoding/json` on purpose.
//
// The use case never sees raw JSON — the binder has already decoded it — so a test that
// passed a hand-built struct would prove nothing about what a client actually sends.
// Decoding the body the way `validation.BindJSON` does, and then asserting the row did
// not move, is what makes this about the wire rather than about a Go type.
//
// THE BODY BELOW ALSO CHANGES A REAL FIELD, and that is the point of the control: a
// body whose `code` is honoured would rename nothing while still editing `quota`, so
// asserting only "200 and no error" would pass for an implementation that quietly
// ignored the code it was supposed to refuse. Asserting `quota` moved PROVES the body
// was read — and proves the code in it was discarded rather than the whole body.
func TestUpdateIgnoresACodeSentInTheBody(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	// A client that believes it may rename: the URL says `tet-2027` and the body
	// says `tet-2028`, alongside a legitimate budget edit.
	var req sub_dto.CampaignUpdateReq
	raw := []byte(`{"version":1,"code":"tet-2028","quota":120}`)
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("the body must decode as a valid update: %v", err)
	}

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", &req, nil)
	if err != nil {
		t.Fatalf("FR-024: a `code` in the body must be ignored, not refused, got %v", err)
	}

	// The budget DID move — so the body was read, and the code in it was discarded.
	if summary.Quota != 120 {
		t.Fatalf("the control failed: quota must have been edited, got %d — the body was never read, "+
			"so this test would pass for the wrong reason", summary.Quota)
	}

	// And the KEY is still the one the URL named, in the row, in the response and in
	// the audit payload — the three places a renamed code would show up.
	row := f.repo.row("tet-2027")
	if row == nil {
		t.Fatal("the campaign must still be stored under its original code")
	}
	if f.repo.row("tet-2028") != nil {
		t.Error("FR-024: the edit created a second campaign under the code sent in the body")
	}
	if summary.Code != "tet-2027" {
		t.Errorf("the response must name the URL's code, got %q", summary.Code)
	}
	if summary.Version != 2 {
		t.Errorf("the rename must not have produced a second write; expected one increment, got version %d",
			summary.Version)
	}
	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: expected exactly one audit row, got %d", len(records))
	}
	if got := records[0].Payload["campaignCode"]; got != "tet-2027" {
		t.Errorf("FR-063/FR-024: the audit payload must name the URL's code, got %v", got)
	}
}

// TestACodeThatDiffersFromThePathIsNotWrittenAnywhere closes the loop from the other
// side: the stub is asked about BOTH codes, and only the path's code is ever written.
//
// `Update` reads the row by the code from the PATH — the argument it was given — so a
// body value has no way to reach a query, and this asserts that at the level where the
// repository calls happen.
func TestACodeThatDiffersFromThePathIsNotWrittenAnywhere(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	f.repo.markGranted("tet-2027")

	var req sub_dto.CampaignUpdateReq
	// A rename AND a budget change together: the budget change is what FR-020 refuses,
	// and the refusal must be about `tet-2027` — the code the operator was looking at —
	// not about the one in the body.
	if err := json.Unmarshal([]byte(`{"version":1,"code":"tet-2028","quota":120}`), &req); err != nil {
		t.Fatalf("the body must decode as a valid update: %v", err)
	}

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", &req, nil)
	if err == nil {
		t.Fatal("a budget change on a campaign that has granted must still be refused")
	}
	if got := httpStatusOf(t, err); got != 422 {
		t.Errorf("FR-021: expected 422, got %d (%v)", got, err)
	}
	if len(f.repo.ledgerReads) != 1 || f.repo.ledgerReads[0] != "tet-2027" {
		t.Errorf("the ledger must be consulted about the PATH's code only, got %v", f.repo.ledgerReads)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused edit must record nothing, got %d", len(f.audit.all()))
	}
}
