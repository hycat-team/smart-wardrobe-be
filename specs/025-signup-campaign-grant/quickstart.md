# Cẩu Nang Kiểm Chứng — Cơ Chế Chiến Dịch Tặng Gói

Cẩu nang chạy tay để chứng minh tính năng hoạt động đúng. Không phải tài liệu triển khai — thiết kế
nằm ở [data-model.md](./data-model.md) và [contracts/](./contracts/).

> **Không** chạy `make hygiene-scan` khi thực hiện cẩu nang này. Lệnh đó sinh tệp báo cáo làm rối
> kho mã nguồn, và `AGENTS.md` quy định phải được cho phép rõ ràng mới chạy.

> ### ⛔ CỔNG AN TOÀN — BẮT BUỘC ĐỌC TRƯỚC KHI CHẠY MỤC 2 VÀ MỤC 6
>
> **Cổng này bảo vệ hai thứ khác nhau. Đừng chỉ đọc một trong hai.**
>
> **1. Xoá dữ liệu.** Kịch bản **K2.2** và **K2.5** ở mục 2 chạy `TRUNCATE subscription.campaign_claims`
> và tạo **55 tài khoản thật**. Mục 6 sửa `configs/config.yaml` thành 7 cấu hình sai. Các lệnh này
> **xoá vĩnh viễn** sổ lượt cấp.
>
> **Chỉ chạy trên cơ sở dữ liệu kiểm thử. TUYỆT ĐỐI KHÔNG chạy trên production.**
>
> Chiến dịch `launch-2026-10` đang chạy trên production và **đã cấp gói cho đăng ký thật**.
> `TRUNCATE` bảng `campaign_claims` ở production sẽ xoá vĩnh viễn dấu vết quyền lợi đã bán cho
> khách hàng, phá vỡ FR-028, và làm mất khả năng trả lời câu hỏi "tôi có được tặng gói không"
> cho chính những người đó. Không có bước nào khôi phục được.
>
> **Kiểm tra môi trường trước khi chạy bất kỳ kịch bản nào ở mục 2 và mục 6:**
>
> ```sql
> SELECT current_database(), current_user, inet_server_addr();
> ```
>
> **Điều kiện chặn duy nhất: `current_database()` KHÔNG được trùng với `DB_NAME` trong `.env.production`.**
>
> Đó là tín hiệu duy nhất phân biệt được môi trường này với production, và **không** phải địa chỉ
> mạng. Lý do đáng ghi lại: cả môi trường phát triển lẫn production đều chạy bằng Docker Compose với
> `DB_HOST=postgres`, nên `inet_server_addr()` trả về địa chỉ bridge kiểu `172.18.0.5` ở **cả hai**
> bên. Địa chỉ loopback (`127.0.0.1`) chỉ xuất hiện khi bạn `psql` vào container — tức là một vòng lặp
> **bên trong** container, tách khỏi đúng đường mà ứng dụng thật sự kết nối. Kiểm tra loopback vì vậy
> cho kết quả **sai theo cả hai chiều**: nó chặn nhầm môi trường phát triển chạy container hóa, và
> không thể chứng minh được mình không đang ở production.
>
> Cách kiểm tra:
>
> ```powershell
> # production dùng tên nào — lấy từ chính tệp biến môi trường của production
> Select-String -Path .env.production -Pattern '^DB_NAME='
>
> # môi trường hiện tại đang nối tới tên nào
> psql -h <host> -U <user> -d <db> -c "SELECT current_database(), inet_server_addr()"
> ```
>
> Trong kho này: production là `closy_db`, môi trường phát triển cục bộ là `smart_wardrobe_db`.
> **Hai tên khác nhau** — chạy được ở đây. Nếu chúng trùng nhau thì dừng, không có bước nào khôi
> phục được.
>
> **2. Cướp suất của chiến dịch đang chạy.** **TUYỆT ĐỐI KHÔNG** khai báo một chiến dịch thứ hai
> vào `configs/config.production.yaml`. FR-011 chọn chiến dịch có `starts_at` **muộn nhất** trong
> số ứng viên, nên một chiến dịch mẫu thừa sẽ **thắng** chiến dịch ra mắt và cướp mất lượt đăng
> ký thật của người dùng — mà không có báo lỗi nào, vì mọi kiểm tra đều xanh. Chiến dịch mẫu chỉ
> được đặt trong `configs/config.yaml`. Chi tiết ở T089, T106.
>
> Vì vậy **không** bắt buộc hai tệp cấu hình phải giống nhau: đó chính là cơ chế biến việc thử
> nghiệm ở dev thành thay đổi hành vi production.

