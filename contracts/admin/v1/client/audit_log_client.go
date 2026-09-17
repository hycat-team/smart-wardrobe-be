// Package client provides the outbound gRPC adapter for the audit-log contract,
// implementing the same port as the in-process implementation so callers do not
// change when the service boundary is crossed (FR-019).
package client

import (
	"context"

	adminv1 "smart-wardrobe-be/contracts/admin/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	admintransport "smart-wardrobe-be/services/account/contract/admin/transport"
	admindto "smart-wardrobe-be/services/account/dto/admin"

	"google.golang.org/grpc"
)

// AuditLogClient is the remote adapter for IAuditLogContract.
type AuditLogClient struct {
	svc adminv1.AdminServiceClient
}

var _ admincontract.IAuditLogContract = (*AuditLogClient)(nil)

// NewAuditLogClient builds the audit-log contract over gRPC.
func NewAuditLogClient(cc grpc.ClientConnInterface) *AuditLogClient {
	return &AuditLogClient{svc: adminv1.NewAdminServiceClient(cc)}
}

func (c *AuditLogClient) Record(ctx context.Context, input admindto.RecordAuditLogInput) error {
	req, err := admintransport.RecordAuditLogInputToProto(input)
	if err != nil {
		return err
	}
	_, err = c.svc.RecordAuditLog(ctx, req)
	return grpcx.ToAppError(err)
}
