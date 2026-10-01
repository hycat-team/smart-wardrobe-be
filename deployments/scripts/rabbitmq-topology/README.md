# RabbitMQ topology reconcile

`x-queue-type` là argument **bất biến** của một queue trong RabbitMQ. Queue đã tồn tại với
type `classic` thì không thể đổi thành `quorum` — broker trả `406 PRECONDITION_FAILED` và
**từ chối khai báo**, kể cả khi queue rỗng.

Hệ quả: `topology.go` (code) và broker state là **hai nguồn sự thật độc lập**. Sửa code để
declare quorum mà broker vẫn giữ queue classic ⇒ mọi service khai báo topology đó **không
khởi động được**, và phải xoá queue cũ rồi khai báo lại.

Thư mục này là lớp phòng thủ cho đúng tình huống đó.

---

## Kiểm tra (check) — tự động

CI chạy check trước **mọi** deploy, trên cả hai môi trường:

| Workflow | Bước | Broker |
|---|---|---|
| `deploy-vps-services.yml` | `Check RabbitMQ queue topology` | container `rabbitmq` trong network `closy-prod_data` |
| `deploy-aws.yml` | `Check RabbitMQ queue topology` | CloudAMQP (ngoài box, qua internet) |

Check là **read-only** — không đụng queue nào. Exit 1 khi có drift ⇒ **workflow đỏ trước khi
container nào được start**, thay vì phải đọc log crash-loop trên VPS.

`MISSING` (queue chưa tồn tại) **không** làm đỏ CI: service tự tạo queue đó lúc khởi động,
và queue mới thêm vào registry cũng rơi vào trạng thái này ở lần deploy đầu. Dùng `--strict`
nếu muốn kiểm tra cả.

## Migrate — thủ công, có chủ ý

Migrate **xoá và tạo lại queue**, nên nó phải chạy thủ công:

- CI **không** bao giờ tự migrate.
- Migrate **từ chối chạy** nếu queue còn consumer (`Consumers > 0`) — xoá queue dưới chân
  worker đang chạy là race với delivery đang lửng.
- Người vận hành dừng worker → chạy migrate → start lại.

---

## Khi nào cần chạy

Khi `docker compose logs` (hoặc log trên VPS) báo:

```
PRECONDITION_FAILED - inequivalent arg 'x-queue-type' for queue '<tên>' ...
  received the value 'quorum' ... but current is none
```

Hoặc khi CI đỏ ở bước **Check RabbitMQ queue topology** — lúc đó log có đúng danh sách queue
lệch và `want`/`actual`.

Quan trọng: đổi `x-queue-type` là việc **hiếm** và **vĩnh viễn**. Chúng không phải chuyện phải
làm mỗi deploy — lần đầu tiên bị bỏ sót là lần đó tốn thời gian debug log trên VPS. Check tự
động tồn tại để bắt đúng lần đó.

---

## Cách dùng

Cần `TOOL_IMAGE` trỏ tới image tool mà CI vừa build (digest được in ra trong log workflow).
Có thể override qua biến môi trường, hoặc sửa trong `common.sh`:

```bash
export TOOL_IMAGE=ghcr.io/hycat/rabbitmq-topology@sha256:<digest>
```

Digest nằm ở bước **Resolve image digest** của workflow `deploy-vps-services.yml` /
`deploy-aws.yml`. Dùng `:latest` cũng được nếu tool gần đây không đổi, nhưng digest mới là
đúng thứ CI đã kiểm chứng.

### 1. Xem drift (read-only)

```bash
bash deployments/scripts/rabbitmq-topology/01-check-drift.sh vps
bash deployments/scripts/rabbitmq-topology/01-check-drift.sh aws
```

Exit: `0` không lệch · `1` có drift · `2` lỗi setup/broker không tới được.

### 2. Migrate

**VPS** (`/opt/closy`, broker là container trong compose):

```bash
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh
# xem trước phần sẽ làm, chưa chạy
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh --yes
```

Script tự dừng `account commerce ai` (trên VPS mỗi container chạy **cả** http lẫn worker), rồi
migrate, rồi verify lại.

