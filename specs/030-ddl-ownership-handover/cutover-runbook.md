# Runbook cutover production — bàn giao quyền sở hữu DDL (030)

**Spec**: [spec.md](spec.md) | **Bằng chứng kiểm chứng**: [verification-log.md](verification-log.md)

Tài liệu này DÀNH CHO THAO TÁC TAY của chủ hệ thống trên production.
Agent không chạy các bước dưới đây (tasks.md T032).

**Không còn placeholder.** Mọi giá trị dưới đây là giá trị thật của hệ thống này.
Mọi lệnh copy-paste được nguyên khối vào shell trên VPS.

> **Nguyên tắc cao nhất**: rollback khi hỏng = khôi phục từ backup.
> KHÔNG BAO GIỜ chạy `Down` trên database có dữ liệu (`Down` của 2 baseline là
> `DROP TABLE` toàn bộ 7+3 bảng — destructive; chỉ an toàn trên DB dựng trắng,
> đã chứng minh ở T025).
> KHÔNG bao giờ chạy `docker compose down` / `down -v` trên production.
> Không sửa chữa thủ công trên dữ liệu sống.

---

## 0. Giá trị thật của môi trường (không cần suy luận thêm)

| Thành phần | Giá trị |
|---|---|
| Thư mục làm việc trên VPS | `/opt/closy` |
| Compose file | `deployments/docker-compose.prod.yml` (project name `closy-prod`) |
| Env files (luôn kèm cả hai) | `.env.production`, `.env.deploy` |
| Container PostgreSQL | `closy-prod-postgres-1` |
| Database | `closy_db` |
| DB user | `closy` (mật khẩu: `DB_PASSWORD` trong `.env.production`) |
| Container dịch vụ | `closy-prod-account-1`, `closy-prod-ai-1`, `closy-prod-commerce-1` |
| Container gateway | `closy-prod-nginx-1` |
| Smoke URL | `https://api.closy.hycat.online/api/v1/<svc>/health` |
| Cửa sổ | 1–2 giờ full-stop, chủ hệ thống tự chọn thời điểm |

Nếu tên container khác (ví dụ dùng `-p` khác), lấy lại bằng:

```bash
docker ps --format '{{.Names}}\t{{.Image}}' | grep -E 'closy-prod|postgres|rabbitmq|redis'
```

**Khởi tạo biến cho toàn bộ runbook** (chạy 1 lần, dùng cho mọi bước sau):

```bash
cd /opt/closy
DC="docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml"
set -a; . ./.env.production; set +a
export PGPASSWORD="$DB_PASSWORD"
CT=closy-prod-postgres-1
U="$DB_USER"
D="$DB_NAME"
P="docker exec -e PGPASSWORD=$PGPASSWORD $CT psql -U $U -d $D -t -A -c"
```

---

## 0b. Điều kiện tiên quyết — thiếu 1 điều là chưa được bắt đầu

1. **Rehearsal dev xanh 100% (cổng SC-007)**: V5 đã pass trên database dev
   (versions `35 / 4 / 2`, chỉ còn constraint tên mới, đếm dòng trước/sau bằng
   nhau, 3 service boot xanh, smoke xanh).
2. **Backup production có 2 bản**: 1 trong container, 1 trên host
   (`/opt/closy/prod_before_030.dump`), đã verify `pg_restore --list` đọc được.
   Thiếu bản host là **chưa đủ điều kiện** — xem §1.
3. **Backup `.env.deploy`** (đường về image cũ): có file `.env.deploy.bak.<ngày-giờ>`.
4. **Hạ tầng phụ đang healthy**: `postgres`, `redis`, `rabbitmq` — nếu broker
   stop, bước check topology của workflow sẽ đỏ (xem §3b).
5. **Cửa sổ full-stop** đã chọn, không có user truy cập.
6. **Ngưỡng abort**: `lock_timeout = 10s` (đo rename thật 0.415 ms; cả file
   baseline ~7–40 ms).
7. **Precaution pre-giờ G** đã chạy (§0c) và kết quả khớp kỳ vọng.

---

## 0c. Precaution pre-giờ G — đối chiếu prod với drift đã biết (chỉ đọc)

