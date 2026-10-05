# shared/tools/rabbitmq-topology/Dockerfile — image RIÊNG cho tool reconcile
# RabbitMQ topology. Không nhét tool vào image của service: nó chạy thủ công
# trên box (migrate) và trong CI (check), không phải một workload dài hạn.
#
# Build context MUST là repo root (để thấy go.work + shared/):
#   docker build -f shared/tools/rabbitmq-topology/Dockerfile .

FROM golang:1.26 AS builder
WORKDIR /src
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY services/account/go.mod services/account/go.sum ./services/account/
# go.work sync cần thấy go.mod của MỌI module (chỉ copy go.mod, không copy source).
COPY services/commerce/go.mod services/commerce/go.sum ./services/commerce/
COPY services/ai/go.mod services/ai/go.sum ./services/ai/
RUN go work sync
# Chỉ phụ thuộc go.mod/go.sum đã copy ở trên: sửa source không kéo lại layer này.
RUN for m in shared contracts services/account services/commerce services/ai; do \
        (cd "$m" && go mod download); \
    done
COPY shared/ ./shared/
COPY contracts/ ./contracts/
WORKDIR /src/shared
RUN CGO_ENABLED=0 go build -trimpath -o /out/rabbitmq-topology ./tools/rabbitmq-topology

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/rabbitmq-topology /rabbitmq-topology
ENTRYPOINT ["/rabbitmq-topology"]