-- ============================================================================
-- SCRIPT MIGRATION: CẬP NHẬT CÁC URL CLOUDINARY SANG NAMED TRANSFORMATIONS
-- (Đã cập nhật theo Database Schema mới nhất: smart_wardrobe_db-schema.sql)
-- ============================================================================
-- Quy tắc thay thế:
--   Loại 1 (Xóa nền):         /upload/e_background_removal/e_trim:10,f_png,q_auto/ -> /upload/t_bg_remove/
--   Loại 2 (Cắt lề nội bộ):   /upload/e_trim:10/                                    -> /upload/t_trim_10/
--   Loại 3 (Fetch ảnh ngoài): /fetch/f_png,e_trim:20/                              -> /fetch/t_fetch_trim/
-- ============================================================================

BEGIN;

-- 1. Table: users (avatar_url)
UPDATE users
SET avatar_url = REPLACE(
    REPLACE(
        REPLACE(avatar_url, 
            '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
        '/upload/e_trim:10/', '/upload/t_trim_10/'),
    '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
)
WHERE avatar_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR avatar_url LIKE '%/upload/e_trim:10/%'
   OR avatar_url LIKE '%/fetch/f_png,e_trim:20/%';

-- 2. Table: fashion_items (image_url)
UPDATE fashion_items
SET image_url = REPLACE(
    REPLACE(
        REPLACE(image_url, 
            '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
        '/upload/e_trim:10/', '/upload/t_trim_10/'),
    '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
)
WHERE image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR image_url LIKE '%/upload/e_trim:10/%'
   OR image_url LIKE '%/fetch/f_png,e_trim:20/%';

-- 3. Table: outfits (cover_image_url)
UPDATE outfits
SET cover_image_url = REPLACE(
    REPLACE(
        REPLACE(cover_image_url, 
            '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
        '/upload/e_trim:10/', '/upload/t_trim_10/'),
    '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
)
WHERE cover_image_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR cover_image_url LIKE '%/upload/e_trim:10/%'
   OR cover_image_url LIKE '%/fetch/f_png,e_trim:20/%';

-- 4. Table: brands (logo_url & background_url)
UPDATE brands
SET logo_url = REPLACE(
    REPLACE(
        REPLACE(logo_url, 
            '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
        '/upload/e_trim:10/', '/upload/t_trim_10/'),
    '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
)
WHERE logo_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR logo_url LIKE '%/upload/e_trim:10/%'
   OR logo_url LIKE '%/fetch/f_png,e_trim:20/%';

UPDATE brands
SET background_url = REPLACE(
    REPLACE(
        REPLACE(background_url, 
            '/upload/e_background_removal/e_trim:10,f_png,q_auto/', '/upload/t_bg_remove/'),
        '/upload/e_trim:10/', '/upload/t_trim_10/'),
    '/fetch/f_png,e_trim:20/', '/fetch/t_fetch_trim/'
)
WHERE background_url LIKE '%/upload/e_background_removal/e_trim:10,f_png,q_auto/%'
   OR background_url LIKE '%/upload/e_trim:10/%'
   OR background_url LIKE '%/fetch/f_png,e_trim:20/%';

COMMIT;
