# Quy Tắc Kiến Trúc Hệ Thống (Architectural Rules)
**Ngày tạo tài liệu:** 2026-07-04 (Cập nhật: 2026-07-05)

Tài liệu này tổng hợp toàn bộ các quy tắc thiết kế kiến trúc hệ thống (Modular Monolith) đã được thảo luận và thống nhất cho dự án Smart Wardrobe Backend. Mọi kế hoạch refactor hoặc phát triển tính năng mới cần phải tuân thủ nghiêm ngặt các quy tắc này.

Xem thêm:
- [Kế Hoạch Refactor Kiến Trúc](20260704_refactor_plan.md)
- [Báo Cáo Phân Tích Vi Phạm Kiến Trúc](architecture_analysis_report.md)

---

## 1. Quy Tắc Về Ranh Giới Module & Layer

### 1.1. Tương tác chéo module (Cross-Module Visibility)
- Sự tương tác, giao tiếp giữa các module khác nhau **chỉ được phép thông qua Contract và DTO**.
- Một module tuyệt đối **không được phép** import trực tiếp các layer mang tính private của module khác, bao gồm:
  - Domain Entities (`domain/entities/...`)
  - Domain Repositories (`domain/repositories/...`)
  - Infrastructure Persistence (`infrastructure/persistence/...`)
  - Application Errors (`application/errors/...`)
  - Usecase Helpers / Mappers nội bộ (`application/usecase/...` hoặc `application/mapper/...`)

### 1.2. Tập trung DTO về `application/dto`
- Tất cả các DTO (Data Transfer Objects) phục vụ cho request, response, hoặc trao đổi thông tin chéo module **phải được tập trung định nghĩa tại thư mục `application/dto/`** của mỗi module.
- Không định nghĩa DTO rải rác trong package `contract` hoặc trực tiếp trong usecase. Package `contract` sẽ import DTO từ `application/dto/` và sử dụng trong khai báo hàm.

---

## 2. Quy Tắc Thiết Kế Contract (Presentation Layer)

Bản chất của Contract là cổng giao tiếp phục vụ tương tác với bên ngoài (sau này khi scale lên Microservices có thể chuyển đổi thành gRPC). Do đó:

### 2.1. Vị trí của Contract trong cấu trúc thư mục
- Contract **không phải là một layer riêng biệt ở root của module**, mà là một package nằm bên trong tầng presentation: **`presentation/contract/`** (ví dụ: `internal/modules/brand/presentation/contract/`).
- Tầng presentation của mỗi module sẽ bao gồm hai nhiệm vụ giao tiếp chính:
  - `presentation/handler/`: Phục vụ các yêu cầu HTTP REST API trực tiếp từ Client/Web.
  - `presentation/contract/`: Phục vụ các yêu cầu tương tác nghiệp vụ từ các module khác gọi sang.

### 2.2. Interface và Struct Implementation trong Contract
- Package `presentation/contract/` phải tự định nghĩa interface và tự triển khai struct implementation của interface đó (ví dụ: `IBrandContract` và struct impl `BrandContract` đều nằm trong `presentation/contract/`).
- Struct triển khai Contract **phải inject các UseCase interface từ tầng `application/interface/usecase/`** để thực thi nghiệp vụ (tương tự cách hoạt động của HTTP Handler). 
- Contract **tuyệt đối không được gọi/inject trực tiếp tầng Repository hoặc Database**.
- Tầng `application` (Usecase) hoàn toàn không biết và không được phép implement các interface của tầng `contract`.

### 2.3. Cấm Inject Contract nội bộ
- Tầng `application` (Usecase, Mapper, Helper, Worker) hoặc tầng `presentation` (Handler) của module `X` **tuyệt đối không được inject hoặc lưu giữ thuộc tính có kiểu dữ liệu là contract của chính module `X`** (`X/presentation/contract.IContractInterface`).
- Nếu các layer nội bộ cần gọi lẫn nhau, chúng phải gọi trực tiếp qua UseCase/Repository nội bộ, không được đi vòng qua Contract.

---

## 3. Quy Tắc Về Hằng Số (Constants) & Lỗi (Errors) Dùng Chung

### 3.1. Hằng số dùng chung toàn hệ thống
- Các hằng số hoặc kiểu enum được chia sẻ chéo module (ví dụ: `roleslug` của module `identity` cần dùng ở module `brand` và `wardrobe`) **phải được di chuyển và khai báo tập trung tại thư mục `internal/shared/domain/constants/`**.
- Không được import trực tiếp constants chéo từ thư mục domain của module này sang module khác.

### 3.2. Quản lý lỗi ứng dụng (Errors)
- Mỗi module tự quản lý lỗi của mình thông qua package errors nội bộ (ví dụ: `fashion/application/errors/`).
- Khi gọi sang contract của module khác và nhận về lỗi, module gọi không được import trực tiếp package errors của module bị gọi để đối chiếu. Hãy sử dụng cơ chế xử lý lỗi chung (`apperror` tiêu chuẩn của hệ thống) hoặc tự định nghĩa lỗi tương ứng trong module của mình.
