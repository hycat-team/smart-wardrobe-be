# 👕 SmartWardrobe Backend (Golang Modular Monolith)

SmartWardrobe Backend là hệ thống API dịch vụ được phát triển bằng ngôn ngữ **Golang** theo kiến trúc **Modular Monolith** sạch sẽ (Clean Architecture), hiệu năng cao và dễ bảo trì. Hệ thống cung cấp các nghiệp vụ quản lý tủ đồ thông minh, gợi ý trang phục bằng AI, quản lý gói dịch vụ và các dịch vụ định danh bảo mật (Identity).

---

## 🛠️ Các công cụ cần có trên máy (Prerequisites)

Trước khi bắt đầu, hãy đảm bảo máy tính của bạn đã cài đặt các công cụ sau:

1. **Go Compiler (v1.24.5+)**
    - Tải về và cài đặt tại: [golang.org/dl](https://golang.org/dl/)
    - Xác nhận cài đặt: `go version`
2. **Docker & Docker Desktop (hoặc Docker Compose)**
    - Cần thiết để chạy PostgreSQL (có pgvector) và Redis ở môi trường local.
    - Tải về và cài đặt tại: [docker.com](https://www.docker.com/)
3. **GNU Make**
    - _Windows:_ Nên cài đặt qua [Chocolatey](https://chocolatey.org/) (`choco install make`) hoặc sử dụng Git Bash / WSL.
    - _macOS:_ Đã có sẵn hoặc cài qua Homebrew (`brew install make`).
    - _Linux:_ `sudo apt install make` (Ubuntu/Debian).

---

## 🚀 Hướng dẫn khởi chạy dự án Local

Hãy làm theo các bước dưới đây để thiết lập và chạy dự án trên máy tính của bạn:

### Bước 1: Sao chép dự án & Cấu hình môi trường

1. Di chuyển vào thư mục dự án:
    ```bash
    cd smart-wardrobe-be
    ```
2. Tạo file môi trường `.env` từ file ví dụ:
    ```bash
    cp .env.example .env
    ```
    _Lưu ý:_ Mở file `.env` vừa tạo và chỉnh sửa cấu hình kết nối database, redis, JWT secret, hoặc Gmail SMTP phục vụ việc gửi mã OTP thực tế (nếu cần).

### Bước 2: Khởi chạy dự án bằng Docker Compose

Bạn có hai lựa chọn để chạy ứng dụng local bằng Docker Compose:

#### Lựa chọn A: Chỉ khởi chạy Database & Redis (Được khuyến nghị khi phát triển code Go trực tiếp)

```bash
docker-compose up postgres redis -d
```

_Lưu ý:_ Sau đó chạy ứng dụng Go ở local bằng lệnh `make dev` hoặc `go run`.

#### Lựa chọn B: Khởi chạy toàn bộ stack bao gồm cả Backend App

```bash
docker-compose up -d --build
```

Lệnh này sẽ tự động build image Docker của Backend Go siêu nhẹ (sử dụng multi-stage build chỉ khoảng ~15MB) và chạy kết nối đồng bộ hoàn toàn với database và cache.

_Kiểm tra trạng thái:_ Chạy lệnh `docker ps` để đảm bảo 3 container (`postgres-smartwardrobe`, `redis-smartwardrobe`, `backend-smartwardrobe`) đang hoạt động bình thường.

### Bước 3: Cài đặt các công cụ phát triển chuyên dụng của Go

Dự án sử dụng **Google Wire** (Dependency Injection) và **Swag** (Swagger Generator). Hãy cài đặt chúng bằng lệnh Make được chuẩn bị sẵn:

```bash
make install-tools
```

_(Hoặc chạy thủ công các lệnh sau nếu không có Make)_:

```bash
go install github.com/google/wire/cmd/wire@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## 💻 Quy trình Phát triển & Chạy dự án

Dự án tích hợp sẵn các lệnh tự động hóa trong `Makefile` để tối giản hóa thao tác của lập trình viên.

### Lệnh phát triển nhanh nhất (Full Flow)

Khi bắt đầu code hoặc sau khi sửa bất kỳ phần nào liên quan đến Dependency Injection (khai báo các Provider/Usecase mới) hoặc cập nhật Swagger API:

```bash
make dev
```

Lệnh này sẽ tự động:

1. `go mod tidy` dọn dẹp và tải các dependencies.
2. `wire` sinh mã Dependency Injection tự động.
3. `swag` cập nhật lại tài liệu Swagger UI.
4. Biên dịch và khởi chạy server ngay lập tức.

---

### Chi tiết các lệnh Make khả dụng

| Lệnh                 | Ý nghĩa                                                  | Lệnh chạy tay tương ứng                                                           |
| :------------------- | :------------------------------------------------------- | :-------------------------------------------------------------------------------- |
| `make install-tools` | Cài đặt `wire` và `swag` CLI lên máy                     | `go install ...`                                                                  |
| `make tidy`          | Đồng bộ và tải các package Go bị thiếu                   | `go mod tidy`                                                                     |
| `make wire`          | Sinh mã tự động cho Dependency Injection                 | `wire ./internal/di/...`                                                          |
| `make swagger`       | Tạo lại tài liệu đặc tả API Swagger                      | `swag init -g cmd/server/main.go --output docs --parseDependency --parseInternal` |
| `make build`         | Biên dịch dự án thành file thực thi trong thư mục `bin/` | `go build -o bin/main.exe cmd/server/main.go`                                     |
| `make run`           | Chạy ứng dụng đã biên dịch                               | `./bin/main.exe`                                                                  |
| `make dev`           | Chạy toàn bộ quy trình từ tidy, wire, swagger đến run    | _(Tổ hợp các lệnh trên)_                                                          |
| `make clean`         | Dọn dẹp thư mục binary `bin/`                            | `rm -rf bin/`                                                                     |

---

## 🔍 Kiểm tra kết quả & Trải nghiệm API

Khi server khởi động thành công, console sẽ in ra giao diện log bắt mắt chứa liên kết mở nhanh tài liệu API:

```text
==========================================================
SmartWardrobe BE is running on port: 8080
Swagger UI is available at: http://localhost:8080/swagger
==========================================================
```

- **Tài liệu API (Swagger UI):** Giữ phím `Ctrl` và nhấp vào liên kết [http://localhost:8080/swagger](http://localhost:8080/swagger) để mở trực tiếp tài liệu trên trình duyệt.
