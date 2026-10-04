-- =============================================================================
-- 13-preprod_cleanup.sql — GHI
--
-- Dọn dữ liệu seed/demo ở production để chuẩn bị chạy thật. Toàn bộ nằm trong
-- MỘT transaction: lỗi giữa chừng ⇒ ROLLBACK toàn bộ, không để lại trạng thái
-- nửa vời.
--
-- ⚠️  BA ĐIỀU KIỆN PHẢI ĐÚNG, nếu không script dừng trước khi ghi bất cứ thứ gì:
--
--   1. Bắt buộc truyền `--set apply=1`. Không có thì script thoát ngay.
--   2. Bắt buộc truyền `--set kept_users="..."` — danh sách tài khoản được giữ.
--      Nội dung lấy từ docs/temp/preprod-kept-users.txt (thư mục này gitignore).
--      Sau khi xoá, tập tài khoản còn lại phải khớp DANH TÍCH danh sách đó, không
--      chỉ khớp số lượng. Lệch một tên, thừa một tên, hoặc thiếu danh sách ⇒
--      rollback.
--   3. Baseline config phải còn nguyên sau khi xoá. Sai số ⇒ rollback.
--
-- Vì sao danh sách không nằm thẳng trong file này: đó là tài khoản người dùng
-- thật, và repo không nên chứa nó. Danh sách nằm ngoài git, truyền vào lúc
-- chạy.
--
-- Chạy file 12 (dry-run) trước, đối chiếu số dòng, rồi mới chạy file này:
--
--   psql "host=<host> user=closy password=<pwd> dbname=closy_db port=<port> sslmode=disable" \
--        -v apply=1 -v kept_users="admin,user,..." \
--        -f database/scripts/13-preprod_cleanup.sql
--
-- Trên PowerShell (không có toán tử `<`):
--
--   $kept = (Get-Content docs/temp/preprod-kept-users.txt | Where-Object { $_ -and $_ -notmatch '^\s*#' }) -join ','
--   Get-Content -Raw -Encoding UTF8 database/scripts/13-preprod_cleanup.sql `
--     | docker exec -i -e PGPASSWORD=$env:PGPROD_PW postgres-closy-dev `
--         psql -h <host> -p <port> -U closy -d closy_db -v apply=1 -v kept_users="$kept" -q
--
-- Cần CÓ dump backup trước khi chạy. Script này xoá vĩnh viễn.
-- =============================================================================

\set ON_ERROR_STOP on
\pset pager off
\timing off

-- -----------------------------------------------------------------------------
-- GUARD 1 — phải có apply=1
--
-- Không có dòng này thì một sự kiện kiểu "psql -f 13-...sql" gõ nhầm tay là
-- đủ để chạy toàn bộ lệnh xoá. Đây là loại tai nạn không hoàn tác được.
-- -----------------------------------------------------------------------------
\if :{?apply}
\else
    \echo ''
    \echo '!!! DUONG: chay lai voi: -v apply=1'
    \echo ''
    \quit
\endif

-- So sanh chuoi bang \gset chu khong phai bang \if: \if ep gia tri sang
-- boolean, nen -v apply=0 se bao loi "Boolean expected" thay vi dung la
-- "gia tri khong hop le". \gset tra ve t/f ma \if doc duoc.
SELECT :'apply' = '1' AS apply_is_one \gset

\if :apply_is_one
\else
    \echo ''
    \echo '!!! apply phai co gia tri "1". Khong co gia tri khac duoc chap nhan.'
    \echo ''
    \quit
\endif

\echo '=== BAT DAU CLEANUP ==='
SELECT now() AS started_at, current_database() AS db, current_user AS db_user;

BEGIN;

-- Kiểm đầu vào TRƯỚC khi tạo bất kỳ bảng tạm nào. Xem giải thích ở khối
-- "ĐIỀU KIỆN ĐẦU VÀO" ngay dưới đây.
\if :{?kept_users}
\else
    \echo ''
    \echo '!!! DUONG: thieu -v kept_users="<danh sach username, phan tach bang dau phay>"'
    \echo '!!! Danh sach mau: docs/temp/preprod-kept-users.txt'
    \echo ''
    \quit
