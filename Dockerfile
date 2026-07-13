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
COPY --from=builder --chown=appuser:appgroup /src/config/*.yaml /app/config/

ENV TZ=Asia/Ho_Chi_Minh

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/main"]
