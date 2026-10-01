# Hợp đồng API quản trị cổng 2 (AI Cost Policy)

> Giao diện quản trị **cổng 2** (`aiexecutor.ai_cost_policy_operations` - FR-004b, FR-004c, FR-004d, FR-004e).
> Cổng 1 được mô tả ở [admin-grant-api.md](./admin-grant-api.md). Hai cổng là hai bảng khác nhau, hai dịch vụ khác nhau.

## 1. Ai gọi, qua đường nào

| | |
|---|---|
| Bên gọi | **Bảng điều khiển web của quản trị viên** (client đối tác), không phải service khác |
| Đường đi | HTTP thẳng vào dịch vụ AI |
| Tiền tố | `/api/v1` (do `services/ai/internal/bootstrap/router.go` gắn) |
| Xác thực | Kế thừa `r.auth.Handle()` + `shared_middleware.RolesAuthorize(roleslug.Admin)` của nhóm `/admin/ai` **sẵn có** |
| Nginx | Đã ánh xạ sẵn tại `deployments/nginx/local.conf` → `location ^~ /api/v1/admin/ai` |

**Không tạo đường xác thực riêng.** Không đặt route ngoài nhóm `/admin/ai`.

## 2. Vì sao đặt trong mô-đun này

Bảng `ai_cost_policy_operations` thuộc sở hữu của dịch vụ AI. Theo Constitution §I, một module chỉ được ghi bảng của chính mình — vì vậy **không** đặt route này trong module `admin` của Account, và Account **không** được ghi thẳng vào bảng này.

## 3. `GET /api/v1/admin/ai/cost-policies` — xem cổng 2

Trả về **toàn bộ** chính sách cùng bốn thao tác và chế độ thi hành, để quản trị viên thấy được cổng chết mà không cần mở SQL.

**Response 200**

```json
{
  "message": "Danh sách chính sách chi phí AI",
  "data": [
    {
      "code": "premium-default",
      "name": "Premium AI Policy",
      "version": 1,
      "enforcementMode": "strict",
      "periodDays": 30,
      "hardCostMicroVnd": 30000000000,
      "operations": [
        {
          "operation": "outfit",
          "normalRoute": "paid",
          "reducedRoute": "paid",
          "freeRoute": "free",
          "maxPaidAttemptsPerDay": null,
          "paidFallbackEnabled": false,
          "isEnabled": true
        },
        {
          "operation": "chat",
          "normalRoute": "paid",
          "reducedRoute": "paid",
          "freeRoute": "free",
          "maxPaidAttemptsPerDay": null,
          "paidFallbackEnabled": false,
          "isEnabled": true
        }
      ]
    }
  ]
}
```

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `code` | string | mã chính sách: `free-default` hoặc `premium-default` |
| `enforcementMode` | string | `strict` hoặc `observe_only` |
| `hardCostMicroVnd` | integer hoặc `null` | `null` = không có trần cứng |
| `maxPaidAttemptsPerDay` | integer hoặc `null` | `null` = chính sách này không dùng giới hạn lượt |

**Lưu ý bắt buộc về ngữ nghĩa:** trường `maxPaidAttemptsPerDay` chỉ **có tác dụng** ở chế độ `observe_only`. Ở chế độ `strict` nó không được đọc (FR-005e). `null` ở `strict` là trạng thái đúng và mong muốn — không phải dữ liệu thiếu.

## 4. `PUT /api/v1/admin/ai/cost-policies/:id/operations/:operation` — sửa một thao tác

Sửa **một** trường quyền lợi của **một** thao tác. `:id` là UUID của bản ghi trong `ai_cost_policy_operations`, `:operation` là một trong `outfit`, `chat`, `rewriter`, `summary`.

**Request body**

```json
{
  "maxPaidAttemptsPerDay": 4
}
```

Gửi `null` (hoặc bỏ trường kèm `"clearMaxPaidAttempts": true`) nghĩa là muốn đặt thành `NULL` — chỉ chấp nhận ở chế độ `strict`, xem mục 5.2.

**Response 200** — ngoài dữ liệu đã cập nhật, phải kèm **cảnh báo bắt buộc** (FR-004b(2)):