\endif

CREATE TEMP TABLE expected_kept_users ON COMMIT DROP AS
SELECT btrim(x) AS username
FROM unnest(string_to_array(:'kept_users', ',')) AS x
WHERE btrim(x) <> '';

-- Danh sách rỗng cũng là sai — ví dụ truyền nhầm `-v kept_users=""`. Bắt ở đây
-- luôn để không phải chờ tới guard 2 sau khi đã xoá.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM expected_kept_users) THEN
        RAISE EXCEPTION 'ROLLBACK: danh sach tai khoan duoc giu rong (kept_users rong)';
    END IF;
END $$;

-- Tạo danh sách 26 tài khoản được giữ NGAY TRONG TRANSACTION, một lần duy nhất.
-- Mọi lệnh DELETE và mọi guard phía sau đọc từ bảng tạm này. Nhờ vậy phạm vi
-- xoá là cố định trong suốt transaction: không có câu nào phải tự viết lại điều
-- kiện "không phải user seed", nên không có chỗ nào hai bên lệch nhau.
CREATE TEMP TABLE kept_users ON COMMIT DROP AS
SELECT id, username
FROM identity.users
WHERE NOT (username LIKE 'fb\_cust\_%'
       OR username LIKE 'conv\_user\_%'
       OR username IN ('owner_branda', 'staff_branda')
       OR username LIKE 'demo%');

CREATE TEMP TABLE doomed_users ON COMMIT DROP AS
SELECT id, username
FROM identity.users
WHERE username LIKE 'fb\_cust\_%'
   OR username LIKE 'conv\_user\_%'
   OR username IN ('owner_branda', 'staff_branda')
   OR username LIKE 'demo%';

-- Ảnh chụp số dòng trước khi xoá, để in báo cáo "đã xoá N" chính xác.
-- KHÔNG dùng ON COMMIT DROP cho bảng này: báo cáo đối chiếu chạy SAU COMMIT,
-- nên bảng phải còn sống tới lúc đó. kept_users và doomed_users thì ngược lại —
-- chúng chỉ phục vụ các lệnh DELETE và guard trong transaction, và tự biến mất
-- khi commit là đúng.
CREATE TEMP TABLE before_counts AS
SELECT 'brand.loyalty_point_lots' AS target_table, count(*)::bigint AS n FROM brand.loyalty_point_lots
UNION ALL SELECT 'brand.brands', count(*) FROM brand.brands
UNION ALL SELECT 'brand.brand_members', count(*) FROM brand.brand_members
UNION ALL SELECT 'brand.brand_customers', count(*) FROM brand.brand_customers
UNION ALL SELECT 'brand.loyalty_point_transactions', count(*) FROM brand.loyalty_point_transactions
UNION ALL SELECT 'brand.digital_sample_responses', count(*) FROM brand.digital_sample_responses
UNION ALL SELECT 'dashboard.dashboard_admin_daily_snapshots', count(*) FROM dashboard.dashboard_admin_daily_snapshots
UNION ALL SELECT 'dashboard.dashboard_brand_daily_snapshots', count(*) FROM dashboard.dashboard_brand_daily_snapshots
UNION ALL SELECT 'dashboard.dashboard_user_wardrobe_summaries', count(*) FROM dashboard.dashboard_user_wardrobe_summaries
UNION ALL SELECT 'identity.users', count(*) FROM identity.users
UNION ALL SELECT 'wardrobe.wardrobe_items', count(*) FROM wardrobe.wardrobe_items
UNION ALL SELECT 'fashion.fashion_items', count(*) FROM fashion.fashion_items;

\echo ''
\echo '--- BUOC 1/5: brand.loyalty_point_lots'
-- Bắt buộc đứng trước brand.brands. Cột earn_transaction_id tham chiếu
-- loyalty_point_transactions bằng ON DELETE RESTRICT, tức là Postgres từ chối
-- xoá một giao dịch điểm khi còn lot trỏ tới nó. Xoá brands trước sẽ kéo theo
-- xoá transactions trong cùng cascade và chạm RESTRICT ⇒ lỗi. Xoá lots trước
-- là hết đường vướng.
DELETE FROM brand.loyalty_point_lots;

