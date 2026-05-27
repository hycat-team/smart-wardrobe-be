# ==========================================================
# STAGE 1: Build stage (Sử dụng Alpine-Go để build tối ưu)
# ==========================================================
FROM golang:1.23-alpine AS builder

# Cài đặt các công cụ hệ thống cần thiết cho quá trình build
RUN apk update && apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency manifests và tải trước để tận dụng Docker Cache
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ mã nguồn dự án
COPY . .

# Biên dịch ứng dụng sang binary tĩnh (statically linked binary)
# Sử dụng flag -s -w để loại bỏ thông tin debug và symbol table giúp giảm ~40% dung lượng file binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main cmd/server/main.go

# ==========================================================
# STAGE 2: Final stage (Sử dụng Alpine siêu nhẹ chỉ ~7MB làm runtime)
# ==========================================================
FROM alpine:3.20

# Cài đặt ca-certificates để gọi HTTPS/API bên ngoài và tzdata để xử lý múi giờ chính xác
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy file thực thi tĩnh từ builder
COPY --from=builder /app/main .

# Copy thư mục docs tĩnh phục vụ tài liệu Swagger UI
COPY --from=builder /app/docs ./docs

# Expose port (sẽ trùng với SERVER_PORT cấu hình trong .env)
EXPOSE 8080

# Chạy ứng dụng backend
CMD ["./main"]
