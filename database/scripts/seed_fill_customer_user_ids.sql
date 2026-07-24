-- ================================================================
-- Seed: Tạo users và gán user_id cho brand_customers đang thiếu
-- Chạy sau seed_feature_006_dashboard.sql
-- Password: 123456 (giống user 'user')
-- ================================================================

BEGIN;

DO $$
DECLARE
    cust RECORD;
    new_uid UUID;
    uname TEXT;
    uemail TEXT;
    idx INT := 0;
BEGIN
    FOR cust IN
        SELECT bc.id, bc.customer_name, bc.phone_e164, bc.brand_id
        FROM brand_customers bc
        WHERE bc.user_id IS NULL
        ORDER BY bc.brand_id, bc.phone_e164
    LOOP
        idx := idx + 1;
        uname := 'fb_cust_' || idx || '_' || REPLACE(cust.phone_e164, '+', '');
        uemail := uname || '@closy.local';

        SELECT id INTO new_uid FROM users WHERE LOWER(username) = LOWER(uname);
        IF NOT FOUND THEN
            INSERT INTO users (id, username, email, password_hash, first_name, last_name, role_slug, status, is_deleted, created_at, updated_at)
            VALUES (
                gen_random_uuid(),
                uname,
                uemail,
                '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
                SPLIT_PART(COALESCE(cust.customer_name, 'Customer'), ' ', 1),
                COALESCE(NULLIF(SPLIT_PART(COALESCE(cust.customer_name, 'Customer'), ' ', 2), ''), SPLIT_PART(COALESCE(cust.customer_name, 'Customer'), ' ', 1)),
                'user',
                0,
                false,
                NOW(),
                NOW()
            )
            RETURNING id INTO new_uid;
        END IF;

        UPDATE brand_customers
        SET user_id = new_uid, updated_at = NOW()
        WHERE id = cust.id;
    END LOOP;

    RAISE NOTICE 'Đã xử lý % brand_customers', idx;
END $$;

COMMIT;