\echo '--- BUOC 2/5: brand.brands (CASCADE gom 14 bang brand con)'
-- 3 brands seed: Thoi Trang ABC, Fashion XYZ, Demo Regis. Mọi thứ về brand —
-- members, customers, loyalty, benefits, vouchers, hội thoại CS, sản phẩm —
-- đều là dữ liệu demo feature 006, không có brand nào được đăng ký thật.
--
-- Brand nằm TRƯỚC bước xoá user vì brands.created_by_user_id là ON DELETE
-- RESTRICT: xoá user admin/user trước sẽ bị chặn bởi chính các brand này.
DELETE FROM brand.brands;

\echo '--- BUOC 3/5: dashboard'
-- Cả ba bảng đều là số liệu tổng hợp. App tự tính lại: với brand thì không còn
-- gì để tổng hợp, với wardrobe thì chạy lại khi user mở trang. Xoá không mất
-- gì mà giữ lại thì dashboard production sẽ hiện số của dữ liệu demo.
DELETE FROM dashboard.dashboard_admin_daily_snapshots;
DELETE FROM dashboard.dashboard_brand_daily_snapshots;
DELETE FROM dashboard.dashboard_user_wardrobe_summaries;

\echo '--- BUOC 4/5: 70 tai khoan seed'
-- CASCADE gom: user_subscriptions 70, user_daily_quotas 2, user_wallets 2,
-- refresh_tokens 21, brand_members 2, wardrobe_items 16.
--
-- KHÔNG có user thật nào nằm trong bốn mẫu này. Mẫu lấy từ chính script đã
-- sinh ra chúng: fb_cust_* và *_branda từ seed feature 006, conv_user_* từ
-- feature 006, demo* từ seed @hycat.online.
DELETE FROM identity.users WHERE id IN (SELECT id FROM doomed_users);

\echo '--- BUOC 5/5: fashion_items khong con ai dung'
-- Điều kiện đọc từ kept_users, tức là "không còn wardrobe_item nào của 26 tài
-- khoản được giữ tham chiếu". KHÔNG viết theo kiểu "không còn wardrobe_item
-- nào" — bản đó sẽ nuốt luôn item của tài khoản thật, vì tại thời điểm này 16
-- wardrobe_item của tài khoản seed đã bị xoá bởi bước 4 và các item tương ứng
-- đã trở thành mồ côi một cách giả.
--
-- 9 item ảnh placehold.co của seed 006 nằm trong tập này, và 13 item mà
-- brand.brand_items trỏ tới (những brand_items đó đã bị xoá ở bước 2).
DELETE FROM fashion.fashion_items f
WHERE NOT EXISTS (
    SELECT 1 FROM wardrobe.wardrobe_items w
    JOIN kept_users k ON k.id = w.user_id
    WHERE w.fashion_item_id = f.id
);

-- -----------------------------------------------------------------------------
-- GUARD 2 — tập tài khoản được giữ phải khớp CHÍNH XÁC
--
-- So từng tên với danh sách trong expected_kept_users. Danh sách nằm ở
-- docs/temp/preprod-kept-users.txt — thư mục đó gitignore, nên tên người dùng
-- thật không nằm trong lịch sử git.
--
-- Vì sao phải so TỪNG TÊN chứ không chỉ so `count(*)`: tình huống xoá nhầm một
-- tài khoản rồi có người mới đăng ký đúng lúc đó sẽ cho tổng vẫn bằng 26, trong
-- khi dữ liệu đã hỏng. So từng tên thì bắt được.
--
-- Cả hai chiều đều kiểm:
--   * thiếu — có tên trong danh sách mà không còn trong DB (đã bị xoá nhầm)
--   * thừa  — còn tài khoản trong DB mà không có trong danh sách (lọt vào, hoặc
--              danh sách bị sửa thiếu)
--
-- So với `kept_users` chứ không so trực tiếp với identity.users: kept_users là
-- tập script SẼ giữ lại. Bằng cách so hai bảng tạm, ta đồng thời bắt được lỗi ở
-- điều kiện lọc và lỗi ở danh sách — không cần tin rằng chúng khớp sẵn.
-- -----------------------------------------------------------------------------
DO $$
DECLARE
    missing text[];
    extra   text[];
