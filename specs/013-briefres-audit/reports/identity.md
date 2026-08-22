# Audit Report — Module Identity (BriefRes)

**Feature**: 013-briefres-audit | **Loại**: Audit/Analysis — KHÔNG sửa code | **Ngày**: 2026-08-22

## Tổng quan

Module `identity` có **10 response struct** tại `internal/modules/identity/application/dto/` (admin_dto.go, auth_res.go, dashboard_dto.go, profile_res.go). Các endpoint user-facing nằm ở `me_handler.go` (5 endpoint) và `auth_handler.go` (12 endpoint); các endpoint admin nằm ở `admin_handler.go` (8 endpoint).

Phát hiện chính:

- **1 struct overdelivery trọng yếu**: `UserRes` (14 field top-level + nhúng `UserSubscriptionRes` 8 field + `UserBodyProfileRes` 6 field → ~26 leaf field). Bị push qua contract `IUserContract` sang module **brand** cho mọi response member/customer/chat, trong khi brand chỉ cần 5–7 field (ID, username, firstName, lastName, gender, avatarUrl, email).
- **Underdelivery**: Trong phạm vi endpoint của identity, KHÔNG có response trả bare UUID FK thiếu nested data (mọi FK đều đi kèm display info — ví dụ `Subscription.PlanID` luôn có `PlanName`/`PlanSlug`). Tuy nhiên tồn tại underdelivery gián tiếp ở consumer (brand): `BrandMemberRes` chỉ nhận được `userFullName`/`userEmail` do brand tự flatten, **thiếu `avatarUrl`** — gốc rễ là contract identity trả `UserRes` quá thô, brand không dùng được trực tiếp.
- **1 struct dead code**: `UserQuotaRes` được định nghĩa nhưng không được dùng bởi bất kỳ usecase/handler nào.
- Các struct admin/dashboard (AdminUserListRes, AdminUserDetailRes, UserSessionItemDTO, UserAdminStatsDTO, CreateAdminUserReq…) theo policy được **giữ nguyên full fields, không đề xuất refactor**.

## Danh sách struct

| # | Struct | File | Số field | Context sử dụng | Kết luận |
|---|--------|------|----------|-----------------|----------|
| 1 | `TokenRes` | auth_res.go:3 | 2 | login/refresh | ✅ OK — không thay đổi |
| 2 | `UserRes` | profile_res.go:13 | 14 (+8+6 nested) | `/api/v1/me*` (4 endpoint), admin (6 endpoint), contract → brand | ⚠️ Overdelivery ở contract & avatar PUT; /me giữ nguyên |
| 3 | `UserSubscriptionRes` | profile_res.go:35 | 8 | nested trong UserRes | ⚠️ Hợp lệ ở profile, thừa ở contract (giải quyết qua UserBriefRes) |
| 4 | `UserBodyProfileRes` (+ `UserBodyMeasurementsRes`, `UserInferredBodyRes`) | profile_res.go:58 | 6+3+2 | nested trong UserRes | ⚠️ Chỉ cần ở context body-profile, đi kèm mọi UserRes là thừa |
| 5 | `UserQuotaRes` | profile_res.go:46 | 3 | — (không dùng) | 🗑️ Dead code — đề xuất xóa |
| 6 | `UserStyleProfileRes` | profile_res.go:52 | 3 | contract GetStyleProfile → fashion AI | ✅ Nội bộ AI, không phải display — giữ nguyên |
| 7 | `AdminUserListRes` (= `PaginationResult[*UserRes]`) | admin_dto.go:20 | — | admin GET /users | ✅ Admin — không refactor |
| 8 | `AdminUserDetailRes` | admin_dto.go:37 | 2 (User + Subscription `any`) | admin GET /users/{id} | ✅ Admin — không refactor |
| 9 | `UserSessionItemDTO` | admin_dto.go:42 | 8 | admin GET sessions | ✅ Admin — không refactor |
| 10 | `UserAdminStatsDTO` | dashboard_dto.go:5 | 4 | dashboard stats | ✅ Admin/dashboard — không refactor |

## Overdelivery

### 1. `UserRes` (profile_res.go:13) — mức **cao** (ở contract), **trung bình** (ở avatar PUT)

Struct hiện tại (14 field): `ID, Username, Email, RoleSlug, FirstName, LastName, DateOfBirth, Address, Gender, Status, CreatedAt, Subscription, BodyProfile, AvatarUrl`.

