// Recovery interceptors for the gRPC servers.
//
// # WHY THIS FILE EXISTS
//
// The HTTP servers have had panic recovery for a long time: the shared error
// middleware calls recover() and answers with apperror.PanicRecoveryMessage. The
// gRPC servers did not. `grpc.NewServer()` was called bare, in every service that
// hosts one.
//
// The gap is not cosmetic. A panic in a gRPC handler is unrecovered in the
// handler's own goroutine, which brings down the WHOLE PROCESS — not one request.
// A single nil pointer dereference in one handler therefore removed an entire
// service for every user at once. That is exactly what happened to the AI
// service: one account with no policy grant reached `EntitlementReader.Read`,
// which dereferenced the nil grant while building a log field, and the container
// came back with RestartCount=6 and a panic repeating in the log.
//
// So this file is not hardening in the abstract. It is the difference between one
// failed request and one outage.
//
// WHAT THE CLIENT ACTUALLY SEES — stated plainly, because it is the part that is
// easy to over-promise.
//
// The status returned here is codes.Internal carrying PanicRecoveryMessage. The
// client-side `ToAppError` in this same package maps codes.Internal to
// apperror.GenericInternalMessage, so a remote caller ends up seeing the generic
// sentence, not the panic message. That mapping is deliberate (FR-005 phase 9:
// never leak raw internal text) and it is NOT changed here.
//
// So the value of these interceptors is: the process survives, and the log line
// carries the method, the panic value and the stack. The message improvement is a
// side effect at best, and promising otherwise would be a lie written into a
// comment where the next person would rely on it.
//
// # WHY NO NEW DEPENDENCY
//
// grpc-ecosystem/go-grpc-middleware/recovery does this and is battle-tested. It
// is not in go.mod, and apperror.PanicRecoveryMessage — the message this
// repository already decided a client should see after a panic — is not
// configurable in it. Writing the two interceptors here keeps the panic policy in
// one place with the rest of the error mapping, and keeps the dependency count
// unchanged. The cost is roughly forty lines that are easy to read.
package grpcx

import (
	"context"
	"fmt"
	"runtime/debug"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// recoveredMessage is the log line. It is always Error: a panic is a defect in
// this repository, not a condition an operator can act on, and a line that only
// appears at Warn would read as noise.
const recoveredMessage = "recovered from panic while serving a gRPC call"

// panicStatus is what the caller receives. It is built with status.Error rather
// than ToGRPCError, because ToGRPCError's default branch would replace the
// message with GenericInternalMessage before it left this function — which is the
// same message anyway at the client, but only after an avoidable round of
// flattening that hides which of the two messages this function chose.
func panicStatus() error {
	return status.Error(codes.Internal, apperror.PanicRecoveryMessage)
}

// reportPanic writes the one record that makes the outage diagnosable.
//
// The stack is included because without it a panic in a gRPC handler is close to
// undebuggable: the goroutine that died is not the goroutine that received the
// request, so there is no other trail back to the handler. The method name is
// included because it narrows the search before anyone reads the stack.
func reportPanic(log logger.Interface, method string, recovered any) {
	if log == nil {
		return
	}
	log.Error(recoveredMessage,
		zap.String("method", method),
		zap.String("panic", fmt.Sprint(recovered)),
		zap.String("stack", string(debug.Stack())),
	)
}

// UnaryRecoveryInterceptor converts a panic in a unary handler into an error.
//
// The `err` result is the load-bearing part: returning it is what lets the gRPC
// server finish the RPC cleanly. A named return is required for the deferred
// recover to modify it.
func UnaryRecoveryInterceptor(log logger.Interface) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				reportPanic(log, info.FullMethod, r)
				resp, err = nil, panicStatus()
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor is the streaming counterpart, and it is the one that
// mattered in the incident: the AI chat RPC is server-streaming.
//
// LIMITATION, because a comment claiming this fully recovers a stream would be
// wrong. Once a stream has written its response headers, the status is already
// committed to the wire and the client will see a transport-level error or a
// truncated stream rather than the status set here. That is inherent to gRPC, not
// to this implementation: there is no way to replace bytes already sent. What this
// interceptor still guarantees is the important half — the process does not die,
// so every OTHER in-flight call and the next request are unaffected.
//
// The wrapped error is returned so an early panic, before anything is written,
// still reaches the caller as a clean status.
func StreamRecoveryInterceptor(log logger.Interface) grpc.StreamServerInterceptor {
	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				reportPanic(log, info.FullMethod, r)
				err = panicStatus()
			}
		}()
		return handler(srv, stream)
	}
}
