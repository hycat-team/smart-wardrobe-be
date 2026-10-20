# Policy grant windows — one definition for the reader and the checker

**Scope**: `aiexecutor.ai_policy_grants` and every component that answers "does
this account have a grant in force at instant `t`?".

**Status**: canonical. If this document and a code comment disagree, this
document is the rule and the comment is the defect.

## 1. The invariant

A row is **readable at instant `t`** when, and only when:

```sql
effective_from <= t
AND (effective_to IS NULL OR effective_to > t)
```

`status` is **not part of the predicate**. It never has been on the read path,
and it must not become one.

The database `CHECK` (`ai_policy_grants_effective_window_check`) guarantees
only that the window is non-empty (`effective_to > effective_from`). It says
nothing about whether `status` matches the window, and the read path does not
consult `status`, so the two are independent by design.

## 2. Why status is not filtered

A grant can be `closed` while its window is still in force. That happens when a
newer grant closes the previous one at the cutoff of the new event, but the
customer paid through the end of the old period. The reader must still return
the old row during the remainder of that period; filtering `status = 'active'`
would silently remove paid access and send the customer to the free model.

The reverse is also real: an `active` row whose `effective_to` has passed is not
readable, because its paid window is over. The window is authoritative; status is
bookkeeping written alongside it.

The reasoning that produced this split is [ADR 014](../decisions/014-relax-grant-ordering-invariant.md)
§2 and [ADR 013](../decisions/013-deferred-lazy-grant-self-heal.md) §2. The
consequence that matters day to day: a `closed` row inside its window is live
data, not history.

## 3. The two consumers

### 3.1 Reader — `GetCurrentGrant`

`services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_policy_grant_writer.go`

```go
Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).
Order("effective_from DESC").
First(&grant)
```

It returns the newest readable row and ignores `status`. `effective_from DESC`
is FR-040's ordering: the latest window wins, including when an older row is
still technically readable.

### 3.2 Whole-window checker — `07-verify_policy_grants.sql`

`database/scripts/07-verify_policy_grants.sql`, section 4

The checker counts rows whose `policy_snapshot` is empty or incomplete **among
the rows the reader can return**. It uses the same predicate and deliberately has
no status filter (027 FR-014/FR-015). Section 4b reports the complementary set —
unreadable snapshots whose window has ended — as **informational only**; that
count is expected to be non-zero after a repair and is not one of the six
invariants (FR-016).

If the checker narrowed to `status = 'active'`, a `closed` row with a live window
would be excluded from the check while still being returned to users. That is
the exact defect FR-015 exists to prevent.

## 4. Status × window, at instant `t`

| Status | Window relation | Readable? | Meaning |
|---|---|---|---|
| `active` | `effective_from <= t < effective_to` (or open-ended) | yes | the ordinary live grant |
| `active` | `effective_to <= t` | no | live status but the paid window is over; must have been closed by the writer |
| `future` | `effective_from > t` | no | scheduled, not started |
| `future` | window contains `t` | yes | the window, not the label, decides |
| `closed` | window contains `t` | **yes** | superseded for the future, still paid for the current period |
| `closed` | window ended, or never started | no | history |

The third and last rows are the counter-intuitive ones and the reason this file
exists.

## 5. Rules for changing the code

1. Any query that decides "is this grant in force at `t`" MUST use the predicate
   in §1. Do not add a status filter "for safety"; it is the defect, not the
   guard.
2. The reader and the checker MUST be changed together. A predicate change in one
   without the other makes the operator report describe a different table state
   from the one customers experience.
3. `status` transitions remain bookkeeping. A change to how statuses are written
   does not require touching the read predicate, and must not be used to narrow
   it.
4. Tests that pin this behaviour:
   - `services/ai/internal/modules/aiexecutor/application/usecase/grant/grant_window_resolution_integration_test.go`
     (window ordering, overlaps, closed-prior handling);
   - `database/scripts/07-verify_policy_grants.sql` section 4 and its 4b
     informational companion (run in the database verification step);
   - `services/ai/internal/modules/aiexecutor/infrastructure/persistence` tests
     for `GetCurrentGrant`.
5. A `CHECK` constraint only enforces a non-empty window. It cannot enforce that
   `status` agrees with the window; do not claim it does in comments or runbooks.

## 6. See also

- [ADR 013 — deferred lazy grant self-heal](../decisions/013-deferred-lazy-grant-self-heal.md)
- [ADR 014 — relaxing the strict ordering invariant](../decisions/014-relax-grant-ordering-invariant.md)
- [AI cost control](ai-cost-control.md) — where grants fit in the request pipeline
- [Grant restore runbook](../operations/grant-restore-runbook.md) — operator procedure
- `specs/027-grant-repair-unblock/spec.md` FR-014, FR-015, FR-016
