-- =============================================================================
-- 12-dry_run_preprod_cleanup.sql — XEM, KHÔNG GHI
--
-- Đo trước phạm vi dọn dữ liệu demo ở production, chuẩn bị chạy thật.
-- Chạy file này NGAY TRƯỚC `13-preprod_cleanup.sql` và đối chiếu với báo cáo
-- mà file 13 in ra sau khi xoá. Hai bảng phải khớp từng dòng.
--
-- File này chỉ gọi SELECT. Không BEGIN/COMMIT, không có side effect.
--
--   docker exec -i -e PGPASSWORD=... postgres-closy-dev psql \
--     -h host.docker.internal -p 15432 -U closy -d closy_db -q -f database/scripts/12-dry_run_preprod_cleanup.sql
--
-- Trên PowerShell không dùng toán tử `<`; dùng đường ống:
--
--   Get-Content -Raw -Encoding UTF8 database/scripts/12-dry_run_preprod_cleanup.sql `
--     | docker exec -i -e PGPASSWORD=... postgres-closy-dev psql `
--         -h host.docker.internal -p 15432 -U closy -d closy_db -q
-- =============================================================================

\pset pager off
\timing off

-- KHU VỰC 1 — 70 tài khoản seed sẽ bị xoá
-- Bốn mẫu username dưới đây là toàn bộ nguồn gốc dữ liệu demo, mỗi mẫu
-- truy được về đúng một script seed trong thư mục này. Tài khoản nào không
-- khớp cả bốn mẫu thì KHÔNG bị đụng tới — kể cả tài khoản thật.
\echo '=== [1] TAI KHOAN SE BI XOA ==='

SELECT pattern, count(*) AS users, min(username) AS first_username, max(username) AS last_username
FROM (
    SELECT CASE
               WHEN username LIKE 'fb\_cust\_%'      THEN 'fb_cust_*   (seed_fill_customer_user_ids.sql)'
               WHEN username LIKE 'conv\_user\_%'   THEN 'conv_user_*  (seed_feature_006_dashboard.sql)'
               WHEN username IN ('owner_branda', 'staff_branda')
                                                    THEN '*_branda     (seed_feature_006_dashboard.sql)'
               WHEN username LIKE 'demo%'          THEN 'demo*        (@hycat.online)'
           END AS pattern,
           username
    FROM identity.users
    WHERE username LIKE 'fb\_cust\_%'
       OR username LIKE 'conv\_user\_%'
       OR username IN ('owner_branda', 'staff_branda')
       OR username LIKE 'demo%'
) t
GROUP BY pattern
ORDER BY pattern;

-- Tài khoản bị xoá, liệt kê đầy đủ. Đọc danh sách này trước khi chạy file 13:
-- nếu có tài khoản thật lọt vào đây thì dừng lại, không chạy file 13.
\echo '=== [1b] DANH SACH DAY DU 70 TAI KHOAN BI XOA ==='

SELECT username, email, provider, status, created_at::date AS created
FROM identity.users
WHERE username LIKE 'fb\_cust\_%'
   OR username LIKE 'conv\_user\_%'
   OR username IN ('owner_branda', 'staff_branda')
   OR username LIKE 'demo%'
ORDER BY created_at, username;

-- KHU VỰC 2 — 26 tài khoản được giữ, và dữ liệu của họ
\echo '=== [2] 26 TAI KHOAN DUOC GIU (KHONG BI DUNG TOI) ==='

SELECT username, email, provider, status, created_at::date AS created,
       (SELECT count(*) FROM wardrobe.wardrobe_items w WHERE w.user_id = u.id) AS wardrobe_items,
       (SELECT count(*) FROM wardrobe.outfits o WHERE o.user_id = u.id)        AS outfits,
       (SELECT count(*) FROM subscription.user_subscriptions s WHERE s.user_id = u.id) AS subscriptions,
       (SELECT count(*) FROM subscription.deposit_transactions d WHERE d.user_id = u.id) AS deposits
FROM identity.users u
WHERE NOT (username LIKE 'fb\_cust\_%'
       OR username LIKE 'conv\_user\_%'
       OR username IN ('owner_branda', 'staff_branda')
       OR username LIKE 'demo%')
ORDER BY created_at, username;

