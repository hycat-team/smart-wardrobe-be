-- ============================================================================
-- SCRIPT ROLLBACK: KHÔI PHỤC CÁC URL CLOUDINARY VỀ CẤU TRÚC TAG CŨ
-- (Đã cập nhật theo Database Schema mới nhất: smart_wardrobe_db-schema.sql)
-- ============================================================================
-- Quy tắc đảo ngược:
--   Loại 1 (Xóa nền):         /upload/t_bg_remove/  -> /upload/e_background_removal/e_trim:10,f_png,q_auto/
--   Loại 2 (Cắt lề nội bộ):   /upload/t_trim_10/    -> /upload/e_trim:10/
--   Loại 3 (Fetch ảnh ngoài): /fetch/t_fetch_trim/  -> /fetch/f_png,e_trim:20/
-- ============================================================================

BEGIN;

-- 1. Table: users (avatar_url)
UPDATE users
SET avatar_url = REPLACE(
    REPLACE(
        REPLACE(avatar_url, 
            '/upload/t_bg_remove/', '/upload/e_background_removal/e_trim:10,f_png,q_auto/'),
        '/upload/t_trim_10/', '/upload/e_trim:10/'),
    '/fetch/t_fetch_trim/', '/fetch/f_png,e_trim:20/'
)
WHERE avatar_url LIKE '%/upload/t_bg_remove/%'
   OR avatar_url LIKE '%/upload/t_trim_10/%'
   OR avatar_url LIKE '%/fetch/t_fetch_trim/%';

-- 2. Table: fashion_items (image_url)
UPDATE fashion_items
SET image_url = REPLACE(
    REPLACE(
        REPLACE(image_url, 
            '/upload/t_bg_remove/', '/upload/e_background_removal/e_trim:10,f_png,q_auto/'),
        '/upload/t_trim_10/', '/upload/e_trim:10/'),
    '/fetch/t_fetch_trim/', '/fetch/f_png,e_trim:20/'
)
WHERE image_url LIKE '%/upload/t_bg_remove/%'
   OR image_url LIKE '%/upload/t_trim_10/%'
   OR image_url LIKE '%/fetch/t_fetch_trim/%';

-- 3. Table: outfits (cover_image_url)
UPDATE outfits
SET cover_image_url = REPLACE(
    REPLACE(
        REPLACE(cover_image_url, 
            '/upload/t_bg_remove/', '/upload/e_background_removal/e_trim:10,f_png,q_auto/'),
        '/upload/t_trim_10/', '/upload/e_trim:10/'),
    '/fetch/t_fetch_trim/', '/fetch/f_png,e_trim:20/'
)
WHERE cover_image_url LIKE '%/upload/t_bg_remove/%'
   OR cover_image_url LIKE '%/upload/t_trim_10/%'
   OR cover_image_url LIKE '%/fetch/t_fetch_trim/%';

-- 4. Table: brands (logo_url & background_url)
UPDATE brands
SET logo_url = REPLACE(
    REPLACE(
        REPLACE(logo_url, 
            '/upload/t_bg_remove/', '/upload/e_background_removal/e_trim:10,f_png,q_auto/'),
        '/upload/t_trim_10/', '/upload/e_trim:10/'),
    '/fetch/t_fetch_trim/', '/fetch/f_png,e_trim:20/'
)
WHERE logo_url LIKE '%/upload/t_bg_remove/%'
   OR logo_url LIKE '%/upload/t_trim_10/%'
   OR logo_url LIKE '%/fetch/t_fetch_trim/%';

UPDATE brands
SET background_url = REPLACE(
    REPLACE(
        REPLACE(background_url, 
            '/upload/t_bg_remove/', '/upload/e_background_removal/e_trim:10,f_png,q_auto/'),
        '/upload/t_trim_10/', '/upload/e_trim:10/'),
    '/fetch/t_fetch_trim/', '/fetch/f_png,e_trim:20/'
)
WHERE background_url LIKE '%/upload/t_bg_remove/%'
   OR background_url LIKE '%/upload/t_trim_10/%'
   OR background_url LIKE '%/fetch/t_fetch_trim/%';

COMMIT;
