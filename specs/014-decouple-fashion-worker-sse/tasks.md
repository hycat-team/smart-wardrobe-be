# Danh sách Công việc (Tasks): Refactor Decouple Fashion Worker & SSE Events

**Input**: Tài liệu thiết kế từ `/specs/014-decouple-fashion-worker-sse/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/event_contracts.md](./contracts/event_contracts.md)

---

## Danh sách Task Thực thi

### Phase 1: Setup (Hằng số Sự kiện & Khai báo DTO Chuẩn)

**Mục tiêu**: Chuẩn bị sẵn sàng các Topics, Routing Keys và DTO Event Payload dùng chung giữa các module.

- [X] T001 Khai báo `TopicFashionAnalyzed` và `TopicFashionFailed` trong `internal/shared/application/constants/eventconstants/eventconstants.go`
- [X] T002 Cấu hình `RoutingKeyFashionAnalyzed` và `RoutingKeyFashionFailed` trong `internal/shared/infrastructure/messaging/rabbitmq_constants.go`
- [X] T003 [P] Định nghĩa struct `FashionAnalyzedEventPayload` trong `internal/modules/fashion/application/dto/contract_dtos.go`

---

### Phase 2: Foundational (Tách biệt Fashion AI Worker)

**Mục tiêu**: Loại bỏ hoàn toàn sự phụ thuộc của `FashionWorkerUseCase` vào `WardrobeContract`, `BrandContract` và `sseManager`.

**⚠️ ĐIỀU KIỆN TIÊN QUYẾT**: Phải hoàn thành giai đoạn này trước khi triển khai các Consumer của User Story 1 & 2.

- [X] T004 Cập nhật `FashionWorkerUseCase` struct, xóa bỏ các field `wardrobeContract`, `brandContract`, `sseManager` trong `internal/modules/fashion/application/usecase/worker/worker_uc.go`
- [X] T005 Chuyển đổi logic hoàn tất phân tích AI sang phát Domain Event `TopicFashionAnalyzed` / `TopicFashionFailed` qua Transactional Outbox Pattern (`uow.Execute` + `eventPublisher.Publish`) trong `internal/modules/fashion/application/usecase/worker/worker_uc.go`
- [X] T006 Xóa bỏ các hàm mapping mượn tạm `mapWardrobeItemToBrief` và `mapBrandItemToBrief` khỏi `internal/modules/fashion/application/usecase/worker/worker_uc.go`
- [X] T007 Cập nhật constructor và unit test của worker tương ứng trong `internal/modules/fashion/application/usecase/worker/worker_uc_test.go`

**Checkpoint**: `FashionWorkerUseCase` đã trở thành Pure Worker độc lập, chỉ ghi nhận vào CSDL `fashion_items` và phát ra event.

---

### Phase 3: User Story 1 - Xử lý Sự kiện & Phát SSE Tủ đồ Cá nhân (Priority: P1) 🎯 MVP

**Mục tiêu**: Module `wardrobe` tự tiếp nhận sự kiện phân tích AI, cập nhật trạng thái tủ đồ và phát SSE cho Client B2C với đầy đủ trường `fashionItem`.

**Independent Test**: Gửi yêu cầu batch upload vào tủ đồ cá nhân, mở stream SSE `/api/v1/wardrobe-items/tasks/{taskId}/sse` và xác nhận nhận được event `completed` chứa `data.fashionItem` hợp lệ.

- [X] T008 [US1] Thêm phương thức xử lý kết quả phân tích AI `HandleFashionAnalyzedResult` trong `internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go`
- [X] T009 [US1] Khai báo phương thức `HandleFashionAnalyzed` trong interface `internal/modules/wardrobe/application/interface/usecase/wardrobe_uc.go`
- [X] T010 [US1] Tạo `WardrobeFashionEventWorker` lắng nghe queue `wardrobe_fashion_analyzed_queue`, xử lý message và gọi use case trong `internal/modules/wardrobe/presentation/worker/fashion_event_worker.go`
- [X] T011 [US1] Cấu hình phát SSE qua `sseManager.Publish` trong `WardrobeFashionEventUseCase`, đảm bảo tính Idempotency (bỏ qua UPDATE DB nếu item đã xong nhưng vẫn phát SSE nếu channel mở) trong `internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go`
- [X] T012 [US1] Viết unit test cho `WardrobeFashionEventUseCase` trong `internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc_test.go`

**Checkpoint**: Hoàn tất MVP! Luồng tải lên tủ đồ cá nhân B2C hoạt động hoàn toàn độc lập qua Event-Driven.

---

### Phase 4: User Story 2 - Xử lý Sự kiện & Phát SSE Sản phẩm Brand (Priority: P2)

**Mục tiêu**: Module `brand` tự tiếp nhận sự kiện phân tích AI, cập nhật sản phẩm thương hiệu và phát SSE cho Brand Portal.

**Independent Test**: Gửi yêu cầu tạo sản phẩm thương hiệu, mở stream SSE `/api/v1/brand-portal/brands/{brandId}/items/tasks/{taskId}/sse` và nhận được event cập nhật tương ứng.

- [X] T013 [US2] Thêm phương thức xử lý kết quả phân tích AI `HandleFashionAnalyzed` trong `internal/modules/brand/application/usecase/item/fashion_event_uc.go`
- [X] T014 [US2] Khai báo phương thức `HandleFashionAnalyzed` trong interface `internal/modules/brand/application/interface/usecase/brand_item_uc.go`
- [X] T015 [US2] Tạo `BrandFashionEventWorker` lắng nghe queue `brand_fashion_analyzed_queue`, xử lý message và gọi use case trong `internal/modules/brand/presentation/worker/fashion_event_worker.go`
- [X] T016 [US2] Cấu hình phát SSE qua `sseManager.Publish` trong `BrandFashionEventUseCase`, đảm bảo nhúng `FashionBrief` và tính Idempotency trong `internal/modules/brand/application/usecase/item/fashion_event_uc.go`
- [X] T017 [US2] Viết unit test cho `BrandFashionEventUseCase` trong `internal/modules/brand/application/usecase/item/fashion_event_uc_test.go`

**Checkpoint**: Hoàn tất User Story 2! Luồng xử lý sản phẩm của nhãn hàng B2B hoạt động độc lập qua Event-Driven.

---

### Phase 5: User Story 3 - Wire DI & Lifecycle Background Worker (Priority: P3)

**Mục tiêu**: Đăng ký các Consumer mới vào hệ thống khởi chạy và cấu hình DI với Google Wire.

**Independent Test**: Chạy `make dev`, kiểm tra logs xác nhận cả 2 worker consumer `WardrobeFashionEventWorker` và `BrandFashionEventWorker` đều khởi động thành công và lắng nghe RabbitMQ.

- [X] T018 [US3] Khai báo `WardrobeFashionEventWorker` và `BrandFashionEventWorker` vào provider tương ứng tại `internal/modules/wardrobe/provider.go` và `internal/modules/brand/provider.go`
- [X] T019 [US3] Cập nhật dependency injection container trong `internal/di/wire.go` và chạy lệnh `wire ./internal/di` để sinh lại `internal/di/wire_gen.go`
- [X] T020 [US3] Đăng ký khởi chạy background worker cho các Consumer mới trong `internal/bootstrap/app.go`

---

### Phase 6: Polish & Thẩm định Toàn diện (Polish & Cross-Cutting Concerns)

**Mục tiêu**: Hoàn thiện tài liệu kiến trúc, kiểm tra toàn bộ test suite và xác minh chất lượng.

- [X] T021 [P] Cập nhật tài liệu thiết kế hệ thống và mô hình phân tách event
- [X] T022 Chạy toàn bộ test suite: `go test ./...`
- [X] T023 Xác minh hợp đồng sự kiện và kịch bản end-to-end theo `specs/014-decouple-fashion-worker-sse/quickstart.md`

---

## Thứ tự Thực thi & Quan hệ Phụ thuộc (Dependencies)

```text
Phase 1: Setup (T001, T002, T003)
   │
   ▼
Phase 2: Foundational Worker Refactor (T004 -> T005 -> T006 -> T007)
   │
   ├────────────────────────────────────────┐
   ▼                                        ▼
Phase 3: User Story 1 (Wardrobe Consumer) Phase 4: User Story 2 (Brand Consumer)
   (T008 -> T009 -> T010 -> T011 -> T012)   (T013 -> T014 -> T015 -> T016 -> T017)
   │                                        │
   └───────────────────┬────────────────────┘
                       │
                       ▼
Phase 5: Wire DI & Worker Lifecycle (T018 -> T019 -> T020)
                       │
                       ▼
Phase 6: Polish & Verification (T021, T022, T023)
```

---

## Cơ hội Triển khai Song song (Parallel Opportunities)

1. **Trong Phase 1**: Task T003 có thể làm song song với T001, T002.
2. **Sau Phase 2**: User Story 1 (Phase 3) và User Story 2 (Phase 4) có thể triển khai song song hoàn toàn bởi 2 lập trình viên khác nhau vì nằm ở 2 module riêng biệt (`internal/modules/wardrobe/` và `internal/modules/brand/`).
3. **Trong Phase 6**: Task cập nhật tài liệu T021 có thể làm song song với việc chạy test T022.