| Context | Fields giữ | Fields bỏ (đề xuất) |
|---------|-----------|---------------------|
| **Contract → brand** (member/customer/chat display) | `ID, Username, FirstName, LastName, Gender, AvatarUrl, Email` | `RoleSlug, DateOfBirth, Address, Status, CreatedAt, Subscription, BodyProfile` |
| **PUT /api/v1/me/avatar** (client vừa set avatar, chỉ cần xác nhận) | `ID, FirstName, LastName, AvatarUrl` (hoặc UserBriefRes) | `Email, RoleSlug, DateOfBirth, Address, Gender, Status, CreatedAt, Subscription, BodyProfile` |
| **GET /api/v1/me** (trang profile đầy đủ) | Toàn bộ | Không bỏ (context hợp lệ — giữ nguyên) |

Lý do: `MapToUserRes` (user_mapper.go:11) fetch cả subscription + body profile cho **mọi** lần gọi contract (`GetByID`, `GetByIDs`, `GetByUsername`, `GetByUsernamesOrEmails`). Brand module chỉ đọc 5–7 field nhưng nhận nguyên 26 leaf field → lãng phí payload trên mọi response member/customer/chat.

### 2. `UserBodyProfileRes` (profile_res.go:58) — mức **trung bình**

6 field + 2 struct con (Measurements 3 field, InferredByAI 2 field). Đi kèm mọi `UserRes` (kể cả response `PUT /api/v1/me/avatar` chỉ để xác nhận avatar). Chỉ thực sự cần ở trang profile/body-profile.

- Giữ: khi trả trong `/api/v1/me` và `/api/v1/me/body-profile`.
- Bỏ: khỏi `UserRes` dùng cho contract & response avatar PUT (không thuộc `UserBriefRes`).

### 3. `UserSubscriptionRes` (profile_res.go:35) — mức **trung bình** (được giải quyết kèm UserBriefRes)

8 field (PlanID, PlanName, PlanSlug, ExpiresAt, MaxWardrobeItems, MaxOutfits, AiOutfitDailyQuota, AiChatDailyQuota). Hợp lệ ở trang profile (widget quota), nhưng bị đẩy sang brand qua contract mà brand không dùng field nào.

### 4. `UserQuotaRes` (profile_res.go:46) — mức **thấp**

Dead code: không được tham chiếu bởi handler/usecase/mapper nào. Đề xuất xóa (hoặc wiring nếu có kế hoạch expose quota riêng).

## Underdelivery

- **Trong phạm vi endpoint identity**: không phát hiện trường hợp trả bare UUID FK thiếu nested data. Mọi FK đi kèm display info đầy đủ (ví dụ `UserRes.Subscription.PlanID` luôn có `PlanName`/`PlanSlug`; `UserSessionItemDTO` không cần nested entity).
- **Underdelivery gián tiếp (consumer brand)**: gốc rễ từ contract identity. `IUserContract` chỉ có method trả `*UserRes` (struct quá lớn, khó nhúng), nên brand tự flatten thành field chuỗi `UserFullName`/`UserEmail` trong `BrandMemberRes` (brand.go:95-96) — **thiếu `avatarUrl`**, buộc frontend không hiển thị được avatar member/customer nếu không gọi API khác. Cách xử lý từ phía identity: cung cấp `UserBriefRes` (có `avatarUrl`) cho contract thay vì để brand tự map.
- `UserStyleProfileRes` trả `UserID` thuần + embedding — đây là context AI nội bộ, không phục vụ frontend display → **không tính underdelivery**.
- `AdminUserDetailRes.Subscription` là field `any` (weak type) — thuộc admin, chỉ ghi nhận, không đề xuất.

## Đề xuất BriefRes

### `UserBriefRes` (mới, đặt tại `identity/application/dto/profile_res.go`)

```go
type UserBriefRes struct {
	ID        uuid.UUID     `json:"id"`
	Username  string        `json:"username"`
	Email     string        `json:"email,omitempty"`
	FirstName string        `json:"firstName"`
	LastName  string        `json:"lastName,omitempty"`
	Gender    gender.Gender `json:"gender"`
	AvatarUrl *string       `json:"avatarUrl,omitempty"`
}
```

Lý do chọn 7 field này:

- Khớp **chính xác** với struct `BrandCustomerUserRes` mà brand đang tự duy trì (brand/application/dto/customer.go:36) — brand có thể bỏ struct thủ công và dùng chung `UserBriefRes`.
- Đủ cho mọi context hiển thị member/customer/chat (tên, email, avatar).
- `avatarUrl` được đưa vào để vá lỗi underdelivery của `BrandMemberRes`.

