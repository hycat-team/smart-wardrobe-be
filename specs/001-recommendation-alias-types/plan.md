# Implementation Plan: Định nghĩa các kiểu Alias riêng cho Nguồn Gợi ý và Nguồn Từ khóa Truy vấn

**Branch**: `001-recommendation-alias-types` | **Date**: 2026-07-13 | **Spec**: [spec.md](spec.md)

## Summary

Chúng ta cần tái cấu trúc cách định nghĩa nguồn ứng viên (candidate source) và nguồn từ khóa truy vấn (retrieval term source) bằng cách thay thế kiểu dữ liệu `string` thuần bằng các kiểu alias tùy chỉnh (`CandidateSource` và `RetrievalTermSource`). Hơn nữa, chúng ta sẽ chia tách file `types.go` hiện tại thành hai file `aliases.go` và `structs.go` để đáp ứng ràng buộc cấu trúc mới. Cuối cùng, chúng ta sẽ dọn dẹp các hằng số bị trùng lặp trong package `ranking` (file `scoring.go`) để đảm bảo nguyên tắc DRY.

## Technical Context

- **Language/Version**: Go (v1.25.+)
- **Primary Dependencies**: `internal/modules/fashion/domain/repositories`, `internal/modules/wardrobe/application/dto`, `internal/modules/brand/application/dto`
- **Storage**: N/A
- **Testing**: `go test -v ./internal/modules/fashion/...`
- **Target Platform**: Go Runtime / Backend Service
- **Project Type**: Web Service / API / Go Modules
- **Performance Goals**: Không suy giảm hiệu suất hoặc thời gian xử lý khi so sánh chuỗi
- **Constraints**: Tuân thủ cấu trúc tách biệt giữa Alias và Struct trong package `types`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Nguyên tắc I & II (Modular Monolith & Module Boundaries)**: Đạt yêu cầu. Thay đổi giới hạn trong module `fashion`, sử dụng DI qua Wire và cấu trúc DTO rõ ràng.
- **Quy tắc ngôn ngữ (Language boundaries)**: Đạt yêu cầu. Comments và định danh code bằng tiếng Anh, tài liệu đặc tả/kế hoạch bằng tiếng Việt.

## Project Structure

### Documentation (this feature)

```text
specs/001-recommendation-alias-types/
├── spec.md              # Đặc tả tính năng
├── plan.md              # Kế hoạch này
└── checklists/
    └── requirements.md  # Bảng chất lượng đặc tả
```

### Source Code

```text
internal/modules/fashion/application/usecase/ai/recommendation/
├── types/
│   ├── aliases.go       # [NEW] Định nghĩa alias và các hằng số
│   └── structs.go       # [NEW] Định nghĩa các struct dùng chung
└── ranking/
    ├── scoring.go       # [MODIFY] Chấm điểm ứng viên (bỏ hằng số trùng lặp)
    └── pool_expansion.go # [MODIFY] Mở rộng tập ứng viên
```

**Structure Decision**: Single module refactoring within `internal/modules/fashion/application/usecase/ai/recommendation`.

## Complexity Tracking

Không có vi phạm hiến pháp cần theo dõi độ phức tạp.
