package grpcx

import (
	"context"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// These tests exist because of one fact: before the recovery interceptors, a panic
// in a gRPC handler killed the whole process. The AI service proved it — one nil
// grant reached a log statement, the container restarted six times, and every user
// lost AI rather than one user losing a request.
//
// A test that only called the interceptor function directly would not prove the
// process survives, because the process dying is the behaviour under test and it
// happens in gRPC's own goroutine. So every test here runs a real server on a real
// loopback socket and speaks to it with a real client.
//
// The pattern that makes these tests meaningful: the handler panics ONCE and
// succeeds afterwards. A restart would also make the first call fail, so "the
// first call failed" proves nothing on its own — it is the SECOND call, on the same
// connection, that distinguishes "we converted a panic into an error" from "the
// server died and the test happened to reconnect".

// panicOnce returns a handler that panics on its first invocation and behaves
// normally on every later one.
type panicOnce struct {
	calls atomic.Int32
	// panicValue is what the first call panics with. It is configurable so a test
	// can prove the text does not travel to the client.
	panicValue string
}

func (p *panicOnce) run() error {
	if p.calls.Add(1) == 1 {
		panic(p.panicValue)
	}
	return nil
}

func (p *panicOnce) callCount() int32 { return p.calls.Load() }

// startServer boots a gRPC server on a loopback port with the recovery
// interceptors installed, and returns a client connected to it.
func startServer(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	return startServerWithLog(t, testLogger(), register)
}

func startServerWithLog(t *testing.T, log logger.Interface, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryRecoveryInterceptor(log)),
		grpc.StreamInterceptor(StreamRecoveryInterceptor(log)),
	)
	register(srv)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func testLogger() logger.Interface {
	return logger.New("test", "", "info", false)
}

// TestUnaryPanicBecomesAnErrorAndTheServerSurvives is the core regression.
func TestUnaryPanicBecomesAnErrorAndTheServerSurvives(t *testing.T) {
	po := &panicOnce{panicValue: "nil pointer dereference on the grant"}
	conn := startServer(t, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			if err := po.run(); err != nil {
				return nil, err
			}
			return wrapperspb.String("ok"), nil
		}, nil)
	})

	ctx := context.Background()

	_, err := invokeUnary(ctx, conn)
	if err == nil {
		t.Fatal("a panicking handler must produce an error, not a silent success")
	}
	if got := status.Code(err); got != codes.Internal {
		t.Errorf("the panic must surface as codes.Internal, got = %v", got)
	}

	// The load-bearing assertion: the server is still serving.
	reply, err := invokeUnary(ctx, conn)
	if err != nil {
		t.Fatalf("the server must survive a recovered panic: %v", err)
	}
	if reply.GetValue() != "ok" {
		t.Errorf("the call after a recovered panic must return the real response, got = %q", reply.GetValue())
	}
	if po.callCount() < 2 {
		t.Fatalf("the survival check must have reached the handler again, calls = %d", po.callCount())
	}
}

// The streaming case is the one that mattered in the incident, because the AI chat
// RPC is server-streaming and the panic happened inside it.
func TestStreamPanicBecomesAnErrorAndTheServerSurvives(t *testing.T) {
	po := &panicOnce{panicValue: "nil pointer dereference on the grant"}
	conn := startServer(t, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, nil, func(s *recoveryTestStream) error {
			return po.run()
		})
	})

	ctx := context.Background()

	if err := invokeStream(ctx, conn); err == nil {
		t.Fatal("a panicking stream handler must produce an error")
	}

	if err := invokeStream(ctx, conn); err != nil {
		t.Fatalf("the server must survive a recovered stream panic: %v", err)
	}
	if po.callCount() < 2 {
		t.Fatalf("the survival check must have reached the handler again, calls = %d", po.callCount())
	}
}

// A panic raised by ordinary code rather than an explicit `panic()` is the common
// case — nothing in a handler writes `panic()` on purpose, the runtime does it —
// and it must be recovered by the same defer. A write to a map that was declared
// but never initialised is a real defect of the kind that has happened here.
func TestPanicFromOrdinaryFaultyCodeIsAlsoRecovered(t *testing.T) {
	var calls atomic.Int32
	conn := startServer(t, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			if calls.Add(1) == 1 {
				var m map[string]string
				m["written"] = "value" // runtime panic: assignment to entry in nil map
			}
			return wrapperspb.String("ok"), nil
		}, nil)
	})

	if _, err := invokeUnary(context.Background(), conn); err == nil {
		t.Fatal("a nil map write must be recovered into an error")
	}
	if _, err := invokeUnary(context.Background(), conn); err != nil {
		t.Fatalf("the server must survive it: %v", err)
	}
}

// The client-side mapping is unchanged and still flattens Internal to the generic
// message. This test exists so the behaviour is RECORDED rather than assumed: if
// someone later opens `ToAppError`'s default branch, this fails and they have to
// decide deliberately whether an internal error may now say more.
func TestRecoveredPanicReachesTheClientAsTheGenericMessage(t *testing.T) {
	po := &panicOnce{panicValue: "boom"}
	conn := startServer(t, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			if err := po.run(); err != nil {
				return nil, err
			}
			return wrapperspb.String("ok"), nil
		}, nil)
	})

	_, err := invokeUnary(context.Background(), conn)
	if err == nil {
		t.Fatal("expected an error")
	}

	if mapped := ToAppError(err); mapped.Error() != apperror.GenericInternalMessage {
		t.Errorf("a recovered panic must reach a remote caller as the generic message, got = %q", mapped.Error())
	}
}

