# services/commerce/Dockerfile — image RIÊNG của Commerce Core service (FR-042).
# Build context MUST là repo root (để thấy go.work + shared/ + contracts/):
#   docker build -f services/commerce/Dockerfile .
# Không dùng chung ARG SERVICE — mỗi service build độc lập thật sự.

FROM golang:1.26 AS builder
WORKDIR /src
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY services/commerce/go.mod services/commerce/go.sum ./services/commerce/
# go.work sync cần thấy go.mod của MỌI module (chỉ copy go.mod, không copy source).
COPY services/account/go.mod services/account/go.sum ./services/account/
COPY services/ai/go.mod services/ai/go.sum ./services/ai/
RUN go work sync
# Chỉ phụ thuộc go.mod/go.sum đã copy ở trên: sửa source không kéo lại layer này.
RUN for m in shared contracts services/account services/commerce services/ai; do \
        (cd "$m" && go mod download); \
    done
COPY shared/ ./shared/
COPY contracts/ ./contracts/
COPY services/commerce/ ./services/commerce/
# Cross-service contract deps (Constitution §II): contracts/.../client imports
# services/account/{contract,dto} (+transport → account internal) and
# services/ai/{contract,dto}, so their sources must be present to compile.
COPY services/account/ ./services/account/
COPY services/ai/ ./services/ai/
WORKDIR /src/services/commerce
RUN CGO_ENABLED=0 go build -trimpath -o /out/commerce ./cmd/commerce

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/commerce /commerce
COPY --from=builder /src/services/commerce/database/migrations /migrations
COPY --from=builder /src/services/commerce/api/swagger /api/swagger
COPY configs/ /app/config/
# Runtime contract: cấu hình 100% qua env; PORT do Cloud Run inject (mặc định 8080).
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/commerce", "-healthcheck"]
ENTRYPOINT ["/commerce"]
