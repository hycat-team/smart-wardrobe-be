# Báo cáo go/no-go — Thử nghiệm Ollama trên EC2 (feature 032)

**Feature**: `specs/032-ollama-lightsail-trial/` | **Ngày tạo**: 2026-10-26 | **Cập nhật**: 2026-10-27
| **Trạng thái**: **CLOSED — DEMO-ONLY / INCOMPLETE — DECISION: ADJUST**

Báo cáo này tổng hợp dữ liệu thực tế của đợt chạy demo ngày 2026-10-27. Đây **không phải**
báo cáo trial 7 ngày đầy đủ: trial được đóng sớm theo lựa chọn của người vận hành, SC-005
(load profile) được waiver, và một số capability chưa được kiểm tra riêng.

## 1. Thông tin thử nghiệm

| Trường | Giá trị | Ghi chú |
|---|---|---|
| Trial window | `2026-10-27` → `2026-10-27` | Đóng sớm demo-only; chưa đủ 7 ngày |
| App image commit | `aa283cff` | AI/Commerce image deploy qua `deploy-aws.yml` |
| Infra/bootstrap commit | `5ba0cfcc` | Terraform/cloud-init/bootstrap/nginx Ollama |
| Docs commit | `1c647d61` | Runbook + tasks |
| AI image digest | `ghcr.io/hycat-team/ai@sha256:5062db27...6480e7a` | Deployed trên Lightsail |
| Proxy hostname | `ollama.hycat.online` | A record trỏ Elastic IP |
| Elastic IP | `13.213.185.178` | EC2 Ollama |
| EC2 instance | `i-0972535513a16e241` | `c6i.2xlarge`, `ap-southeast-1a` |
| Ollama | `0.40.11` | Pin version |
| Model | `qwen2.5:3b-instruct-q4_K_M` | Pin digest `sha256:5ee4f07...31caa6` |
| TLS | Let's Encrypt | Hạn `2027-01-25` |
| Static IP Lightsail | `54.251.248.132` | Dùng cho SG + Nginx allowlist |
| Người vận hành | `Mavil` | |

Phạm vi: backend vẫn chạy trên Lightsail; Ollama chạy EC2 riêng; không thao tác VPS.
Ngoại lệ đã thực hiện theo yêu cầu vận hành: tăng timeout outfit trên Nginx local/AWS;
không thay đổi VPS, Dockerfile, compose AWS hay pipeline.

## 2. Smoke test và phạm vi

| Hạng mục | Kỳ vọng | Thực tế | Đạt? | Bằng chứng |
|---|---|---|---|---|
| 6 app container Lightsail | 3 http + 3 worker healthy | 6/6 healthy; nginx healthy | Đạt | `docker compose ps` |
| Ba capability text | chat/recommendation/free do Ollama | Chat + recommendation xác nhận `openai`; free_text cấu hình `openai` nhưng chưa gọi riêng | Một phần | AI attempt logs |
| `summary` / `rewriter` | Route free/paid qua Ollama | Chưa kích hoạt riêng trong demo | Chưa thu | — |
| Vision / embedding | Vẫn Gemini | Cấu hình giữ nguyên; chưa retest trên AWS | Chưa thu | Config |
| Chi phí usage event | Text `actual_cost_micro_vnd = 0` | Chưa đối chiếu bảng usage event trên Supabase | Chưa thu | — |
| Artefact deployment | Không đổi VPS/pipeline | Không đổi VPS; có thay đổi Nginx AWS timeout outfit đã được duyệt | Một phần | Git diff |
| Dịch vụ dữ liệu ngoài | Không đổi | Không thay đổi env/kết nối | Đạt | `.env.aws` diff |

## 3. SC-007 — Bảng chỉ số

| # | Chỉ số | Ngưỡng tham chiếu | Giá trị thực tế | Đạt? | Bằng chứng |
|---|---|---|---|---|---|
| 1 | Smoke 6 container | 6/6 healthy; text Ollama khi không fallback | 6/6 healthy; outfit/chat `openai` | Đạt | `docker compose ps`, AI logs |
| 2 | Access/auth | IP ngoài bị chặn; thiếu/sai Bearer bị từ chối; Lightsail được chấp nhận; 11434 không public | Local timeout; Lightsail healthz 200; valid 200; missing/wrong 401; 11434 timeout | Đạt | Runbook B5.1 |
| 3 | Lỗi | Breakdown outcome | Fallback drill ghi nhận `http_5xx` → Gemini; chưa có breakdown 7 ngày | Một phần | AI logs |
| 4 | Độ trễ AI | p95 chat/free ≤15s; recommendation ≤60s/120s | Chat ~13.7s; outfit ~68.7s; fallback ~1.7s; **không có p95** | Không đủ | AI logs |
| 5 | Fallback | Lỗi trước chunk → Gemini đúng 1 lần | Outfit: Ollama fail → Gemini success; chưa chạy full matrix | Một phần | AI logs |
| 6 | Hàng đợi | Queue hữu hạn; không attempt 3 | Chưa đo queue; không thấy attempt 3 | Chưa thu | — |
| 7 | CPU/RAM/disk | CPU >85% 10 phút → điều tra | Chưa thu metric CloudWatch | Chưa thu | — |
| 8 | Chi phí AWS | Đối chiếu Budget 80 USD | EC2 đang chạy; chưa chốt Cost Explorer | Chưa thu | — |

