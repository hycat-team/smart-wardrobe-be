# Hợp đồng: tín hiệu thay đổi gói `subscription.plan.changed`

**Ngày**: 2026-09-29 | **Phía phát**: Account (`services/account`) · **Phía nhận**: AI (`services/ai`)
**Trao đổi**: RabbitMQ, exchange `smart_wardrobe_exchange` (topic), hàng đỗi `subscription_plan_changed_queue`
**Yêu cầu**: FR-011 đến FR-026, FR-033, FR-036, FR-039

> Ghi chú kiến trúc: kiểu DTO **không** được chia sẻ giữa hai dịch vụ — mỗi bên khai báo riêng, và JSON là hợp đồng. Xem `wallet.go:22-25`. Hằng số loại sự kiện thì **có** dùng chung, nằm ở `shared/domain/constants/subscriptioneventtype/` ([research.md](../research.md) R1).

---

## Cấu trúc nội dung — 14 trường

6 trường đang gửi giữ nguyên **tên và ý nghĩa** (FR-025). 8 trường mới được thêm vào.

| # | JSON | Kiểu Go | Bắt buộc | Nguồn ở phía phát | Mô tả |
|---|---|---|---|---|---|
| 1 | `eventVersion` | `int` | ✅ | hằng số | Số phiên bản nội dung. Bản hiện tại: `2` |
| 2 | `userId` | `string` | ✅ | `event.UserID` | UUID tài khoản |
| 3 | `eventType` | `string` | ✅ | `event.EventType` | Xem bảng loại bên dưới. **Lấy từ bản ghi lịch sử, không viết tay** (FR-019) |
| 4 | `toPlanId` | `string` | ✅ | `plan.ID` | UUID của gói đích. **Trường mới** — thiếu chính là nguyên nhân `plan_id` ghi sai |
| 5 | `toPlanCode` | `string` | ✅ | `plan.Slug` | Mã gói dạng chữ |
| 6 | `toTierRank` | `int` | ✅ | `plan.TierRank` | Cấp bậc quyền |
| 7 | `aiCostPolicyId` | `string` | ✅ | `plan.AICostPolicyID` | Chính sách phí áp dụng cho gói đích |
| 8 | `occurredAt` | `string` | ✅ | `event.OccurredAt` | RFC3339Nano — thời điểm xảy ra |
| 9 | `effectiveAt` | `string` | ✅ | `event.EffectiveAt` | RFC3339Nano — thời điểm có hiệu lực |
| 10 | `effectiveTo` | `string` | ⬜ | `sub.ExpiresAt` | RFC3339Nano, hoặc chuỗi rỗng nếu gói không thời hạn |
| 11 | `durationDays` | `int` | ⬜ | `plan.DurationDays` | Số ngày của gói; `0` nếu không thời hạn. **Trường mới** — cần cho quy tắc gia hạn |
| 12 | `sourceEventId` | `string` | ✅ | `event.ID` | UUID bản ghi lịch sử. **Khoá chống xử lý trùng** |
| 13 | `sourceDepositTransactionId` | `string` | ⬜ | `event.SourceDepositTransactionID` | Chuỗi rỗng nếu không phải giao dịch tiền |
| 14 | `fromPlanCode` | `string` | ⬜ | `event.FromPlanCode` | Gói trước; chuỗi rỗng nếu tài khoản chưa có gói |

**Cột trắng** = có thể chuỗi rỗng / `0`, nhưng **khoá phải có mặt trong JSON** để phía nhận phân biệt được "không có" với "phiên bản cũ chưa gửi trường này".

### Trường nào để trống khi nào

| Trường | Rỗng khi |
|---|---|
| `effectiveTo` | Gói không thời hạn (`durationDays = 0`) — tức gói miễn phí hoặc gói trọn đời |
| `durationDays` | Gói không thời hạn |
| `sourceDepositTransactionId` | Không phải mua bằng tiền: cấp thủ công, cấp phát mặc định, thu hồi |
| `fromPlanCode` | Tài khoản chưa từng có gói (lần cấp đầu tiên) |

### Ví dụ — gia hạn kỳ chưa bắt đầu

