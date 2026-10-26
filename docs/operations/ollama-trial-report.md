# Báo cáo go/no-go — Thử nghiệm Ollama trên EC2 (feature 032)

**Feature**: `specs/032-ollama-lightsail-trial/` | **Ngày tạo**: 2026-10-26 | **Trạng thái**: mẫu
(TEMPLATE) — chưa có dữ liệu thử nghiệm

Báo cáo này tổng hợp dữ liệu đã thu trong tuần thử nghiệm và chốt quyết định `go` / `adjust` /
`no-go`. Nguồn dữ liệu chuẩn là
[contracts/trial-telemetry.md](../../specs/032-ollama-lightsail-trial/contracts/trial-telemetry.md)
và quy trình vận hành là `docs/operations/ollama-trial-runbook.md`. Không chỉnh ngưỡng sau khi bắt
đầu đo mà không ghi lý do; không dùng uptime làm tiêu chí quyết định.

## 1. Thông tin thử nghiệm

| Trường | Giá trị | Ghi chú |
|---|---|---|
| Cửa sổ thử nghiệm (`trial_started_at` → ngày 7) | `TODO(operator)` | UTC, khớp runbook mục 1 |
| Commit baseline | `TODO(operator)` | `git rev-parse --short HEAD` (read-only) |
| Image AI đã deploy | `TODO(operator)` | Tag image từ pipeline hiện hữu |
| Proxy hostname / Elastic IP | `TODO(operator)` | Không ghi token/secret ARN đầy đủ |
| Model / Ollama version / digest | `TODO(operator)` | Local và EC2 phải khớp |
| Static IP Lightsail dùng cho allowlist | `TODO(operator)` | Đổi IP thì phải ghi lại mốc thời gian |
| Người vận hành | `TODO(operator)` | Danh tính |
| Phạm vi xác nhận | Không thao tác VPS; không đổi Dockerfile/Compose/Nginx AWS/pipeline; không EKS/ALB/ACM | Tick sau khi rà diff |

## 2. Kết quả smoke test và phạm vi

| Hạng mục | Kỳ vọng | Thực tế | Đạt? | Bằng chứng |
|---|---|---|---|---|
| 6 container Lightsail | 3 `http` + 3 `worker` healthy | `TODO(operator)` | TODO | `docker compose ps` |
| Ba capability text | `chat_text`, `recommendation_text`, `free_text` do `openai`/Ollama phục vụ | `TODO(operator)` | TODO | log provider/model |
| `summary` / `rewriter` | Route free → `free_text`, route paid → `chat_text`; cả hai `openai` | `TODO(operator)` | TODO | log provider/model |
| Vision / embedding | Vẫn do `google` phục vụ | `TODO(operator)` | TODO | log provider/model |
| Chi phí usage event | Ba đường text có `actual_cost_micro_vnd = 0` | `TODO(operator)` | TODO | `ai_usage_events` |
| Artefact deployment | 0 diff Dockerfile/Compose/Nginx AWS/pipeline; 0 tài nguyên EKS | `TODO(operator)` | TODO | `git diff --stat` |
| VPS | 0 lệnh/probe/thay đổi | `TODO(operator)` | TODO | diff + nhật ký phiên |

## 3. SC-007 — Bảng chỉ số bắt buộc

Bảng dưới đây là hợp đồng báo cáo: **100% dòng phải có số liệu thật** trước khi chốt quyết định.
Cột `Giá trị thực tế` để trống chỉ được phép khi báo cáo còn ở trạng thái mẫu.