```bash
# D-01: default của plan_kind / tier_rank (kỳ vọng: KHÔNG có default)
$P "SELECT column_name||'='||coalesce(column_default,'NULL') FROM information_schema.columns
    WHERE table_schema='subscription' AND table_name='subscription_plans'
      AND column_name IN ('plan_kind','tier_rank') ORDER BY 1;"

# D-02: UNIQUE trên gateway_reference (kỳ vọng: CÓ deposit_transactions_gateway_reference_key)
$P "SELECT conname FROM pg_constraint
    WHERE conrelid='subscription.deposit_transactions'::regclass AND contype='u' ORDER BY 1;"

# D-03: index của webhook inbox (kỳ vọng: KHÔNG có idx_webhook_inbox_processing)
$P "SELECT indexname FROM pg_indexes
    WHERE schemaname='subscription' AND tablename='provider_webhook_inbox' ORDER BY 1;"

# D-04: default của started_at (kỳ vọng: KHÔNG có default)
$P "SELECT column_name||'='||coalesce(column_default,'NULL') FROM information_schema.columns
    WHERE table_schema='subscription' AND table_name='user_subscriptions'
      AND column_name='started_at';"
```

| Kết quả | Hành động |
|---|---|
| Khớp drift đã biết (D-01..D-04 như trên) | Đi tiếp |
| Lệch kiểu khác (object lạ, thiếu object baseline cần) | DỪNG, điều tra, chưa bắt đầu |

**D-05 (data)**: owner đã xác nhận giá trị trên prod là giá trị đúng (sửa tay có
chủ ý), baseline seed dùng `ON CONFLICT DO NOTHING` / `WHERE NOT EXISTS` nên
không ghi đè. Không cần đối chiếu thêm.

---

## 1. Backup production + kiểm chứng (đường về duy nhất)

```bash
# 1a. Backup đầy đủ, custom format.
docker exec -e PGPASSWORD="$PGPASSWORD" $CT pg_dump -U "$U" -Fc -f /tmp/prod_before_030.dump "$D"

# 1b. Xác minh file đọc được (BẮT BUỘC trước khi đi tiếp).
docker exec $CT pg_restore --list /tmp/prod_before_030.dump | head -10

# 1c. Bản thứ hai trên HOST — bắt buộc, vì /tmp nằm trong container.
docker cp $CT:/tmp/prod_before_030.dump /opt/closy/prod_before_030.dump
ls -lh /opt/closy/prod_before_030.dump
```

**Gate cứng** — chỉ đi tiếp khi cả 3 đúng:

```bash
test -s /opt/closy/prod_before_030.dump && echo "HOST-BACKUP-OK"
docker exec $CT test -s /tmp/prod_before_030.dump && echo "CONTAINER-BACKUP-OK"
docker exec $CT pg_restore --list /tmp/prod_before_030.dump | grep -c . # > 0
```

Mốc tham chiếu lần chạy 2026-10-20: ~1.5 MB, **399 TOC entries**, dbname `closy_db`.

```bash
# 1d. Backup .env.deploy (đường về image cũ).
cp -p .env.deploy ".env.deploy.bak.$(date +%F-%H%M)"
grep _IMAGE .env.deploy
```

---

## 2. Chụp số liệu trước cutover (để đối chiếu §5)

```bash
# 2a. Sổ version. Kỳ vọng trước cutover: 35 / 3 / chưa có bảng AI.
$P "SELECT 'account='||count(*)||' max='||max(version_id) FROM goose_db_version;"
$P "SELECT 'commerce='||count(*)||' max='||max(version_id) FROM goose_db_version_commerce;"
$P "SELECT 'ai_table='||coalesce(to_regclass('public.goose_db_version_ai')::text,'NULL');"

# 2b. Tên constraint grant. Kỳ vọng: còn tên cũ user_ai_policy_grants_check.
$P "SELECT conname FROM pg_constraint
    WHERE conrelid='aiexecutor.ai_policy_grants'::regclass ORDER BY 1;"

# 2c. Đếm dòng CHÍNH XÁC toàn bộ 56 bảng nghiệp vụ.
$P "SELECT schemaname||'.'||tablename||'='||
      (xpath('/row/c/text()', query_to_xml(
        format('SELECT count(*) AS c FROM %I.%I', schemaname, tablename), false, true, '')))[1]::text
    FROM pg_tables
    WHERE schemaname IN ('identity','subscription','wardrobe','fashion','brand','community','dashboard','aiexecutor')
    ORDER BY 1;"
```