Tài khoản có Premium hết hạn `2026-10-22`, quản trị viên cấp thêm 30 ngày vào ngày `2026-09-29`. `expiresAt` mới = `2026-11-21`. Vì `2026-11-21 − 30 ngày = 2026-10-22` vẫn nằm trong tương lai so với `occurredAt`, bản ghi mới sẽ ở trạng thái `future` và chỉ có hiệu lực từ `2026-10-22`:

```json
{
  "eventVersion": 2,
  "userId": "ad11ad11-ad11-ad11-ad11-ad11ad11ad12",
  "eventType": "EXTENDED",
  "toPlanId": "…",
  "toPlanCode": "premium-monthly",
  "toTierRank": 1,
  "aiCostPolicyId": "aa000000-0000-0000-0000-000000000002",
  "occurredAt": "2026-09-29T08:15:00.123456789Z",
  "effectiveAt": "2026-09-29T08:15:00.123456789Z",
  "effectiveTo": "2026-11-21T08:15:00.123456789Z",
  "durationDays": 30,
  "sourceEventId": "…",
  "sourceDepositTransactionId": "",
  "fromPlanCode": "premium-monthly"
}
```

Đây chính là nội dung mà tín hiệu hiện tại **không** mang: không có `effectiveTo` thì bên nhận không tính được mốc bắt đầu của kỳ chờ tới, và FR-039 không thể thực hiện.

---

## Bảng loại thay đổi và cách phía nhận xử lý

| `eventType` | Có phát ra không | Hành động của phía nhận |
|---|---|---|
| `ACTIVATED` | ✅ | Đóng bản ghi cũ, tạo bản ghi `active` |
| `UPGRADED` | ✅ | Như trên |
| `OVERLAID` | ✅ | Như trên |
| `LIFETIME_FALLBACK_RESTORED` | ✅ | Như trên |
| `EXTENDED` | ✅ | Cắt đuôi bản ghi cũ, tạo bản ghi `active` **hoặc** `future` theo quy tắc mốc bắt đầu |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | ✅ | Đóng mọi bản ghi `active`, xoá mọi bản ghi `future` |
| `AUTO_RENEW_UPDATED` | ✅ | **Không làm gì** — bật/tắt tự gia hạn không đổi gói |
| ~~`RENEWED`~~ | ❌ không tồn tại | Gỡ khỏi mã phía nhận |
| ~~`EXPIRED`~~ | ❌ không tồn tại | Gỡ khỏi mã phía nhận |
| Bất kỳ giá trị lạ nào | — | **Từ chối** + cảnh báo + đẩy hàng chết. Tuyệt đối không rơi vào nhánh mặc định |

`RENEWED` và `EXPIRED` là hai chuỗi phía nhận đang xử lý nhưng **không nơi nào phát ra**. Chúng tồn tại vì danh sách hai bên lệch nhau từ trước — xem [research.md](../research.md) R1 để biết hậu quả cụ thể của từng cặp.

---

## Quy tắc tương thích phiên bản

| So với mức phía nhận hiểu | Hành động |
|---|---|
| `eventVersion` **mới hơn** | **Từ chối** + cảnh báo + hàng chết. Không ghi dữ liệu cho nội dung không hiểu (FR-021) |
| Bằng | Xử lý bình thường |
| **Cũ hơn** | Xử lý với quy tắc trường thiếu: coi như chưa xác định + cảnh báo (FR-022) |

### Trường thiếu ở nội dung phiên bản cũ

Tín hiệu đang nằm trong hàng đỗi tại thời điểm triển khai có đúng 6 trường cũ, không có `eventVersion`. Quy tắc xử lý (FR-020, FR-022):

| Trường thiếu | Xem là | Hệ quả |
|---|---|---|
| `eventVersion` | `1` | Được xử lý theo nhánh "cũ hơn" |
| `toPlanId` | `uuid.Nil` | Bản ghi mới có `plan_id` rỗng — **không được ghi**, vì sẽ phá bất biến I1/I2 |
| `effectiveTo` | `NULL` | Bản ghi không thời hạn |
| `durationDays` | `0` | Không tính được quy tắc `EXTENDED` → coi như không phải gia hạn |
| `sourceEventId` | `""` | **Không chống trùng được** → bỏ qua bản ghi, ghi cảnh báo |
| `fromPlanCode` | `""` | Không ảnh hưởng |

