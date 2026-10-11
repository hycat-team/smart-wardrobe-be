# Deferred: Chi phí AI thật phục vụ báo cáo lợi nhuận (028)

> Ghi nợ ngoài phạm vi phát hiện trong lúc làm feature (hiến pháp §Governance).
> Kết luận đợt này: **không có debt nào cần defer.**

| Phát hiện | Vì sao ngoài phạm vi | Điều gì gỡ được | Trạng thái |
|---|---|---|---|
| Tin đồn `gemini-embedding-001` sunset 14/07/2026 | Thuộc ops/config, không thuộc báo cáo lợi nhuận | Không cần — đã verify live ngày 11/10/2026: `batchEmbedContents` với key production trả HTTP 200 + vector, model còn sống (xem T021 trong `tasks.md`) | Không phải debt, đóng tại chỗ |
| Giá tính theo model primary khi provider fallback thắng | Đã ghiเป็น ngoại lệ đã biết trong FR-001 + `research.md` R7, hành vi có chủ ý | Nếu muốn chính xác tuyệt đối: ghi lại provider thắng qua `executeWithFallback` (thay đổi interface nội bộ shared AI) — đề xuất thành feature riêng nếu bao giờ cần | Không phải debt của đợt này, đã tài liệu hóa |
| `countTokens` cho embedding model nếu Google hỗ trợ sau này | Đã ghi điểm bật lại trong R3 (đúng 1 điểm ở `GenerateEmbeddings`) | Google mở `countTokens` cho embedding → bật lại attempt + test | Không phải debt, là roadmap có điều kiện |

Không có mục nào làm mất hiệu lực bằng chứng của task đã tick.
