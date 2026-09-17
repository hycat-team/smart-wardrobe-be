// Package grpc provides the inbound gRPC adapter for the audit-log contract.
// The server depends on the contract interface — never on use cases directly
// (Constitution II/IV, T039).
package grpc

import (
	"context"

	adminv1 "smart-wardrobe-be/contracts/admin/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	admintransport "smart-wardrobe-be/services/account/contract/admin/transport"
)

// AdminServer is the inbound adapter for AdminService.
type AdminServer struct {
	adminv1.UnimplementedAdminServiceServer
	auditLog admincontract.IAuditLogContract
}

var _ adminv1.AdminServiceServer = (*AdminServer)(nil)

// NewAdminServer builds the admin service bound to the contract port.
func NewAdminServer(auditLog admincontract.IAuditLogContract) *AdminServer {
	return &AdminServer{auditLog: auditLog}
}

func (s *AdminServer) RecordAuditLog(ctx context.Context, req *adminv1.RecordAuditLogRequest) (*adminv1.RecordAuditLogResponse, error) {
	input, err := admintransport.RecordRequestToInput(req)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	if err := s.auditLog.Record(ctx, input); err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &adminv1.RecordAuditLogResponse{}, nil
}
