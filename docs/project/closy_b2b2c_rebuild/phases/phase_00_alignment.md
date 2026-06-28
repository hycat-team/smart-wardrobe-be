# Phase 00 - Alignment and Repository Audit

## Mục tiêu

Chuẩn bị trước khi code. Phase này không tạo feature mới. Agent phải xác minh repo hiện tại, schema hiện tại, auth flow hiện tại, migration tool hiện tại, và các module đang tồn tại để tránh code lệch.

## Không làm trong phase này

```text
- Không tạo migration mới.
- Không drop table.
- Không move code module lớn.
- Không tạo endpoint mới.
- Không đổi auth behavior.
```

## Việc cần làm

### Audit repo structure

Xác định các folder hiện có:

```text
internal/modules/identity hoặc internal/identity
internal/modules/wardrobe hoặc internal/wardrobe
internal/modules/subscription hoặc internal/subscription
internal/shared/domain/entities
internal/shared/infrastructure hoặc internal/infrastructure
migrations hoặc internal/infrastructure/database/migrations
```

Nếu repo đang dùng structure khác, giữ structure hiện tại và map phase plan vào structure đó. Không tự tạo structure song song gây trùng module.

### Audit schema hiện tại

Xác minh các bảng hiện có:

```text
users
refresh_tokens
wardrobe_items
outfits
outfit_items
categories
user_style_profiles
ai quota/subscription/payment tables
community/resale tables nếu còn
```

Ghi lại field thật của:

```text
users
wardrobe_items
outfit_items
```

Đặc biệt kiểm tra:

```text
users.email nullable hay NOT NULL
users.password_hash nullable hay NOT NULL
users.status hiện là type gì
wardrobe_items.embedding dimension bao nhiêu
outfit_items primary key hiện tại là gì
current migration tool là goose embedded hay cách khác
```

### Audit auth/OTP hiện tại

Xác minh OTP đang lưu Redis ở đâu và flow hiện tại tên gì.

Không tạo table OTP.

Cần biết:

```text
- register hiện tại dùng email/password hay phone/OTP?
- login hiện tại dùng email/password hay OTP?
- forgot-password hiện tại dùng email OTP hay cơ chế khác?
- reset-password flow hiện tại gồm endpoint nào?
```

Nếu không rõ, dừng và báo lại.

### Audit AI outfit endpoint hiện tại

Tìm endpoint hiện tại cho outfit recommendation:

```text
POST /api/v1/ai/outfit-recommendations
```

Nếu endpoint khác, ghi lại endpoint thật. Phase sau sẽ mở rộng endpoint hiện tại, không tạo endpoint song song nếu không cần.

### Audit search/vector/Elastic flow

Tìm nơi đang search trên `wardrobe_items`:

```text
- SQL lexical search
- pgvector / HNSW query
- Elasticsearch indexing/sync worker
- AI context builder
```

Ghi lại file và query để Phase 03 cập nhật sang `fashion_items`.

## Output cần tạo sau phase này

Agent phải tạo một file audit nội bộ, ví dụ:

```text
docs/project/closy_b2b2c_rebuild/repo_audit_before_rebuild.md
```

Nội dung gồm:

```text
- Current module structure
- Current migration tool
- Current users schema
- Current wardrobe_items schema
- Current outfit_items schema
- Current auth/OTP flow summary
- Current AI outfit route summary
- Current search/vector/Elastic summary
- Risks / unknowns
```

## Acceptance checklist

- [ ] Đã xác minh current schema thật.
- [ ] Đã xác minh migration tool thật.
- [ ] Đã xác minh OTP dùng Redis, không cần table OTP.
- [ ] Đã xác minh current auth endpoints.
- [ ] Đã xác minh current outfit recommendation endpoint.
- [ ] Đã xác minh nơi code đang đọc `wardrobe_items` metadata.
- [ ] Chưa thay đổi behavior production.

## Lỗi cần tránh

- Không dựa hoàn toàn vào report để code nếu schema thật khác.
- Không tạo table OTP.
- Không tạo module mới ngoài 5 module đã chốt.
- Không tạo endpoint AI outfit mới nếu endpoint cũ đã có.