BEGIN
    IF NOT EXISTS (SELECT 1 FROM expected_kept_users) THEN
        RAISE EXCEPTION
            'ROLLBACK: danh sach tai khoan duoc giu rong. '
            'Truyen -v kept_users="admin,user,..." (danh sach mau: docs/temp/preprod-kept-users.txt)';
    END IF;

    SELECT array_agg(e.username ORDER BY e.username)
      INTO missing
      FROM expected_kept_users e
     WHERE NOT EXISTS (SELECT 1 FROM identity.users u WHERE u.username = e.username);

    SELECT array_agg(k.username ORDER BY k.username)
      INTO extra
      FROM kept_users k
     WHERE NOT EXISTS (SELECT 1 FROM expected_kept_users e WHERE e.username = k.username);

    IF missing IS NOT NULL THEN
        RAISE EXCEPTION 'ROLLBACK: mat tai khoan duoc giu: %', missing;
    END IF;

    IF extra IS NOT NULL THEN
        RAISE EXCEPTION 'ROLLBACK: con tai khoan khong nam trong danh sach duoc giu: %', extra;
    END IF;

    RAISE NOTICE 'guard 2 OK: du % tai khoan duoc giu, khop danh tinh',
        (SELECT count(*) FROM expected_kept_users);
END $$;

-- -----------------------------------------------------------------------------
-- GUARD 3 — baseline config phải còn nguyên
--
-- Không một trong những bảng dưới đây tự phục hồi được:
--
--   * database/init/03-seed.sql được mount vào /docker-entrypoint-initdb.d.
--     Postgres chỉ chạy nó một lần, lúc data dir còn rỗng. Production đã có dữ
--     liệu nên nó không bao giờ chạy lại. Xoá 2 plan là mất vĩnh viễn.
--   * Catalog Validator chặn boot service account nếu không có đúng một plan
--     free đang active, và chặn nếu plan active nào thiếu ai_cost_policy_id
--     (services/account/.../validator/catalog_validator.go).
--   * goose đọc public.goose_db_version để biết migration nào đã chạy. Bảng
--     trống ⇒ goose chạy lại toàn bộ migration từ đầu.
--
-- Ngoài ra: còn tồn tại tài khoản seed nào sót lại, và số item mồ côi có khớp
-- dự đoán của file 12 hay không.
-- -----------------------------------------------------------------------------
DO $$
DECLARE
    plan_count       bigint;
    free_active      bigint;
    plans_no_policy  bigint;
    orphans_left     bigint;
