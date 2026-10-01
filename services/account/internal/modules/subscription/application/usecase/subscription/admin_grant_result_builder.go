package subscription

import (
	"fmt"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T147: turning item outcomes into a response, and the time budget.

// Split out of admin_grant_batch_uc.go for code-hygiene H1. Everything here is a
// pure function of its inputs: no repository, no transaction, no clock. That is
// why they are grouped - they are the part of the path a test can exercise
// without a database, and keeping them together says so.

func grantSuccessMessage(plan *entities.SubscriptionPlan, expiresAt *time.Time) string {
	if expiresAt == nil {
		return fmt.Sprintf(mapper.MsgAdminGrantNoExpiry, plan.Name)
	}
	return fmt.Sprintf(mapper.MsgAdminGrantSuccess, plan.Name, expiresAt.Local().Format("02/01/2006 15:04"))
}

// adminGrantSingleItemBudget and adminGrantBulkBudget are FR-076's latency
// bounds: one item answers within 5 seconds, a full list within 60.
//
// They are separated rather than interpolated because the requirement states two
// specific numbers, and a formula would be a third thing the spec never said.
const (
	adminGrantSingleItemBudget = 5 * time.Second
	adminGrantBulkBudget       = 60 * time.Second
)

// grantBudgetFor picks the budget for a list of n items.
func grantBudgetFor(n int) time.Duration {
	if n <= 1 {
		return adminGrantSingleItemBudget
	}
	return adminGrantBulkBudget
}

// markRemainingAsTimedOut reports every item from `from` onward as not attempted.
//
// Without this the response would silently be shorter than the request, and an
// administrator reading "499 of 500 granted" would have no way to know the 500th
// was never tried. Each unfinished line names the reason and says what to do.
func markRemainingAsTimedOut(results []sub_dto.AdminGrantBatchResult, from int, items []sub_dto.AdminGrantBatchItem) {
	for idx := from; idx < len(items) && idx < len(results); idx++ {
		results[idx] = sub_dto.AdminGrantBatchResult{
			Index:          idx,
			UserIdentifier: items[idx].UserIdentifier,
			PlanSlug:       items[idx].PlanSlug,
			Status:         sub_dto.AdminGrantStatusError,
			Reason:         sub_dto.AdminGrantReasonBatchTimeout,
			Message:        mapper.MsgAdminGrantBatchTimeout,
		}
	}
}

func summarizeGrantResults(results []sub_dto.AdminGrantBatchResult) *sub_dto.AdminGrantBatchRes {
	res := &sub_dto.AdminGrantBatchRes{
		Total:   len(results),
		Results: results,
	}
	for _, r := range results {
		switch r.Status {
		case sub_dto.AdminGrantStatusSuccess:
			res.Succeeded++
		case sub_dto.AdminGrantStatusSkipped:
			res.Skipped++
		default:
			res.Failed++
		}
	}
	return res
}

// markDuplicateAccounts flags every item after the first that resolves to an
// already-seen ACCOUNT (FR-066, quickstart edge case "same account, several
// different plan slugs").
//
// Keyed on the resolved user id on purpose. The previous key was
// `identifier + planSlug`, which meant the account was not part of the identity at
// all: two lines for one customer with different plans were treated as two
// different targets and both were granted, stacking periods with no error.
//
// Two kinds of item are NOT counted here, each for a reason:
//
//   - An identifier that does not resolve. It gets its own per-item error below,
//     and counting it would let a typo suppress a later valid line for the same
//     person.
//   - A blank identifier. Two blank lines are not a duplicate account, they are
//     two blank lines, and telling the administrator one duplicates the other
//     would send them looking for an account that does not exist. Each blank
//     line reports `identifier_blank` on its own.
//
// A skipped duplicate is reported, never dropped: a silent drop looks like a
// success the administrator never asked for.
// markDuplicateAccounts flags every item after the first that resolves to an
// already-seen ACCOUNT (FR-066, quickstart edge case "same account, several
// different plan slugs").
//
// Keyed on the resolved user id on purpose. The previous key was
// `identifier + planSlug`, which meant the account was not part of the identity at
// all: two lines for one customer with different plans were treated as two
// different targets and both were granted, stacking periods with no error.
//
// An identifier that does not resolve is NOT counted here. It gets its own
// per-item error below, and counting it would let a typo suppress a later valid
// line for the same person.
func markDuplicateAccounts(items []sub_dto.AdminGrantBatchItem, resolved map[string]*ResolvedUser) map[int]bool {
	firstSeen := make(map[uuid.UUID]int, len(items))
	duplicates := make(map[int]bool)
	for idx, item := range items {
		entry := resolved[normalizeIdentifier(item.UserIdentifier)]
		if entry == nil || entry.UserID == uuid.Nil {
			continue
		}
		if _, ok := firstSeen[entry.UserID]; !ok {
			firstSeen[entry.UserID] = idx
			continue
		}
		duplicates[idx] = true
	}
	return duplicates
}

func collectIdentifiers(items []sub_dto.AdminGrantBatchItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if id := normalizeIdentifier(item.UserIdentifier); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func classifyGrantIdentifier(raw string, resolved map[string]*ResolvedUser) (reason string, message string, failed bool) {
	key := normalizeIdentifier(raw)
	if key == "" {
		return sub_dto.AdminGrantReasonIdentifierBlank, mapper.MsgAdminGrantIdentifierBlank, true
	}
	// FR-062: reported before the not-found check, because the resolver cannot
	// tell an over-long identifier from an unknown one - it drops both without a
	// lookup. Without this branch the administrator would be told to check the
	// spelling of an identifier that was never searched for.
	if ExceedsIdentifierLength(key) {
		return sub_dto.AdminGrantReasonIdentifierTooLong, mapper.MsgAdminGrantIdentifierTooLong, true
	}
	entry := resolved[key]
	if entry == nil {
		return sub_dto.AdminGrantReasonUserNotFound, mapper.MsgAdminGrantUserNotFound, true
	}
	if IsAmbiguous(entry) {
		// FR-048: refused, not guessed. Granting to the wrong account is not
		// something the administrator can discover afterwards.
		return sub_dto.AdminGrantReasonIdentifierAmbiguous, mapper.MsgAdminGrantIdentifierAmbiguous, true
	}
	if entry.User == nil {
		return sub_dto.AdminGrantReasonUserNotFound, mapper.MsgAdminGrantUserNotFound, true
	}
	return "", "", false
}
