-- ============================================================================
-- SEED: dữ liệu kiểm thử FR-088 — tài khoản CÓ gói nhưng THIẾU bản ghi mức quyền
-- (024 T204)
-- ============================================================================
-- FR-088 là công cụ sửa chữa: phát lại tín hiệu kích hoạt cho mọi tài khoản đang
-- có gói mà dịch vụ AI không ghi được bản ghi mức quyền.
--
-- ⚠️ VÌ SAO 08 KHÔNG ĐỦ, DÙ 08 ĐÃ CÓ SẴN.
--
-- 08 tạo 500 tài khoản `t024_*` bằng `INSERT` thẳng vào `identity.users`, cố ý
-- không tạo `user_subscriptions`. Đó đúng là trạng thái cần kiểm thử cho
-- FR-005b/c/d (đường dự phòng), và KHÔNG phải hồi quy.
--
-- Nhưng nó không kiểm thử được FR-088, vì công cụ khôi phục quét
-- `UserSubscriptionRepository.ListAdmin` — tức là bảng
-- `subscription.user_subscriptions` (`restore_policy_grants_uc.go:249`). Danh sách
-- `t024_*` không có dòng nào ở đó, nên chúng **không bao giờ** xuất hiện trong
-- `res.Scanned`, kể cả trước T196. Nói cách khác: công cụ khôi phục không có dữ
-- liệu nào để khôi phục, và `restored` luôn bằng 0 không phải vì nó hoạt động mà
-- vì không có gì để làm. Trước T204, KHÔNG tồn tại fixture nào thật sự kiểm thử
-- được FR-088.
--
-- Tệp này tạo đúng trạng thái đó, theo đúng tiêu chuẩn mà `ListAdmin` quét. Với
-- bộ lọc rỗng (`AdminUserSubscriptionFilter{}`) mà `scanSubscriptions` truyền vào,
-- `ListAdmin` không lọc gì cả: nó đọc MỌI dòng của `subscription.user_subscriptions`
-- rồi sắp xếp theo `started_at DESC` và chia trang
-- (`user_subscription_repo_impl.go:159-188`). Entity `UserSubscription` không có
-- cột xoá mềm, nên tài khoản bị `is_deleted` vẫn được quét. Điều duy nhất còn
-- lọc lại là trong `applyRestoreRun`: gói phải có trong danh mục, và kỳ đăng ký
-- không được đã hết hạn.
--
-- Ba điều kiện vì vậy là bắt buộc, và cả ba đều phải đúng:
--
--   * có dòng `subscription.user_subscriptions`;
--   * `current_plan_code` trỏ tới một gói CÓ THẬT trong
--     `subscription.subscription_plans` — không thì `loadRestorePlans` không tìm
--     thấy và tài khoản bị báo `plan_not_in_catalog`, tức là đang kiểm thử nhầm
--     một nhánh khác;
--   * KHÔNG có dòng nào trong `aiexecutor.ai_policy_grants`.
--
-- Sáu tài khoản, chia theo từng nhánh cần chứng minh:
--
--   | tài khoản         | gói             | kết quả mong đợi                    |
--   |-------------------|-----------------|-------------------------------------|
--   | t024restore01..03 | free            | `restored` — FR-088 không phân biệt gói miễn phí |
--   | t024restore04..05 | premium-monthly | `restored` — gói trả phí            |
--   | t024restore06     | premium-monthly | `skipped` / `subscription_expired`  |
--
-- Tài khoản thứ sáu là tài khoản ĐÃ HẾT HẠN, và nó ở đây có chủ đích: một công cụ
-- khởi phục chỉ báo "đã khôi phục N" mà không nói lý do của phần còn lại thì không
-- phân biệt được "đã sửa" với "cố ý không sửa". Nhánh bỏ qua cũng là một hành vi
-- cần chứng minh.
--
-- Sau khi chạy, tài khoản thứ 3 trong mỗi nhóm sẽ KHÔNG còn thiếu grant nữa, và
-- lần chạy thứ hai của cùng công cụ phải báo `already_restored` cho chúng — đó là
-- FR-088 "chạy lại nhiều lần được mà không tạo tác dụng phụ". Muốn kiểm thử lại từ
-- đầu thì chạy lại tệp này ở chế độ `--set cleanup=1` rồi tạo lại.
--
-- KHOÁ DỮ LIỆU
--
-- Tiền tố `t024restore_` và sáu UUID cố định `f0240001-…`. Hai khoá cùng nhằm
-- một việc: nhận diện và xoá sạch mà không đụng tới dữ liệu khác, nên tệp này chạy
-- được song song với 08, với dữ liệu thật, và với những lần chạy thử khác. Cố ý
-- KHÔNG dùng tiền tố `t024_`: 08 xoá theo `LIKE 't024\_%'`, và `t024restore01`
-- KHÔNG khớp mẫu đó — ký tự đứng ngay sau `t024` là `r`, không phải `_`.
--
-- CÁCH CHẠY (ON_ERROR_STOP đã bật sẵn trong file, không cần truyền thêm)
--   psql -f database/scripts/11-seed_restore_test_data.sql
--
-- CÁCH XOÁ
--   psql -f database/scripts/11-seed_restore_test_data.sql --set cleanup=1
--
-- ⚠️ file này GHI vào cơ sở dữ liệu. Chỉ chạy trên môi trường phát triển.
-- ============================================================================

