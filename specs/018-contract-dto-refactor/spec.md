# Đặc tả Tính Năng: Tách Biệt Contract DTO Và Chuẩn Hóa Ranh Giới Dữ Liệu Giữa Các Module

**Feature Branch**: `018-contract-dto-refactor`

**Created**: 2026-09-03

**Status**: Draft

**Input**: User description: "Refactor tách riêng contract_dto.go trong application/dto cho các module, ưu tiên giải quyết dứt điểm việc tách FashionItem khỏi WardrobeItem và BrandItem trong contract"

---

## Bối Cảnh & Vấn Đề

Sau khi triển khai nguyên lý **Pure Contract** (Hiến pháp dự án §II và `docs/system-design/contract-purity-pattern.md`), các Contract Use Case phục vụ giao tiếp chéo giữa các module đã được thuần hóa (không tự ý gọi ngầm module khác để enrich dữ liệu). Tuy nhiên, tầng dữ liệu trao đổi (DTO) hiện tại vẫn đang tồn tại mâu thuẫn kiến trúc:

1. **Dùng chung DTO giữa API Response và Contract (Over-sharing DTO):**
   - Hai struct `WardrobeItemBriefRes` / `WardrobeItemRes` (module Wardrobe) và `BrandItemBriefRes` / `BrandItemRes` (module Brand) đang được dùng chung cho cả 2 mục đích: vừa làm kiểu dữ liệu trả về cho API Handler (cho Mobile/Web Client), vừa làm kiểu dữ liệu trả về qua Contract (giao tiếp nội bộ giữa các module).
2. **DTO "nửa nạc nửa mỡ" trong Contract:**
   - Khi trả về qua Contract, trường `FashionItem` luôn luôn là `nil` (do Contract Use Case là Pure, không enrich). Điều này tạo ra ảo tưởng sai lệch về dữ liệu, làm tăng nguy cơ bug runtime và vi phạm ranh giới module.
   - Ngược lại, API Response cho người dùng cuối bắt buộc phải có `FashionItem` để hiển thị hình ảnh, phân loại và chi tiết sản phẩm.
3. **Phụ thuộc kiểu dữ liệu không cần thiết (Type Coupling):**
   - Module DTO của Brand và Wardrobe phải trực tiếp import `fashion/application/dto` ngay cả trong các ngữ cảnh chỉ giao tiếp thuộc tính cốt lõi của mình.
4. **Thiếu sự nhất quán tổ chức file DTO giữa các module:**
   - Chưa có một file quy chuẩn (ví dụ `contract_dto.go`) trong `application/dto/` để phân định ranh giới giữa "Dữ liệu phục vụ giao tiếp chéo module (Contract)" và "Dữ liệu phục vụ luồng trình bày Client (HTTP Presentation Handler)".

Mục tiêu của đợt refactor này là chuẩn hóa toàn diện cấu trúc DTO cho Contract trên toàn bộ các module, với trọng tâm ưu tiên cao nhất là giải quyết dứt điểm việc tách `FashionItem` khỏi `WardrobeItem` và `BrandItem` trong Contract.


---

## Clarifications

### Session 2026-09-03

- Q: Tổ chức Package và Vị trí của Contract DTO như thế nào? → A: Đặt tập tin `contract_dto.go` trực tiếp trong package `application/dto` của từng module, các struct dùng hậu tố `ContractDTO` (ví dụ `WardrobeItemContractDTO`, `BrandItemContractDTO`), tuân thủ Clean Architecture và cơ chế import `application/dto` của Hiến pháp §II mà không tạo thêm subpackage.
- Q: Chiến lược phân kỳ triển khai (Phasing Strategy) như thế nào? → A: Triển khai theo 2 Phase tuần tự — Phase 1: Xử lý dứt điểm Wardrobe & Brand cùng toàn bộ callers/mappers/workers liên quan; Phase 2: Chuẩn hóa `contract_dto.go` cho các module còn lại (`identity`, `subscription`, `fashion`, `admin`, `notification`) nhằm cô lập rủi ro và dễ kiểm thử tích hợp.
- Q: Cơ chế tương thích kiểu dữ liệu trong nội bộ Contract (Contract Migration Approach) như thế nào? → A: Chuyển đổi dứt khoát (Strict Cutover) trực tiếp trên interface của `IWardrobeContract` và `IBrandContract`, tận dụng trình biên dịch Go để phát hiện và cập nhật toàn diện 100% callers, không duy trì các method deprecated/song song gây dư thừa code rác.

