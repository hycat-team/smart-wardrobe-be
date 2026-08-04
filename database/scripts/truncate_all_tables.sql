-- =============================================================
-- TRUNCATE toàn bộ bảng data trong DB (postgres).
-- Quét mọi schema module (identity, wardrobe, fashion, brand,
-- subscription, aiexecutor, dashboard, shared, public) và
-- TRUNCATE ... CASCADE để xử lý FK xuyên schema.
-- RESTART IDENTITY để reset lại auto-increment về 1.
-- =============================================================

-- -------------------------------------------------------------
-- SCRIPT 1: Truncate TẤT CẢ bảng, GIỮ LẠI goose_db_version.
-- Bảng version migration được giữ nguyên nên goose không chạy
-- lại migration. Sau đó chạy seed để nạp dữ liệu mẫu.
-- -------------------------------------------------------------
DO $$
DECLARE
    tbls text;
BEGIN
    SELECT string_agg(format('%I.%I', schemaname, tablename), ', ')
    INTO tbls
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
      AND schemaname NOT LIKE 'pg\_%'
      AND NOT (schemaname = 'public' AND tablename = 'goose_db_version');

    IF tbls IS NOT NULL THEN
        EXECUTE 'TRUNCATE TABLE ' || tbls || ' RESTART IDENTITY CASCADE';
    END IF;
END $$;

-- -------------------------------------------------------------
-- SCRIPT 2: Truncate TẤT CẢ bảng, GỒM CẢ goose_db_version.
-- Xóa bảng version migration -> goose sẽ coi như CHƯA có
-- migration nào và chạy lại toàn bộ từ đầu (sẽ fail nếu
-- migration không idempotent). CHỈ dùng khi cần rebuild từ 0.
-- -------------------------------------------------------------
DO $$
DECLARE
    tbls text;
BEGIN
    SELECT string_agg(format('%I.%I', schemaname, tablename), ', ')
    INTO tbls
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
      AND schemaname NOT LIKE 'pg\_%';

    IF tbls IS NOT NULL THEN
        EXECUTE 'TRUNCATE TABLE ' || tbls || ' RESTART IDENTITY CASCADE';
    END IF;
END $$;