-- KHU VỰC 3 — dòng bị xoá theo từng bảng
-- Số ở cột "will_delete" là dự đoán. File 13 in ra con số thực sau khi xoá;
-- hai cột này phải bằng nhau, nếu lệch thì dữ liệu đã dịch chuyển giữa lúc bạn
-- chạy file 12 và lúc bạn chạy file 13 — chạy lại file 12.
\echo '=== [3] SO DONG BI XOA THEO BANG ==='

WITH doomed_users AS (
    SELECT id, username FROM identity.users
    WHERE username LIKE 'fb\_cust\_%'
       OR username LIKE 'conv\_user\_%'
       OR username IN ('owner_branda', 'staff_branda')
       OR username LIKE 'demo%'
),
kept_users AS (
    SELECT id, username FROM identity.users
    WHERE NOT (username LIKE 'fb\_cust\_%'
           OR username LIKE 'conv\_user\_%'
           OR username IN ('owner_branda', 'staff_branda')
           OR username LIKE 'demo%')
)
SELECT step, target_table, will_delete, note
FROM (
    -- Thứ tự phải khớp với file 13. Lý do thứ tự nằm ở file 13.
    SELECT 1 AS step, 'brand.loyalty_point_lots' AS target_table,
           (SELECT count(*) FROM brand.loyalty_point_lots) AS will_delete,
           'RESTRICT sang loyalty_point_transactions — phai xoa truoc' AS note
    UNION ALL
    SELECT 2, 'brand.brands',
           (SELECT count(*) FROM brand.brands),
           'CASCADE gom 14 bang brand con lai (xem bang [4])'
    UNION ALL
    SELECT 3, 'dashboard.dashboard_admin_daily_snapshots',
           (SELECT count(*) FROM dashboard.dashboard_admin_daily_snapshots),
           'Du lieu tong hop, app tinh lai tu wardrobe'
    UNION ALL
    SELECT 3, 'dashboard.dashboard_brand_daily_snapshots',
           (SELECT count(*) FROM dashboard.dashboard_brand_daily_snapshots),
           'Brand se bi xoa het nen snapshot cung vo nghia'
    UNION ALL
    SELECT 3, 'dashboard.dashboard_user_wardrobe_summaries',
           (SELECT count(*) FROM dashboard.dashboard_user_wardrobe_summaries),
           'Derived — tinh lai khi user mo trang'
    UNION ALL
    SELECT 4, 'identity.users',
           (SELECT count(*) FROM doomed_users),
           'CASCADE gom subscriptions/quotas/wallets/tokens/brand_members'
    UNION ALL
    -- Điều kiện này KHÔNG phụ thuộc đã xoá user hay chưa: một item bị coi là
    -- mồ côi khi không còn wardrobe_item nào của 26 tài khoản được giữ tham chiếu.
    -- Nhờ vậy file 13 chạy đúng dù thứ tự lệnh có đổi.
    SELECT 5, 'fashion.fashion_items',
           (SELECT count(*) FROM fashion.fashion_items f
            WHERE NOT EXISTS (
                SELECT 1 FROM wardrobe.wardrobe_items w
                JOIN kept_users k ON k.id = w.user_id
                WHERE w.fashion_item_id = f.id)),
           'Chi xoa item khong con ai dung — KHONG xoa item cua 26 tai khoan giu'
) t
ORDER BY step, target_table;

-- KHU VỰC 4 — dòng bị xoá bởi CASCADE, không cần viết lệnh riêng
-- Những bảng này không có câu DELETE nào trong file 13; chúng bị dọn bởi
-- ON DELETE CASCADE từ brand.brands (bước 2) và identity.users (bước 4).
\echo '=== [4] DONG BI XOA BOI CASCADE (khong co lenh DELETE rieng) ==='

