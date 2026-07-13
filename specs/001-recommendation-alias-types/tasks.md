# Danh sách công việc: Định nghĩa các kiểu Alias riêng cho Nguồn Gợi ý và Nguồn Từ khóa Truy vấn

**Đầu vào**: Các tài liệu thiết kế từ `/specs/001-recommendation-alias-types/`

**Tiền đề**: [plan.md](plan.md) (yêu cầu), [spec.md](spec.md) (yêu cầu)

**Tổ chức**: Các tác vụ được nhóm theo phân pha để thực hiện và kiểm thử độc lập.

## Định dạng: `- [ ] [ID] [P?] [Story] Mô tả chi tiết`

- **[P]**: Có thể chạy song song (các file khác nhau, không có phụ thuộc)
- **[Story]**: Thuộc kịch bản người dùng nào (ví dụ: US1, US2)
- Các mô tả đều chứa đường dẫn file cụ thể cần chỉnh sửa.

---

## Giai đoạn 1: Thiết lập cấu trúc (Setup)

**Mục đích**: Chuẩn bị cấu trúc file chia tách trong package `types`.

- [x] T001 Xóa file cũ tại [types.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types/types.go) và tạo file mới [aliases.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types/aliases.go) chứa các định nghĩa kiểu alias `CandidateSource` và `RetrievalTermSource` cùng các hằng số liên quan.
- [x] T002 Tạo file mới [structs.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types/structs.go) chứa các định nghĩa struct và interface đã có, chuyển đổi các trường kiểu `string` sang `CandidateSource` và `RetrievalTermSource`.

---

## Giai đoạn 2: Kịch bản 1 - Đảm bảo An toàn Kiểu dữ liệu (Priority: P1) 🎯 MVP

**Mục tiêu**: Cập nhật tất cả các tham chiếu struct/constant trong mã nguồn để sử dụng kiểu alias mới một cách nhất quán.

**Kiểm thử Độc lập**: Biên dịch module thành công, đảm bảo các trường nhận kiểu dữ liệu an toàn.

### Triển khai cho Kịch bản 1

- [x] T003 [P] [US1] Cập nhật tham chiếu kiểu dữ liệu và hằng số mới trong [pool_expansion.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/ranking/pool_expansion.go).
- [x] T004 [P] [US1] Cập nhật tham chiếu kiểu dữ liệu và hằng số mới trong [candidate_helpers.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go).
- [x] T005 [P] [US1] Cập nhật tham chiếu kiểu dữ liệu và hằng số mới trong [workflow.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/workflow.go).
- [x] T006 [P] [US1] Cập nhật tham chiếu kiểu dữ liệu và hằng số mới trong [parser.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/parser/parser.go).

**Điểm kiểm soát**: Kết thúc phần này, kịch bản 1 hoàn tất, hệ thống sử dụng alias an toàn.

---

## Giai đoạn 3: Kịch bản 2 - Áp dụng Nguyên tắc DRY (Priority: P2)

**Mục tiêu**: Loại bỏ hằng số trùng lặp trong package ranking để trỏ về package types.

**Kiểm thử Độc lập**: Đảm bảo không còn hằng số candidate source trùng lặp cục bộ trong package ranking.

### Triển khai cho Kịch bản 2

- [x] T007 [US2] Chỉnh sửa [scoring.go](file:///C:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring.go) để xóa bỏ các hằng số trùng lặp và sử dụng trực tiếp các hằng số từ package `types`.

**Điểm kiểm soát**: Hoàn thành việc tối ưu DRY cho các hằng số.

---

## Giai đoạn cuối: Hoàn thiện & Đối chiếu

**Mục đích**: Đảm bảo toàn bộ hệ thống biên dịch sạch sẽ và vượt qua mọi unit test.

- [x] T008 Chạy lệnh biên dịch toàn bộ dự án `go build ./...` để kiểm tra tĩnh.
- [x] T009 Chạy unit test của module fashion bằng lệnh `go test -v ./internal/modules/fashion/...`.
- [x] T010 Chạy kiểm tra định dạng code và linting nếu có để đảm bảo chất lượng.

---

## Phụ thuộc & Thứ tự Thực hiện

- T001 & T002 là tiền đề bắt buộc trước khi thực hiện các task cập nhật tham chiếu (T003 - T007).
- Các task US1 (T003 - T006) có thể thực hiện song song sau khi hoàn tất Giai đoạn 1.
- Giai đoạn cuối (T008 - T010) được thực hiện sau khi tất cả các task trước đó hoàn thành.
