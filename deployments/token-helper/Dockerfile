# deployments/token-helper/Dockerfile — chạy trên VPS cùng Nginx (FR-039).
# Build context = chính thư mục này (module độc lập):
#   docker build -f deployments/token-helper/Dockerfile deployments/token-helper
FROM golang:1.26 AS builder
WORKDIR /src
# go.mod/go.sum nằm trong context (không dùng go.work) → build standalone.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/token-helper .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/token-helper /token-helper
# Lắng nghe mọi interface để Nginx (container khác) gọi qua service name;
# chỉ expose trong compose network, không publish ra ngoài.
ENV PORT=8082
EXPOSE 8082
ENTRYPOINT ["/token-helper"]
