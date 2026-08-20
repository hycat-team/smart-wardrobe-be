package mapper

import (
	"smart-wardrobe-be/internal/modules/admin/application/dto"
	"smart-wardrobe-be/internal/modules/admin/domain/entities"
)

func MapAdminAuditLogToDTO(log *entities.AdminAuditLog) *dto.AdminAuditLogDTO {
	return &dto.AdminAuditLogDTO{
		ID:          log.ID,
		ActorUserID: log.ActorUserID,
		Action:      log.Action,
		TargetType:  log.TargetType,
		TargetID:    log.TargetID,
		Payload:     map[string]any(log.Payload),
		RequestIP:   log.RequestIP,
		CreatedAt:   log.CreatedAt,
	}
}

func MapAdminAuditLogsToDTO(logs []*entities.AdminAuditLog) []*dto.AdminAuditLogDTO {
	res := make([]*dto.AdminAuditLogDTO, len(logs))
	for i, l := range logs {
		res[i] = MapAdminAuditLogToDTO(l)
	}
	return res
}
