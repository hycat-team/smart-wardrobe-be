# Feature Specification: Giải quyết nợ kỹ thuật định tuyến AI Routing Policy

**Feature Branch**: `004-fix-ai-routing-policy`

**Created**: 2026-07-17

**Status**: Draft

**Input**: User description: "giải quyết nợ kỹ thuật này (AI Executor Routing Policy). Sửa vị trí .env.example:L98-L105 trong .env.example và .env.production.example và các nơi liên quan để nó nằm dưới cùng của các model env."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tự động định tuyến động dựa trên hạn mức (Priority: P1)

Khi người dùng thực hiện một yêu cầu gọi AI (như trò chuyện hoặc tạo gợi ý trang phục), hệ thống phải tự động quyết định xem yêu cầu đó đi qua kênh Paid (trả phí) hay Free (miễn phí) dựa vào ví hạn mức và luật Policy, sau đó ánh xạ chính xác model tương ứng chứ không dùng model tĩnh.

**Why this priority**: Đây là giá trị cốt lõi nhất để giải quyết nợ kỹ thuật, giúp kiểm soát chi phí gọi API AI của hệ thống.

**Independent Test**: Có thể kiểm tra bằng cách lập cấu hình một policy có ví hạn mức cạn kiệt, khi người dùng gọi AI, hệ thống tự động ánh xạ sang model Gemini Flash Lite (miễn phí) thay vì Gemini Pro (trả phí).

**Acceptance Scenarios**:

1. **Given** Người dùng thuộc gói gói cước có hạn mức và ví vẫn còn đủ hạn mức thanh toán, **When** Người dùng gửi yêu cầu chat, **Then** Hệ thống quyết định định tuyến `"paid"` và gọi model được cấu hình cho Paid (ví dụ: `gemini-3.1-flash`).
2. **Given** Người dùng đã sử dụng hết hạn mức Premium hoặc bị áp chính sách chỉ dùng gói miễn phí, **When** Người dùng gửi yêu cầu chat, **Then** Hệ thống quyết định định tuyến `"free"` và tự động chuyển hướng gọi model được cấu hình cho Free (ví dụ: `gemini-2.5-flash-lite`).

---

### User Story 2 - Bỏ qua gọi API AI và chạy thuật toán cục bộ khi định tuyến Local (Priority: P2)

Khi hệ thống quyết định định tuyến là `"local"`, yêu cầu gọi AI sẽ được bỏ qua và chuyển giao cho các thuật toán nội bộ xử lý cục bộ bằng code Go thuần (như HSL Color Matching và Local Keyword Extraction).

**Why this priority**: Đảm bảo hệ thống vẫn hoạt động và đưa ra kết quả dự phòng (fallback) ngay cả khi không có kết nối LLM hoặc người dùng bị chặn hoàn toàn khỏi dịch vụ đám mây.

**Independent Test**: Cấu hình ví hạn mức ở chế độ chặn toàn bộ API, gửi yêu cầu gợi ý trang phục và kiểm tra kết quả trả về có cờ `IsFallback: true` và được phối bằng thuật toán HSL cục bộ.

**Acceptance Scenarios**:

1. **Given** Cấu hình định tuyến của người dùng bị hạ cấp về `"local"`, **When** Người dùng gửi yêu cầu gợi ý trang phục, **Then** Hệ thống không gọi bất kỳ LLM API nào mà chạy trực tiếp thuật toán phối màu HSL cục bộ và trả về kết quả phối đồ dự phòng.

---

### Edge Cases

- **Mô hình Free bị lỗi kết nối**: Khi gọi mô hình Free gặp sự cố và `PaidFallbackEnabled` được cấu hình là `true`, hệ thống phải tự động thử lại bằng mô hình Paid (nếu người dùng còn đủ hạn mức).
- **Mô hình cấu hình trong .env bị trống hoặc sai**: Khi hệ thống định tuyến động đến một cấu hình model bị thiếu hoặc sai thông tin trong `.env`, hệ thống phải ghi nhận lỗi chi tiết và an toàn trả về kết quả fallback local.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST dịch chuyển logic chọn Model/Provider từ tĩnh sang động dựa vào `AIRoute` được phân giải từ `CostUseCase.Prepare`.
- **FR-002**: Khi `route` phân giải là `"paid"`, hệ thống MUST ánh xạ đến các cấu hình model tương ứng của operation (ví dụ: `CHAT_TEXT_PRIMARY`).
- **FR-003**: Khi `route` phân giải là `"free"`, hệ thống MUST ánh xạ đến mô hình `FREE_TEXT_PRIMARY` (Gemma / Gemini Flash Lite) thay vì gọi mô hình trả phí.
- **FR-004**: Khi `route` phân giải là `"local"`, hệ thống MUST bỏ qua việc gọi API LLM và chuyển hướng thực thi thuật toán fallback cục bộ bằng Go thuần.
- **FR-005**: Hệ thống MUST di chuyển và sắp xếp lại các biến môi trường cấu hình model Free (`FREE_TEXT_PRIMARY_...`, `FREE_TEXT_FALLBACK_...`) xuống dưới cùng của nhóm cấu hình AI trong `.env.example`, `.env.production.example` và các file cấu hình liên quan để đồng bộ cấu trúc.

### Key Entities *(include if feature involves data)*

- **AIModelConfig**: Lưu trữ thông tin đơn giá và model động từ cơ sở dữ liệu.
- **AICostPolicyOperation**: Định nghĩa các route áp dụng cho từng operation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% các cuộc gọi AI của người dùng hết hạn mức trả phí được định tuyến chính xác sang mô hình miễn phí/mô hình cục bộ mà không bị gọi nhầm vào mô hình đám mây trả phí.
- **SC-002**: Các biến cấu hình môi trường trong các tệp `.env.example` và `.env.production.example` được sắp xếp ngăn nắp, đồng bộ và không gây lỗi khởi chạy ứng dụng.

## Assumptions

- Việc thay đổi cấu trúc định tuyến và sắp xếp lại tệp tin cấu hình không thay đổi các API Response của module AI Executor đối với các module gọi bên ngoài.
- Client hoặc thiết bị cục bộ không bắt buộc phải tải mô hình LLM về máy để chạy route `local` (thuật toán cục bộ chạy bằng Go thuần trên Server).