-- Dừng ngay tại câu lệnh hỏng. Lý do như 08: `psql` mặc định vẫn chạy tiếp, nên
-- một script nhiều câu lệnh sẽ vừa báo lỗi vừa tiếp tục xoá dữ liệu phía sau.
\set ON_ERROR_STOP on

\if :{?cleanup}
\else
  \set cleanup 0
\endif

\if :cleanup

BEGIN;

\echo 'XOA du lieu kiem thu FR-088 (t024restore_*)...'

-- Xoá theo thứ tự để không vi phạm khoá ngoại.
--
-- ⚠️ `shared.outbox_records` KHÔNG có khoá ngoại nào tới bảng sự kiện — nó là
-- bảng outbox, liên kết với sự kiện chỉ qua `payload->>'sourceEventId'` theo đúng
-- đặc tả hợp đồng, không phải qua ràng buộc. Thứ tự này vì vậy KHÔNG do khoá
-- ngoại mà do ý nghĩa: bản ghi sự kiện phải còn đúng lúc xoá outbox, vì chính
-- `sourceEventId` trong payload là thứ đang được dùng để tìm bản ghi sự kiện.
--
-- Phần CREATE của tệp này cố ý KHÔNG tạo bản ghi sự kiện nào — nó chỉ dựng
-- trạng thái trước khi sửa chữa. Bản ghi sự kiện và bản ghi outbox CHỈ xuất hiện
-- sau khi quy trình khôi phục thật sự chạy, tức là câu lệnh này xoá 0 dòng ngay
-- sau lần tạo, và xoá đúng số dòng đó sau khi đã kiểm thử. Đó là hành vi mong
-- muốn, không phải lỗi: nó làm cho fixture tự dọn trọn vẹn sau cả vòng kiểm thử.
DELETE FROM shared.outbox_records
 WHERE payload->>'sourceEventId' IN (
    SELECT e.id::text
      FROM subscription.user_subscription_events e
      JOIN identity.users u ON u.id = e.user_id
     WHERE u.username LIKE 't024restore\_%'
);

DELETE FROM subscription.user_subscription_events e
  USING identity.users u
  WHERE u.id = e.user_id AND u.username LIKE 't024restore\_%';

-- Bảng ledger phải đi trước grant: nó trỏ tới grant, và cả hai đều có
-- `ON DELETE CASCADE` từ `identity.users` nên câu `DELETE` cuối cùng cũng đủ —
-- nhưng ghi tường minh ở đây để thứ tự đúng ngay cả khi ai đó đọc tệp này mà
-- không biết tới các khoá ngoại.
DELETE FROM aiexecutor.ai_usage_period_ledgers l
  USING identity.users u
  WHERE u.id = l.user_id AND u.username LIKE 't024restore\_%';

DELETE FROM aiexecutor.ai_policy_grants g
  USING identity.users u
  WHERE u.id = g.user_id AND u.username LIKE 't024restore\_%';

DELETE FROM subscription.user_daily_quotas q
  USING identity.users u
  WHERE u.id = q.user_id AND u.username LIKE 't024restore\_%';

DELETE FROM subscription.user_subscriptions s
  USING identity.users u
  WHERE u.id = s.user_id AND u.username LIKE 't024restore\_%';

-- `shared.admin_audit_logs.actor_user_id` là khoá ngoại KHÔNG có hành động
-- (NO ACTION), nên một dòng kiểm toán còn trỏ tới tài khoản này sẽ làm câu
-- `DELETE` cuối cùng thất bại. Xoá trước cho chắc. Với tài khoản loại `user` ở
-- đây nó xoá 0 dòng — có đó là để một lần chạy thử đăng nhập bằng tài khoản thử
-- không làm hỏng lần dọn sau.
DELETE FROM shared.admin_audit_logs a
  USING identity.users u
  WHERE u.id = a.actor_user_id AND u.username LIKE 't024restore\_%';

DELETE FROM identity.users WHERE username LIKE 't024restore\_%';

