// Package grpcx bridges service errors and gRPC status codes so a contract
// observes the same error shape in-process or remote (Constitution II / V).
// Clients use ToAppError; servers use ToGRPCError.
package grpcx

import (
	"errors"
	"net/http"

	"smart-wardrobe-be/shared/application/constants/apperror"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToAppError maps a gRPC transport error to a centralized apperror.
// Server-produced apperror messages (already Vietnamese) are preserved;
// unrecognized codes degrade to a generic message.
func ToAppError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return appErr
	}

	st, ok := status.FromError(err)
	if !ok {
		return apperror.NewUpstreamUnavailable(err)
	}

	switch st.Code() {
	case codes.NotFound:
		return apperror.NewNotFound(st.Message())
	case codes.InvalidArgument, codes.FailedPrecondition:
		return apperror.NewBadRequest(st.Message())
	case codes.Unauthenticated:
		return apperror.NewUnauthorized(st.Message())
	case codes.PermissionDenied:
		return apperror.NewForbidden(st.Message())
	case codes.AlreadyExists, codes.Aborted:
		return apperror.NewConflict(st.Message())
	case codes.ResourceExhausted:
		return apperror.NewTooManyRequest(st.Message())
	case codes.Unavailable, codes.DeadlineExceeded:
		return apperror.NewUpstreamUnavailable(err)
	default:
		fallback := apperror.NewInternalError(apperror.GenericInternalMessage)
		fallback.Cause = err
		return fallback
	}
}

// ToGRPCError converts a centralized apperror into a gRPC status error, so
// remote clients receive the same client-facing message the server intended.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		return status.Error(codes.Internal, apperror.GenericInternalMessage)
	}

	switch appErr.Status {
	case http.StatusNotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case http.StatusBadRequest:
		return status.Error(codes.InvalidArgument, appErr.Message)
	case http.StatusUnauthorized:
		return status.Error(codes.Unauthenticated, appErr.Message)
	case http.StatusForbidden:
		return status.Error(codes.PermissionDenied, appErr.Message)
	case http.StatusConflict:
		return status.Error(codes.AlreadyExists, appErr.Message)
	case http.StatusTooManyRequests:
		return status.Error(codes.ResourceExhausted, appErr.Message)
	case http.StatusServiceUnavailable:
		return status.Error(codes.Unavailable, appErr.Message)
	default:
		return status.Error(codes.Internal, apperror.GenericInternalMessage)
	}
}