**Quyết định cần nêu rõ**: tín hiệu phiên bản 1 thiếu `sourceEventId` nên **không thể** chống trùng. Hai lựa chọn — xử lý bằng cách khác, hoặc từ chối. Ta chọn **từ chối kèm cảnh báo** (FR-020 nói "MUST NOT dừng hoạt động", không nói phải ghi dữ liệu). Lý do: nếu chấp nhận ghi thì mỗi lần giao lại của hàng đỗi sẽ tạo một bản ghi trùng — vi phạm đúng bất biến mà cả tính năng này sinh ra để chữa. Dữ liệu tín hiệu cũ trong hàng đỗi là dữ liệu phát triển, nên mất nó không mất dữ liệu người dùng.

Có thể vận hành an toàn vì: đường ghi đồng bộ vẫn còn (chưa gỡ cho tới GĐ5) nên tín hiệu cũ không phải nguồn duy nhất; và sau GĐ5, công cụ khôi phục ở GĐ8 sẽ tạo lại tín hiệu **phiên bản 2** cho mọi tài khoản thiếu bản ghi.

---

## Xử lý lỗi và giao lại

| Tình huống | Phân loại | Kết quả |
|---|---|---|
| `eventVersion` mới hơn mức hiểu | `PermanentError` | Hàng chết + cảnh báo |
| `eventType` lạ | `PermanentError` | Hàng chết + cảnh báo |
| `occurredAt` không mới hơn **đồng hồ thứ tự** của bản ghi mới nhất của tài khoản | `PermanentError` | Hàng chết + cảnh báo `stale_event` — chống tín hiệu đến sai thứ tự hồi sinh mức quyền tính phí đã bị thu hồi. Phạm vi so sánh là bản ghi **mới nhất** (kể cả đang chờ tới hoặc đã đóng), không chỉ bản ghi đang hiệu lực — xem FR-033. **Đính chính bởi spec 027:** đồng hồ thứ tự là **thời điểm tạo của chính bản ghi** (`created_at`), và khi `created_at` không dùng được thì lùi về `effective_from`. Bản ghi **không** đọc `occurred_at` từ ảnh chụp chính sách để sắp thứ tự — chính cột đó đã gây ra sự cố. Trên **đường tạo bản ghi mới**, một sự kiện đến muộn **không** còn bị từ chối vĩnh viễn: mốc bắt đầu của bản ghi mới được dịch lên ngay sau sàn thứ tự. Từ chối `stale_event` chỉ còn đúng ở nhánh hạ cấp tới một gói không định giá được |
| Thiếu `userId` / `eventType` / `toPlanCode` / `toTierRank` / `aiCostPolicyId` / `occurredAt` | `PermanentError` | Hàng chết + cảnh báo |
| Thiếu `sourceEventId` (nội dung phiên bản 1) | `PermanentError` | Hàng chết + cảnh báo |
| `aiCostPolicyId` không tồn tại trong bảng `aiexecutor.ai_cost_policies` | `PermanentError` | Hàng chết + cảnh báo `ai_cost_policy_not_found` — FR-036, không được âm thầm bỏ qua |
| Lỗi cơ sở dữ liệu, lỗi mạng | `RetryableError` | Giao lại, tối đa 5 lần rồi hàng chết |
| Đã có bản ghi cùng `sourceEventId` | **Không phải lỗi** | Bỏ qua, ghi nhận thành công, không sinh tác dụng phụ |

Hàng đỗi phải là **quorum** với `x-delivery-limit: 5`, nếu không thì `RetryableError` sẽ bị giao lại vô hạn. Xem [research.md](../research.md) R2.

**Vì sao dòng FR-036 nói về chính sách phí chứ không nói về danh mục gói (024 T172).** Dòng này trước đây mô tả việc tra `toPlanCode` trong một danh mục, và mô tả một kiểm tra mà dịch vụ AI **không thực hiện**. Dịch vụ AI tra **chính sách phí của chính nó** (`aiexecutor.ai_cost_policies`, theo `aiCostPolicyId`) và **không** tra danh mục gói, vì Constitution §I quy định mỗi miền dữ liệu có một chủ sở hữu: danh mục gói thuộc dịch vụ Account, và đọc bảng của dịch vụ khác sẽ phá vỡ ranh giới đó.

