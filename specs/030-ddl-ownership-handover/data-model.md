# Data Model: Bàn giao quyền sở hữu DDL (030)

**Spec**: [spec.md](spec.md) | **Research**: [research.md](research.md)

Tài liệu này không tạo entity nghiệp vụ mới. "Data" của tính năng là bản đồ sở hữu, sổ version và ma trận hội tụ — 3 thực thể dưới đây là toàn bộ bề mặt cần đúng.

## 1. Bản đồ sở hữu schema (chuẩn sau bàn giao)

| Schema | Chủ sở hữu | Cây migration | Ghi chú |
|---|---|---|---|
| `identity`, `subscription` | Account | `services/account/database/migrations/` | giữ nguyên |
| `wardrobe`, `fashion`, `brand`, `dashboard`, `community` | Commerce | `services/commerce/database/migrations/` | `dashboard` là phần mới bàn giao |
| `aiexecutor` | AI | `services/ai/database/migrations/` | mới: 7 bảng, xem §2 |
| `shared`, `public` | Hạ tầng chung (Account giữ tạm) | Account | chấp nhận có ý thức, ngoài phạm vi spec này |

Ngoại lệ duy nhất được phép: khóa ngoại tham chiếu `identity.users(id)` từ schema khác (kiến trúc §VII giữ FK user xuyên schema). Mọi `JOIN`/truy vấn xuyên module khác vẫn bị cấm theo hiến pháp §I/§II.

## 2. Ma trận hội tụ `aiexecutor.*` (7 bảng)

| Bảng | Nguồn cũ (Account, sẽ gỡ) | Nguồn mới (AI, tự đứng được) | Điểm hội tụ bắt buộc |
|---|---|---|---|
| `ai_cost_policies` | `20260622131631:3` tạo | baseline AI tạo lại nguyên hình | định nghĩa cột/enum đồng nhất |
| `ai_cost_policy_operations` | `20260622131631:15` tạo; `20260930120000:57-62` nullable + set NULL premium | baseline AI tạo ở trạng thái cuối (nullable, không preset NULL sai) | database cũ đã có NULL premium → no-op |
| `ai_policy_grants` | `20260622131631:41-47` tạo + CHECK nội tuyến; `20260704062159:19` rename; `20260930120000:47` index | baseline AI tạo với tên tường minh `ai_policy_grants_effective_window_check` + index `source_event_id` | database cũ: rename 1 lần duy nhất (check `pg_constraint`) |
| `ai_usage_period_ledgers` | `20260622131631:51` tạo; `20260930120000:50-55` nullable + partial unique | baseline AI tạo ở trạng thái cuối | database cũ: `grant_id` đã nullable → no-op |
| `ai_usage_events` | `20260622131631:60` tạo | baseline AI tạo lại nguyên hình | — |
| `ai_model_configs` | `20260704062159:3` tạo | baseline AI tạo lại nguyên hình | — |
| `ai_cost_policy_audit_logs` | `20260930120000:64` tạo | baseline AI tạo lại nguyên hình + 2 index | — |

Chuyển schema (`SET SCHEMA` ở `20260801000000:73-78`) không cần replay: baseline mới tạo thẳng trong `aiexecutor`.

## 3. Ma trận hội tụ `dashboard.*` (3 bảng, Commerce)

| Bảng | Nguồn cũ (Account, sẽ gỡ) | Nguồn mới (Commerce) |
|---|---|---|
| `dashboard_admin_daily_snapshots` | `20260723000001` tạo; `20261011090002:7-10` thêm 3 cột payos | 1 file baseline: tạo bảng đã gồm 3 cột payos |
| `dashboard_brand_daily_snapshots` | `20260723000001` tạo | cùng file baseline trên |
| `dashboard_user_wardrobe_summaries` | `20260723000001` tạo | cùng file baseline trên |

## 4. Sổ version (bất biến nghiệm thu)

| Bảng version | Trước | Sau | Quy tắc |
|---|---|---|---|
| `goose_db_version` (Account) | 35 dòng, max `20261011090002` | **35 dòng, cùng max** | 0 file mới, 0 dòng mới; nội dung 11 file viết lại giữ nguyên version |
| `goose_db_version_commerce` | 3 dòng, max `20260927120000` | 3 + k dòng baseline dashboard | k = số file baseline Commerce (spec chốt ở implement, kỳ vọng 1) |
| `goose_db_version_ai` | chưa tồn tại | m dòng baseline AI | m = số file baseline AI (kỳ vọng 1–2); bảng được tạo lần đầu khi AI boot |

## 5. Quy tắc validation rút từ bảng trên

- Mọi file baseline: `IF NOT EXISTS` cho CREATE; `DO` block check `pg_constraint`/`information_schema` cho ALTER/rename; `Down` hoàn nguyên đúng thứ tự ngược.
- Fresh DB replay: Account (đã gỡ) → AI baseline → Commerce baseline phải ra schema cuối; DB cũ boot bản mới: Account `Up` 0 đáng kể, AI/Commerce hội tụ no-op sau lần đầu.
- Cấm tuyệt đối trong spec này: `DROP TABLE/SCHEMA` trên dữ liệu sống, `DELETE` dòng bảng version, lệnh xóa volume, target database production.