---

## 0. Điều kiện tiên quyết

| Cần | Kiểm tra |
|---|---|
| Go 1.26.0 | `go version` |
| PostgreSQL đang chạy | `make migrate-status svc=account` |
| `goose` đã cài | `goose -version`, nếu thiếu thì `make install-goose` |
| RabbitMQ, Redis đang chạy | `make dev` sẽ báo lỗi nếu thiếu |

Chạy kiểm thử hiện có trước, để biết nền sạch:

```powershell
make wire && make lint && make test-race
```

**Điều kiện đạt**: mã trả về 0. Nếu đỏ ngay từ đầu thì dừng — lỗi có sẵn không thuộc phạm vi
tính năng này.

---

## 1. Chuẩn bị dữ liệu

### 1.1 Chạy migration

```powershell
make migrate-up svc=account
```

Xác minh bằng SQL:

```sql
SELECT column_name FROM information_schema.columns
 WHERE table_schema = 'identity' AND table_name = 'users' AND column_name = 'created_by_admin';

SELECT indexname FROM pg_indexes
 WHERE schemaname = 'subscription' AND tablename = 'campaign_claims';
```

**Điều kiện đạt**: câu 1 trả về đúng 1 dòng. Câu 2 trả về 4 chỉ mục: `ux_campaign_claims_user`, `ix_campaign_claims_campaign_bucket`,
`ux_campaign_claims_watermark`, `ix_campaign_claims_campaign_granted`.

**Nếu hỏng**: đặc tả lược đồ ở `data-model.md` mục 2 và mục 3, có cả phần hoàn tác.

### 1.2 Khai báo chiến dịch

Thêm vào **cả** `configs/config.yaml` **và** `configs/config.production.yaml`. Bộ nạp cấu hình dùng
chế độ khớp chính xác, nên một tệp thiếu khoá sẽ không báo lỗi mà âm thầm lấy giá trị rỗng.

```yaml
campaigns:
  - code: launch-2026-10
    plan_slug: premium-monthly
    quota: 50
    reserve: 5
    starts_at: "2026-10-02T00:00:00+07:00"
    ends_at: null
```

> Khi thử nghiệm, đặt `starts_at` ở một thời điểm **đã qua** (ví dụ `"2020-01-01T00:00:00+07:00"`).
> Chiến dịch nằm ngoài cửa sổ thời gian sẽ bị từ chối ở bước điều kiện và không tạo lượt cấp nào.

### 1.3 Khởi động

```powershell
make dev
```

**Điều kiện đạt**:
- Nhật ký khởi động không có cảnh báo liên quan tới chiến dịch.
- Có hai dòng xác nhận: một bộ nhận thông báo đã đăng ký, một cơ chế quét bù đã lên lịch.
- Hàng đỗi mới tồn tại:

```powershell
rabbitmqctl list_queues name messages | Select-String "campaign_grant"
```

Phải thấy `campaign_grant_user_created_queue` và `campaign_grant_user_created_queue.dlq`.

---

## 2. Tám kịch bản bắt buộc

Đây là bằng chứng cho các yêu cầu khó nhất. Chạy theo thứ tự.

| Kịch bản | Chứng minh |
|---|---|
| K2.1 | SC-001, SC-001b, SC-001c, FR-022, FR-031 |
| K2.2 | SC-002, FR-025 |
| K2.3 | SC-003, FR-024, FR-039 |
| K2.4 | SC-016, FR-079, FR-080, FR-089 |
| K2.5 | SC-001d — thứ tự thông điệp không ảnh hưởng kết quả |
| K2.6 | SC-007, SC-006 — hết hạn tự hạ về `free`, 0 đồng bị |
| K2.7 | SC-006 — cấp gói không trừ đồng bị |
| K2.8 | SC-004 — không đoạn nào chậm hơn 10% |