```json
{
  "message": "Đã cập nhật giới hạn lượt của thao tác outfit",
  "data": {
    "policyCode": "free-default",
    "operation": "outfit",
    "maxPaidAttemptsPerDay": 4,
    "warning": {
      "replayRequired": true,
      "frozenGrantCount": 13,
      "openGrantCount": 13,
      "message": "Trường này thuộc nhóm quyền lợi đã đóng băng trong ảnh chụp. 13 bản ghi mức quyền hiện có không tự làm mới, trong đó 13 bản ghi effectiveTo = null sẽ không bao giờ tự làm mới. Cần chạy phát lại tín hiệu theo FR-029c."
    }
  }
}
```

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `replayRequired` | boolean | luôn là `true` khi sửa `maxPaidAttemptsPerDay` |
| `frozenGrantCount` | integer | tổng số bản ghi mức quyền đã tồn tại, vẫn giữ giá trị cũ |
| `openGrantCount` | integer | trong đó có bao nhiêu bản ghi `effective_to IS NULL` — nhóm này **không bao giờ tự làm mới** |

## 5. Các trường hợp từ chối — bốn trường hợp bắt buộc

Mỗi trường hợp dùng constructor có sẵn trong `shared/application/constants/apperror`. Toàn bộ trả về qua `shared_pres.APIResponse` chuẩn với `GlobalErrorHandler` xử lý.

| # | Điều kiện | Lỗi | HTTP | Constructor | Nội dung phải nói rõ |
|---|---|---|---|---|---|
| 1 | `maxPaidAttemptsPerDay = null` khi chính sách ở `observe_only` | `NewConflict` | **409** | `apperror.NewConflict(detail)` | `NULL` ở `observe_only` sẽ **tắt** cơ chế đếm lượt; chỉ `strict` mới được phép `NULL` (FR-005e) |
| 2 | Chuyển sang `strict` khi `hard_cost_micro_vnd` còn trống | `NewConflict` | **409** | `apperror.NewConflict(detail)` | không có trần chi phí thì `strict` sẽ không chặn được gì cả |
| 3 | `:id` không tồn tại hoặc không thuộc thao tác được hỗ trợ | `NewNotFound` | **404** | `apperror.NewNotFound(detail)` | nêu rõ `:id` và `:operation` không hợp lệ |
| 4 | `:id` sai định dạng UUID, hoặc body thiếu trường bắt buộc | `NewBadRequest` | **400** | qua `validation.BindJSON` / `BindURI` | trả về đúng nguyên văn validator đã dịch tiếng Việt |

**Không** dùng `500` cho bốn trường hợp này — đều là lỗi người gọi, phải trả về mã 4xx để quản trị viên hiểu và sửa được.

## 6. Ràng buộc kiểm tra dữ liệu

Validate ở tầng DTO với tag `binding` của Gin (đúng convention đang dùng, ví dụ `services/commerce/internal/modules/brand/application/dto/brand.go`):

```go
type UpdateCostPolicyOperationReq struct {
	MaxPaidAttemptsPerDay *int `json:"maxPaidAttemptsPerDay" binding:"omitempty,min=0" label:"giới hạn lượt trả phí mỗi ngày"`
	ClearMaxPaidAttempts  bool `json:"clearMaxPaidAttempts,omitempty" label:"đặt giới hạn lượt thành NULL"`
}
```

Luật bổ sung: gửi cả `maxPaidAttemptsPerDay` lẫn `clearMaxPaidAttempts: true` cùng lúc là **mâu thuẫn**, phải từ chối `400` — nếu không, ý định của người gọi không xác định.

## 7. Ghi nhận ký — hai bản ghi trong một giao dịch

Mỗi lần `PUT` thành công tạo **hai** bản ghi trong **cùng một giao dịch** (FR-004c):

| # | Bản ghi | Bảng | Ai ghi |
|---|---|---|---|
| 1 | Kiểm toán cục bộ | `aiexecutor.ai_cost_policy_audit_logs` | dịch vụ AI |
| 2 | Outbox | `shared.outbox_records`, `topic = admin.audit.log` | dịch vụ AI |

**Nếu ghi bản ghi outbox thất bại thì phải rollback cả bản ghi kiểm toán.** Chấp nhận bản ghi kiểm toán mà không có event là trạng thái nửa vời: màn hình kiểm toán tập trung sẽ không thấy thay đổi đã xảy ra.

### 7.0 Cách bảo đảm hai bản ghi nằm chung một giao dịch

Bọc cả hai lệnh ghi trong `IUnitOfWork.Execute(ctx, func(ctx context.Context) error {...})`. Hàm này mở transaction GORM rồi tiêm nó vào `ctx` qua `db.InjectTx`, nên **cả hai** repository tự động ghi vào cùng transaction mà không cần truyền tham số `tx` riêng:

| Đường ghi | Cách lấy transaction |
|---|---|
| `IAICostPolicyAuditRepository.Create` | `getDB(ctx)` → `db.GetTx(ctx)`; không có tx trong ctx thì rơi về DB gốc |
| `IEventPublisher.Publish` | gọi `OutboxRepositoryImpl.Create`, cũng qua `getDB(ctx)` → `db.GetTx(ctx)` |