> **Không dùng `pg_stat_user_tables.n_live_tup`**: đó là ước lượng, chỉ cập nhật
> khi autovacuum/ANALYZE, nên nhiều bảng báo `0` sai (đã gặp thực tế với
> `ai_cost_policies`, `subscription_plans`). Luôn dùng `count(*)` như trên.

Lưu output vào sổ giờ G. **Quy tắc so sánh**:

| Nhóm | Bảng | Quy tắc |
|---|---|---|
| Tĩnh/cấu hình | `aiexecutor.ai_cost_policies`, `ai_cost_policy_operations`, `ai_model_configs`, `subscription.subscription_plans`, `campaigns`, `campaign_sweep_runs`, `campaign_claims`, `fashion.categories`, toàn bộ `brand.*` | **bằng nhau tuyệt đối** |
| Có traffic | `identity.users`, `refresh_tokens`, `user_activity_events`, `aiexecutor.ai_policy_grants`, `ai_usage_events`, `ai_usage_period_ledgers`, `subscription.*` (user/deposit/wallet/event), `shared.outbox_records`, `admin_audit_logs`, `wardrobe.*`, `community.*` | bằng hoặc **tăng** (user thật/smoke) — tăng là bình thường, ghi riêng |

---

## 3. Deploy: Account → AI → Commerce

### Cách chuẩn (dùng CI, không build trên VPS)

GitHub → **Actions** → workflow **Deploy VPS Services** → **Re-run all jobs**.

> **Bắt buộc chọn "Re-run all jobs", KHÔNG chọn "Re-run failed jobs".**
> Job `deploy` dùng `needs.plan.outputs.matrix` và `needs.build-tool.outputs.ref`;
> khi chỉ re-run job lỗi, các output này rỗng → `docker pull ''` hoặc matrix rỗng.

Hoặc bằng CLI:

```bash
gh workflow run deploy-vps-services.yml \
  -R hycat-team/smart-wardrobe-be \
  --ref 030-ddl-ownership-handover \
  -f services=all
```

Phải **approve** environment `vps-production` (job `deploy` và `reload-nginx`),
nếu không run đứng im.

Thứ tự job thực tế: `plan` → `build-push` + `build-tool` → `deploy` →
`reload-nginx` → `smoke-all`.

`deploy` chạy tuần tự **account → commerce → ai**. Khác thứ tự lý thuyết
`account → ai → commerce`, nhưng **an toàn trên prod**: baseline Commerce chỉ
chạm schema `dashboard`, baseline AI chỉ chạm `aiexecutor`, hai schema độc lập
và cả hai đều idempotent; Account (giữ toàn bộ lịch sử cũ) vẫn chạy đầu tiên.

### Kiểm tra sau khi deploy

```bash
$DC ps
docker logs --tail 40 closy-prod-account-1   # "no migrations to run", current 20261011090002
docker logs --tail 40 closy-prod-ai-1        # OK 20261016213000_adopt_aiexecutor_ownership.sql
docker logs --tail 40 closy-prod-commerce-1  # OK 20261016213001_adopt_dashboard_ownership.sql

# Boot lần 2 phải no-op ở cả 3 cây (chứng minh hội tụ — không bắt buộc nếu log đã sạch)
```

---

## 3b. Sự cố đã gặp thật — tra nhanh