Mỗi kịch bản dưới đây nêu rõ kỳ vọng kiểm chứng được, không chỉ nêu hành động.

### K2.1 — Trần cứng không bị vượt khi thông điệp tới ngược thứ tự

**Chứng minh**: SC-001, SC-001b, SC-001c, FR-022, FR-031

Cách làm thông điệp tới ngược thứ tự: tắt worker, tạo 55 tài khoản **theo thứ tự ngược** (tài
khoản cuối cùng trước), rồi bật lại worker. Hàng đỗi phát ra theo thứ tự ghi nên thứ tự xử lý là
ngược thứ tự tạo tài khoản.

Chờ hết hàng đỗi, chạy thêm một vòng quét bù, rồi:

```sql
SELECT COUNT(*) AS total,
       COUNT(*) FILTER (WHERE grant_bucket = 'main')    AS main_count,
       COUNT(*) FILTER (WHERE grant_bucket = 'reserve') AS reserve_count,
       COUNT(*) FILTER (WHERE fills_main_quota)         AS watermark_rows
  FROM subscription.campaign_claims
 WHERE campaign_code = 'launch-2026-10';

SELECT registered_at FROM subscription.campaign_claims
 WHERE campaign_code = 'launch-2026-10' AND fills_main_quota;
```

| Kiểm tra | Kỳ vọng |
|---|---|
| `total` | **từ 50 đến 55**. Tuyệt đối không vượt 55 |
| `main_count` | **không vượt 50** |
| `reserve_count` | **không vượt 5**. Đây là kiểm tra then chốt: nếu `reserve_count = 0` trong khi `total > 50` thì phần dự phòng đang hỏng |
| `main_count + reserve_count` | Bằng đúng `total`. Không lượt cấp nào mất dấu |
| `watermark_rows` | **đúng 1** |
| **Lượt cấp dự phòng** có `registered_at` ≥ mốc ưu tiên | **0**. Đây là điều FR-031 thật sự ràng buộc, và nó chỉ áp cho phần dự phòng |
| Lượt cấp **chính** có `registered_at` > mốc ưu tiên | **không giới hạn, và đó là hành vi đúng**. Ở thứ tự ngược, các tài khoản 7 tới 55 đều tạo sau mốc (tài khoản 6) mà vẫn được cấp từ phần chính, vì mốc ưu tiên chỉ được chốt tại lượt thứ 50 — tức sau khi cả 50 lượt đó đã quyết định |
| Số tài khoản tạo **sau** mốc ưu tiên mà được cấp **từ phần dự phòng** | **0** |

**Cảnh báo khi chạy kịch bản này**: tuyệt đối **không** viết phép kiểm tra "không tài khoản nào tạo
sau mốc ưu tiên được cấp" áp lên **toàn bộ** lượt cấp. Phép kiểm tra đó sai, và sẽ báo đỏ một cách
hợp lệ khi mã nguồn đang **đúng**. Phải lọc theo `grant_bucket` trước. Lý do: FR-031 chỉ ràng buộc
nhánh dự phòng.

Bốn dòng đếm tách bạch là cố ý. Nếu cài sai cột và đếm phần chính bằng
`COUNT(*) FILTER (WHERE NOT fills_main_quota)` thì kịch bản này vẫn xanh ở `total` và
`watermark_rows`, và chỉ lộ ra ở `main_count` — vốn là nơi lỗi thật nằm. Vì vậy bốn dòng phải được
đối chiếu cùng nhau, không kiểm tra lướt.

### K2.2 — Giao lại cùng một thông điệp 100 lần

**Chứng minh**: SC-002, FR-025

Đặt lại bảng về rỗng, rồi phát lại cùng một thông điệp 100 lần:

```sql
TRUNCATE subscription.campaign_claims;
```

```sql
SELECT current_plan_code, expires_at, is_auto_renew_enabled, version
  FROM subscription.user_subscriptions WHERE user_id = '<mã tài khoản>';
```