**Cấm tuyệt đối** ghi hai lệnh rời rạc ngoài transaction. Khi đó nếu lệnh thứ hai hỏng thì bản ghi thứ nhất đã nằm xuống vĩnh viễn và không còn cách gỡ.

### 7.1 Dùng lại hạ tầng có sẵn — không tạo mới

| Thành phần | Hằng số / đường dẫn | Vị trí |
|---|---|---|
| Topic | `eventconstants.TopicAuditLog` = `"admin.audit.log"` | `shared/application/constants/eventconstants/eventconstants.go` |
| Queue | `eventconstants.QueueAuditLog` = `"audit_log_queue"` | `shared/application/constants/eventconstants/queues.go` |
| Binding + DLQ | đã khai báo trong topology | `shared/infrastructure/messaging/topology.go:80` và `:176` |
| Publisher | `event.IEventPublisher.Publish` — tự ghi `OutboxRecord` | `shared/infrastructure/messaging/rabbitmq_pubsub.go:109` |
| Contract ghi log | `services/account/contract/admin/audit_log_contract.go` | đã có `.Record(ctx, input)` |
| Consumer | `AuditLogConsumerWorker` — đã đăng ký `RegisterTypedConsumer` | `services/account/internal/modules/admin/presentation/worker/audit_log_consumer.go` |
| DTO | `dto.RecordAuditLogInput` | `services/account/dto/admin` |

**Không** tạo topic mới, **không** tạo queue mới, **không** viết consumer mới.

### 7.2 Payload — bắt buộc đúng cấu trúc có sẵn

```json
{
  "actorUserId": "8b1d1c3e-4f5a-4b2c-9d8e-7a6b5c4d3e2f",
  "action": "ai.cost_policy_update",
  "targetType": "ai_cost_policy",
  "targetId": null,
  "requestIp": "203.0.113.42",
  "payload": {
    "policyCode": "free-default",
    "operation": "outfit",
    "fieldName": "maxPaidAttemptsPerDay",
    "oldValue": 3,
    "newValue": 4
  }
}
```

| Trường | Giá trị bắt buộc | Ghi chú |
|---|---|---|
| `actorUserId` | UUID | lấy từ `contextutils.GetUserId(c)` |
| `requestIp` | chuỗi | lấy từ `c.ClientIP()` |
| `action` | `ai.cost_policy_update` | ký hiệu chấm, theo đúng convention `adminaudit.AuditAction` đang dùng (`ai.pricing_update`, `ai.grant_create`) |
| `targetType` | `ai_cost_policy` | theo đúng convention `adminaudit.AuditTargetType` (`ai_model_config`, `ai_policy_grant`) |
| `payload` | 5 trường ở trên | để màn hình hiển thị được trường nào đổi từ giá trị nào sang giá trị nào |

**Hai hằng số phải thêm vào `shared/domain/constants/adminaudit/`:**

```go
// action.go
AICostPolicyUpdate AuditAction = "ai.cost_policy_update"

// targettype.go
AICostPolicy AuditTargetType = "ai_cost_policy"
```

**Payload sai cấu trúc sẽ bị đưa vào DLQ và không thử lại.** `RegisterTypedConsumer` unmarshal thẳng vào `dto.RecordAuditLogInput`; sai schema sinh `PermanentError` → message vào dead-letter queue. Đây là lý do cấu trúc phải giống hệt, không được tự chế.

**Riêng trường `action` sai giá trị thì không gây lỗi** — nó là chuỗi, nên được ghi thẳng xuống `shared.admin_audit_logs` một giá trị không nằm trong danh mục mà không có báo động nào. Vì vậy phải dùng hằng số chứ không gõ chuỗi thủ công.

### 7.3 Chống ghi trùng khi outbox thử lại

Bảng `shared.admin_audit_logs` **không có** ràng buộc duy nhất tự nhiên cho tình huống này, nên phải chặn trùng bằng điều kiện:

> Trước khi ghi, kiểm tra đã có bản ghi nào trùng **định danh ổn định** của thay đổi chưa. Nếu có thì bỏ qua (coi như đã ghi).

Định danh ổn định gồm: `actorUserId` + `policyCode` + `operation` + `fieldName` + `newValue` + mốc thời gian.

Chọn **check-then-insert** chứ không phải khoá duy nhất, vì hai thay đổi hợp lệ có thể trùng tên trường và giá trị nhưng khác thời điểm — khoá duy nhất sẽ nuốt mất thay đổi thứ hai. Đánh đổi của lựa chọn này: hai event đến đồng thời vẫn có thể trùng, và mức chấp nhận được là không đáng kể vì cùng một admin hiếm khi sửa cùng một trường trong cùng một mili giây. Ghi rõ giới hạn này trong test.

