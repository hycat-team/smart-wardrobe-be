# deployments/monolith-legacy/Dockerfile — image rollback monolith FROZEN (FR-043).
#
# File này sống trên nhánh 020 nhưng BUILD TỪ working tree của tag `monolith-legacy`
# (không phải từ nhánh hiện tại). Quy trình (xem Makefile target
# `image-build-monolith-rollback` — T007):
#   1. git worktree add /tmp/monolith-legacy monolith-legacy
#   2. cp deployments/monolith-legacy/Dockerfile /tmp/monolith-legacy/Dockerfile.legacy
#   3. docker build -f /tmp/monolith-legacy/Dockerfile.legacy /tmp/monolith-legacy
#   4. push GHCR monolith-legacy:<tag-sha> MỘT LẦN trước khi hết cửa sổ funded (T097a).
#
# Nội dung mirror root Dockerfile tại thời điểm tag (alpine + cmd/server).
# Chỉ phục vụ nhánh rollback (compose profile `rollback`) — MUST NOT nhận code mới.

# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/main ./cmd/server

FROM alpine:3.20 AS production

RUN apk add --no-cache ca-certificates tzdata && addgroup -S appgroup && adduser -S -G appgroup appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /out/main /app/main
COPY --from=builder --chown=appuser:appgroup /src/api /app/api
COPY --from=builder --chown=appuser:appgroup /src/configs/*.yaml /app/config/

ENV TZ=Asia/Ho_Chi_Minh

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/main"]
