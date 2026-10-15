# Hợp đồng gRPC — delta của 029 (additive)

> Hai file proto thay đổi, toàn bộ là **field mới cộng thêm** (không đổi số field cũ, không xoá).
> Regen: `make buf-gen`. Chuỗi cập nhật adapter: xem `../research.md` R2.

## 1. `contracts/subscription/v1/subscription.proto`

```proto
message AdminSubscriptionStats {
  string total_subscription_revenue = 1;   // KHÔNG đổi: chỉ direct_purchase, status=success
  int32 payos_transaction_count = 2;       // ĐỔI NGỮ NGHĨA: cả direct_purchase + wallet_topup
  int32 payos_success_count = 3;           // ĐỔI NGỮ NGHĨA: như trên
  int32 payos_pending_count = 4;           // ĐỔI NGỮ NGHĨA: như trên
  int32 payos_failed_count = 5;            // ĐỔI NGỮ NGHĨA: như trên
  int32 payos_voided_count = 6;            // ĐỔI NGỮ NGHĨA: như trên
  string total_payos_success_value = 7;    // MỚI: SUM(amount) status=success, cả hai type
}

message GetAdminSubscriptionStatsRequest {
  google.protobuf.Timestamp snapshot_date = 1;  // giữ cho tương thích wire (server cũ dùng)
  google.protobuf.Timestamp from_date = 2;      // MỚI: cận dưới (bao gồm), 00:00 VN
  google.protobuf.Timestamp to_date = 3;        // MỚI: cận trên (loại trừ), 00:00 VN ngày kế tiếp
}

message GetSubscriptionDistributionRequest {
  google.protobuf.Timestamp from_date = 1;      // MỚI (rỗng = hiện trạng)
  google.protobuf.Timestamp to_date = 2;        // MỚI
}

message GetRenewalStatsRequest {
  google.protobuf.Timestamp from_date = 1;      // MỚI (rỗng = toàn thời gian)
  google.protobuf.Timestamp to_date = 2;        // MỚI
}
```

Quy tắc phía server (account gRPC `SubscriptionDashboardServer`):
- `GetAdminSubscriptionStats`: có `from_date`/`to_date` → dùng cặp `[from, to)`; ngược lại dùng
  `VNDayRange(snapshot_date)` (hành vi cũ).
- `GetSubscriptionDistribution`, `GetRenewalStats`: có cặp ngày → truyền xuống usecase; rỗng →
  zero-time (giữ hành vi cũ).

## 2. `contracts/ai/v1/ai.proto`

```proto
message GetAIErrorBreakdownRequest {
  google.protobuf.Timestamp from_date = 1;   // MỚI (rỗng = all-time, hành vi cũ)
  google.protobuf.Timestamp to_date = 2;     // MỚI
}
```

Quy tắc phía server (AI gRPC `AIDashboardServer.GetAIErrorBreakdown`): có cặp ngày → lọc
`created_at >= from AND created_at < to`; rỗng → giữ all-time.

## 3. Interface Go (hai phía phải đổi cùng lúc khi build)

```go
// services/account/contract/subscription/dashboard_reader.go
GetAdminSubscriptionStats(ctx, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error)
GetSubscriptionDistribution(ctx, from, to time.Time) ([]dto.SubscriptionDistributionDTO, error)
GetRenewalStats(ctx, from, to time.Time) (*dto.RenewalStatsDTO, error)
// GetRevenueBreakdown giữ chữ ký; đổi ngữ nghĩa: to là cận nửa mở

// services/ai/contract/aiexecutor/dashboard_reader.go
GetAIErrorBreakdown(ctx, from, to time.Time) ([]dto.AIErrorBreakdownDTO, error)
```

- In-process adapter: `services/account/contract/subscription/dashboard_reader.go`,
  `services/ai/contract/aiexecutor/dashboard_reader.go`.
- gRPC client: `contracts/subscription/v1/client/dashboard_reader_client.go`,
  `contracts/ai/v1/client/dashboard_reader_client.go`.
- Transport mapper: cập nhật `AdminStatsToProto/DTO` (field 7), thêm map cặp ngày.

## 4. Tương thích rolling deploy

- Field additive: service cũ bỏ qua field mới (không lỗi wire).
- Client mới + server cũ: server trả kết quả phạm vi cũ (ngày snapshot / all-time) — chấp nhận
  tạm cho dashboard nội bộ; thứ tự deploy khuyến nghị: account/ai trước commerce.