Hai hệ quả thực tế, vì đây không chỉ là chuyện câu chữ:

- Một `aiCostPolicyId` không tồn tại bị **từ chối**, không phải bỏ qua âm thầm — đó là điều FR-036 đòi hỏi, chỉ là áp dụng cho chính sách phí.
- Một `toPlanId` không tồn tại **không** bị phía AI phát hiện, và điều đó là có chủ ý. Không có gì ở phía sau kiểm tra nó, nên `toPlanId` rỗng bị chặn ngay trong bộ phân tích nội dung và trong bước dựng bản ghi quyền. Ghi chép trước đây nói một bản ghi như vậy "bị bỏ qua bởi kiểm tra danh mục của FR-036" — khẳng định sai rằng có gì đó ở phía sau sẽ bắt được nó.

---

## Bắt buộc quan sát (FR-086, FR-087)

Mỗi lần từ chối hoặc thất bại phải ghi lại: mã tài khoản, mã lý do, thời điểm. Cần bộ đếm theo từ định danh tín hiệu và cảnh báo khi bộ đếm tăng. Việc ghi nhận này **không** phụ thuộc vào việc có thao tác cấp gói nào đi kèm — vì không phải tín hiệu nào cũng đi kèm thao tác cấp gói.

Mã lý do tối thiểu:

| Mã | Nghĩa |
|---|---|
| `unsupported_event_version` | `eventVersion` mới hơn mức hiểu |
| `unknown_event_type` | Không thuộc 7 loại hợp lệ |
| `missing_required_field` | Thiếu trường bắt buộc |
| `missing_source_event_id` | Nội dung phiên bản 1 |
| `ai_cost_policy_not_found` | `aiCostPolicyId` không tồn tại trong bảng chính sách phí của dịch vụ AI (024 T159 đổi tên từ `plan_not_in_catalog`) |
| `stale_event` | Sự kiện không mới hơn đồng hồ thứ tự của bản ghi mới nhất — thời điểm tạo bản ghi (`created_at`), lùi về `effective_from` khi `created_at` không dùng được (FR-033; đính chính bởi spec 027) |
| `policy_snapshot_unavailable` | Không dựng được ảnh chụp chính sách |
| `duplicate_source_event` | Đã xử lý rồi (không phải lỗi) |
| `policy_grant_constraint_violation` | Ghi bản ghi mức quyền vi phạm ràng buộc `CHECK` (SQLSTATE 23514); từ chối vĩnh viễn, **không** thử lại (spec 027, FR-009..FR-011) |
| `transient_failure` | Lỗi cơ sở dữ liệu hoặc mạng |

### Sửa một lỗi quan sát phát hiện được

Nhật ký hiện ghi tiêu đề `correlationId` (`rabbitmq_pubsub.go:121`) nhưng bên nhận đọc `x-correlation-id` (`consumer.go:23`) — nên **mã định danh tương quan luôn rỗng**. Với yêu cầu truy vết được mọi lần từ chối, điều này phải sửa. Sửa ở GĐ8, sửa phía nhận đọc cả hai tên khoá để không phụ thuộc thứ tự triển khai.

---

## Yêu cầu bảo đảm cho phía phát

Để hợp đồng này được giữ đúng, phía phát bắt buộc:

1. **Đúng một nơi dựng nội dung** — không tự viết chuỗi JSON ở từng nơi phát (FR-011).
2. `eventType` lấy từ bản ghi lịch sử vừa ghi, không viết tay (FR-019).
3. `toPlanId` lấy từ `plan.ID` — trường này đã nằm trong phạm vi ở cả `wallet.go:150` và mọi điểm phát mới, chỉ là chưa được gửi đi.
4. `sourceEventId` lấy từ `event.ID` **sau khi** ghi bản ghi lịch sử — GORM sinh khoá trong lúc `INSERT` nên đọc được ngay.
5. Ghi bản ghi lịch sử **và** bản ghi outbox trong cùng một giao dịch (FR-015).
6. Phát tín hiệu cho **mọi** loại thay đổi gói, kể cả loại mà bên nhận chọn không làm gì (FR-012).
