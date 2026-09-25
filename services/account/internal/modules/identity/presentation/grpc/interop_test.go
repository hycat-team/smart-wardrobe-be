package grpc_test

import (
	"context"
	"net"
	"testing"

	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	client "smart-wardrobe-be/contracts/identity/v1/client"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	identitygrpc "smart-wardrobe-be/services/account/internal/modules/identity/presentation/grpc"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type stubBriefContract struct {
	brief *identitydto.UserBriefContractDTO
}

func (s *stubBriefContract) GetByID(context.Context, uuid.UUID) (*identitydto.UserBriefContractDTO, error) {
	return s.brief, nil
}

func (s *stubBriefContract) GetByIDs(context.Context, []uuid.UUID) ([]*identitydto.UserBriefContractDTO, error) {
	return []*identitydto.UserBriefContractDTO{s.brief}, nil
}

func (s *stubBriefContract) GetByUsernamesOrEmails(context.Context, []string) ([]*identitydto.UserBriefContractDTO, error) {
	return []*identitydto.UserBriefContractDTO{s.brief}, nil
}

func (s *stubBriefContract) SearchUserBriefs(context.Context, string, int) ([]*identitydto.UserBriefContractDTO, error) {
	return []*identitydto.UserBriefContractDTO{s.brief}, nil
}

var _ identitycontract.IUserBriefContract = (*stubBriefContract)(nil)

// TestClientServerInterop proves the remote adapter and the inbound server
// agree on the wire contract: a DTO served in-process round-trips through a
// real gRPC server back into an equal DTO (T055).
func TestClientServerInterop(t *testing.T) {
	avatar := "https://cdn.example.com/a.png"
	brief := &identitydto.UserBriefContractDTO{
		ID:        uuid.New(),
		Username:  "alice",
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Nguyen",
		Gender:    gender.Female,
		Status:    userstatus.Active,
		AvatarUrl: &avatar,
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(server, identitygrpc.NewIdentityServer(&stubBriefContract{brief: brief}))
	go func() { _ = server.Serve(lis) }()
	defer server.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	c := client.NewUserBriefClient(conn)
	got, err := c.GetByID(context.Background(), brief.ID)
	if err != nil {
		t.Fatalf("client call failed: %v", err)
	}

	if got.ID != brief.ID || got.Username != brief.Username || got.Email != brief.Email {
		t.Fatalf("interop identity mismatch: %+v", got)
	}
	if got.Gender != brief.Gender || got.Status != brief.Status {
		t.Fatalf("interop enum mismatch: %+v", got)
	}
	if got.AvatarUrl == nil || *got.AvatarUrl != avatar {
		t.Fatalf("interop avatar mismatch: %+v", got.AvatarUrl)
	}
}