BEGIN
    SELECT count(*) INTO plan_count FROM subscription.subscription_plans;
    SELECT count(*) INTO free_active FROM subscription.subscription_plans
     WHERE slug = 'free' AND is_active;
    SELECT count(*) INTO plans_no_policy FROM subscription.subscription_plans p
     WHERE p.is_active AND p.ai_cost_policy_id IS NULL;

    IF plan_count <> 2 THEN
        RAISE EXCEPTION 'ROLLBACK: subscription_plans = %, phai la 2', plan_count;
    END IF;
    IF free_active <> 1 THEN
        RAISE EXCEPTION 'ROLLBACK: can dung 1 plan free dang active, hien co %', free_active;
    END IF;
    IF plans_no_policy <> 0 THEN
        RAISE EXCEPTION 'ROLLBACK: % plan active thieu ai_cost_policy_id — account se khong boot',
            plans_no_policy;
    END IF;

    IF (SELECT count(*) FROM fashion.categories) <> 9 THEN
        RAISE EXCEPTION 'ROLLBACK: fashion.categories = %, phai la 9',
            (SELECT count(*) FROM fashion.categories);
    END IF;
    IF (SELECT count(*) FROM aiexecutor.ai_cost_policies) <> 2 THEN
        RAISE EXCEPTION 'ROLLBACK: ai_cost_policies = %, phai la 2',
            (SELECT count(*) FROM aiexecutor.ai_cost_policies);
    END IF;
    IF (SELECT count(*) FROM aiexecutor.ai_cost_policy_operations) <> 8 THEN
        RAISE EXCEPTION 'ROLLBACK: ai_cost_policy_operations = %, phai la 8',
            (SELECT count(*) FROM aiexecutor.ai_cost_policy_operations);
    END IF;
    IF (SELECT count(*) FROM aiexecutor.ai_model_configs) <> 7 THEN
        RAISE EXCEPTION 'ROLLBACK: ai_model_configs = %, phai la 7',
            (SELECT count(*) FROM aiexecutor.ai_model_configs);
    END IF;
    IF (SELECT count(*) FROM public.goose_db_version) <> 32 THEN
        RAISE EXCEPTION 'ROLLBACK: goose_db_version = %, phai la 32',
            (SELECT count(*) FROM public.goose_db_version);
    END IF;
    IF (SELECT count(*) FROM public.goose_db_version_commerce) <> 3 THEN
        RAISE EXCEPTION 'ROLLBACK: goose_db_version_commerce = %, phai la 3',
            (SELECT count(*) FROM public.goose_db_version_commerce);
    END IF;

    -- Còn sót tài khoản seed nào không? Nếu có thì bước 4 chưa làm hết việc.
    IF EXISTS (SELECT 1 FROM identity.users u
                JOIN doomed_users d ON d.id = u.id) THEN
        RAISE EXCEPTION 'ROLLBACK: con tai khoan seed chua bi xoa';
    END IF;

    -- Sau khi xoá, không được còn item nào mà 26 tài khoản được giữ đang dùng
    -- mà không có trong bảng fashion_items.
    SELECT count(*) INTO orphans_left
      FROM wardrobe.wardrobe_items w
      JOIN kept_users k ON k.id = w.user_id
      LEFT JOIN fashion.fashion_items f ON f.id = w.fashion_item_id
     WHERE f.id IS NULL;

    IF orphans_left <> 0 THEN
        RAISE EXCEPTION 'ROLLBACK: % wardrobe_item tro toi fashion_item da bi xoa', orphans_left;
    END IF;

    RAISE NOTICE 'guard 3 OK: baseline nguyen ven, khong con tai khoan seed';
END $$;

-- COMMIT đặt ở CUỐI FILE, sau cả khối báo cáo. Lý do: nếu câu báo cáo hỏng
-- (lỗi cú pháp, mất kết nối) thì ON_ERROR_STOP dừng script trước COMMIT và
-- toàn bộ việc dọn bị hoàn tác. Đặt COMMIT trước báo cáo nghĩa là báo cáo hỏng
-- không cứu được dữ liệu — bạn mất dữ liệu mà không có bản đối chiếu để biết
-- đã mất gì.

-- -----------------------------------------------------------------------------
-- BÁO CÁO SAU CLEANUP — chỉ để đọc
--
-- Đối chiếu "rows_deleted" ở đây với cột "will_delete" của file 12. Lệch ở
-- bất kỳ dòng nào nghĩa là dữ liệu đã dịch chuyển giữa lúc chạy file 12 và lúc
-- chạy file 13.
-- -----------------------------------------------------------------------------
\echo ''
\echo '=== BÁO CAO DA XOA (doi chieu voi file 12) ==='