## 8. Chú thích Swagger bắt buộc

Handler mới đặt trong `presentation/handler/`, kế thừa mẫu của `admin_handler.go`. Bắt buộc có đủ chú thích để `make generate svc=ai` sinh ra Swagger:

```go
// ListCostPolicies
// @Summary Danh sách chính sách chi phí AI (cổng 2)
// @Description Quản trị viên xem cổng 2 của từng gói: chế độ thi hành, trần chi phí và giới hạn lượt của từng thao tác.
// @Tags Admin AI
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.AICostPolicyDTO}
// @Router /api/v1/admin/ai/cost-policies [get]

// UpdateCostPolicyOperation
// @Summary Cập nhật giới hạn lượt của một thao tác AI (cổng 2)
// @Description Sửa maxPaidAttemptsPerDay. Trường này thuộc nhóm quyền lợi đã đóng băng nên cần phát lại tín hiệu (FR-029c) mới có tác dụng.
// @Tags Admin AI
// @Accept json
// @Produce json
// @Param id path string true "Operation ID" format(uuid)
// @Param operation path string true "Tên thao tác" Enums(outfit, chat, rewriter, summary)
// @Param body body dto.UpdateCostPolicyOperationReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.UpdateCostPolicyOperationResp}
// @Router /api/v1/admin/ai/cost-policies/{id}/operations/{operation} [put]
```

## 9. Kiểm thử bắt buộc

| # | Kiểm thử | Kỳ vọng |
|---|---|---|
| 1 | `GET` trả về cả hai chính sách, đủ 4 thao tác mỗi cái | khớp `ai_cost_policy_operations` |
| 2 | `GET` với `strict` trả `maxPaidAttemptsPerDay: null` | `null` đúng, không phải lỗi |
| 3 | `PUT` đặt `null` khi policy ở `observe_only` | **409** |
| 4 | `PUT` chuyển `strict` khi `hardCostMicroVnd` trống | **409** |
| 5 | `PUT` với `:id` không tồn tại | **404** |
| 6 | `PUT` với `:id` không phải UUID | **400** |
| 7 | `PUT` gửi cả giá trị lẫn `clearMaxPaidAttempts: true` | **400** |
| 8 | `PUT` thành công | trả `warning.replayRequired = true`, đủ `frozenGrantCount` và `openGrantCount` |
| 9 | Sau `PUT`, bảng `ai_cost_policy_audit_logs` có đủ **5** trường: actor, thời điểm, IP, giá trị cũ, giá trị mới | đủ 5 |
| 10 | Sau `PUT`, `shared.outbox_records` có record `topic = admin.audit.log` | có, cùng giao dịch |
| 11 | Cho outbox ghi lỗi | **rollback** cả bản ghi kiểm toán, cờ 2 còn 0 |
| 12 | Nhận **cùng một event hai lần** | `admin_audit_logs` chỉ có **một** dòng |
| 13 | Payload event unmarshal được vào `dto.RecordAuditLogInput` | không rơi vào DLQ |
| 14 | Người không có role `admin` gọi `GET` | **403** từ `RolesAuthorize` |
| 15 | `PUT` gửi `clearMaxPaidAttempts: true` **không kèm** giá trị, khi chính sách ở `strict` | đặt thành `NULL` thành công, trả `maxPaidAttemptsPerDay: null` |
| 16 | `PUT` gửi `clearMaxPaidAttempts: true` **không kèm** giá trị, khi chính sách ở `observe_only` | **409** - từ chối vì `observe_only` không được phép `NULL` (FR-005e) |

**Cách chạy từng mục.** Các mục dựa trên quyết định logic thuần — 1, 2, 3, 4, 5, 6, 7, 8, 13, 14, 15, 16 — **kiểm thử tự động được** với repository giả lập, không cần cơ sở dữ liệu thật. Bốn mục cần đọc cơ sở dữ liệu thật — **9, 10, 11, 12** — là **xác minh thủ công bằng `psql`**, theo các bước có sẵn trong `quickstart.md`, mục "Xác minh ghi nhận ký cổng 2".

Lý do không kiểm thử tự động phần cơ sở dữ liệu nằm ở mục "Chiến lược kiểm thử" của `spec.md`: toàn dự án hiện không có hạ tầng kiểm thử DB, và mục 11 còn cần bảng `ai_cost_policy_audit_logs` đã tồn tại — nó được tạo ở T020, sau T011 và T012.
