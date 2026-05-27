# 🛠️ SmartWardrobe Backend - Development & Technical Guidelines

Tài liệu này đóng vai trò là **Cẩm nang Kỹ thuật & Chuẩn Phát triển (Development Handbook)** bắt buộc áp dụng cho toàn bộ các lập trình viên khi xây dựng các module và tính năng mới trên hệ thống SmartWardrobe Backend.

Chúng ta lấy **Module Identity** làm chuẩn mực (baseline/gold standard) cho cấu trúc thư mục, quy trình luân chuyển dữ liệu, xử lý lỗi, viết tài liệu API Swagger và Dependency Injection.

---

## 📁 1. Cấu trúc Module Chuẩn (Lấy Identity làm gốc)

Mỗi Module phải được đóng gói gọn gàng bên trong thư mục `internal/modules/<module_name>` và chia thành các tầng độc lập theo nguyên lý Clean Architecture:

```text
📁 internal/modules/identity
│
├── 📁 domain                      # 1. Tầng Nghiệp vụ cốt lõi (Domain Layer)
│    └── 📁 repositories           # Chứa interface định nghĩa các Repository (ví dụ: user_repo.go)
│
├── 📁 application                 # 2. Tầng Ứng dụng (Application Layer)
│    ├── 📁 contract               # Interface & Implementation giao tiếp liên module (Loose Coupling)
│    ├── 📁 dto                    # Data Transfer Objects (Request/Response structs)
│    ├── 📁 mapper                 # Chuyển đổi dữ liệu (ví dụ: Mapper Entity <-> DTO)
│    ├── 📁 usecase                # Nghiệp vụ dòng chảy (AuthUseCase, UserUseCase)
│    └── 📄 provider.go            # Wire Provider Set của riêng tầng Application
│
├── 📁 infrastructure              # 3. Tầng Hạ tầng (Infrastructure Layer)
│    ├── 📁 persistence            # Triển khai thực tế các repository bằng GORM/Postgres DB
│    ├── 📁 caching                # Triển khai Redis cache (ví dụ: lưu OTP)
│    ├── 📁 security               # Các dịch vụ bảo mật (Bcrypt hashing, blacklist token)
│    ├── 📁 communication          # Các dịch vụ gửi Mail SMTP, SMS
│    └── 📄 provider.go            # Wire Provider Set của riêng tầng Infrastructure
│
├── 📁 presentation                # 4. Tầng Giao tiếp (Presentation Layer)
│    ├── 📁 handler                # Bộ xử lý Request/Response (AuthHandler, MeHandler)
│    └── 📄 provider.go            # Wire Provider Set của riêng tầng Presentation
│
└── 📄 provider.go                 # 5. File gom nhóm Provider toàn module (Outermost Provider Set)
```

---

## 🏗️ 2. Quy chuẩn Dependency Injection (Google Wire)

Không khai báo thủ công hay chồng chéo các Provider. Mỗi module phải có cấu trúc Provider phân cấp:

1. **Từng Layer con (`application`, `infrastructure`, `presentation`)**: Tự gom nhóm các service của mình vào biến `ProviderSet` tương ứng tại file `provider.go` của layer đó.
2. **File `provider.go` ngoài cùng của module**: Gom nhóm các `ProviderSet` của từng layer con:

    ```go
    package identity

    import (
        "smart-wardrobe-be/internal/modules/identity/application"
        "smart-wardrobe-be/internal/modules/identity/infrastructure"
        "smart-wardrobe-be/internal/modules/identity/presentation"
        "github.com/google/wire"
    )

    var ProviderSet = wire.NewSet(
        presentation.ProviderSet,
        application.ProviderSet,
        infrastructure.ProviderSet,
    )
    ```

3. **DI toàn cục**: Đăng ký `identity.ProviderSet` tại tệp tin `internal/di/wire.go`.

---

## 📡 3. Tiêu chuẩn viết Handler (Presentation Layer)

Đây là chuẩn mực **bắt buộc** khi triển khai các Handler để đảm bảo tính nhất quán của API Response, cơ chế quản lý lỗi tập trung và tự động đồng bộ hóa tài liệu Swagger.

### 🚫 Quy tắc 1: Tuyệt đối KHÔNG sử dụng `c.JSON()` trực tiếp

Để tránh tình trạng mỗi API trả về một định dạng dữ liệu khác nhau, các Handler **không được phép** tự gọi `c.JSON(http.StatusOK, ...)` của Gin. Thay vào đó, phải gọi các hàm tiện ích đã chuẩn hóa tại package `shared_pres` (`smart-wardrobe-be/internal/shared/presentation`):