SC-005 (load profile 50 request/capability, concurrency 2, 30 phút) **WAIVED — not run by choice**.
Không có p95 và không đánh giá PASS cho chỉ số độ trễ.

## 4. Sự cố trong đợt chạy

| Thời điểm | Sự cố | Ảnh hưởng | Xử lý | Cần tồn? |
|---|---|---|---|---|
| 2026-10-27 | `user_data` vượt 16 KiB | EC2 không tạo được | Chuyển sang `base64gzip` | Không |
| 2026-10-27 | Ubuntu 24.04 thiếu package `awscli` | Bootstrap fail | Cài AWS CLI v2 official zip | Không |
| 2026-10-27 | EBS Nitro không có `/dev/sdf` | Bootstrap fail | Dò disk non-root | Không |
| 2026-10-27 | Ollama CLI thiếu `HOME` | Version check fail | Export `HOME=/root` | Không |
| 2026-10-27 | So sánh model ID với digest | Bootstrap fail | Đọc digest từ `ollama show --modelfile` | Không |
| 2026-10-27 | Nginx trùng default server + map bucket 64 | Nginx không reload | Xoá site default; `map_hash_bucket_size 128` | Không |
| 2026-10-27 | Ollama trả 403 vì Host public | AI request fail qua Nginx | Upstream `Host 127.0.0.1:11434` | Không |
| 2026-10-27 | Fallback drill | Ollama stop tạm thời | Gemini phục vụ; start lại Ollama | Không |

Không ghi nhận mất dữ liệu đã xác nhận.

## 5. Chi phí

| Khoản | Ước tính | Thực tế | Ghi chú |
|---|---|---|---|
| EC2 `c6i.2xlarge` | ~0.27 USD/giờ; ~46 USD/7 ngày | Chưa lấy Cost Explorer | Instance đang chạy |
| EBS gp3 66 GiB | Tính theo giờ | Chưa lấy | Root 16 GiB + data 50 GiB |
| Elastic IP | Tính khi gắn/không gắn | Chưa lấy | Giữ cho demo |
| Secrets Manager / SSM / CloudWatch | Nhỏ | Chưa lấy | Log retention 14 ngày |
| **Tổng** | — | **Chưa chốt** | Đối chiếu Budget 80 USD sau demo |

## 6. Quyết định go/no-go

| Trường | Giá trị |
|---|---|
| Quyết định | **`adjust`** |
| Người quyết định | `Mavil` |
| Thời điểm | `2026-10-27` |
| Căn cứ | Demo Ollama + fallback + security + rollback PASS; SC-005 waived; chưa đủ 7 ngày; chưa test đủ free_text/summary/rewriter/vision; chưa chốt cost |
| Bằng chứng | Runbook B2.1, B3.1, B5.1, B6.1, B7.1 + commit `1c647d61` |
| Follow-ups | Chạy load test khi có quota/harness; theo dõi 7 ngày; test đủ capability; chốt cost; cleanup sau demo |

Quyết định `adjust` không tự động rollback, stop hay teardown. EC2 và app Ollama tiếp tục chạy
đến khi demo kết thúc.

## 7. Hạng mục cần thiết trước production

- Load test SC-005 và p95 thực tế.
- Theo dõi đủ 7 ngày, gồm lỗi/fallback/queue/CPU/RAM/disk.
- Test riêng `free_text`, `summary`, `rewriter`; retest vision/embedding.
- Đối chiếu `ai_usage_events` và Cost Explorer/Budget.
- Quy trình rotate Bearer token, backup/restore `.env.aws`, capacity planning.
- Chốt cleanup/teardown sau demo.

## 8. Checklist vận hành

Checklist 7 ngày **không chạy đầy đủ** vì trial đóng sớm demo-only. Các mục B1–B10 của
template gốc không được coi là PASS. Riêng SC-010 read-only đã kiểm tra PASS; T051 `make check`
vẫn blocked do thiếu CGO/gcc.

## 9. SC-010 — read-only

Đã kiểm tra read-only ngày 2026-10-27: không có `aws_autoscaling_schedule`,
`aws_scheduler_schedule`, `lifecycle { prevent_destroy }`, auto-teardown hay `terraform destroy`
trong artefact feature. Kết quả: **PASS (read-only)**. Không có lệnh teardown tự động.

## 10. Trạng thái tài nguyên sau báo cáo

- EC2 Ollama, EIP, EBS, Secrets Manager, CloudWatch vẫn đang chạy để phục vụ demo.
- App Lightsail đang dùng Ollama cho text; vision/embedding vẫn Gemini.
- DNS `ollama.hycat.online` vẫn trỏ EIP.
- Cleanup sẽ thực hiện sau demo theo runbook: rollback env, xoá A record, `terraform destroy`,
  xác nhận EIP/secret/log group.
