# Hợp đồng: Phân loại lỗi cơ sở dữ liệu

**Ngày**: 2026-10-07 | **Kế hoạch**: [plan.md](plan.md) | **Đặc tả**: [spec.md](spec.md)

Hợp đồng phân biệt **sự cố** với **quyết định**. Hàng đợi xử lý khác nhau cho hai loại đó, và
một lỗi xếp sai nhóm sẽ quay vô hạn hoặc chết sớm.

---

## 1. Vấn đề

`23514` (vi phạm `CHECK`) đang được bọc trong `sharedevent.RetryableError` bởi
`GrantUseCase.transient` (`grant_uc.go:313`). Hệ quả quan sát được trên production
ngày 2026-10-07: 10 tài khoản không được sửa, mỗi tài khoản quay vòng tới giới hạn thử lại,
rồi thư chết.

Vi phạm `CHECK effective_to > effective_from` là hệ quả tất định của các giá trị đã tính.
Giao dịch đã lùi, dữ liệu không đổi giữa hai lần giao. Thử lại nó sẽ cho **cùng** kết quả.

---

## 2. Hợp đồng

### 2.1 Hàm phân loại

```go
// grant_error_class.go, tầng ứng dụng
func isCheckViolation(err error) bool
```

| Thuộc tính | Yêu cầu |
| --- | --- |
| Không chạm vào hàng đợi, hàng đợi hay RabbitMQ | Chỉ nhận `error` |
| Trả `false` với `nil` | Không ném lỗi |
| Đi qua `Unwrap` của mọi lớp bọc | `errors.As`, không phải kiểm tra kiểu trực tiếp |
| Chỉ `true` cho `SQLState == "23514"` | Xem mục 2.3 |
| Không đụng chuỗi lỗi | `SQLState` ổn định; thông điệp đã đổi qua nhiều phiên bản Postgres |

### 2.2 Hằng

```go
const pgCheckViolationCode = "23514"
```

Hằng phải nằm cạnh hàm, kèm chú thích nêu lý do **chỉ bắt `23514` chứ không bắt cả lớp
`23`**. Đây là điểm dễ bị người đọc sau "sửa cho gọn" thành
`strings.HasPrefix(code, "23")`, và cái sửa đó sẽ hỏng FR-010 mà không có dấu hiệu gì.

### 2.3 Bảng phân loại

| `SQLState` | Nghĩa | Kết quả | Có requeue không |
| --- | --- | --- | --- |
| `23514` | `check_violation` | Từ chối vĩnh viễn + mã lý do riêng (FR-009) | **Không** |
| `23505` | `unique_violation` | `transient` | Có |
| `23503` | `foreign_key_violation` | `transient` | Có |
| `23502` | `not_null_violation` | `transient` | Có |
| `40001`, `40P01` | serialization failure, deadlock | `transient` | Có |
| `57014`, `57P03` | query canceled, admin shutdown | `transient` | Có |
| mọi mã khác | — | `transient` | Có, **như hành vi hiện tại** |

Hàng cuối cùng là điểm quan trọng: 027 chỉ thêm một trường hợp. Mọi lỗi chưa được nhận diện
vẫn hành xử như trước, nên một `SQLState` lạ không biến thành thư chết.

### 2.4 Vì sao `23505` phải ở nhóm tạm thời

`GetGrantBySourceEventID` được gọi **dưới khoá tài khoản**, nên một `23505` khi thêm bản ghi
không phải là đua thông thường — nó có thể là sự kiện tới từ một nguồn khác đang giữ khoá
theo cách khác, hoặc khoá tư vấn không được dùng nhất quán. Dù lý do là gì, thử lại vẫn là
đúng: `23505` báo rằng có hàng đang tồn tại, và một lần thử nữa sau khi hàng kia biến mất
sẽ thành công. Xếp nó thành vĩnh viễn sẽ giết một sự kiện hợp lệ vì lý do không liên quan.

---

## 3. Điểm nối

Trong khối xử lý lỗi của giao dịch (`grant_uc.go:274-284`), nhánh phân loại mới đứng
**trước** nhánh `transient`:

```text
uow.Execute(...) trả err
  │
  ├─ err là *RejectionError?  ──► trả về nguyên văn        (giữ nguyên hiện tại)
  │
  ├─ isCheckViolation(err)?   ──► ghi mã lý do mới, trả PermanentError
  │                               (MỚI — FR-009)
  │
  └─ còn lại                ──► transient, RetryableError   (giữ nguyên hiện tại)
```

