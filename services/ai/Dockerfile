# services/ai/Dockerfile — image RIÊNG của AI Engine service (FR-042).
# Build context MUST là repo root (để thấy go.work + shared/ + contracts/):
#   docker build -f services/ai/Dockerfile .
# Không dùng chung ARG SERVICE — mỗi service build độc lập thật sự.

FROM golang:1.25 AS builder
WORKDIR /src
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY shared/go.mod ./shared/
# NOTE: thêm `COPY shared/go.sum ./shared/` khi shared/ có external dependency (T009+).
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY services/ai/go.mod services/ai/go.sum ./services/ai/
# go.work sync cần thấy go.mod của MỌI module (chỉ copy go.mod, không copy source).
COPY services/account/go.mod services/account/go.sum ./services/account/
COPY services/commerce/go.mod services/commerce/go.sum ./services/commerce/
RUN go work sync
COPY shared/ ./shared/
COPY contracts/ ./contracts/
COPY services/ai/ ./services/ai/
WORKDIR /src/services/ai
RUN CGO_ENABLED=0 go build -trimpath -o /out/ai ./cmd/ai

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/ai /ai
COPY --from=builder /src/services/ai/database/migrations /migrations
# Runtime contract: cấu hình 100% qua env; PORT do Cloud Run inject (mặc định 8080).
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/ai", "-healthcheck"]
ENTRYPOINT ["/ai"]