---

## User Scenarios & Testing *(mandatory)*

<!--
  Trong bài toán Refactoring kiến trúc backend, các tác nhân chính (Actors) là:
  - Consumer Modules: Các module gọi contract (ví dụ Fashion, Dashboard, Workers).
  - Provider Modules: Các module cung cấp contract (Wardrobe, Brand, Identity, Subscription, v.v.).
  - End Users / Clients: Web/Mobile App (đảm bảo không bị ảnh hưởng hành vi/dữ liệu).
-->

### User Story 1 - Chuẩn Hóa Contract DTO Của Module Wardrobe (Priority: P1)

Là một Consumer Module (ví dụ module Fashion hoặc Background Worker), tôi muốn khi gọi `IWardrobeContract` thì dữ liệu trả về là `WardrobeItemContractDTO` thuần khiết, chỉ chứa thông tin tủ đồ và `FashionItemID`, không chứa con trỏ `FashionItem` rỗng, giúp hợp đồng rõ ràng và không bị sai lệch dữ liệu.

**Why this priority**: Module Wardrobe là trung tâm của luồng xử lý trang phục người dùng. Việc DTO chứa con trỏ `FashionItem` luôn `nil` qua contract gây nhầm lẫn nghiêm trọng cho các use case AI phối đồ (Recommendation) và Worker dọn dẹp dữ liệu (`CleanupFailedItems`, `RecoverStaleProcessingItems`).

**Independent Test**: Có thể kiểm thử độc lập bằng cách gọi các phương thức trong `IWardrobeContract`, kiểm tra kiểu trả về là `WardrobeItemContractDTO`, xác nhận các trường dữ liệu nội tại của Wardrobe và `FashionItemID` được điền chính xác, không còn phụ thuộc vào struct của module Fashion.

**Acceptance Scenarios**:

1. **Given** một người dùng có các trang phục trong tủ đồ, **When** module Fashion gọi `IWardrobeContract.GetWardrobeItemsByUserID`, **Then** kết quả trả về là danh sách `*dto.WardrobeItemContractDTO` chứa đầy đủ ID, UserID, Price, Status, IsLocked và `FashionItemID *uuid.UUID`, không có trường `FashionItem`.
2. **Given** trang phục tải lên bị lỗi (chưa kịp tạo bản ghi Fashion), **When** Worker gọi `IWardrobeContract.GetFailedItemsForCleanup`, **Then** contract trả về `WardrobeItemContractDTO` với `FashionItemID == nil`, hệ thống xử lý dọn dẹp tủ đồ an toàn mà không phát sinh lỗi nil pointer.
3. **Given** người dùng truy cập API lấy danh sách tủ đồ `/api/v1/wardrobe/items`, **When** Handler gọi `WardrobeItemUseCase`, **Then** kết quả API trả về cho Client vẫn là `WardrobeItemBriefRes` với đầy đủ thông tin `FashionItem` đã được enrich, không làm thay đổi giao diện người dùng.

---

### User Story 2 - Chuẩn Hóa Contract DTO Của Module Brand (Priority: P1)

Là một Consumer Module (ví dụ module Fashion khi tìm sản phẩm đối tác để gợi ý), tôi muốn khi gọi `IBrandContract` thì dữ liệu trả về là `BrandItemContractDTO` thuần khiết, chỉ chứa thông tin sản phẩm nhãn hàng và `FashionItemID`, không chứa con trỏ `FashionItem` lồng nhau.