\echo 'Da xoa du lieu kiem thu FR-088.'

COMMIT;

\echo '--- xac nhan con lai (mong doi 0 o ca ba dong) ---'
SELECT
    (SELECT count(*) FROM identity.users WHERE username LIKE 't024restore\_%')                  AS con_tai_khoan,
    (SELECT count(*) FROM subscription.user_subscriptions s
       JOIN identity.users u ON u.id = s.user_id
      WHERE u.username LIKE 't024restore\_%')                                                  AS con_dang_ky,
    (SELECT count(*) FROM subscription.user_subscription_events e
       JOIN identity.users u ON u.id = e.user_id
      WHERE u.username LIKE 't024restore\_%')                                                  AS con_su_kien;

\else

BEGIN;

\echo 'TAO du lieu kiem thu FR-088...'

-- Chặn sớm nếu danh mục gói không đúng. Không có gói free đang hoạt động thì cả
-- nhóm `restored` sẽ rỗng và nhóm `premium` rỗng nếu `premium-monthly` mất —
-- script vẫn "chạy thành công" và người vận hành sẽ tưởng FR-088 hỏng.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM subscription.subscription_plans
         WHERE plan_kind = 0 AND is_active
    ) THEN
        RAISE EXCEPTION
            'khong co goi free mac dinh dang hoat dong (plan_kind = 0, is_active = true) - fixture FR-088 khong the tao';
    END IF;
    -- Điều kiện phải có `plan_kind = 1`, không chỉ `slug`. Cả hai INSERT của nhánh
    -- `t024restore04..06` lấy `current_plan_kind` từ danh mục nhưng truyền
    -- `expires_at` khác NULL, mà `chk_user_subscription_kind_expiry`
    -- (`closy_db-schema.sql:1082`) chỉ cho phép điều đó khi gói là hữu hạn. Một
    -- `premium-monthly` đổi sang lifetime sẽ làm CHECK đỏ giữa `BEGIN`, và
    -- `ON_ERROR_STOP` dừng script với lỗi ràng buộc thay vì thông báo rằng
    -- fixture không thể tạo.
    IF NOT EXISTS (
        SELECT 1 FROM subscription.subscription_plans
         WHERE slug = 'premium-monthly' AND plan_kind = 1
    ) THEN
        RAISE EXCEPTION
            'khong co goi premium-monthly trong danh muc (slug = premium-monthly, plan_kind = 1) - fixture FR-088 khong the tao';
    END IF;
END $$;

