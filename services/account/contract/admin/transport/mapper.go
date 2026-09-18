package transport

import (
	adminv1 "smart-wardrobe-be/contracts/admin/v1"
	admindto "smart-wardrobe-be/services/account/dto/admin"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

// RecordRequestToInput converts the wire audit record into the contract input.
func RecordRequestToInput(req *adminv1.RecordAuditLogRequest) (admindto.RecordAuditLogInput, error) {
	var out admindto.RecordAuditLogInput
	if req == nil {
		return out, apperror.NewBadRequest("Thiếu nội dung nhật ký kiểm toán.")
	}

	actorID, err := uuid.Parse(req.GetActorUserId())
	if err != nil {
		return out, apperror.From(err)
	}
	out.ActorUserID = actorID
	out.Action = req.GetAction()
	out.TargetType = req.GetTargetType()

	if req.GetHasTargetId() {
		targetID, err := uuid.Parse(req.GetTargetId())
		if err != nil {
			return out, apperror.From(err)
		}
		out.TargetID = &targetID
	}
	if req.GetPayload() != nil {
		out.Payload = req.GetPayload().AsMap()
	}
	if req.GetHasRequestIp() {
		ip := req.GetRequestIp()
		out.RequestIP = &ip
	}
	return out, nil
}

// RecordAuditLogInputToProto converts the contract input into its wire form.
// The free-form payload crosses as google.protobuf.Struct (JSON-compatible).
func RecordAuditLogInputToProto(input admindto.RecordAuditLogInput) (*adminv1.RecordAuditLogRequest, error) {
	out := &adminv1.RecordAuditLogRequest{
		ActorUserId: input.ActorUserID.String(),
		Action:      input.Action,
		TargetType:  input.TargetType,
	}
	if input.TargetID != nil {
		out.TargetId = input.TargetID.String()
		out.HasTargetId = true
	}
	if len(input.Payload) > 0 {
		fields, err := structpb.NewStruct(input.Payload)
		if err != nil {
			wrapErr := apperror.NewInternalError(apperror.GenericInternalMessage)
			wrapErr.Cause = err
			return nil, wrapErr
		}
		out.Payload = fields
	}
	if input.RequestIP != nil {
		out.RequestIp = *input.RequestIP
		out.HasRequestIp = true
	}
	return out, nil
}