| # | Chỉ số bắt buộc | Nguồn | Đơn vị / ngưỡng tham chiếu | Giá trị thực tế | Đạt? | Bằng chứng |
|---|---|---|---|---|---|---|
| 1 | Smoke test 6 container Lightsail | `docker compose ps` + smoke script | 6/6 healthy; 100% text do Ollama phục vụ khi không rơi fallback đủ điều kiện (SC-001) | `TODO(operator)` | TODO | TODO |
| 2 | Kiểm soát truy cập / xác thực | `proxy_401_total`, `proxy_5xx_total`, test từ IP ngoài | 100% IP ngoài allowlist bị chặn; 100% thiếu/sai Bearer bị từ chối; 100% Lightsail được chấp nhận; `11434` không public (SC-003) | `TODO(operator)` | TODO | TODO |
| 3 | Lỗi | Breakdown `outcome` trong structured log | Liệt kê đủ `timeout`, `transport`, `http_4xx`, `http_429`, `http_5xx`, `malformed`, `schema_invalid`, `max_tokens`, `canceled` | `TODO(operator)` | TODO | TODO |
| 4 | Độ trễ AI | `ai_ttft_ms`, `ai_latency_ms` | p95: `chat_text`/`free_text` TTFT ≤15s; `recommendation_text` TTFT ≤60s và hoàn tất ≤120s; ≥95% attempt Ollama ≤90s (SC-005) | `TODO(operator)` | TODO | TODO |
| 5 | Tỷ lệ fallback | `ai_fallback_total` / tổng request, theo `fallback_reason` | Mỗi fallback đủ điều kiện được đếm riêng; 100% timeout/429/5xx/malformed trước chunk đầu gọi Gemini đúng 1 lần; cancel/4xx = 0 lần (SC-009) | `TODO(operator)` | TODO | TODO |
| 6 | Hàng đợi | `OLLAMA_MAX_QUEUE`, log queue full, 429/503 | Queue hữu hạn; không có attempt thứ ba; không treo backend (SC-005, SC-009) | `TODO(operator)` | TODO | TODO |
| 7 | CPU / RAM / disk EC2 | CloudWatch `ec2_cpu_percent`, `ec2_memory_bytes`, `ec2_disk_free_bytes` | CPU >85% kéo dài 10 phút → điều tra; RAM >90% → điều tra; disk free <10 GiB → điều tra | `TODO(operator)` | TODO | TODO |
| 8 | Chi phí AWS | AWS Cost Explorer + Budget 80 USD (cảnh báo 50/80/100%) | Ghi chi phí thực tế trong tuần; vượt 80 USD thì ghi nhận, không tự hạ cấu hình/teardown | `TODO(operator)` | TODO | TODO |

**Quy tắc điền bảng**

- `Giá trị thực tế` phải lấy từ dữ liệu đã thu; không suy đoán, không nội suy.
- Mỗi dòng phải tham chiếu được một bằng chứng (file log/metric/ảnh) trong runbook mục 7.
- `thinking_tokens > 0` trên provider `openai` là lỗi cấu hình: dừng đo, sửa, ghi sự cố; không được
  coi trial là hợp lệ (FR-026, contracts/trial-telemetry.md §5).
- Dòng chưa có dữ liệu phải ghi rõ `chưa thu được` + lý do, không đánh `Đạt`.

## 4. Sự cố trong tuần

| Thời điểm (UTC) | Sự cố | Ảnh hưởng | Xử lý | Còn tồn? |
|---|---|---|---|---|
| `TODO(operator)` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` | TODO |

## 5. Chi phí

| Khoản | Ước tính | Thực tế | Ghi chú |
|---|---|---|---|
| EC2 `c6i.2xlarge` on-demand | `TODO(operator)` | `TODO(operator)` | Theo giờ chạy thực tế |
| EBS gp3 (data 50 GiB + root) | `TODO(operator)` | `TODO(operator)` | Tính cả thời gian chờ quyết định |
| Elastic IP | `TODO(operator)` | `TODO(operator)` | Tính khi địa chỉ được cấp nhưng không gắn |
| Secrets Manager / SSM / CloudWatch | `TODO(operator)` | `TODO(operator)` | Retention log 14 ngày |
| **Tổng** | `TODO(operator)` | `TODO(operator)` | Đối chiếu Budget 80 USD |

## 6. Quyết định go/no-go

| Trường | Giá trị |
|---|---|
| Quyết định | `TODO(operator)` — chọn một: `go` / `adjust` / `no-go` |
| Người quyết định (`decided_by`) | `TODO(operator)` (danh tính, không phải tên nhóm chung) |
| Thời điểm (`decided_at`) | `TODO(operator)` (UTC, sau ngày thứ 7) |
| Căn cứ | `TODO(operator)` — tham chiếu bảng SC-007 mục 3, sự cố mục 4, chi phí mục 5; không cảm tính |
| Bằng chứng kèm theo | `TODO(operator)` — link/đường dẫn metric/log/báo cáo |
| Việc tiếp theo (`follow_ups`) | `TODO(operator)` — hạng mục còn thiếu trước production |

**Quy tắc**

- Chưa có quyết định thì cấu hình Ollama tiếp tục active, quan sát tiếp tục, **không** rollback/tắt/
  teardown tự động (SC-010).
- Quyết định phải có đủ ba yếu tố: lựa chọn, danh tính người quyết định, thời điểm — dùng cho audit.
- Nếu `no-go` hoặc `adjust`: vẫn không tự động rollback trong feature này; việc rollback là hành động
  thủ công theo runbook mục 4.2 (SC-006) và phải ghi thời gian thực hiện.

## 7. Hạng mục còn thiếu trước khi dùng production

- `TODO(operator)` — ví dụ: chưa có monitoring dài hạn, chưa có capacity planning, chưa có quy trình
  rotate Bearer token, chưa đo tải thật.