SELECT b.target_table,
       b.n AS before_count,
       CASE b.target_table
           WHEN 'brand.loyalty_point_lots'  THEN (SELECT count(*) FROM brand.loyalty_point_lots)
           WHEN 'brand.brands'               THEN (SELECT count(*) FROM brand.brands)
           WHEN 'brand.brand_members'        THEN (SELECT count(*) FROM brand.brand_members)
           WHEN 'brand.brand_customers'      THEN (SELECT count(*) FROM brand.brand_customers)
           WHEN 'brand.loyalty_point_transactions' THEN (SELECT count(*) FROM brand.loyalty_point_transactions)
           WHEN 'brand.digital_sample_responses'    THEN (SELECT count(*) FROM brand.digital_sample_responses)
           WHEN 'dashboard.dashboard_admin_daily_snapshots'    THEN (SELECT count(*) FROM dashboard.dashboard_admin_daily_snapshots)
           WHEN 'dashboard.dashboard_brand_daily_snapshots'    THEN (SELECT count(*) FROM dashboard.dashboard_brand_daily_snapshots)
           WHEN 'dashboard.dashboard_user_wardrobe_summaries'  THEN (SELECT count(*) FROM dashboard.dashboard_user_wardrobe_summaries)
           WHEN 'identity.users'             THEN (SELECT count(*) FROM identity.users)
           WHEN 'wardrobe.wardrobe_items'    THEN (SELECT count(*) FROM wardrobe.wardrobe_items)
           WHEN 'fashion.fashion_items'      THEN (SELECT count(*) FROM fashion.fashion_items)
       END AS after_count,
       b.n - CASE b.target_table
           WHEN 'brand.loyalty_point_lots'  THEN (SELECT count(*) FROM brand.loyalty_point_lots)
           WHEN 'brand.brands'               THEN (SELECT count(*) FROM brand.brands)
           WHEN 'brand.brand_members'        THEN (SELECT count(*) FROM brand.brand_members)
           WHEN 'brand.brand_customers'      THEN (SELECT count(*) FROM brand.brand_customers)
           WHEN 'brand.loyalty_point_transactions' THEN (SELECT count(*) FROM brand.loyalty_point_transactions)
           WHEN 'brand.digital_sample_responses'    THEN (SELECT count(*) FROM brand.digital_sample_responses)
           WHEN 'dashboard.dashboard_admin_daily_snapshots'    THEN (SELECT count(*) FROM dashboard.dashboard_admin_daily_snapshots)
           WHEN 'dashboard.dashboard_brand_daily_snapshots'    THEN (SELECT count(*) FROM dashboard.dashboard_brand_daily_snapshots)
           WHEN 'dashboard.dashboard_user_wardrobe_summaries'  THEN (SELECT count(*) FROM dashboard.dashboard_user_wardrobe_summaries)
           WHEN 'identity.users'             THEN (SELECT count(*) FROM identity.users)
           WHEN 'wardrobe.wardrobe_items'    THEN (SELECT count(*) FROM wardrobe.wardrobe_items)
           WHEN 'fashion.fashion_items'      THEN (SELECT count(*) FROM fashion.fashion_items)
       END AS rows_deleted
FROM before_counts b
ORDER BY rows_deleted DESC, b.target_table;

\echo ''
\echo '=== 26 TAI KHOAN DUOC GIU ==='
SELECT username, email, provider, status,
       (SELECT count(*) FROM wardrobe.wardrobe_items w WHERE w.user_id = u.id) AS wardrobe_items,
       (SELECT count(*) FROM wardrobe.outfits o WHERE o.user_id = u.id)        AS outfits
FROM identity.users u
ORDER BY u.created_at, u.username;

\echo ''
\echo '=== BASELINE CON NGUYEN ==='
SELECT 'subscription_plans' AS row_type, count(*) AS rows_left FROM subscription.subscription_plans
UNION ALL SELECT 'fashion.categories',            count(*) FROM fashion.categories
UNION ALL SELECT 'ai_cost_policies',              count(*) FROM aiexecutor.ai_cost_policies
UNION ALL SELECT 'ai_cost_policy_operations',     count(*) FROM aiexecutor.ai_cost_policy_operations
UNION ALL SELECT 'ai_model_configs',              count(*) FROM aiexecutor.ai_model_configs
UNION ALL SELECT 'goose_db_version',              count(*) FROM public.goose_db_version
UNION ALL SELECT 'goose_db_version_commerce',     count(*) FROM public.goose_db_version_commerce
UNION ALL SELECT 'subscription_plans (free active)', count(*) FROM subscription.subscription_plans
     WHERE slug = 'free' AND is_active;

-- Mọi câu lệnh phía trên đã chạy xong trong transaction và đều đúng. Đến đây
-- mới ghi. Đây là dòng duy nhất trong file làm thay đổi dữ liệu vĩnh viễn.
COMMIT;

\echo ''
\echo '=== HOAN TAT (da COMMIT) ==='