package contract

import "github.com/gin-gonic/gin"

// IAuthMiddleware xác thực JWT cho request. Các module khác và NewEngine
// chỉ phụ thuộc interface này, không import implementation
// (identity/presentation/middleware).
type IAuthMiddleware interface {
	Handle() gin.HandlerFunc         // Bắt buộc xác thực; 401 nếu thiếu/sai JWT
	OptionalHandle() gin.HandlerFunc // Xác thực nếu có JWT; không fail nếu thiếu
}
