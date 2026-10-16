# Ownership Contract: Ai sở hữu schema nào (030)

**Spec**: [../spec.md](../spec.md) | **Data model**: [../data-model.md](../data-model.md)

Hợp đồng này là căn cứ duy nhất để gate FR-009判定 đỏ/xanh. Mọi thay đổi cấu trúc database sau thời điểm bàn giao MUST tuân thủ bảng dưới đây.

## 1. Canonical ownership map

| Schema | Owner service | Migration tree |
|---|---|---|
| `identity`, `subscription` | account | `services/account/database/migrations/` |
| `wardrobe`, `fashion`, `brand`, `dashboard`, `community` | commerce | `services/commerce/database/migrations/` |
| `aiexecutor` | ai | `services/ai/database/migrations/` |
| `shared`, `public` | infra chung (account giữ tạm) | `services/account/database/migrations/` |

## 2. Gate rules (FR-009)

- **G-ACCT**: cây Account MUST NOT chứa `ALTER/CREATE/DROP` trên `aiexecutor.|wardrobe.|fashion.|brand.|dashboard.|community.`. Ngoại lệ: `REFERENCES identity.users(` (FK user xuyên schema được phép).
- **G-COMM**: cây Commerce MUST NOT chứa thay đổi trên `identity.|subscription.|aiexecutor.`. Ngoại lệ: `REFERENCES identity.users(`.
- **G-AI**: cây AI MUST NOT chứa thay đổi ngoài `aiexecutor.`. Không ngoại lệ.

Mẫu thử đỏ tối thiểu cho mỗi luật (dùng ở quickstart V3): 1 file migration chứa đúng 1 câu vi phạm → gate phải đỏ và nêu tên luật + file + dòng.

## 3. Điều khoản chuyển tiếp (có hiệu lực tới khi spec đóng)

- 11 file Account bị viết lại nội dung (R2 trong research) được miễn G-ACCT **cho đúng các version đó**, vì nội dung mới đã gỡ DDL ngoài chủ. Mọi file mới vẫn chịu gate đầy đủ.
- Baseline mới ở cây AI/Commerce được miễn kiểm tra "trùng định nghĩa với lịch sử cũ" — tính đúng đắn của chúng do convergence audit (SC-002) chứng minh, không do gate.
- Lệnh ngừng bắn FR-011 (quy ước thủ công) hết hiệu lực khi spec đóng; 3 luật trên có hiệu lực vĩnh viễn từ lúc gate merge.

## 4. Versioning của hợp đồng này

Mọi sửa đổi map ở §1 sau spec này (ví dụ chuyển `shared` cho service khác, thêm schema mới) MUST đi qua 1 spec riêng có ADR, không sửa lặng lẽ. Gate đọc map từ file này (single source of truth) thay vì hardcode danh sách schema trong script.