**Thay đổi kèm theo (phía identity)**: bổ sung hoặc đổi method contract để các lookup display trả `*UserBriefRes`/`[]*UserBriefRes`:

- `IUserContract.GetByID / GetByIDs / GetByUsername / GetByUsernamesOrEmails` → trả `*UserBriefRes` (breaking change theo policy) — hoặc thêm method mới `GetBriefByIDs` giữ nguyên method cũ nếu admin vẫn cần `UserRes` qua contract.
- `MapToUserBrief` (mapper mới trong `application/mapper/user_mapper.go`) chuyển `*entities.User` → `*UserBriefRes` (không cần fetch subscription/body profile).

## Ưu tiên

| Ưu tiên | Mục | Lý do |
|---------|-----|-------|
| **Cao** | Tạo `UserBriefRes` + đổi contract `IUserContract` các method display về `UserBriefRes` | Giảm payload lớn (26 → 7 leaf field) trên contract xuyên module; ảnh hưởng hàng loạt endpoint brand (member, customer, chat); đồng thời vá underdelivery `avatarUrl`. Cần quyết định sớm vì là breaking change. |
| **Trung bình** | `PUT /api/v1/me/avatar` trả `UserBriefRes` thay vì `UserRes` | Client vừa upload avatar, không cần toàn bộ profile (kèm subscription + body profile) trong response. Giảm payload rõ rệt trên endpoint gọi thường xuyên. |
| **Trung bình** | Tách `UserBodyProfileRes`/`UserSubscriptionRes` khỏi `UserRes` cho context không cần (contract, avatar PUT) | Giảm nested depth & payload trên các response không phải trang profile. |
| **Thấp** | Xóa `UserQuotaRes` (dead code) | Không có consumer; giữ lại chỉ tăng bảo trì. |
| **Thấp** | `UserStyleProfileRes` — giữ nguyên | Internal AI contract, không phải display; không nằm trong scope BriefRes. |

## API bị ảnh hưởng

### Endpoint thuộc identity module

| Endpoint | Response hiện tại | Response đề xuất |
|----------|-------------------|------------------|
| `GET /api/v1/me` | `UserRes` (full) | Giữ nguyên `UserRes` — context trang profile hợp lệ (không refactor) |
| `PUT /api/v1/me` | `UserRes` (full) | Giữ nguyên `UserRes` (không refactor) |
| `PUT /api/v1/me/body-profile` | `UserRes` (full) | Giữ nguyên hoặc trả riêng `UserBodyProfileRes` (tùy chọn — trung bình) |
| `PUT /api/v1/me/avatar` | `UserRes` (full) | `UserBriefRes` (trung bình) |
| `POST /api/v1/auth/login` | `TokenRes` (qua cookie) | Không đổi |
| `POST /api/v1/auth/refresh-token` | `TokenRes` (qua cookie) | Không đổi |
| Các endpoint admin `/api/v1/admin/users*` | `UserRes` / `AdminUserListRes` / `AdminUserDetailRes` / `UserSessionItemDTO` | **Không đề xuất refactor** (giữ full fields — policy admin) |

### Endpoint downstream (module brand — ghi nhận để phối hợp, không thuộc scope identity)

| Endpoint (brand) | Response hiện tại (dữ liệu user nhận từ contract identity) | Sau khi đổi contract sang `UserBriefRes` |
|------------------|-------------------------------------------------------------|------------------------------------------|
| Brand member list/detail, customer list/detail, chat conversations/messages, brand admin | `UserRes` (26 leaf field) bị brand tự flatten → `userFullName`/`userEmail` (thiếu `avatarUrl`) | Nhúng `UserBriefRes` (7 field, có `avatarUrl`) — frontend hiển thị avatar được ngay, không cần gọi thêm API |

## Open Questions (đề xuất đưa lên plan)

1. **Breaking change contract**: đổi trực tiếp signature `IUserContract.GetByID` về `*UserBriefRes` (breaking, theo policy) hay thêm method mới `GetBriefByIDs` giữ `UserRes` cho các context cần full (ví dụ brand admin đang dùng `UserRes` để hiển thị thêm email/status)? → ảnh hưởng tới scope sửa brand module.
2. **`/api/v1/me/avatar`**: đồng ý trả `UserBriefRes` hay giữ `UserRes` cho nhất quán client?
3. **`UserQuotaRes`**: xóa hay giữ chờ feature expose quota?

---

*File này là tài liệu audit/analysis — không thay đổi code hay database schema. Không tạo task.*