- Trả về phản hồi thành công thường (`200 OK`):
    ```go
    shared_pres.Success(c, "Thông điệp thành công", data)
    ```
- Trả về phản hồi tạo mới thành công (`211 Created`):
    ```go
    shared_pres.Created(c, "Tạo mới dữ liệu thành công", data)
    ```

### 🚫 Quy tắc 2: Handler phải trả về `error` & Sử dụng `WrapHandler`

Mỗi phương thức của Handler phải có signature chuẩn: `func (h *MyHandler) MyMethod(c *gin.Context) error`.

- **Khi thành công**: Trả về `nil` ở cuối hàm sau khi đã gọi `shared_pres.Success`.
- **Khi có lỗi**: Trả về trực tiếp đối tượng lỗi (`return err`) để hệ thống tự động bắt và xử lý tập trung, tuyệt đối không tự bắt lỗi rồi viết Response tại Handler.
- **Đăng ký Route**: Phải bọc Handler bằng `shared_pres.WrapHandler`:
    ```go
    group.POST("/login", shared_pres.WrapHandler(h.authHandler.Login))
    ```

---

## 📝 4. Tiêu chuẩn viết Swagger Annotations (swag)

Mỗi phương thức Handler phải được viết chú thích Swagger bằng tiếng Việt rõ ràng, đầy đủ các trường thông tin. Cấu trúc chuẩn:

```go
// Register register a new user account
// @Summary Đăng ký tài khoản               <- Tóm tắt ngắn gọn tính năng
// @Description Đăng ký tài khoản mới cho người dùng và gửi OTP xác thực qua email <- Mô tả chi tiết
// @Tags Auth                               <- Nhóm API (Auth, Me, v.v.)
// @Accept json                              <- Định dạng request nhận vào
// @Produce json                             <- Định dạng response trả ra
// @Param body body dto.RegisterReq true "Thông tin đăng ký"  <- Dữ liệu đầu vào
// @Success 200 {object} shared_pres.APIResponse <- Định dạng phản hồi khi thành công
// @Router /api/v1/auth/register [post]     <- Đường dẫn và phương thức HTTP
func (h *AuthHandler) Register(c *gin.Context) error { ... }
```

> [!IMPORTANT]
> Đối với các API có trả về dữ liệu (Data) cụ thể, chú thích `@Success` phải khai báo đúng DTO đại diện để Swagger render cấu trúc dữ liệu chính xác cho Frontend:
> `// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes} "Thông tin người dùng"`

---

## ⚠️ 5. Quy chuẩn Xử lý Lỗi & Xác thực dữ liệu (Error Handling & Validation)

### A. Xác thực Dữ liệu Đầu vào (Validation)

Mọi dữ liệu dạng JSON gửi lên từ client phải được validate tại Handler bằng Gin Binding (`c.ShouldBindJSON`). Nếu xảy ra lỗi validate, phải dịch lỗi sang ngôn ngữ dễ hiểu và trả về thông qua hàm tiện ích `validation.TranslateValidationError`:

```go
var input dto.RegisterReq
if err := c.ShouldBindJSON(&input); err != nil {
    // Dịch lỗi và trả trực tiếp lỗi validate về cho WrapHandler xử lý
    return validation.TranslateValidationError(err)
}
```

### B. Trả về Lỗi Nghiệp vụ từ Usecase/Service

Các tầng bên dưới (`Usecase`, `Service`, `Repository`) khi xảy ra lỗi nghiệp vụ phải ném ra lỗi có cấu trúc chuẩn từ gói `errorcode` (`smart-wardrobe-be/internal/shared/application/constants/errorcode`):

- Lỗi không tìm thấy dữ liệu:
    ```go
    return errorcode.NewNotFound("Không tìm thấy người dùng với ID đã cung cấp.")
    ```
- Lỗi dữ liệu xung đột (đã tồn tại):
    ```go
    return errorcode.NewConflict("Email này đã được đăng ký trên hệ thống.")
    ```
- Lỗi không có quyền truy cập:
    ```go
    return errorcode.NewUnauthorized("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.")
    ```
- Lỗi dữ liệu không hợp lệ:
    ```go
    return errorcode.NewBadRequest("Thông tin xác thực OTP không chính xác.")
    ```
- Và các lỗi khác tại gói errorcode...

Hạ tầng `WrapHandler` phối hợp cùng Middleware Xử lý Lỗi toàn cục (`GlobalErrorHandler`) sẽ tự động bắt các lỗi trên, phân tích mã HTTP phù hợp và serialize thành JSON phản hồi đồng bộ cho Client. Người lập trình tuyệt đối không viết code parse thủ công HTTP status tại Handler!