| Kiểm tra | Kỳ vọng |
|---|---|
| Số lượt cấp của tài khoản đó | **1** |
| `expires_at` | **1 tháng kể từ lúc được cấp gói**, không phải 2 tháng. Cũng không phải tính từ `registered_at` — xem FR-034 |
| `is_auto_renew_enabled` | `false` |
| `version` | `1` |
| Số dòng lịch sử thay đổi gói | 1 dòng duy nhất |

### K2.3 — Nhiều phiên bản chạy song song

**Chứng minh**: FR-022 (FR-023 đã được gộp vào FR-022 và không còn nội dung riêng), SC-001

1. Để còn đúng 1 suất trong phần chính.
2. Chạy 5 phiên bản dịch vụ `account` trên 5 cổng khác nhau.
3. Đăng ký 20 tài khoản gần như đồng thời.

```sql
SELECT s.current_plan_code, COUNT(*)
  FROM subscription.user_subscriptions s
  JOIN identity.users u ON u.id = s.user_id
 WHERE u.created_at > now() - interval '10 minutes'
 GROUP BY 1;
```

| Kiểm tra | Kỳ vọng |
|---|---|
| Số tài khoản được cấp trong nhóm 20 | **đúng 1** |
| 19 tài khoản còn lại | Vẫn ở gói `free` |

### K2.4 — Tắt worker 10 phút rồi bật lại

**Chứng minh**: SC-016, FR-079, FR-080, FR-089

1. Ghi lại số suất đã cấp.
2. Đặt `APP_MODE=http` để không khởi động worker, rồi khởi động lại.
3. Tạo 20 tài khoản đủ điều kiện.
4. Trả `APP_MODE` về `all`, khởi động lại.
5. **Không** chạy quét bù thủ công — chờ đủ một chu kỳ lịch.

| Kiểm tra | Kỳ vọng |
|---|---|
| Số tài khoản đủ điều kiện mất quyền lợi | **0** |
| Nhật ký cơ chế quét bù | Có một lần chạy thành công, ghi số tài khoản đã xử lý |
| Nhật ký cho thấy quét bù dùng chung bộ luật hạn mức với đường thông điệp | **Có** (FR-080) |

---

### K2.5 — Thứ tự thông điệp không ảnh hưởng tính đúng đắn (SC-001d)

K2.1 chạy thông điệp **ngược** thứ tự. Kịch bản này chạy **thuận** thứ tự rồi so kết quả.

**Tiêu chí so sánh không phải "cùng một tập tài khoản được cấp"** — mà là **mốc ưu tiên và trần cứng
không đổi theo thứ tự xử lý**. Tập tài khoản thì **có** khác, và khác là chấp nhận được (xem bảng
"Giới hạn đã biết" trong `spec.md`).

1. Đặt lại bảng lượt cấp về rỗng, xoá hàng đỗi.
2. Tạo 55 tài khoản **theo đúng thứ tự** (tài khoản đầu tiên trước), không tắt worker.
3. Chờ hết hàng đỗi, chạy một vòng quét bù.
4. Chạy lại câu SQL đếm của K2.1.

| Kiểm tra | Kỳ vọng |
|---|---|
| `main_count` | **50** — giống hệt K2.1 |
| `watermark_rows` | **đúng 1** — giống hệt K2.1 |
| `total` | **50** |
| Mốc ưu tiên | Trỏ về tài khoản tạo **sớm nhất** (ở thứ tự thuận là tài khoản 50) |
| Lượt cấp dự phòng có `registered_at` ≥ mốc | **0** |

**So sánh hai kịch bản — cột nào phải giống, cột nào được phép khác:**

| Đại lượng | K2.1 (ngược) | K2.5 (thuận) | Phải giống? |
|---|---|---|---|
| `main_count` | 50 | 50 | **Có** — đây là phần chính, không phụ thuộc thứ tự |
| `watermark_rows` | 1 | 1 | **Có** |
| `total` | **55** | **50** | **Không** — và đây là hành vi đúng, xem bên dưới |
| `reserve_count` | 5 | 0 | **Không** |
| Tập được cấp | 1 tới 55 (tất cả) | 1 tới 50 | **Không** |
| Trần cứng 55 không bị vượt | Không bị vượt | Không bị vượt | **Có** |