**AWS** (`/opt/closy-aws`, broker là CloudAMQP bên ngoài):

```bash
bash deployments/scripts/rabbitmq-topology/03-migrate-aws.sh
bash deployments/scripts/rabbitmq-topology/03-migrate-aws.sh --yes
```

Khác biệt so với VPS:

- **Không có** `--network`: tool tới CloudAMQP qua internet (`amqps`, management TLS — đọc từ
  `.env.aws`).
- Chỉ dừng `account-worker commerce-worker ai-worker`; container `-http` không consume queue.
- Cần `RABBITMQ_VHOST` đúng. Sai vhost ⇒ CloudAMQP trả `403 no access to this vhost`, và script
  báo lỗi setup chứ không báo "mọi queue đều MISSING".

Một queue cụ thể:

```bash
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh --yes --queue subscription_plan_changed_queue
```

### 3. Verify

```bash
bash deployments/scripts/rabbitmq-topology/04-verify.sh vps
```

In ra trạng thái type **và** độ dài các DLQ — queue đúng type vẫn có thể đang âm thầm fail
vào DLQ, và đó là thứ đáng biết sau một lần đổi topology.

---

## Message có bị mất không?

Không. Quy trình là **drain → xoá → khai báo lại → replay**:

1. **Drain** — mỗi message được ghi ra file JSONL **rồi mới** ack. Nếu tool crash giữa chừng,
   message chưa ack vẫn nằm trong queue; file JSONL còn lại để replay tay.
2. **Xoá** queue cũ (binding đi theo).
3. **Khai báo lại** đúng args từ `topology.go` + bind lại từng routing key.
4. **Replay** — publish qua **default exchange** với routing key là tên queue.

Bước 4 dùng default exchange (không dùng exchange của queue): message đã drain mang routing
key lúc **publish**, không nhất thiết trùng binding key của queue đó. Gửi lại qua topic
exchange có thể không khớp binding nào và **bị drop âm thẩm** — queue ra khỏi trống sau một lần
"migrate thành công". Default exchange định tuyến theo tên queue nên message về đúng queue
đang khôi phục. Headers và message ID được giữ nguyên nên consumer không nhận ra khác biệt.

File spill nằm ở `/opt/closy-rabbitmq-drain` trên host (mount vào container để không mất khi
container thoát). Giữ lại để audit; xoá sau khi đã xác nhận OK.

**Đã đánh đổi biết:** replay đi qua DLX với chính message đã dead-letter trước đó, nên chuỗi
header `x-death` bị nhân bản. Chấp nhận được vì migrate chỉ chạy khi queue rỗng hoặc gần rỗng.

---

## Ngân sách connection trên CloudAMQP

Gói CloudAMQP đang dùng là **shared plan**: trần connection thấp hơn nhiều so với ghi trong
runbook trước đây (xem mục Limits trên dashboard). Migrate mở thêm một connection để drain;
tool in số connection đang mở **trước khi** migrate để bạn thấy còn dư đà không.

Nếu số connection đang sát trần, dừng worker trước (script đã làm) — giảm phần lớn các
connection của app trước khi tool kết nối.

---

## Rollback

Sau khi migrate, queue là type mới. Không thể quay lại type cũ mà không làm lại drain/delete/
redeclare — dùng lại đúng các script này sau khi đổi `topology.go` về giá trị cũ.

Rollback **deploy** (image cũ) sau khi đã migrate là an toàn: image cũ khai báo topology cũ,
nhưng queue trên broker đã là type mới. Kiểm tra bằng `01-check-drift.sh` xem image cũ có
còn khớp trước khi rollback.

---

## Liên quan

- `shared/infrastructure/messaging/topology.go` — registry, nguồn sự thật duy nhất
- `shared/tools/rabbitmq-topology/` — tool Go (`check` / `migrate`)
- `Makefile` — `rabbitmq-topology-check`, `rabbitmq-topology-migrate`, `image-build-tool`
- `docs/development/messaging-guide.md` — thêm queue/topic, đổi queue type
- `docs/decisions/016-rabbitmq-refactor.md` — quyết định chấp nhận downtime drain