-- Sáu tài khoản.
--
-- Mật khẩu lấy từ tài khoản quản trị sẵn có của môi trường phát triển, y hệt 08;
-- chỉ dùng để kiểm thử. `ON CONFLICT (id) DO NOTHING` vì khoá chính là `id`: lần
-- chạy thứ hai trùng cả sáu id nên không sinh trùng, và chạy lại không ghi đè lên
-- một tài khoản đã bị sửa tay.
INSERT INTO identity.users (id, username, email, password_hash, role_slug, email_verified)
VALUES
    ('f0240001-0000-4000-8000-000000000001', 't024restore01', 't024restore01@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true),
    ('f0240002-0000-4000-8000-000000000002', 't024restore02', 't024restore02@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true),
    ('f0240003-0000-4000-8000-000000000003', 't024restore03', 't024restore03@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true),
    ('f0240004-0000-4000-8000-000000000004', 't024restore04', 't024restore04@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true),
    ('f0240005-0000-4000-8000-000000000005', 't024restore05', 't024restore05@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true),
    ('f0240006-0000-4000-8000-000000000006', 't024restore06', 't024restore06@closy.local',
     (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1), 'user', true)
ON CONFLICT (id) DO NOTHING;

-- Gói miễn phí: `expires_at` phải NULL và tự gia hạn phải tắt, vì ràng buộc kiểm
-- tra `chk_user_subscription_kind_expiry` / `chk_user_subscription_auto_renew` bắt
-- buộc đúng vậy cho `plan_kind = 0`. Nhờ vậy `applyRestoreRun` không bỏ qua nhánh
-- "đã hết hạn" và tài khoản thật sự được khôi phục.
--
-- Cột `version` đặt 1 (không phải 0) để khoá sự kiện
-- `RESTORE_GRANT:{user_id}:{version}` của fixture này không trùng với khoá của
-- một tài khoản mới được cấp phát mặc định ở version 0.
INSERT INTO subscription.user_subscriptions (
    user_id, subscription_plan_id, expires_at, is_auto_renew_enabled,
    current_plan_code, current_tier_rank, current_plan_kind,
    current_benefit_snapshot, started_at, version, created_at, updated_at)
SELECT u.id, p.id, NULL, false, p.slug, p.tier_rank, p.plan_kind,
       '{}'::jsonb, now(), 1, now(), now()
  FROM identity.users u
  CROSS JOIN subscription.subscription_plans p
 WHERE p.plan_kind = 0 AND p.is_active
   AND u.username IN ('t024restore01', 't024restore02', 't024restore03')
ON CONFLICT (user_id) DO NOTHING;

-- Gói trả phí còn hiệu lực. `plan_kind = 1` nên `expires_at` BẮT BUỘC khác NULL
-- (cùng ràng buộc kiểm tra ở trên), và đặt trong tương lai để không rơi vào nhánh
-- bỏ qua.
INSERT INTO subscription.user_subscriptions (
    user_id, subscription_plan_id, expires_at, is_auto_renew_enabled,
    current_plan_code, current_tier_rank, current_plan_kind,
    current_benefit_snapshot, started_at, version, created_at, updated_at)
SELECT u.id, p.id, now() + INTERVAL '30 days', true, p.slug, p.tier_rank, p.plan_kind,
       '{}'::jsonb, now() - INTERVAL '30 days', 1, now(), now()
  FROM identity.users u
  CROSS JOIN subscription.subscription_plans p
 WHERE p.slug = 'premium-monthly'
   AND u.username IN ('t024restore04', 't024restore05')
ON CONFLICT (user_id) DO NOTHING;

-- Gói trả phí ĐÃ HẾT HẠN — nhánh `subscription_expired`.
--
-- `started_at` đặt trong quá khứ và `expires_at` ở quá khứ, nên nếu nhánh bỏ qua
-- không chạy, tài khoản này sẽ bị "khôi phục" thành công và nhận một grant
-- `active` có ngày kết thúc ở quá khứ — đúng thứ mà FR-088 cấm và mà kiểm chứng 5
-- của `07-verify_policy_grants.sql` phải báo.
INSERT INTO subscription.user_subscriptions (
    user_id, subscription_plan_id, expires_at, is_auto_renew_enabled,
    current_plan_code, current_tier_rank, current_plan_kind,
    current_benefit_snapshot, started_at, version, created_at, updated_at)
SELECT u.id, p.id, now() - INTERVAL '1 day', false, p.slug, p.tier_rank, p.plan_kind,
       '{}'::jsonb, now() - INTERVAL '31 days', 1, now(), now()
  FROM identity.users u
  CROSS JOIN subscription.subscription_plans p
 WHERE p.slug = 'premium-monthly'
   AND u.username = 't024restore06'
ON CONFLICT (user_id) DO NOTHING;

\echo '--- so lieu vua tao ---'
\echo '--- Mong doi: 3 / 2 / 1. Tong cong 6 dong subscription.user_subscriptions. ---'
SELECT
    count(*) FILTER (WHERE s.current_plan_kind = 0)                          AS mien_phi,
    count(*) FILTER (WHERE s.current_plan_kind = 1 AND s.expires_at > now())  AS tra_phi_con_hieu_luc,
    count(*) FILTER (WHERE s.current_plan_kind = 1 AND s.expires_at <= now()) AS tra_phi_da_het_han
FROM subscription.user_subscriptions s
JOIN identity.users u ON u.id = s.user_id
WHERE u.username LIKE 't024restore\_%';

\echo '--- TRANG THAI CAN KIEM THU: co dang ky nhung THIEU ban ghi muc quyen AI ---'
\echo '--- Mong doi: 6. Neu la 0 thi fixture khong con kiem thu duoc FR-088. ---'
SELECT count(*) AS accounts_missing_grant
FROM subscription.user_subscriptions s
JOIN identity.users u ON u.id = s.user_id
WHERE u.username LIKE 't024restore\_%'
  AND NOT EXISTS (
      SELECT 1 FROM aiexecutor.ai_policy_grants g
       WHERE g.user_id = s.user_id
         AND g.status IN ('active', 'future')
  );

\echo '--- nhung tai khoan CON DUOC quet boi cong cu khoi phuc (ListAdmin, bo loc rong) ---'
\echo '--- Mong doi: 6 dong. Ca 6 deu duoc quet; t024restore06 se bi BO QUA ---'
\echo '--- vi da het han, va 5 tai khoan con lai se duoc khai phuc. ---'
SELECT s.user_id, u.username, s.current_plan_code, s.current_plan_kind, s.expires_at
FROM subscription.user_subscriptions s
JOIN identity.users u ON u.id = s.user_id
WHERE u.username LIKE 't024restore\_%'
ORDER BY s.started_at DESC;

\echo 'Da tao du lieu kiem thu FR-088. Bay gio chay cong cu khoi phuc (xem quickstart.md, muc GD8).'

COMMIT;

\endif