| Triệu chứng | Nguyên nhân | Cách xử lý |
|---|---|---|
| `read queue types: ... dial tcp: lookup rabbitmq on 127.0.0.11:53: server misbehaving` ở bước check topology | Container `rabbitmq` đang stop → hostname `rabbitmq` không resolve trong network `closy-prod_data` | `$DC up -d postgres redis rabbitmq`, chờ healthy, rồi re-run **all jobs** |
| `docker pull ''` / matrix rỗng ở job deploy | Đã bấm "Re-run failed jobs" thay vì "Re-run all jobs" | Re-run **all jobs** |
| Deploy xong nhưng không truy cập được | `nginx` bị stop tay và chưa được recreate | `$DC up -d --force-recreate nginx` (workflow cũng tự làm ở job `reload-nginx`) |
| Job `deploy` báo `deploy failed for: <svc>` | Pull/up service đó lỗi | Xem log job, sửa nguyên nhân rồi re-run; nếu không cứu được → rollback §6 |
| `lookup rabbitmq ...` lặp lại dù broker đã up | Broker chưa healthy (start_period 30s) | Chờ `docker exec closy-prod-rabbitmq-1 rabbitmq-diagnostics -q ping` trả OK |

Chạy lại đúng bước check bằng tay (cùng lệnh workflow dùng):

```bash
docker run --rm --network closy-prod_data --env-file .env.production \
  ghcr.io/hycat-team/rabbitmq-topology@sha256:389a16df95c6f1c4811a30a3d662e93d035490a9ecf510591efb5a7f3e273718 \
  --mode check
```

---

## 4. Giám sát lock trong lúc rename + ngưỡng abort

Rename giữ `AccessExclusive` trên `aiexecutor.ai_policy_grants` trong chốc lát.
Mở session riêng trước khi deploy AI:

```sql
-- 4a. Session đang giữ/chờ lock.
SELECT pid, usename, wait_event_type, wait_event, left(query, 80)
FROM pg_stat_activity WHERE query ILIKE '%ai_policy_grants%';
SELECT locktype, mode, granted, pid
FROM pg_locks WHERE relation = 'aiexecutor.ai_policy_grants'::regclass;
-- 4b. Ngưỡng abort.
SET lock_timeout = '10s';
```

Chạy nhanh bằng CLI:

```bash
$P "SELECT pid, state, wait_event_type, left(query,80) FROM pg_stat_activity
    WHERE query ILIKE '%ai_policy_grants%';"
```

**Tiêu chí abort (1 điều đỏ = dừng + rollback §6, không sửa tiếp):**

- Service boot đỏ (migration failed, không listening, health ≠ 200).
- Đếm dòng nhóm tĩnh lệch so với §2.
- Constraint không đúng kỳ vọng §9.
- Smoke §5 đỏ.
- Lock chờ vượt `10s` hoặc có deadlock.
- Bất kỳ lệnh nào trỏ nhầm database hoặc xóa volume.

---

## 5. Smoke sau cutover + checklist nghiệm thu

Smoke qua đúng ingress công khai (container là distroless, **không** curl được
vào `:8080` từ host):

```bash
for s in account ai commerce; do
  echo "== $s =="
  curl -4 --fail --silent "https://api.closy.hycat.online/api/v1/$s/health"; echo
done
```

Điều kiện pass: body chứa `"service":"<svc>"` (không chỉ HTTP 200).

Smoke nghiệp vụ (dùng tài khoản kiểm chứng đánh dấu rõ, KHÔNG dùng tài khoản
người thật): đăng ký → confirm OTP → login; xem danh sách gói; tạo + liệt kê
phiên chat AI; gọi dashboard admin bằng tài khoản thường → **403** (RBAC còn
nguyên).

**Checklist (ghi từng dòng vào sổ giờ G):**

- [ ] `goose_db_version`: **35 dòng**, max `20261011090002` (không đổi).
- [ ] `goose_db_version_commerce`: **4 dòng**, max `20261016213001`.
- [ ] `goose_db_version_ai`: **2 dòng** (dòng `0` + baseline), max `20261016213000`.
- [ ] Constraint chỉ còn `ai_policy_grants_effective_window_check` cho CHECK cửa
      sổ hiệu lực; 4 tên legacy (`..._pkey`, `..._policy_id_fkey`,
      `..._status_check`, `..._user_id_fkey`) **giữ nguyên có chủ ý**.
