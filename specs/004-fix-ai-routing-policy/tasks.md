# Tasks: Giải quyết nợ kỹ thuật định tuyến AI Routing Policy

**Input**: Design documents from `/specs/004-fix-ai-routing-policy/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Sắp xếp lại tệp cấu hình môi trường.

- [x] T001 Sắp xếp các biến cấu hình `FREE_TEXT_PRIMARY_...` xuống dưới cùng trong [.env.example](../../.env.example) và [.env.production.example](../../.env.production.example)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Đảm bảo dự án sạch lỗi trước khi tiến hành code.

- [x] T002 Chạy lệnh `make check` để kiểm tra trạng thái biên dịch và định dạng của codebase hiện tại.

---

## Phase 3: User Story 1 - Tự động định tuyến động dựa trên hạn mức (Priority: P1) 🎯 MVP

**Goal**: Cho phép Orchestrator tự động chuyển hướng model sang cấu hình `FreeTextPrimary` khi `CostUseCase.Prepare` trả về route `free`.

**Independent Test**: Thay đổi chính sách ví sang `free_only`, gọi API và kiểm tra xem hệ thống có tự động gọi sang model Gemma hoặc Gemini Flash Lite thay vì model Premium hay không.

### Implementation for User Story 1

- [x] T003 [P] [US1] Cập nhật hàm `resolveProviderModel` trong [plan.go](../../internal/modules/aiexecutor/application/usecase/orchestrator/plan.go) nhận thêm tham số `route airoute.AIRoute` và trả về `cfg.AI.FreeTextPrimary` khi route là `free` hoặc `local`.
- [x] T004 [US1] Cập nhật các hàm `GenerateChatText`, `GenerateRecommendationText`, `GenerateChatTextStream` trong [orchestrator.go](../../internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go) để cập nhật lại `plan.Provider` và `plan.Model` sang mô hình Free khi nhận `decision.Route == airoute.Free`.

**Checkpoint**: Định tuyến Free hoạt động độc lập và chính xác, tự động hạ cấp chất lượng model tương ứng.

---

## Phase 4: User Story 2 - Bỏ qua gọi API AI và chạy thuật toán cục bộ khi định tuyến Local (Priority: P2)

**Goal**: Khi định tuyến được quyết định là `local`, hệ thống lập tức ngắt cuộc gọi ngoại vi và kích hoạt fallback thuật toán cục bộ.

**Independent Test**: Đặt hạn mức sử dụng về 0, gửi yêu cầu gợi ý trang phục, hệ thống trả về kết quả HSL cục bộ với cờ `IsFallback: true`.

### Implementation for User Story 2

- [x] T005 [US2] Cập nhật logic trong [orchestrator.go](../../internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go) để kiểm tra `decision.Route == airoute.Local`, trả về lỗi `errors.New("ai route resolved to local fallback: bypassing remote api call")` mà không thực hiện gọi API LLM.

**Checkpoint**: Khi gặp lỗi hoặc cạn kiệt tài khoản, hệ thống chuyển giao thành công cho thuật toán cục bộ xử lý.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Đồng bộ hóa code sinh ra, chạy test kiểm tra.

- [x] T006 Chạy lệnh `make generate` để cập nhật lại dependency injection của Google Wire.
- [x] T007 Thực hiện biên dịch dự án (`go build ./...`) để đảm bảo không lỗi cú pháp.
- [x] T008 Chạy unit tests (`go test ./...`) để kiểm tra logic định tuyến AI Executor.
- [x] T009 Xác thực kịch bản bằng tay theo [quickstart.md](quickstart.md).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Hoàn thành ngay (T001 đã được thực hiện trước).
- **Foundational (Phase 2)**: Chạy kiểm tra nền tảng.
- **User Stories (Phase 3 & 4)**: Cần thực hiện tuần tự để kiểm thử dễ dàng (US1 trước, sau đó tới US2).
- **Polish (Phase 5)**: Chạy cuối cùng sau khi hoàn tất code.

### Parallel Opportunities

- Việc định nghĩa lại tham số trong `plan.go` (T003) và cập nhật logic `orchestrator.go` (T004) có thể được thực hiện song song hoặc tuần tự bởi cùng một dev.