**Vì sao `reserve_count` khác nhau mà vẫn đúng**: ở thứ tự thuận, lượt thứ 50 là tài khoản 50, nên 5
tài khoản còn lại (51 tới 55) đều tạo **sau** mốc ưu tiên — không ứng viên hợp lệ cho dự phòng.
Ở thứ tự ngược, lượt thứ 50 là tài khoản 6, nên 5 tài khoản còn lại (5, 4, 3, 2, 1) đều tạo
**trước** mốc — cả 5 vào dự phòng.

Điều kiện để hai kịch bản này **đều đúng** là: `main_count` bằng nhau, `watermark_rows` bằng 1,
và `total` không vượt 55. Nếu `main_count` lệch nhau thì phần chính đã phụ thuộc thứ tự, tức vi
phạm FR-022. Nếu `watermark_rows` khác 1 thì vi phạm FR-030.

---

### K2.6 — Hết hạn rồi tự hạ về gói miễn phí, không trừ đồng bộ (SC-007, SC-006)

Kịch bản này kiểm tra hành vi **không phải phần mới** — nó thuộc worker gia hạn có sẵn — nhưng nó
quyết định khách hàng có thực sự nhận được quyền lợi dài hạn hay không. Không kiểm thì cả chiến dịch
có thể "thành công" mà khách hàng mất quyền sau một tháng.

1. Cấp gói cho một tài khoản, ghi lại `expires_at` (kỳ vọng `granted_at + 30 ngày`, **không** phải `registered_at + 30 ngày` — FR-034; `granted_at` nằm trong `subscription.campaign_claims`).
2. Dời thời gian hệ thống về sau `expires_at` (đặt `starts_at`/`ends_at` đã qua, hoặc dùng cấu hình
   `premium-monthly` ngắn hạn ở môi trường kiểm thử).
3. Chờ một chu kỳ của worker gia hạn.

| Kiểm tra | Kỳ vọng |
|---|---|
| Gói sau khi hết hạn | `free` |
| `is_auto_renew_enabled` | **false** — không được tự mua thêm |
| **Số đồng bị** (ngân sách hằng ngày) | **0**. Không tự hạ cấp thì không được trừ bất kỳ đồng bị nào |
| Sự kiện phát ra khi hạ về `free` | đúng 1 dòng `subscription.plan.changed` với `toPlanId` của gói `free` |
| Quyền AI theo sau | Thu hẹp về chính sách của gói `free` |

Bảng "Số đồng bị" là dòng quan trọng nhất. Nếu nó khác 0 thì cơ chế tự hạ về gói miễn phí đang trừ
đồng bị mà không cấp gói, tức là phạt khách hàng vì hết hạn quà tặng.

---

### K2.7 — Không trừ đồng bị khi cấp gói (SC-006)

1. Ghi lại số đồng bị trước khi cấp.
2. Cấp gói cho một tài khoản mới.
3. So sánh ngay sau khi cấp và sau một chu kỳ quét bù.

| Kiểm tra | Kỳ vọng |
|---|---|
| Số đồng bị thay đổi | **0**, ở cả hai mốc |
| Lý do | Cấp gói chiến dịch không phải giao dịch mua, nên không sinh chi phí cho ví |

---

### K2.8 — Không đoạn nào chậm hơn 10% (SC-004)

SC-004 là tiêu chí **không hồi quy**, nên nó chỉ có nghĩa khi so với một mốc chuẩn đo trước. Đo
mốc này **trước khi** bật cơ chế chiến dịch, rồi so sau.

1. **Trước khi bật**: đăng ký 20 tài khoản, ghi lại trung vị thời gian xử lý đăng ký và trung vị thời
   gian khởi tạo gói mặc định.
2. **Sau khi bật**: lặp lại đúng quy trình, cùng số lượng tài khoản, cùng khoảng thời gian.
3. So sánh trung vị của từng đoạn.

| Đoạn | Ngưỡng |
|---|---|
| Xử lý POST đăng ký (từ lúc nhận tới lúc trả) | Chậm hơn mốc trước **không quá 10%** |
| Khởi tạo gói mặc định | Chậm hơn **không quá 10%** |
| Thời gian giao dịch `campaign_claims` | Dưới 50 ms |