- [ ] Đếm dòng nhóm tĩnh bằng §2 tuyệt đối; nhóm traffic bằng hoặc tăng (ghi riêng).
- [ ] 3 service + nginx + postgres + redis + rabbitmq đều `healthy` (`$DC ps`).
- [ ] Smoke 3 health + nghiệp vụ xanh.
- [ ] Log 30 phút gần nhất không có `migration failed`, `deadlock`,
      `does not exist`, `PRECONDITION_FAILED`, `panic`:

```bash
for s in account ai commerce; do
  echo "== $s =="
  docker logs --since 30m closy-prod-$s-1 2>&1 \
    | grep -iE "migration failed|deadlock|does not exist|PRECONDITION_FAILED|panic" | head -20
done
```

---

## 6. Rollback = restore từ backup (KHÔNG dùng Down)

```bash
# 6a. Về image cũ trước (không đụng dữ liệu).
cp -p .env.deploy.bak.<ngày-giờ> .env.deploy
$DC pull account commerce ai nginx
$DC up -d account commerce ai nginx

# 6b. Chỉ khi dữ liệu bị ảnh hưởng: restore DB từ bản backup TRÊN HOST.
docker cp /opt/closy/prod_before_030.dump $CT:/tmp/prod_before_030.dump
docker exec -e PGPASSWORD="$PGPASSWORD" $CT pg_restore -U "$U" -d "$D" --clean -c /tmp/prod_before_030.dump

# 6c. Boot lại, chạy lại checklist §5 trên dữ liệu đã khôi phục (khớp §2).
```

Không chạy `Down` trên database có dữ liệu — `Down` của 2 baseline là
`DROP TABLE` 7+3 bảng, sẽ xóa dữ liệu chứ không "quay về tên cũ".

---

## 7. Môi trường bị quên, sót (staging bỏ xó, laptop dev, backup cũ)

Không cần phẫu thuật tay: boot bản mới tự hội tụ nhờ migration idempotent
(`IF NOT EXISTS` + seed guard + rename có điều kiện `pg_constraint`) — môi
trường nào boot bản mới là tự chữa lành, kẹt mới báo lỗi rõ ràng để thử lại.

---

## 8. Đóng cửa sổ

- Mở lại traffic (workflow tự recreate nginx); quan sát log 15 phút.
- Ghi sổ giờ G: thời gian full-stop thực tế, số liệu §2/§5, kết quả smoke,
  mọi abort (nếu có) và cách xử lý.
- **Giữ** `/opt/closy/prod_before_030.dump` và `.env.deploy.bak.*` ít nhất vài
  ngày sau cutover (đường về nếu phát sinh lỗi muộn).
- Backup V5 của dev (bản rehearsal SC-007) **xóa** sau khi cutover xong theo
  quyết định của chủ hệ thống.
- **Không** push tag `vps-*` sau khi merge (quyết định của chủ hệ thống).

---

## 9. Bảng giá trị kỳ vọng sau cutover (tra trực tiếp, không cần hỏi ai)

| Hạng mục | Kỳ vọng |
|---|---|
| `goose_db_version` | 35 dòng, max `20261011090002` |
| `goose_db_version_commerce` | 4 dòng, max `20261016213001` |
| `goose_db_version_ai` | 2 dòng, max `20261016213000` |
| Constraint `aiexecutor.ai_policy_grants` | `ai_policy_grants_effective_window_check` + `user_ai_policy_grants_pkey` + `user_ai_policy_grants_policy_id_fkey` + `user_ai_policy_grants_status_check` + `user_ai_policy_grants_user_id_fkey` |
| Nhóm tĩnh (mốc 2026-10-20) | policies `2`, operations `8`, model_configs `7`, plans `2`, campaigns `1`, sweep_runs `1`, claims `17`, categories `9`, brand_items `0` |
| Drift schema đã chấp nhận | D-01..D-04 (`subscription`) — không sửa trong 030 |
| Drift data đã chấp nhận | D-05 (model_configs = 7 do có `gemini-3.5-flash-lite`) |

Kiểm tra constraint:

```bash
$P "SELECT conname FROM pg_constraint
    WHERE conrelid='aiexecutor.ai_policy_grants'::regclass ORDER BY 1;"
```
