package service

import (
	"context"
	"encoding/json"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditService interface {
	Log(ctx context.Context, nodeID uuid.UUID, action, resource, status string, details interface{}) error
	GetLogs(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.AuditLog, int64, error)
}

type auditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) AuditService {
	return &auditService{db: db}
}

func (s *auditService) Log(ctx context.Context, nodeID uuid.UUID, action, resource, status string, details interface{}) error {
	detailsJSON := ""
	if details != nil {
		data, err := json.Marshal(details)
		if err == nil {
			detailsJSON = string(data)
		}
	}

	log := model.AuditLog{
		NodeID:   nodeID,
		Action:   action,
		Resource: resource,
		Details:  detailsJSON,
		Status:   status,
	}

	return s.db.WithContext(ctx).Create(&log).Error
}

func (s *auditService) GetLogs(ctx context.Context, nodeID uuid.UUID, limit, offset int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := s.db.WithContext(ctx).Where("node_id = ?", nodeID)

	if err := query.Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
