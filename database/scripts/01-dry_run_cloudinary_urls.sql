-- ============================================================================
-- SCRIPT DRY-RUN: KIỂM TRA & RÀ SOÁT CÁC URL CLOUDINARY CẦN CẬP NHẬT
-- (Đã cập nhật theo Database Schema mới nhất: smart_wardrobe_db-schema.sql)
-- ============================================================================
-- Các cột rà soát (5 cột thuộc 4 bảng):
--   1. users.avatar_url
--   2. fashion_items.image_url
--   3. outfits.cover_image_url
--   4. brands.logo_url
--   5. brands.background_url
--
-- Quy tắc rà soát:
--   Loại 1 (Xóa nền):         /upload/e_background_removal/e_trim:10,f_png,q_auto/ -> /upload/t_bg_remove/
--   Loại 2 (Cắt lề nội bộ):   /upload/e_trim:10/                                    -> /upload/t_trim_10/
--   Loại 3 (Fetch ảnh ngoài): /fetch/f_png,e_trim:20/                              -> /fetch/t_fetch_trim/
-- ============================================================================

-- 1. TỔNG BẢNG THỐNG KÊ SỐ LƯỢNG BẢN GHI BỊ ẢNH HƯỞNG THEO TỪNG BẢNG & LOẠI TAG

WITH summary AS (
    -- 1. users (avatar_url)
    SELECT 'users' AS table_name, 'avatar_url' AS column_name,
        COUNT(CASE WHEN avatar_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%' THEN 1 END) AS type1_count,
        COUNT(CASE WHEN avatar_url LIKE '%/upload/e_trim:10/%' THEN 1 END) AS type2_count,
        COUNT(CASE WHEN avatar_url LIKE '%/fetch/f_png,e_trim:20/%' THEN 1 END) AS type3_count
    FROM users

    UNION ALL

    -- 2. fashion_items (image_url)
    SELECT 'fashion_items', 'image_url',
        COUNT(CASE WHEN image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%' THEN 1 END),
        COUNT(CASE WHEN image_url LIKE '%/upload/e_trim:10/%' THEN 1 END),
        COUNT(CASE WHEN image_url LIKE '%/fetch/f_png,e_trim:20/%' THEN 1 END)
    FROM fashion_items

    UNION ALL

    -- 3. outfits (cover_image_url)
    SELECT 'outfits', 'cover_image_url',
        COUNT(CASE WHEN cover_image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%' THEN 1 END),
        COUNT(CASE WHEN cover_image_url LIKE '%/upload/e_trim:10/%' THEN 1 END),
        COUNT(CASE WHEN cover_image_url LIKE '%/fetch/f_png,e_trim:20/%' THEN 1 END)
    FROM outfits

    UNION ALL

    -- 4. brands (logo_url)
    SELECT 'brands', 'logo_url',
        COUNT(CASE WHEN logo_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%' THEN 1 END),
        COUNT(CASE WHEN logo_url LIKE '%/upload/e_trim:10/%' THEN 1 END),
        COUNT(CASE WHEN logo_url LIKE '%/fetch/f_png,e_trim:20/%' THEN 1 END)
    FROM brands

    UNION ALL

    -- 5. brands (background_url)
    SELECT 'brands', 'background_url',
        COUNT(CASE WHEN background_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%' THEN 1 END),
        COUNT(CASE WHEN background_url LIKE '%/upload/e_trim:10/%' THEN 1 END),
        COUNT(CASE WHEN background_url LIKE '%/fetch/f_png,e_trim:20/%' THEN 1 END)
    FROM brands
)
SELECT 
    table_name,
    column_name,
    type1_count AS "Loại 1 (Bg Remove)",
    type2_count AS "Loại 2 (Trim 10)",
    type3_count AS "Loại 3 (Fetch Trim)",
    (type1_count + type2_count + type3_count) AS "Tổng bị ảnh hưởng"
FROM summary
ORDER BY table_name, column_name;


-- 2. CHI TIẾT BẢN GHI MẪU SẼ DỰ KIẾN BỊ THAY ĐỔI (SAMPLE PREVIEW - MAXIMUM 5 BẢN GHI/BẢNG)

-- 1. users (avatar_url)
SELECT 'users' AS table_name, id::text, avatar_url AS old_url,
    REPLACE(
        REPLACE(
            REPLACE(avatar_url, 
                '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
            '/upload/e_trim:10/', '/upload/t_trim_10/'),
        '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
    ) AS new_url
FROM users
WHERE avatar_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR avatar_url LIKE '%/upload/e_trim:10/%'
   OR avatar_url LIKE '%/fetch/f_png,e_trim:20/%'
LIMIT 5;

-- 2. fashion_items (image_url)
SELECT 'fashion_items' AS table_name, id::text, image_url AS old_url,
    REPLACE(
        REPLACE(
            REPLACE(image_url, 
                '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
            '/upload/e_trim:10/', '/upload/t_trim_10/'),
        '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
    ) AS new_url
FROM fashion_items
WHERE image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR image_url LIKE '%/upload/e_trim:10/%'
   OR image_url LIKE '%/fetch/f_png,e_trim:20/%'
LIMIT 5;

-- 3. outfits (cover_image_url)
SELECT 'outfits' AS table_name, id::text, cover_image_url AS old_url,
    REPLACE(
        REPLACE(
            REPLACE(cover_image_url, 
                '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
            '/upload/e_trim:10/', '/upload/t_trim_10/'),
        '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
    ) AS new_url
FROM outfits
WHERE cover_image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR cover_image_url LIKE '%/upload/e_trim:10/%'
   OR cover_image_url LIKE '%/fetch/f_png,e_trim:20/%'
LIMIT 5;

-- 4. brands (logo_url & background_url)
SELECT 'brands (logo_url)' AS table_name, id::text, logo_url AS old_url,
    REPLACE(
        REPLACE(
            REPLACE(logo_url, 
                '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
            '/upload/e_trim:10/', '/upload/t_trim_10/'),
        '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
    ) AS new_url
FROM brands
WHERE logo_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR logo_url LIKE '%/upload/e_trim:10/%'
   OR logo_url LIKE '%/fetch/f_png,e_trim:20/%'
LIMIT 5;

SELECT 'brands (background_url)' AS table_name, id::text, background_url AS old_url,
    REPLACE(
        REPLACE(
            REPLACE(background_url, 
                '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
            '/upload/e_trim:10/', '/upload/t_trim_10/'),
        '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
    ) AS new_url
FROM brands
WHERE background_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR background_url LIKE '%/upload/e_trim:10/%'
   OR background_url LIKE '%/fetch/f_png,e_trim:20/%'
LIMIT 5;
