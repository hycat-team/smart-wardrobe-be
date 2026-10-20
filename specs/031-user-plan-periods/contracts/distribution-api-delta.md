# Contract Delta — Admin subscription distribution

**Feature**: `031-user-plan-periods` | **Endpoint**: `GET /api/v1/admin/dashboard/subscriptions/distribution`

**Change type**: additive. No existing field is renamed, retyped, or removed. Two optional fields are
added and the counting source changes from the current-plan projection to validity intervals.

## 1. HTTP response — before and after

### Before (029)

```json
{
  "distribution": [
    { "planCode": "premium-monthly", "userCount": 12, "percentage": 60 },
    { "planCode": "free", "userCount": 8, "percentage": 40 }
  ],
  "totalUsers": 20
}
```

`userCount` came from `user_subscriptions.current_plan_code` grouped over users whose **current**
window overlapped the period. A user who downgraded after the period was in no bucket.

### After (031)

```json
{
  "distribution": [
    { "planCode": "premium-monthly", "userCount": 14, "percentage": 58.33 },
    { "planCode": "free", "userCount": 10, "percentage": 41.67 }
  ],
  "totalUsers": 24,
  "dataAvailableFrom": "2026-10-20",
  "dataCompleteness": "partial"
}
```

- The same user may now appear in more than one bucket when they held two plans inside the period
  (mid-period switch). `totalUsers` is the **sum of buckets**, so it can exceed the number of distinct
  users; percentages still add up to 100% across the returned buckets (clarification 2026-10-20).
- `dataAvailableFrom` is a **string** date in VN time (`YYYY-MM-DD`), not an RFC 3339 timestamp.
- `dataCompleteness` describes the requested period against that date.

| `dataCompleteness` | Meaning | `distribution` / `totalUsers` |
|---|---|---|
| `complete` | the whole period is at or after `dataAvailableFrom`; or no period was supplied (live snapshot) | normal |
| `partial` | the period starts before `dataAvailableFrom` and ends after it (or is open-ended) | computed from `[max(from, dataAvailableFrom), to)` only |
| `unavailable` | the period ends at or before `dataAvailableFrom` | `[]` and `0` as placeholders; the enum is the signal, the client must show an empty state |

A zero/absent bound means **unbounded**, never the zero timestamp: `to` absent means "up to now" and
the query also excludes intervals starting in the future. When the deployment has no cutover row, a period
request returns `dataCompleteness = "unavailable"` and omits only `dataAvailableFrom` (the date is
unknown); the no-period snapshot stays `complete` and may omit both fields.

## 2. Behaviour change per request shape

| Request | Before | After |
|---|---|---|
| no `fromDate`/`toDate` | active subscriptions at `now()`, grouped by current plan | snapshot at `now()` from intervals; same meaning, plus completeness fields |
| `fromDate` only | `expires_at IS NULL OR expires_at >= from` — no upper cap, future rows could count | intervals overlapping `[from, ∞)` **and** `valid_from <= now()` |
| `toDate` only | `started_at < to` — no expiry predicate, long-expired rows counted | intervals overlapping `(-∞, to)` |
| both | current window overlap | interval overlap, including plans the user changed away from; `from` clipped up to the cutover when the result is `partial` |
| malformed / `fromDate > toDate` | HTTP 400 | unchanged, HTTP 400 |

## 3. gRPC delta (`contracts/subscription/v1/subscription.proto`)

Additive, field numbers 2 and 3 on the existing response; no request change.

```proto
enum DistributionDataCompleteness {
  DISTRIBUTION_DATA_COMPLETENESS_UNSPECIFIED = 0;
  DISTRIBUTION_DATA_COMPLETENESS_COMPLETE = 1;
  DISTRIBUTION_DATA_COMPLETENESS_PARTIAL = 2;
  DISTRIBUTION_DATA_COMPLETENESS_UNAVAILABLE = 3;
}

message SubscriptionDistributionList {
  repeated SubscriptionDistribution items = 1;
  // Cutover instant. The Commerce edge formats it to the VN calendar date;
  // absent when the deployment has no cutover row.
  google.protobuf.Timestamp data_available_from = 2;
  DistributionDataCompleteness data_completeness = 3;
}
```

- An old Account server answering the new Commerce client leaves fields 2/3 unset; the adapter maps
  `UNSPECIFIED` + absent timestamp to "omit both JSON fields".
- A new Account server answering an old client is backwards compatible: the client reads `items` and
  ignores unknown fields.
- A compatibility test pins both directions: decode of the old JSON shape and tolerance of unknown
  proto fields.

## 4. Internal DTO delta

Account (`services/account/dto/subscription/dashboard_dto.go`):

```go
type SubscriptionDistributionResultDTO struct {
    Items             []SubscriptionDistributionDTO
    DataAvailableFrom *time.Time // formatted to YYYY-MM-DD (VN) at the Commerce edge
    DataCompleteness  string     // complete | partial | unavailable
}
```

`IAdminStatsUseCase.GetSubscriptionDistribution` and
`ISubscriptionDashboardReader.GetSubscriptionDistribution` return this result instead of a bare slice.
Both adapters (in-process `SubscriptionDashboardReader`, gRPC
`SubscriptionDashboardReaderClient`) map it to/from the proto message. Commerce keeps the percentage
computation and adds `dataAvailableFrom` (`*string`, VN `YYYY-MM-DD`) and `dataCompleteness` (`string`)
to `SubscriptionDistributionRes` with `omitempty`.

## 5. Examples the client must handle

1. Period after the cutover, users switched plans: buckets include both plans; `totalUsers` is the
   bucket sum; `dataCompleteness = complete`.
2. Period straddling the cutover: `partial` + `dataAvailableFrom`; counts cover only the part from the
   cutover onward; render with a caveat.
3. Period before the cutover: `unavailable`, empty buckets; render an empty state, not a zero chart.
4. No period: current snapshot; `complete` (and `dataAvailableFrom` if the row exists).
5. Cutover row missing: period request is `unavailable` with `dataAvailableFrom` omitted; snapshot
   remains `complete` (the completeness field is emitted) and omits only `dataAvailableFrom`.
