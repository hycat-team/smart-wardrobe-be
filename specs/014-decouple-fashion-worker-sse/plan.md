# Kế hoạch Triển khai (Implementation Plan): Decouple Fashion Worker & SSE Events

**Branch**: `014-decouple-fashion-worker-sse` | **Date**: 2026-08-26 | **Spec**: [spec.md](./spec.md)

---

## 1. Tóm tắt Kỹ thuật (Summary)

Chuyển đổi kiến trúc xử lý kết quả phân tích AI từ mô hình **Orchestration** (Fashion Worker gọi ngược Contract của Wardrobe/Brand và tự quản lý SSE) sang mô hình **Event Choreography (Event-Driven)**:
- `FashionWorkerUseCase`: Sau khi hoàn thành phân tích AI, chỉ lưu kết quả vào CSDL nội bộ và phát sự kiện `fashion.event.analyzed` qua Transactional Outbox Pattern (`RabbitMQ`).
- `Wardrobe Module` & `Brand Module`: Bổ sung Consumer lắng nghe `fashion.event.analyzed`, tự cập nhật trạng thái thực thể của mình, tự map DTO có nhúng đầy đủ `fashionItem` và tự phát tín hiệu qua `ISSEManager` tới Client.
- Triệt tiêu hoàn toàn **Runtime Call Cycle** và vòng lặp **Read-After-Write**, đồng thời sửa lỗi thiếu trường `fashionItem` trong sự kiện SSE mà không làm thay đổi hợp đồng API phía Client.

---

## 2. Bối cảnh Kỹ thuật (Technical Context)

- **Ngôn ngữ & Runtime**: Go (Golang v1.25.+) với Gin Web Framework.
- **Dependency Injection**: Google Wire.
- **Message Broker & Pattern**: RabbitMQ với Transactional Outbox Pattern (`outbox_records` table).
- **Lưu trữ CSDL**: PostgreSQL với GORM.
- **Realtime**: Server-Sent Events (SSE) với `ISSEManager` (Thread-safe in-memory channel manager).
- **Ranh giới Kiến trúc**: Tuân thủ Modular Monolith và Clean Architecture (§II, §III, §IV trong `constitution.md`).

---

## 3. Đánh giá Tuân thủ Hiến pháp (Constitution Check)

| Nguyên tắc | Đánh giá | Trạng thái |
| :--- | :--- | :--- |
| **§I. Pragmatic Modular Monolith** | Giữ nguyên monolithic deployment, không tạo cross-module DB join. | ✅ Đạt |
| **§II. Strict Module Boundaries & DTO Contracts** | Loại bỏ hoàn toàn việc Fashion Worker phụ thuộc vào Wardrobe/Brand Contract. Giao tiếp chuyển sang DTO Event qua RabbitMQ. | ✅ Đạt |
| **§III. Asynchronous Event-Driven Processing** | Triển khai đúng chuẩn Outbox Pattern cho các side effect giữa các module. | ✅ Đạt |
| **§IV. Clean Architecture & Strict Layer Dependency** | Consumer đặt tại `presentation/worker`, gọi vào UseCase nội bộ, không bypass layer. | ✅ Đạt |

---

## 4. Cấu trúc Dự án & Các File Thay đổi

### Tài liệu Đặc tả (Documentation)
```text
specs/014-decouple-fashion-worker-sse/
├── spec.md                  # Bản đặc tả yêu cầu & phân tích tác động
├── plan.md                  # Kế hoạch triển khai kỹ thuật (file này)
├── research.md              # Quyết định kỹ thuật & lý do lựa chọn
├── data-model.md            # Cấu trúc payload sự kiện và SSE DTO
├── quickstart.md            # Hướng dẫn kiểm thử end-to-end
├── checklists/
│   └── requirements.md      # Checklist thẩm định chất lượng
└── contracts/
    └── event_contracts.md   # Đặc tả Queue, Topic và Event Lifecycle
```

### Mã nguồn Thực tế (Source Code Changes)

```text
internal/
├── shared/
│   └── application/
│       └── constants/
│           └── eventconstants/
│               └── eventconstants.go          # Bổ sung TopicFashionAnalyzed, TopicFashionFailed
├── modules/
│   ├── fashion/
│   │   ├── application/
│   │   │   ├── dto/
│   │   │   │   └── contract_dtos.go          # Thêm FashionAnalyzedEventPayload
│   │   │   └── usecase/
│   │   │       └── worker/
│   │   │           ├── worker_uc.go           # Refactor: Xóa wardrobe/brand contract, xóa sseManager, phát Outbox Event
│   │   │           └── worker_uc_test.go      # Cập nhật unit test theo signature mới
│   │   └── presentation/worker/               # Dọn dẹp DI dependencies
│   ├── wardrobe/
│   │   ├── application/
│   │   │   └── usecase/
│   │   │       └── wardrobe/
│   │   │           └── item/
│   │   │               ├── item_uc_write.go   # Thêm logic xử lý kết quả AI phân tích (Update status & map brief)
│   │   │               └── item_uc_interface.go
│   │   └── presentation/
│   │       └── worker/
│   │           └── wardrobe_analyze_consumer.go # [NEW] Consumer nhận event và phát SSE cho tủ đồ
│   ├── brand/
│   │   ├── application/
│   │   │   └── usecase/
│   │   │       └── item/
│   │   │           └── usecase.go             # Thêm logic xử lý kết quả AI cho brand item
│   │   └── presentation/
│   │       └── worker/
│   │           └── brand_analyze_consumer.go    # [NEW] Consumer nhận event và phát SSE cho Brand Portal
│   └── di/
│       ├── wire.go                            # Cập nhật Wire DI bindings cho các worker/consumer mới
│       └── wire_gen.go                        # Generate lại code DI
```

---

## 5. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

1. **Unit Tests**:
   - Test `FashionWorkerUseCase` đảm bảo khi phân tích xong sẽ lưu DB và gọi `eventPublisher.Publish` đúng event payload.
   - Test `WardrobeAnalyzeConsumer` và `BrandAnalyzeConsumer` kiểm tra việc cập nhật status và publish SSE với payload đầy đủ `FashionItem`.
2. **Integration & Manual Test**:
   - Khởi động server với `make dev`.
   - Chạy kịch bản trong [`quickstart.md`](./quickstart.md) qua `curl` và kiểm tra stream SSE realtime.
