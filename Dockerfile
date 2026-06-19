# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.11

FROM golang:${GO_VERSION}-alpine3.24 AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/server/main.go

FROM alpine:3.24.1 AS production

RUN apk add --no-cache ca-certificates tzdata && addgroup -S appgroup && adduser -S -G appgroup appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /out/backend /app/backend
COPY --from=builder --chown=appuser:appgroup /src/docs /app/docs

ENV TZ=Asia/Ho_Chi_Minh

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/backend"]