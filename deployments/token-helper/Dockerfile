# deployments/token-helper/Dockerfile — chạy trên VPS cùng Nginx (FR-039).
# Build context = repo root:  docker build -f deployments/token-helper/Dockerfile .
FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY deployments/token-helper/ ./deployments/token-helper/
RUN CGO_ENABLED=0 go build -trimpath -o /out/token-helper ./deployments/token-helper

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/token-helper /token-helper
# Bind localhost only; Nginx gọi qua service name trong compose network.
ENV PORT=8082
EXPOSE 8082
ENTRYPOINT ["/token-helper"]
