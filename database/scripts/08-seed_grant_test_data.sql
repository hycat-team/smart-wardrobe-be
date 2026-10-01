-- ============================================================================
-- SEED: dữ liệu kiểm thử cho feature 024 (T104, T106, FR-005b/c/d)
--
-- Tạo:
--   * 1 tài khoản quản trị viên thứ hai  → để kiểm tra hai admin cùng gọi một
--     Idempotency-Key song song (T104).
--   * 500 tài khoản khách hàng trên gói free, CHƯA có bản ghi mức quyền AI →
--     đây là nhóm chạy qua bộ giải quyết dự phòng, dùng để đo thời gian một
--     yêu cầu 500 phần tử (T106) và xác minh hạn mức lượt của gói miễn phí
--     khi không có grant (FR-005b/c/d).
--
-- Mọi tài khoản đều có tiền tố `t024_` để nhận diện và xoá sạch được.
-- Chạy lại nhiều lần không sinh trùng: `ON CONFLICT (lower(username))`, vì unique
-- index của bảng nằm trên biểu thức `lower(username)` chứ không phải trên cột.
--
-- CÁCH CHẠY (ON_ERROR_STOP đã bật sẵn trong file, không cần truyền thêm)
--   psql -f database/scripts/08-seed_grant_test_data.sql
--
-- CÁCH XOÁ
--   psql -f database/scripts/08-seed_grant_test_data.sql --set cleanup=1
--
-- ⚠️ file này GHI vào cơ sở dữ liệu. Chỉ chạy trên môi trường phát triển.
-- ============================================================================

-- Dừng ngay tại câu lệnh hỏng.
--
-- ⚠️ ĐÂY KHÔNG PHẢI CHUYỆN NHỎ. `psql` mặc định VẪN CHẠY TIẾP sau một câu lệnh
-- hỏng. Một script nhiều câu lệnh sẽ vừa báo lỗi vừa tiếp tục xoá dữ liệu ở
-- những câu lệnh phía sau, và người chạy chỉ thấy một dòng ERROR ở giữa rồi tưởng
-- đã dừng. Khi đó việc dọn dữ liệu diễn ra một cách âm thầm và không thể đảo.
-- Biến này làm lỗi thành lỗi.
\set ON_ERROR_STOP on

\if :{?cleanup}
\else
  \set cleanup 0
\endif

\if :cleanup

BEGIN;

\echo 'XOA du lieu kiem thu T024...'

-- Xoá theo thứ tự để không vi phạm khoá ngoại.
--
-- Outbox đi trước KHÔNG vì khoá ngoại — `shared.outbox_records` không có khoá
-- ngoại nào tới bảng sự kiện, và hướng liên kết cũng là ngược lại: bản ghi mức
-- quyền mới là thứ trỏ tới sự kiện. Thứ tự này là vì ý nghĩa: bản ghi sự kiện
-- phải còn đúng lúc xoá outbox, vì chính `payload->>'sourceEventId'` là thứ đang
-- dùng để tìm bản ghi sự kiện. Liên kết đi qua biểu thức đó chứ không phải một
-- cột, theo đúng đặc tả hợp đồng sự kiện.
--
-- (024 T204) Câu này xoá 0 dòng ngay sau lần tạo, và đó là ĐÚNG. Phần CREATE của
-- tệp này cố ý không tạo bản ghi sự kiện nào: các tài khoản `t024_*` được dựng
-- bằng `INSERT` thẳng nên không có lịch sử đăng ký. Câu này chỉ bắt đầu xoá được
-- sau khi ai đó chạy công cụ sửa chữa FR-088 — lúc đó bản ghi sự kiện và bản ghi
-- outbox mới thật sự tồn tại cho những tài khoản này. Giữ câu lại là đúng; chỉ
-- lý do nêu trong nó là sai, và lý do sai thì tệ hơn không có lý do.
DELETE FROM shared.outbox_records
 WHERE payload->>'sourceEventId' IN (
    SELECT e.id::text
      FROM subscription.user_subscription_events e
      JOIN identity.users u ON u.id = e.user_id
     WHERE u.username LIKE 't024\_%'
 );

DELETE FROM subscription.user_subscription_events e
 USING identity.users u
 WHERE u.id = e.user_id AND u.username LIKE 't024\_%';

DELETE FROM subscription.user_daily_quotas q
 USING identity.users u
 WHERE u.id = q.user_id AND u.username LIKE 't024\_%';

DELETE FROM subscription.user_subscriptions s
 USING identity.users u
 WHERE u.id = s.user_id AND u.username LIKE 't024\_%';

DELETE FROM shared.admin_audit_logs a
 USING identity.users u
 WHERE u.id = a.actor_user_id AND u.username LIKE 't024\_%';

DELETE FROM identity.users WHERE username LIKE 't024\_%';

\echo 'Da xoa du lieu kiem thu T024.'

COMMIT;

\else

BEGIN;

\echo 'TAO du lieu kiem thu T024...'

-- Quản trị viên thứ hai. Mật khẩu đặt giống tài khoản admin sẵn có của môi
-- trường phát triển; chỉ dùng để kiểm thử.
INSERT INTO identity.users (username, email, password_hash, role_slug, email_verified)
VALUES (
    't024_admin2',
    't024_admin2@closy.local',
    (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1),
    'admin',
    true
)
ON CONFLICT (lower(username)) DO NOTHING;

-- 500 tài khoản khách hàng trên gói free.
--
-- ⚠️ Cố Ý KHÔNG tạo bản ghi user_subscriptions và không tạo grant: đó chính là
-- trạng thái cần kiểm thử. Lượt dùng của họ phải đi qua bộ giải quyết dự phòng,
-- và đó là đường cần biết là có ép đúng hạn mức lượt hay không.
INSERT INTO identity.users (username, email, password_hash, role_slug, email_verified)
SELECT
    't024_u' || lpad(n::text, 4, '0'),
    't024_u' || lpad(n::text, 4, '0') || '@closy.local',
    (SELECT password_hash FROM identity.users WHERE role_slug = 'admin' AND is_deleted = false LIMIT 1),
    'user',
    true
FROM generate_series(1, 500) AS n
ON CONFLICT (lower(username)) DO NOTHING;

\echo '--- so lieu vua tao ---'
SELECT
    count(*) FILTER (WHERE role_slug = 'admin') AS admins,
    count(*) FILTER (WHERE role_slug = 'user')  AS customers
FROM identity.users
WHERE username LIKE 't024\_%';

\echo '--- xac nhan KHONG con ban ghi muc quyen AI ( dung duong du phong ) ---'
SELECT count(*) AS customers_with_grant
FROM identity.users u
JOIN aiexecutor.ai_policy_grants g ON g.user_id = u.id
WHERE u.username LIKE 't024\_%';

\echo 'Da tao du lieu kiem thu T024.'

COMMIT;

\endif