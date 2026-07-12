# Phase 05e - Brand Chat

## Mục tiêu

Tạo chat MVP giữa user và brand để phục vụ customer service. Đây là chat trực tiếp đơn giản, không phải support ticket workflow.

## Không làm trong phase này

```text
- Không tạo support_tickets.
- Không tạo return_exchange_requests.
- Không tạo marketing broadcast.
- Không tạo chat cho offline brand_customer chưa linked tài khoản Closy.
- Không cho brand staff xem AI chat riêng của user.
```

## Schema target

### brand_conversations

```text
id UUID PK
brand_id UUID FK brands(id)
user_id UUID FK users(id)
status VARCHAR(50)
last_message_at TIMESTAMP NULL
created_at
updated_at
unique(brand_id, user_id)
```

Status:

```text
OPEN
CLOSED
```

### brand_conversation_messages

```text
id UUID PK
conversation_id UUID FK brand_conversations(id)
sender_user_id UUID FK users(id) NULL
sender_role VARCHAR(50)
message TEXT
created_at
```

Sender role:

```text
CUSTOMER
BRAND_STAFF
SYSTEM
```

`sender_user_id` có thể nullable cho SYSTEM message nếu codebase muốn.

## Rules

User-facing chat:

```text
- User must be ACTIVE.
- Brand must be ACTIVE.
- User should be brand_customer ACTIVE or conversation creation should create/join brand_customer depending business.
- MVP: require brand_customer ACTIVE to chat.
```

Brand Portal chat:

```text
- Staff must be brand_member ACTIVE.
- Staff role OWNER/MANAGER/SUPPORT_STAFF can reply.
- MARKETER cannot reply unless business allows.
```

Conversation visibility:

```text
- Brand staff only sees conversations for their brand.
- User only sees own conversation.
```

Message creation:

```text
- insert message
- update brand_conversations.last_message_at
- if conversation CLOSED and new message sent, either reopen or reject. MVP: reopen to OPEN.
```

## APIs

User-facing:

```text
GET /api/v1/brands/:brandId/conversation
POST /api/v1/brands/:brandId/conversation/messages
```

Brand portal:

```text
GET /api/v1/brand-portal/brands/:brandId/conversations
GET /api/v1/brand-portal/conversations/:conversationId/messages
POST /api/v1/brand-portal/conversations/:conversationId/messages
```

## Response DTO

Conversation list item:

```json
{
  "id": "uuid",
  "brand_id": "uuid",
  "user_id": "uuid",
  "customer_name": "nullable",
  "user_display_name": "nullable",
  "status": "OPEN",
  "last_message_at": "timestamp"
}
```

Message:

```json
{
  "id": "uuid",
  "conversation_id": "uuid",
  "sender_role": "CUSTOMER",
  "sender_user_id": "uuid",
  "message": "text",
  "created_at": "timestamp"
}
```

## Tests

- ACTIVE user can open conversation with brand where they are brand_customer ACTIVE.
- Offline brand_customer chưa linked Closy account không thể dùng chat trong app.
- Staff from other brand cannot view conversation.
- Disabled staff cannot reply.
- Sending message updates last_message_at.
- Closed conversation reopens or rejects based on implemented MVP rule.

## Acceptance checklist

- [ ] Chat tables created.
- [ ] User chat requires ACTIVE user.
- [ ] Brand portal chat checks brand_members.
- [ ] No support_tickets.
- [ ] No marketing broadcast.
- [ ] No access to AI chat private data.
