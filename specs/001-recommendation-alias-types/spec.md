# Đặc tả tính năng: Định nghĩa các kiểu Alias riêng cho Nguồn Gợi ý và Nguồn Từ khóa Truy vấn

**Nhánh tính năng**: `001-recommendation-alias-types`

**Ngày tạo**: 2026-07-13

**Trạng thái**: Bản thảo (Draft)

**Đầu vào**: Mô tả của người dùng: "tạo các loại alias riêng cho từng cái này thay vì để là string cho những nơi dùng const và nơi định nghĩa struct thì thì để là alias đó thay vì string tiếp theo để đảm bảo DRY thì cho những nơi này dùng type đó"

## Kịch bản Người dùng & Kiểm thử *(bắt buộc)*

### Kịch bản 1 - Đảm bảo An toàn Kiểu dữ liệu (Type-safety) cho Nguồn Ứng viên và Nguồn Từ khóa (Độ ưu tiên: P1)

Lập trình viên cần sự an toàn về kiểu dữ liệu (type safety) và khả năng tự động gợi ý (auto-completion) từ IDE khi làm việc với các nguồn ứng viên (candidate sources) và nguồn từ khóa truy vấn (retrieval term sources), thay vì sử dụng các chuỗi văn bản thuần (string) vốn rất dễ gõ sai chính tả.

**Lý do ưu tiên**: Đây là yêu cầu cốt lõi của việc tái cấu trúc code nhằm giảm thiểu lỗi do gõ sai giá trị chuỗi (string literal typos).

**Kiểm thử Độc lập**: Xác minh rằng mã nguồn biên dịch thành công và bất kỳ phép gán giá trị chuỗi không hợp lệ nào cho các trường yêu cầu kiểu alias này đều sẽ bị cảnh báo bởi trình phân tích tĩnh (static analyzer) hoặc trình biên dịch.

**Kịch bản Nghiệm thu**:

1. **Cho** các kiểu alias mới là `CandidateSource` và `RetrievalTermSource` được định nghĩa trong `types.go`, **Khi** bộ công cụ gợi ý trang phục khởi tạo và chấm điểm các ứng viên, **Thì** tất cả các trường trong struct và các hằng số liên quan đều sử dụng các kiểu alias này thay vì `string`.

---

### Kịch bản 2 - Áp dụng Nguyên tắc DRY cho các Hằng số (Độ ưu tiên: P2)

Lập trình viên muốn tránh việc định nghĩa trùng lặp các hằng số ở nhiều package khác nhau để dễ dàng bảo trì từ một nguồn duy nhất (single source of truth).

**Lý do ưu tiên**: Đảm bảo rằng bất kỳ thay đổi nào đối với nguồn ứng viên trong tương lai chỉ cần cập nhật ở một nơi duy nhất.

**Kiểm thử Độc lập**: Xác minh rằng file `scoring.go` không định nghĩa lại các hằng số nguồn ứng viên trùng lặp mà sử dụng trực tiếp các hằng số đã định nghĩa trong `types.go`.

**Kịch bản Nghiệm thu**:

1. **Cho** các hằng số nguồn ứng viên trong `types.go` được định nghĩa với kiểu `CandidateSource`, **Khi** chấm điểm ứng viên trong `scoring.go`, **Thì** package này sẽ tham chiếu đến `types.CandidateSource...` thay vì tự định nghĩa lại.

---

### Các Trường hợp Biên (Edge Cases)

- Điều gì xảy ra nếu các ranh giới bên ngoài (cơ sở dữ liệu hoặc API ngoại vi) vẫn trả về chuỗi raw string?
  - Hệ thống cần chuyển đổi hoặc cho phép chuyển đổi an toàn khi cần thiết. Do các kiểu alias được định nghĩa dựa trên kiểu nền tảng là `string` (ví dụ: `type CandidateSource string`), việc chuyển đổi kiểu hoặc gán từ hằng số chuỗi trong Go vẫn diễn ra rất dễ dàng và an toàn.

## Yêu cầu *(bắt buộc)*

### Yêu cầu Chức năng

- **FR-001**: Hệ thống KHÔNG ĐƯỢC để một file `types.go` duy nhất mà PHẢI chia tách thư mục `types/` thành 2 file:
  - `aliases.go`: chứa định nghĩa các kiểu alias mới và các hằng số liên quan.
  - `structs.go`: chứa định nghĩa các struct và interface dùng chung cho luồng gợi ý trang phục.
- **FR-002**: Hệ thống PHẢI định nghĩa kiểu dữ liệu mới `CandidateSource` (kiểu nền tảng là `string`) và `RetrievalTermSource` (kiểu nền tảng là `string`) trong file `aliases.go`.
- **FR-003**: Hệ thống PHẢI cập nhật tất cả các hằng số trong `aliases.go` để sử dụng các kiểu alias mới này một cách rõ ràng.
- **FR-004**: Hệ thống PHẢI cập nhật các trường trong các struct của `structs.go` (`CandidateForRanking`, `RankedCandidate`, `RecommendationRetrievalQuery`, `RetrievalTerm`, v.v.) đang lưu trữ nguồn ứng viên hoặc nguồn từ khóa truy vấn để sử dụng kiểu alias tương ứng từ `aliases.go` thay vì `string` thuần.
- **FR-005**: Hệ thống PHẢI loại bỏ các định nghĩa hằng số nguồn ứng viên trùng lặp cục bộ trong `scoring.go` và tham chiếu đến các hằng số toàn cục tương ứng từ package `types` để đảm bảo nguyên tắc DRY.
- **FR-006**: Hệ thống PHẢI cập nhật tất cả các tham chiếu trong các file khác (ví dụ: `scoring.go`, `pool_expansion.go`, v.v.) để sử dụng các kiểu dữ liệu và hằng số mới từ package `types`.

### Các Thực thể Chính *(nếu tính năng có liên quan đến dữ liệu)*

- **CandidateSource**: Alias của string đại diện cho nguồn gốc của một ứng viên trang phục (hybrid search, fallback, strict-fallback, relaxed-fallback, general-fallback).
- **RetrievalTermSource**: Alias của string đại diện cho nguồn gốc của từ khóa tìm kiếm (local dictionary, raw user input, taxonomy).

## Tiêu chí Thành công *(bắt buộc)*

### Kết quả Đo lường được

- **SC-001**: 100% hằng số và các trường struct liên quan đến nguồn ứng viên và nguồn từ khóa trong luồng gợi ý trang phục được chuyển đổi sang kiểu alias mới.
- **SC-002**: Mã nguồn biên dịch sạch sẽ không có lỗi hoặc cảnh báo liên quan.
- **SC-003**: 100% các bài kiểm thử đơn vị (unit test) và tích hợp (integration test) liên quan đều vượt qua thành công.

## Giả định

- Chúng ta giả định `type CandidateSource string` và `type RetrievalTermSource string` là đủ để đại diện cho các trạng thái này.
- Chúng ta giả định tất cả các file sử dụng và các bài test sẽ biên dịch thành công sau khi cập nhật các kiểu dữ liệu này.