// reportPanic must not be able to take down the thing it exists to protect. A nil
// logger is reachable when a server is built by a tool or a test that has no logger
// wired, and a nil dereference inside the recovery path would reintroduce exactly
// the crash it was added to stop.
func TestNilLoggerDoesNotCrashTheRecoveryPath(t *testing.T) {
	po := &panicOnce{panicValue: "boom"}
	conn := startServerWithLog(t, nil, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			if err := po.run(); err != nil {
				return nil, err
			}
			return wrapperspb.String("ok"), nil
		}, nil)
	})

	if _, err := invokeUnary(context.Background(), conn); err == nil {
		t.Fatal("expected an error even with no logger installed")
	}
	if _, err := invokeUnary(context.Background(), conn); err != nil {
		t.Fatalf("the server must survive a panic with no logger: %v", err)
	}
}

// The recovered message must not carry the panic text to the client. The server
// logs the panic value and the stack; the wire gets a generic sentence.
func TestRecoveredPanicTextIsNotSentToTheClient(t *testing.T) {
	po := &panicOnce{panicValue: "pq: password authentication failed for user closy_app"}
	conn := startServer(t, func(srv *grpc.Server) {
		registerRecoveryTestService(srv, func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			if err := po.run(); err != nil {
				return nil, err
			}
			return wrapperspb.String("ok"), nil
		}, nil)
	})

	_, err := invokeUnary(context.Background(), conn)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, secret := range []string{"pq:", "password", "closy_app"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("panic text %q must not reach the client, got = %q", secret, err.Error())
		}
	}
}

// ---- test service plumbing ----

// The service below is hand-written rather than generated so this test needs no
// protoc. It is the smallest shape that still exercises both interceptor kinds: one
// unary method and one server-streaming method.
//
// Its messages are the well-known types from the protobuf module rather than local
// structs. gRPC's default codec requires a real proto.Message, and an empty request
// plus a string reply is all this needs — the payloads are irrelevant, the panics
// are the subject.

type recoveryTestService interface {
	Unary(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error)
	Stream(grpc.ServerStream) error
}

type recoveryTestStream struct {
	grpc.ServerStream
	send func(*wrapperspb.StringValue) error
}

func (s *recoveryTestStream) Send(m *wrapperspb.StringValue) error { return s.send(m) }

type recoveryTestServer struct {
	unary  func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error)
	stream func(*recoveryTestStream) error
}

func (s *recoveryTestServer) Unary(ctx context.Context, req *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return s.unary(ctx, req)
}

// Stream adapts the wire stream to the test's own type so a test handler can call
// Send without importing grpc metadata.
func (s *recoveryTestServer) Stream(stream grpc.ServerStream) error {
	return s.stream(&recoveryTestStream{ServerStream: stream})
}

var recoveryTestServiceDesc = grpc.ServiceDesc{
	ServiceName: "recovery.Test",
	HandlerType: (*recoveryTestService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Unary",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(emptypb.Empty)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(recoveryTestService).Unary(ctx, in)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/recovery.Test/Unary"}
				handler := func(ctx context.Context, req any) (any, error) {
					return srv.(recoveryTestService).Unary(ctx, req.(*emptypb.Empty))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName: "Stream",
			Handler: func(srv any, stream grpc.ServerStream) error {
				return srv.(recoveryTestService).Stream(stream)
			},
			ServerStreams: true,
		},
	},
	Metadata: "recovery_test",
}

func registerRecoveryTestService(
	srv *grpc.Server,
	unary func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error),
	stream func(*recoveryTestStream) error,
) {
	// The descriptor requires both handlers to be present to build, so the one
	// under test gets a no-op sibling.
	if unary == nil {
		unary = func(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
			return wrapperspb.String("ok"), nil
		}
	}
	if stream == nil {
		stream = func(*recoveryTestStream) error { return nil }
	}
	srv.RegisterService(&recoveryTestServiceDesc, &recoveryTestServer{unary: unary, stream: stream})
}

func invokeUnary(ctx context.Context, conn *grpc.ClientConn) (*wrapperspb.StringValue, error) {
	out := new(wrapperspb.StringValue)
	if err := conn.Invoke(ctx, "/recovery.Test/Unary", &emptypb.Empty{}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// invokeStream performs one server-streaming call and drains it to completion.
func invokeStream(ctx context.Context, conn *grpc.ClientConn) error {
	desc := &grpc.StreamDesc{StreamName: "Stream", ServerStreams: true}
	cs, err := conn.NewStream(ctx, desc, "/recovery.Test/Stream")
	if err != nil {
		return err
	}
	if err := cs.SendMsg(&emptypb.Empty{}); err != nil && err != io.EOF {
		return err
	}
	if err := cs.CloseSend(); err != nil {
		return err
	}
	for {
		if err := cs.RecvMsg(&wrapperspb.StringValue{}); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
