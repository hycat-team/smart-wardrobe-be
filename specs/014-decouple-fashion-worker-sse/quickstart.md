# Hướng dẫn Xác minh Nhanh (Quickstart Validation Guide)

**Feature**: [spec.md](./spec.md) | **Date**: 2026-08-26

Tài liệu này cung cấp các kịch bản chạy thử nghiệm để kiểm chứng toàn bộ quy trình Event-Driven và phát sự kiện SSE sau khi refactor.

---

## 1. Chuẩn bị Môi trường (Prerequisites)

1. Khởi động Docker containers (PostgreSQL, Redis, RabbitMQ):
   ```bash
   make docker-up
   ```
2. Khởi động backend server và các background worker:
   ```bash
   make dev
   ```

---

## 2. Kịch bản Kiểm thử End-to-End: Tải lên Hàng loạt Tủ đồ Cá nhân (B2C)

### Bước 1: Mở luồng lắng nghe SSE từ Client
Sử dụng công cụ `curl` để mở kết nối stream:
```bash
curl -N -H "Authorization: Bearer <USER_TOKEN>" \
  http://localhost:8080/api/v1/wardrobe-items/tasks/<TEST_TASK_ID>/sse
```

### Bước 2: Gửi yêu cầu tải lên hàng loạt ảnh
Gửi request batch upload với cùng `taskId`:
```bash
curl -X POST http://localhost:8080/api/v1/wardrobe-items/batch-upload \
  -H "Authorization: Bearer <USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "imageUrl": "https://res.cloudinary.com/.../shirt.jpg",
        "imagePublicId": "shirt_01"
      }
    ]
  }'
```

### Bước 3: Quan sát đầu ra SSE
Kết nối SSE ở Bước 1 phải nhận được sự kiện:
```text
event: message
data: {"itemId":"...","status":"completed","total":1,"index":1,"data":{"id":"...","price":null,"status":"in_wardrobe","fashionItem":{"id":"...","imageUrl":"...","color":"...","colorHex":"...","style":"...","category":{"id":"...","name":"...","slug":"..."}}}}
```
* **Tiêu chí đạt**: Trường `fashionItem` trong `data` có đầy đủ thông tin màu sắc, danh mục và ảnh, không bị `null`.

---

## 3. Kịch bản Kiểm thử Tự động (Automated Unit & Integration Tests)

Chạy bộ unit test của các worker và consumer:
```bash
go test -v -race ./internal/modules/fashion/application/usecase/worker/...
go test -v -race ./internal/modules/wardrobe/...
go test -v -race ./internal/modules/brand/...
```