Đoạn thứ ba là chỗ dễ trượt nhất. Nó phải **không** tăng tuyến tính theo số lượt cấp: phần chính
chỉ quét 50 dòng, và chỉ mục `ix_campaign_claims_campaign_bucket` bảo đảm điều đó. Nếu thời gian
tăng theo số tài khoản đã cấp thì đang quét bảng không có chỉ mục — xem lại `CountByBucket` ở
`data-model.md` mục 3.3.

Cơ chế chống trùng và khoá tư vấn **không** làm thời gian tăng vì chúng chạy trong transaction
đăng ký sẵn có, không phải một lượt gọi mạng nào thêm.

---

## 3. Kiểm chứng nhóm A — sửa bộ đếm lượt dùng AI

**Chứng minh**: SC-009, FR-053, FR-054, FR-055, FR-057

Đưa một tài khoản lên sát hạn mức: 29 trên 30 lượt trong ngày, rồi chạy 3 thao tác dùng AI song song.

```sql
SELECT outfit_recommend_count, ai_usage_count, last_reset_date
  FROM subscription.user_daily_quotas WHERE user_id = '<mã tài khoản>';
```

| Kỳ vọng | Lý do |
|---|---|
| Tổng số lượt = **30** | FR-053 |
| **Không** phải 29 | đó là lỗi đếm thiếu trước khi sửa |
| **Không** phải 32 | hạn mức vẫn phải được áp dụng |
| Lần dùng AI thứ 31 bị từ chối với thông báo lỗi phù hợp | FR-057 |

Kiểm tra ràng buộc không khoá (FR-055): trong khi một thao tác đang ghi bộ đếm, một yêu cầu
`GET /subscriptions/me/daily-quota` phải trả về **không bị treo**.

Kiểm tra sang ngày mới (FR-054): đặt `last_reset_date` lùi một ngày, gọi lại đường dẫn trên, hai
bộ đếm phải về 0.

---

## 4. Kiểm chứng nhóm B — sự kiện tạo tài khoản

**Chứng minh**: FR-084 tới FR-091 · kịch bản 1 tới 7 trong
[contracts/account-created-event.md](./contracts/account-created-event.md)

Chạy cả ba đường tạo tài khoản:

```sql
SELECT provider, created_by_admin, created_at
  FROM identity.users
 WHERE created_at > now() - interval '30 minutes'
 ORDER BY created_at;
```

```sql
SELECT payload::text FROM shared.outbox_records
 WHERE topic = 'identity.user.created'
 ORDER BY created_at DESC LIMIT 1;
```

| Đường | `created_by_admin` | `createdByAdmin` trong thông điệp |
|---|---|---|
| Đăng ký email | `false` | `false` |
| Đăng nhập Google tạo mới | `false` | `false` |
| Quản trị viên tạo tài khoản | `true` | `true` |

**Điều kiện đạt**: hai cột khớp nhau trên mọi dòng. Đây là kiểm thử chống trôi lệch quan trọng nhất
— hai nguồn suy ra cùng một sự thật từ hai đường khác nhau.

| Kiểm tra thêm | Kỳ vọng |
|---|---|
| Số khoá trong thông điệp | **đúng 4**: `userId`, `provider`, `registeredAt`, `createdByAdmin`. Không có email, tên, ngày sinh (FR-087) |
| Tài khoản do quản trị viên tạo có gói gì | `free`, `expires_at` rỗng (FR-051) |
| Liên kết Google vào tài khoản có sẵn | **0** thông điệp phát ra — tài khoản không phải mới |
| Hai người cùng đăng ký một email | Tài khoản thứ hai không được tạo ⇒ **0** thông điệp thứ hai |
| Thông điệp không đọc được nội dung | Vào hàng đỗi chết, không thử lại vô hạn (FR-063) |

Truy vấn gói của tài khoản do quản trị viên tạo:

```sql
SELECT s.current_plan_code, s.expires_at
  FROM subscription.user_subscriptions s
  JOIN identity.users u ON u.id = s.user_id
 WHERE u.created_by_admin = true
 ORDER BY u.created_at DESC LIMIT 1;
```

