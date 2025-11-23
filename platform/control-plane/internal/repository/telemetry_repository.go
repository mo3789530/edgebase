package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TelemetryRepository interface {
	InsertBatch(ctx context.Context, batch []model.TelemetryData) (int, error)
	GetPendingCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error)
	UpdateCommandStatus(ctx context.Context, commandID uuid.UUID, success bool) error
	GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error)
	RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error)
	UpdateDeviceLastSeen(ctx context.Context, deviceID uuid.UUID) error
}

type telemetryRepository struct {
	db *gorm.DB
}

func NewTelemetryRepository(db *gorm.DB) TelemetryRepository {
	return &telemetryRepository{db: db}
}

func (r *telemetryRepository) InsertBatch(ctx context.Context, batch []model.TelemetryData) (int, error) {
	tx := r.db.WithContext(ctx).Begin()
	inserted := 0

	for _, data := range batch {
		var existing model.TelemetryData
		result := tx.Where("id = ?", data.ID).First(&existing)

		if result.Error == nil {
			// Conflict: use Last-Write-Wins
			if data.Version > existing.Version {
				if err := tx.Model(&existing).Updates(data).Error; err != nil {
					tx.Rollback()
					return 0, err
				}
				inserted++
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			// Insert new
			if err := tx.Create(&data).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
			inserted++
		} else {
			tx.Rollback()
			return 0, result.Error
		}
	}

	if len(batch) > 0 {
		if err := tx.Model(&model.SyncStatus{}).
			Where("device_id = ?", batch[0].DeviceID).
			Updates(map[string]interface{}{
				"last_sync_at":         gorm.Expr("NOW()"),
				"last_sync_status":     "success",
				"total_synced_records": gorm.Expr("total_synced_records + ?", inserted),
			}).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return inserted, tx.Commit().Error
}

func (r *telemetryRepository) GetPendingCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error) {
	var commands []model.Command
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, "pending").
		Order("created_at ASC").
		Limit(100).
		Find(&commands).Error
	return commands, err
}

func (r *telemetryRepository) UpdateCommandStatus(ctx context.Context, commandID uuid.UUID, success bool) error {
	status := "executed"
	if !success {
		status = "failed"
	}
	return r.db.WithContext(ctx).
		Model(&model.Command{}).
		Where("id = ?", commandID).
		Updates(map[string]interface{}{
			"status":      status,
			"executed_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *telemetryRepository) GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error) {
	var status model.SyncStatus
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		First(&status).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (r *telemetryRepository) RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error) {
	deviceID := uuid.New()
	tx := r.db.WithContext(ctx).Begin()

	device := model.Device{
		ID:         deviceID,
		DeviceName: name,
		DeviceType: deviceType,
		Location:   &location,
		Status:     "active",
	}
	if err := tx.Create(&device).Error; err != nil {
		tx.Rollback()
		return uuid.Nil, err
	}

	syncStatus := model.SyncStatus{
		ID:       uuid.New(),
		DeviceID: deviceID,
	}
	if err := tx.Create(&syncStatus).Error; err != nil {
		tx.Rollback()
		return uuid.Nil, err
	}

	return deviceID, tx.Commit().Error
}

func (r *telemetryRepository) UpdateDeviceLastSeen(ctx context.Context, deviceID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ?", deviceID).
		Update("last_seen_at", gorm.Expr("NOW()")).Error
}