**Why this priority**: Module Brand cung cấp các sản phẩm tiếp thị số liên kết trực tiếp với Fashion qua `FashionItemID`. Việc tách riêng DTO cho contract giúp rạch ròi trách nhiệm giữa Brand và Fashion, ngăn ngừa rò rỉ dữ liệu và phục vụ trực tiếp cho tính năng AI Recommendation.

**Independent Test**: Có thể kiểm thử độc lập bằng cách gọi `IBrandContract.GetBrandItemsByFashionItemIDs` và `IBrandContract.ListEligibleBrandItemsForStyling`, xác nhận trả về `*dto.BrandItemContractDTO` có chứa `FashionItemID` rõ ràng, sau đó caller tự thực hiện stitching bằng `sliceutils.IndexBy`.

**Acceptance Scenarios**:

1. **Given** danh sách các `FashionItemID` hợp lệ, **When** module khác gọi `IBrandContract.GetBrandItemsByFashionItemIDs`, **Then** contract trả về danh sách `*dto.BrandItemContractDTO` chứa ID, BrandID, Name, Price, Status và `FashionItemID uuid.UUID`, không chứa trường lồng `FashionItem`.
2. **Given** nhân viên nhãn hàng xem chi tiết sản phẩm qua Brand Portal API, **When** Brand Portal Handler xử lý yêu cầu, **Then** Handler vẫn sử dụng `BrandItemRes` với đầy đủ thông tin `FashionItem` được enrich để hiển thị lên cổng quản trị.

---

### User Story 3 - Chuẩn Hóa Cấu Trúc `contract_dto.go` Cho Toàn Bộ Các Module Khác (Priority: P2)

Là một lập trình viên phát triển hệ thống, tôi muốn toàn bộ các module (`identity`, `subscription`, `fashion`, `admin`, `notification`) đều có file `internal/modules/<module>/application/dto/contract_dto.go` chứa định dạng dữ liệu giao tiếp nội bộ, giúp mã nguồn có tính cấu trúc đồng nhất 100%.

**Why this priority**: Sau khi giải quyết 2 module trọng yếu (Wardrobe và Brand), việc áp dụng mô hình chuẩn `contract_dto.go` cho các module còn lại sẽ hoàn thiện kiến trúc Modular Monolith, ngăn chặn việc tái xuất hiện lỗi trộn lẫn DTO trong tương lai.

**Independent Test**: Quét toàn bộ cây thư mục mã nguồn, đảm bảo mỗi module có `presentation/contract/` đều có file `application/dto/contract_dto.go` tương ứng định nghĩa các struct phục vụ cho interface contract đó.

**Acceptance Scenarios**:

1. **Given** module Identity cung cấp thông tin người dùng tóm tắt, **When** kiểm tra `internal/modules/identity/application/dto/`, **Then** struct phục vụ cho `IUserBriefContract` nằm trong `contract_dto.go` với các thuộc tính định danh thuần túy (`UserBriefContractDTO` hoặc `UserBriefRes` đã được chuẩn hóa).
2. **Given** module Subscription cung cấp hạn mức và gói dịch vụ, **When** kiểm tra `internal/modules/subscription/application/dto/`, **Then** các struct phục vụ cho `IUserSubscriptionContract` và `IUserQuotaContract` được tập trung rõ ràng trong `contract_dto.go`.
3. **Given** module Fashion cung cấp thông tin thời trang cho các module khác, **When** kiểm tra `internal/modules/fashion/application/dto/`, **Then** các DTO phục vụ cho `IFashionContract` được tổ chức trong `contract_dto.go`.

---

### User Story 4 - Đồng Bộ Hóa Mappers Và Bảo Đảm Tương Thích Ngược API (Priority: P3)

