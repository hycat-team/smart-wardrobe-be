# services/account/Dockerfile — image RIÊNG của Account service (FR-042).
# Build context MUST là repo root (để thấy go.work + shared/ + contracts/):
#   docker build -f services/account/Dockerfile .
# Không dùng chung ARG SERVICE — mỗi service build độc lập thật sự.

FROM golang:1.26 AS builder
WORKDIR /src
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY services/account/go.mod services/account/go.sum ./services/account/
# go.work sync cần thấy go.mod của MỌI module (chỉ copy go.mod, không copy source).
COPY services/commerce/go.mod services/commerce/go.sum ./services/commerce/
COPY services/ai/go.mod services/ai/go.sum ./services/ai/
COPY deployments/token-helper/go.mod ./deployments/token-helper/
RUN go work sync
COPY shared/ ./shared/
COPY contracts/ ./contracts/
COPY services/account/ ./services/account/
WORKDIR /src/services/account
RUN CGO_ENABLED=0 go build -trimpath -o /out/account ./cmd/account

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/account /account
COPY --from=builder /src/services/account/database/migrations /migrations
COPY --from=builder /src/services/account/api/swagger /api/swagger
COPY configs/ /app/config/
# Runtime contract: cấu hình 100% qua env; PORT do Cloud Run inject (mặc định 8080).
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/account", "-healthcheck"]
ENTRYPOINT ["/account"]