SELECT 'brand.brand_members' AS cascaded_by_cascade, count(*) AS rows_deleted FROM brand.brand_members
UNION ALL SELECT 'brand.brand_customers',          count(*) FROM brand.brand_customers
UNION ALL SELECT 'brand.brand_benefits',           count(*) FROM brand.brand_benefits
UNION ALL SELECT 'brand.benefit_redemptions',      count(*) FROM brand.benefit_redemptions
UNION ALL SELECT 'brand.brand_conversations',      count(*) FROM brand.brand_conversations
UNION ALL SELECT 'brand.brand_conversation_messages', count(*) FROM brand.brand_conversation_messages
UNION ALL SELECT 'brand.brand_items',              count(*) FROM brand.brand_items
UNION ALL SELECT 'brand.brand_customer_claims',    count(*) FROM brand.brand_customer_claims
UNION ALL SELECT 'brand.loyalty_programs',         count(*) FROM brand.loyalty_programs
UNION ALL SELECT 'brand.loyalty_tiers',            count(*) FROM brand.loyalty_tiers
UNION ALL SELECT 'brand.loyalty_accounts',         count(*) FROM brand.loyalty_accounts
UNION ALL SELECT 'brand.loyalty_point_transactions', count(*) FROM brand.loyalty_point_transactions
UNION ALL SELECT 'brand.digital_sample_responses', count(*) FROM brand.digital_sample_responses
UNION ALL SELECT 'subscription.user_subscriptions', count(*) FROM subscription.user_subscriptions
             WHERE user_id IN (SELECT id FROM identity.users
                               WHERE username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                                  OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')
UNION ALL SELECT 'subscription.user_daily_quotas', count(*) FROM subscription.user_daily_quotas
             WHERE user_id IN (SELECT id FROM identity.users
                               WHERE username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                                  OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')
UNION ALL SELECT 'subscription.user_wallets', count(*) FROM subscription.user_wallets
             WHERE user_id IN (SELECT id FROM identity.users
                               WHERE username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                                  OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')
UNION ALL SELECT 'identity.refresh_tokens', count(*) FROM identity.refresh_tokens
             WHERE user_id IN (SELECT id FROM identity.users
                               WHERE username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                                  OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')
UNION ALL SELECT 'wardrobe.wardrobe_items', count(*) FROM wardrobe.wardrobe_items
             WHERE user_id IN (SELECT id FROM identity.users
                               WHERE username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                                  OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')
ORDER BY 1;

-- KHU VỰC 5 — dữ liệu KHÔNG bị đụng tới
-- Bảng này là những thứ bạn đã quyết định giữ lại. Nếu một bảng ở đây bằng 0
-- mà bạn không xoá nó, hãy dừng lại và đọc lại file 13 trước khi chạy.
\echo '=== [5] DU LIEU GIU NGUYEN ==='

SELECT 'identity.users' AS kept_untouched, count(*) AS rows_kept FROM identity.users
UNION ALL SELECT 'wardrobe.wardrobe_items',           count(*) FROM wardrobe.wardrobe_items
UNION ALL SELECT 'wardrobe.outfits',                  count(*) FROM wardrobe.outfits
UNION ALL SELECT 'wardrobe.outfit_items',            count(*) FROM wardrobe.outfit_items
UNION ALL SELECT 'subscription.deposit_transactions', count(*) FROM subscription.deposit_transactions
UNION ALL SELECT 'subscription.user_subscriptions',   count(*) FROM subscription.user_subscriptions
UNION ALL SELECT 'subscription.user_wallets',         count(*) FROM subscription.user_wallets
UNION ALL SELECT 'subscription.wallet_statements',    count(*) FROM subscription.wallet_statements
UNION ALL SELECT 'subscription.campaigns',            count(*) FROM subscription.campaigns
UNION ALL SELECT 'subscription.campaign_claims',      count(*) FROM subscription.campaign_claims
UNION ALL SELECT 'aiexecutor.ai_usage_events',       count(*) FROM aiexecutor.ai_usage_events
UNION ALL SELECT 'aiexecutor.ai_usage_period_ledgers', count(*) FROM aiexecutor.ai_usage_period_ledgers
UNION ALL SELECT 'aiexecutor.ai_policy_grants',      count(*) FROM aiexecutor.ai_policy_grants
UNION ALL SELECT 'shared.outbox_records',            count(*) FROM shared.outbox_records
UNION ALL SELECT 'community.posts',                   count(*) FROM community.posts
UNION ALL SELECT 'community.post_comments',          count(*) FROM community.post_comments
UNION ALL SELECT 'community.post_likes',             count(*) FROM community.post_likes
UNION ALL SELECT 'community.post_media',             count(*) FROM community.post_media
UNION ALL SELECT 'community.follows',                count(*) FROM community.follows
UNION ALL SELECT 'community.user_follow_stats',       count(*) FROM community.user_follow_stats
UNION ALL SELECT 'fashion.conversational_contexts',  count(*) FROM fashion.conversational_contexts
UNION ALL SELECT 'fashion.messages',                 count(*) FROM fashion.messages
UNION ALL SELECT 'identity.refresh_tokens',          count(*) FROM identity.refresh_tokens
ORDER BY 1;

-- KHU VỰC 6 — baseline phải CÒN sau khi dọn
-- Số ở cột "expected" là giá trị app cần để chạy. Không có một trong những
-- dòng này thì hệ thống hỏng, và KHÔNG tự phục hồi được:
--
--   * `database/init/03-seed.sql` được mount vào /docker-entrypoint-initdb.d,
--     chỉ chạy đúng một lần khi data dir còn rỗng. Production đã có dữ liệu
--     nên file đó không bao giờ chạy lại. Xoá là mất vĩnh viễn.
--   * Subscription Catalog Validator chặn boot service account khi không có
--     đúng một plan free đang active
--     (services/account/internal/modules/subscription/application/validator/catalog_validator.go).
--   * `public.goose_db_version` giữ lịch sử migration. Xoá ⇒ goose coi là
--     chưa migration nào và chạy lại từ đầu.
--
-- File 13 kiểm tra lại đúng bảng này trước khi COMMIT.
\echo '=== [6] BASELINE PHAI CON LAI ==='

SELECT 'subscription.subscription_plans (can free)' AS baseline_row, 2 AS expected,
       (SELECT count(*) FROM subscription.subscription_plans) AS actual,
       (SELECT count(*) FROM subscription.subscription_plans
        WHERE slug = 'free' AND is_active) AS note
UNION ALL
SELECT 'fashion.categories',                     9, count(*), 0 FROM fashion.categories
UNION ALL
SELECT 'aiexecutor.ai_cost_policies',            2, count(*), 0 FROM aiexecutor.ai_cost_policies
UNION ALL
SELECT 'aiexecutor.ai_cost_policy_operations',   8, count(*), 0 FROM aiexecutor.ai_cost_policy_operations
UNION ALL
SELECT 'aiexecutor.ai_model_configs',            7, count(*), 0 FROM aiexecutor.ai_model_configs
UNION ALL
SELECT 'public.goose_db_version',               32, count(*), 0 FROM public.goose_db_version
UNION ALL
SELECT 'public.goose_db_version_commerce',       3, count(*), 0 FROM public.goose_db_version_commerce;

-- KHU VỰC 7 — bất biến sau khi dọn, để đối chiếu với file 13
\echo '=== [7] KY VONG SAU KHI DON ==='

SELECT 'identity.users' AS table_name, 26 AS expected_remaining,
       (SELECT count(*) FROM identity.users
        WHERE NOT (username LIKE 'fb\_cust\_%' OR username LIKE 'conv\_user\_%'
                   OR username IN ('owner_branda','staff_branda') OR username LIKE 'demo%')) AS actual_remaining
UNION ALL
SELECT 'fashion.fashion_items', 317,
       (SELECT count(*) FROM fashion.fashion_items f
        WHERE EXISTS (SELECT 1 FROM wardrobe.wardrobe_items w
                      JOIN identity.users u ON u.id = w.user_id
                      WHERE w.fashion_item_id = f.id
                        AND NOT (u.username LIKE 'fb\_cust\_%' OR u.username LIKE 'conv\_user\_%'
                                 OR u.username IN ('owner_branda','staff_branda') OR u.username LIKE 'demo%')))
UNION ALL
SELECT 'brand.brands', 0, (SELECT count(*) FROM brand.brands)
UNION ALL
SELECT 'dashboard.dashboard_admin_daily_snapshots', 0, (SELECT count(*) FROM dashboard.dashboard_admin_daily_snapshots)
UNION ALL
SELECT 'dashboard.dashboard_brand_daily_snapshots', 0, (SELECT count(*) FROM dashboard.dashboard_brand_daily_snapshots)
UNION ALL
SELECT 'dashboard.dashboard_user_wardrobe_summaries', 0, (SELECT count(*) FROM dashboard.dashboard_user_wardrobe_summaries);