| Yêu cầu | Nguồn |
| --- | --- |
| Từ chối vĩnh viễn, **không** requeue | FR-009 |
| Dùng mã lý do **riêng**, không nhập nhẩm với mã cũ | FR-009 |
| Không thay đổi cách xử lý của sự cố tạm thời thật sự | FR-010 |
| Xuất hiện trong bộ đếm từ chối công khai | FR-011 |
| **Không** tự đặt ngưỡng cảnh báo | FR-011 |
| Không sửa bản ghi nào của tài khoản | FR-012 — giao dịch đã lùi trước khi tới đây |

Hệ quả cần nói rõ: hành vi hàng đợi **không đổi** — một sự kiện bị từ chối vĩnh viễn vẫn
chết thư. Cái đổi là nó **chết ngay với một mã lý do đọc được**, thay vì quay vô hạn trước
khi chết. Một nhân sự vận hành nhìn vào bộ đếm sẽ thấy `policy_grant_constraint_violation`
thay vì `transient_failure` tăng đều.

---

## 4. Rủi ro chưa đóng

`gorm.io/driver/postgres` có bảo toàn chuỗi lỗi không? Trong toàn repo, **không nơi nào** đọc
`pgconn.PgError` hay `SQLState` — kể cả `shared/infrastructure/db/`. Đây là lần đầu trong hệ
thống phân loại lỗi theo mã Postgres, nên chưa có bằng chứng nào trong repo.

Nếu GORM bọc bằng `fmt.Errorf` không dùng `%w`, `errors.As` không bóc được, hàm trả `false`,
lỗi rơi vào nhóm `transient` — và ta quay lại **đúng** lỗi đang sửa, không có tín hiệu cảnh
báo nào.

> **Bắt buộc**: một kiểm thử tích hợp trên Postgres thật, ép vi phạm `CHECK` thật, khẳng
> định `errors.As` bóc được xuyên qua GORM. Kiểm thử đơn vị với `PgError` dựng tay **không
> đủ** và sẽ cho cảm giác an toàn sai.

Ngoài ra `github.com/jackc/pgx/v5` hiện là phụ thuộc **gián tiếp** trong `services/ai/go.mod`
(`// indirect`, dòng 41). Dùng `pgconn.PgError` trong mã ứng dụng nghĩa là nó phải là phụ thuộc
**trực tiếp**. `go mod tidy` sẽ tự dọn; kiểm thử phải chạy sau khi tidy để bắt trường hợp
tệ hơn — dùng một gói mà không khai, thì build trên máy có thể vẫn qua.

---

## 5. Điều KHÔNG đổi

- **Không** thay đổi `sharedevent.PermanentError` hay `RetryableError` — phân loại nằm ở
  chỗ quyết định, không phải ở lớp bọc.
- **Không** thay đổi `transient()` hay `reject()`. Chúng vẫn giữ nguyên chữ ký và hành vi.
- **Không** thêm hàng đợi, không thêm loại thư, không đổi thiết lập dead-letter.
- **Không** phân loại `23514` ở tầng hạ tầng theo tên ràng buộc cụ thể
  (`user_ai_policy_grants_check`). Lý do không phải sở thích mà là bằng chứng: tên đó là
  **dấu vết lịch sử**, không còn khớp với bảng chứa nó. Ràng buộc được tạo bởi migration
  `20260622131631_add_ai_cost_policy.sql` dưới dạng `CHECK` viết nội tuyến không đặt tên,
  trên bảng lúc đó tên là `user_ai_policy_grants`. Sau đó bảng bị đổi tên thành
  `ai_policy_grants` và chuyển sang schema `aiexecutor`
  (`20260801000000_separate_module_schemas.sql` dùng `ALTER TABLE ... SET SCHEMA`). Cả hai
  thao tác đều **giữ nguyên** tên ràng buộc đã sinh tự động — nên production vẫn báo
  `user_ai_policy_grants_check` trong khi bảng là `aiexecutor.ai_policy_grants`. Người đọc
  thấy tên đó sẽ đi tìm một bảng không tồn tại. Mã `23514` là mức **đúng đắn**: nó áp dụng
  cho mọi ràng buộc `CHECK` của bảng này, kể cả ràng buộc được thêm sau này, và không cần
  sửa mã khi có thêm ràng buộc.