Là một nhà phát triển hệ thống, tôi muốn tầng Mapper của các module cung cấp các hàm chuyển đổi chuyên biệt từ Entity sang `ContractDTO` (tách biệt khỏi hàm map sang `API Res`), đồng thời các caller nội bộ tự thực hiện stitching mượt mà, đảm bảo tất cả Unit Tests và API công khai không bị ảnh hưởng.

**Why this priority**: Đảm bảo quá trình refactor không làm phá vỡ (zero regression) chức năng hiện hữu của ứng dụng di động và website, đồng thời giữ vững độ bao phủ kiểm thử (test suite passing).

**Independent Test**: Chạy toàn bộ test suite (`make test-race`, `make lint`) và kiểm tra Swagger / HTTP API responses của các endpoint liên quan đến tủ đồ và trang phục nhãn hàng.

**Acceptance Scenarios**:

1. **Given** hệ thống vừa refactor xong DTO contract, **When** thực thi lệnh `make test-race`, **Then** toàn bộ các unit test và integration test pass 100%.
2. **Given** client gọi các API công khai `/api/v1/wardrobe/items`, `/api/v1/brand/items`, **When** kiểm tra JSON response schema, **Then** cấu trúc JSON trả về cho client giữ nguyên vẹn 100%, không bị mất các trường dữ liệu lồng nhau (`fashionItem`, `brandItem`).

---

### Edge Cases

- **Trang phục tủ đồ thiếu FashionItemID (Upload lỗi / Dữ liệu mồ côi):**
  - Trong `WardrobeItemContractDTO`, trường `FashionItemID` phải là con trỏ `*uuid.UUID` (nullable) kèm nil-guard để các worker dọn dẹp (`CleanupFailedItems`) không bị panic khi truy cập.
- **Sản phẩm nhãn hàng không liên kết với FashionItem:**
  - Trong `BrandItemContractDTO`, nếu có sản phẩm không liên kết, trường `FashionItemID` phải được định nghĩa rõ ràng (uuid rỗng `uuid.Nil` hoặc con trỏ tuỳ theo thiết kế domain).
- **Bộ sưu tập rỗng (Empty Slice vs Nil Slice):**
  - Khi Contract trả về danh sách không có phần tử, phải luôn trả về empty slice `[]*dto.SomeContractDTO{}` thay vì `nil` để tránh lỗi nil pointer ở phía Caller.
- **Hiệu năng Stitching ở phía Caller:**
  - Caller khi nhận danh sách `ContractDTO` phải gom nhóm ID và dùng `sliceutils.IndexBy` hoặc `sliceutils.GroupBy` để stitch dữ liệu đạt độ phức tạp $O(1)$, không lặp lồng $O(N \times M)$.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Module `wardrobe` BẮT BUỘC định nghĩa file `internal/modules/wardrobe/application/dto/contract_dto.go` chứa `WardrobeItemContractDTO`.