---

## 5. Kiểm chứng nhóm E — giao diện tra cứu

**Chứng minh**: FR-065 tới FR-072, SC-011

```powershell
$admin = "<token của tài khoản quản trị viên>"

curl -H "Authorization: Bearer $admin" http://localhost:8080/admin/campaigns

curl -H "Authorization: Bearer $admin" `
  http://localhost:8080/admin/campaigns/launch-2026-10/eligibility/<userId>

curl -H "Authorization: Bearer $admin" http://localhost:8080/admin/campaigns/khong-ton-tai
```

| Kiểm tra | Kỳ vọng |
|---|---|
| Danh sách chiến dịch | Có `status`, `degraded`, `grantedMainCount`, `remainingMain`, `hardCap` |
| Tài khoản đã được cấp | `eligibility = "granted"`, `granted = true` |
| Tài khoản đủ điều kiện, còn suất, nhưng **chưa** được cấp (thông điệp còn trên đường, hoặc quét bù chưa tới) | `eligibility = "eligible_pending"`, `granted = false`, `reason` **rỗng** |
| Tài khoản đủ điều kiện nhưng hết suất | `eligibility = "eligible_but_exhausted"` |
| Tài khoản quản trị viên tạo | `eligibility = "created_by_admin"` |
| Mã chiến dịch sai | 404 với thông điệp tiếng Việt nói mã không tồn tại, **không** phải danh sách rỗng (FR-072) |
| Gọi bằng token người dùng thường | 403, không lộ nội dung gì (FR-070) |

**Điều kiện đạt quan trọng nhất**: hai trường hợp giữa trả về **khác nhau**. Trả về giống nhau là
không đạt FR-068.

Kiểm tra chỉ đọc (FR-069): gọi đường dẫn danh sách 50 lần, `grantedMainCount` không đổi, và không
có bản ghi hàng đỗi chủ đề `notification.email` nào sinh ra.

---

## 6. Kiểm chứng ranh giới — những thứ KHÔNG được phép xảy ra

Đây là các lỗi mà trình biên dịch **không** bắt được. Không cái nào biểu hiện bằng lỗi.

| # | Kiểm tra | Cách kiểm tra | Điều kiện đạt |
|---|---|---|---|
| 1 | Dịch vụ AI chỉ tiêu thụ **đúng một** chủ đề (FR-097) | Kiểm thử trong `shared/infrastructure/messaging` | `Topology` chỉ có **một** liên kết chứa `TopicUserCreated` |
| 2 | Tập loại sự kiện đổi gói không có loại mới (FR-092) | So sánh `subscriptioneventtype.All` với kỳ vọng | Đúng 7 loại, không thêm không bớt |
| 3 | Dịch vụ AI không bị sửa (R7) | `git diff --stat -- services/ai` | Rỗng |
| 4 | Thông điệp không chứa dữ liệu nhận diện cá nhân (FR-087) | Truy vấn ở mục 4 | Đúng 4 khoá |
| 5 | Ghi `created_by_admin = true` chỉ ở một chỗ | Tìm kiếm văn bản trong `services/account` | Đúng **một** vị trí ghi |
| 6 | Cấu hình chiến dịch có ở cả hai tệp cấu hình | So sánh hai tệp trong `configs/` | Cả hai đều có khóa `campaigns` và phân tích được. `config.yaml` có **hai** mã, `config.production.yaml` có **một** — **không** đòi giống nhau (T089, T106) |
| 7 | Lần cấp gói ghi **một** dòng lịch sử (FR-095) | Đếm dòng lịch sử của một tài khoản được cấp | Đúng 1 dòng cho mỗi lần cấp |

### 6.1 Cấu hình sai phải chặn khởi động (FR-010, SC-014)

Sửa `configs/config.yaml`, khởi động lại, kiểm tra dịch vụ **không** khởi động được và thông báo
chỉ đúng trường sai. Thử lần lượt 7 trường hợp:

| Cấu hình lỗi | Kỳ vọng |
|---|---|
| `code` rỗng | Chặn, nói đúng `code` |
| `code` sai định dạng — ví dụ `Launch_2026` (hoa chữ, gạch dưới) hoặc mã 2 ký tự | Chặn, nói đúng `code` |
| Hai chiến dịch trùng `code` | Chặn, nói đúng trùng lặp |
| `quota: 0` | Chặn, nói đúng `quota` |
| `reserve: -1` | Chặn, nói đúng `reserve` |
| `ends_at` trước `starts_at` | Chặn, nói đúng khoảng thời gian |
| `starts_at` không có múi giờ | Chặn, nói đúng định dạng thời gian |

Ca `code` sai định dạng không thừa: ràng buộc `^[a-z0-9][a-z0-9-]{2,63}$` chặn ký tự lạ, và `code`
đi thẳng vào cột `campaign_code` cùng nhật ký kiểm toán — một mã vô tình được ghép vào tên hàng đỗi
sẽ là đường truy cận (FR-038).

Sau mỗi lần, **hoàn tác** cấu hình trước khi thử lần tiếp theo.

**Hai trường hợp của `plan_slug` khác nhau, và chỉ một cái rời khỏi bảng này.**

- **`plan_slug` rỗng** vẫn **chặn khởi động**. Đây là lỗi hình dạng cấu hình, kiểm được mà không
  cần chạm cơ sở dữ liệu.
- **`plan_slug` không tồn tại trong CSDL, hoặc trỏ tới gói không còn active** thì **không** chặn
  khởi động — nạp cấu hình không được chạm cơ sở dữ liệu, nên không thể tra tên gói lúc bật dịch
  vụ. Nó chỉ khiến chiến dịch đó không cấp được gì cho ai, và mỗi lượt bị từ chối đều ghi
  `plan_unavailable` (FR-010).

Kiểm chứng trường hợp thứ hai ở mục 4 chứ không ở đây: khai báo một `plan_slug` không tồn tại,
rồi xác nhận dịch vụ **vẫn khởi động bình thường** và mỗi lượt bị từ chối đều ghi
`plan_unavailable`.

---

## 7. Chạy lại toàn bộ trước khi kết luận

```powershell
make wire && make lint && make test-race
```

**Điều kiện đạt**: cả ba lệnh trả về 0.

Nếu chưa đạt, chạy `make hygiene-scan` **chỉ khi được cho phép rõ ràng** — lệnh đó sinh tệp báo cáo
trong kho mã nguồn.

---

## 8. Danh sách đối chiếu cuối

| Nhóm | Yêu cầu | Tiêu chí | Kịch bản |
|---|---|---|---|
| A | FR-052 tới FR-058 | SC-009, SC-010 | Mục 3 |
| B | FR-084 tới FR-088, FR-093 tới FR-095 | SC-013 | Mục 4 |
| C | FR-021 tới FR-033, FR-078 tới FR-083, FR-089 | SC-001 tới SC-001d, SC-002, SC-003, SC-015, SC-016 | Mục 2 K2.1 tới K2.5 |
| C | FR-034 tới FR-040, FR-091 tới FR-095 | SC-005 tới SC-008, SC-012 | Mục 2, mục 4 |
| C | FR-041 tới FR-043 | SC-004 | Mục 2 **K2.8** — đo độ chậm |
| C | FR-044 tới FR-049 | SC-005 | Mục 2 K2.1 (kiểm tra hộp thư) |
| C | FR-050, FR-051 | SC-006 | Mục 2 **K2.7** |
| C | FR-010 | SC-014 | Mục 6.1 |
| C | FR-059 tới FR-064 | SC-003, SC-012 | Mục 2, mục 5 |
| C | FR-001 tới FR-005, FR-090 | SC-012, SC-013 | Mục 6.1 (chạy được khi không khai báo chiến dịch) |
| D | FR-006 tới FR-009 | SC-010 | Mục 1.2, mục 6.1 |
| D | FR-092, FR-096, FR-097, FR-098 | — | Mục 6 |
| E | FR-065 tới FR-072 | SC-011 | Mục 5 |
| Tất cả | FR-001 tới FR-005 | SC-010, SC-013 | Xem `data-model.md` mục 9 |
