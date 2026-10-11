# Seed giá model thủ công (dev tự chạy, không phải migration)

> Dev tự chạy tay trên database của môi trường cần sửa.
> Chạy lại nhiều lần đều an toàn: model đã có thì **gán đè giá mới**,
> chưa có thì thêm mới. Không lỗi trùng, không bỏ qua.

## Vì sao không dùng `ON CONFLICT (provider, model)`

Bảng `ai_model_configs` không có unique trên cặp `(provider, model)`.
Constraint duy nhất là `uq_provider_model_active (provider, model, effective_from)`
(`services/account/database/migrations/20260704062159_init_ai_executor_module.sql:16`),
và bản giá đang hiệu lực được xác định bằng cửa sổ
`effective_from <= now() AND (effective_to IS NULL OR effective_to > now())`
(`pricing_repo_impl.go:GetActiveByProviderAndModel`).
Vì vậy seed dưới đây dùng ngữ nghĩa "gán đè": `UPDATE` dòng đang hiệu lực,
`INSERT` chỉ khi chưa có dòng nào.

## Đoạn SQL (chạy toàn bộ một lần)

```sql
-- manual-seed 028: upsert giá TOÀN BỘ model AI đang dùng (7 model).
-- Giá đối chiếu bảng giá Google AI công bố ngày 11/10/2026 (paid tier, text):
--   gemini-3.5-flash-lite  $0.30 / $2.50  (ai.google.dev pricing + deepmind model card)
--   gemini-3.1-flash-lite  $0.25 / $1.50  (khớp giá trị đang có trong DB)
--   gemini-2.5-flash-lite  $0.10 / $0.40  (khớp giá trị đang có trong DB)
--   gemini-embedding-2     $0.20 input / $0 output (Google chỉ thu input)
--   gemini-embedding-001   $0.15 input / $0 output (Google chỉ thu input)
--   gemma-4-31b-it / gemma-4-26b-a4b-it: tự vận hành nội bộ -> 0 / 0
-- usd_to_vnd giữ 27500 theo quy ước chung của dự án.
-- LƯU Ý: kiểm tra lại giá trên https://ai.google.dev/gemini-api/docs/pricing
-- trước khi chạy, vì Google có thể đổi giá sau ngày đối chiếu trên.
WITH params(provider, model, input_usd_per_million, output_usd_per_million, usd_to_vnd) AS (
  VALUES
    ('google', 'gemini-3.5-flash-lite',  0.30, 2.50, 27500),
    ('google', 'gemini-3.1-flash-lite',  0.25, 1.50, 27500),
    ('google', 'gemini-2.5-flash-lite',  0.10, 0.40, 27500),
    ('google', 'gemini-embedding-2',    0.20, 0,    27500),
    ('google', 'gemini-embedding-001',   0.15, 0,    27500),
    ('google', 'gemma-4-31b-it',        0,    0,    27500),
    ('google', 'gemma-4-26b-a4b-it',    0,    0,    27500)
),
updated AS (
  UPDATE aiexecutor.ai_model_configs AS c
  SET input_usd_per_million  = p.input_usd_per_million,
      output_usd_per_million = p.output_usd_per_million,
      usd_to_vnd             = p.usd_to_vnd
  FROM params AS p
  WHERE c.provider = p.provider
    AND c.model = p.model
    AND c.effective_to IS NULL
  RETURNING c.provider, c.model
)
INSERT INTO aiexecutor.ai_model_configs
  (provider, model, input_usd_per_million, output_usd_per_million, usd_to_vnd)
SELECT p.provider, p.model,
       p.input_usd_per_million, p.output_usd_per_million, p.usd_to_vnd
FROM params AS p
WHERE NOT EXISTS (
  SELECT 1 FROM aiexecutor.ai_model_configs AS c
  WHERE c.provider = p.provider
    AND c.model = p.model
    AND c.effective_to IS NULL
);
```

## Kiểm tra sau khi chạy

```sql
SELECT provider, model, input_usd_per_million, output_usd_per_million, usd_to_vnd
FROM aiexecutor.ai_model_configs
WHERE provider = 'google'
ORDER BY model;
```

Kỳ vọng: đủ 7 model trên, giá đúng như bảng params.
Nếu bảng ở schema khác (môi trường cũ chưa tách schema), đổi tiền tố
`aiexecutor.` cho khớp `search_path` thực tế.