- **FR-002**: `WardrobeItemContractDTO` BẮT BUỘC chỉ chứa các trường do module Wardrobe sở hữu (ID, UserID, Price, Status, IsLocked, LastUsedAt, TaskID) và `FashionItemID *uuid.UUID`. TUYỆT ĐỐI KHÔNG chứa trường `FashionItem` hay `BrandItem`.
- **FR-003**: Interface `IWardrobeContract` BẮT BUỘC cập nhật toàn bộ các phương thức trả về danh sách trang phục sang kiểu `*dto.WardrobeItemContractDTO`.
- **FR-004**: Module `brand` BẮT BUỘC định nghĩa file `internal/modules/brand/application/dto/contract_dto.go` chứa `BrandItemContractDTO`.
- **FR-005**: `BrandItemContractDTO` BẮT BUỘC chỉ chứa các trường do module Brand sở hữu (ID, BrandID, BrandName, ProductCode, Name, Price, ItemType, Status) và `FashionItemID uuid.UUID`. TUYỆT ĐỐI KHÔNG chứa trường `FashionItem`.
- **FR-006**: Interface `IBrandContract` BẮT BUỘC cập nhật các phương thức liên quan sang kiểu `*dto.BrandItemContractDTO`.
- **FR-007**: Tất cả các Caller gọi đến `IWardrobeContract` và `IBrandContract` (bao gồm `fashion`, `dashboard`, `workers`) BẮT BUỘC cập nhật logic tiếp nhận sang `ContractDTO` và tự thực hiện làm giàu (stitching) dữ liệu nếu có nhu cầu hiển thị.
- **FR-008**: Các module còn lại (`identity`, `subscription`, `fashion`, `admin`, `notification`) BẮT BUỘC chuẩn hóa việc đặt các DTO phục vụ contract vào file `contract_dto.go` thuộc package `application/dto`.
- **FR-009**: Tầng Mapper của các module liên quan BẮT BUỘC bổ sung hàm mapping chuyên biệt (ví dụ `MapToWardrobeItemContractDTO`, `MapToBrandItemContractDTO`).
- **FR-010**: Tầng Presentation HTTP Handler và các API Response DTO phục vụ Client bên ngoài (`WardrobeItemRes`, `BrandItemRes`, v.v.) BẮT BUỘC giữ nguyên vẹn cấu trúc dữ liệu JSON để bảo đảm tương thích ngược 100%.

---

### Key Entities

- **WardrobeItemContractDTO**: Đại diện cho bản ghi tủ đồ trong giao tiếp nội bộ giữa các module. Thuộc tính chính: `ID`, `UserID`, `Price`, `Status`, `IsLocked`, `FashionItemID`, `LastUsedAt`, `TaskID`.
- **BrandItemContractDTO**: Đại diện cho bản ghi sản phẩm nhãn hàng trong giao tiếp nội bộ giữa các module. Thuộc tính chính: `ID`, `BrandID`, `BrandName`, `ProductCode`, `Name`, `Price`, `ItemType`, `Status`, `FashionItemID`.
- **Contract DTO Boundaries**: Tập hợp các struct dữ liệu thuần túy (pure data containers) tại `application/dto/contract_dto.go` của mỗi module, đóng vai trò là hợp đồng dữ liệu chính thức cho `presentation/contract/`.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% các phương thức trong `IWardrobeContract` và `IBrandContract` không trả về bất kỳ struct nào chứa con trỏ struct của module khác (`FashionItem` = 0 trường vi phạm).
- **SC-002**: 100% các module có triển khai Contract nội bộ đều sở hữu file `internal/modules/<module>/application/dto/contract_dto.go` chứa định nghĩa kiểu dữ liệu của contract.
- **SC-003**: 0% breaking change đối với các API endpoints công khai ra ngoài internet (Swagger API specs và JSON responses của Client không bị biến đổi cấu trúc).
- **SC-004**: 100% các bài kiểm thử tự động (`make test-race`) và kiểm tra tĩnh (`make lint`) vượt qua thành công sau khi hoàn tất refactor.
- **SC-005**: Toàn bộ các vị trí ghép dữ liệu (stitching) chéo module ở phía Caller đều sử dụng `sliceutils.IndexBy` hoặc `sliceutils.GroupBy` đạt độ phức tạp thời gian truy xuất $O(1)$.

---

## Assumptions

- Các Client (Web Frontend, Mobile App) chỉ tương tác với hệ thống qua HTTP REST API (thông qua Handler), không tương tác trực tiếp với tầng Contract nội bộ, do đó việc thay đổi Contract DTO không ảnh hưởng tới người dùng cuối.
- Việc caller tự làm giàu dữ liệu (Self-enriching & Stitching) tuân thủ nghiêm ngặt nguyên tắc Pure Contract của Hiến pháp §II.
- Việc phân chia file `contract_dto.go` nằm cùng package `dto` trong thư mục `application/dto/` không làm thay đổi các quy tắc import chéo tầng đã được quy định trong Hiến pháp.
