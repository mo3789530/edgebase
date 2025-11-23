package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/edgebase/platform/control-plane/internal/storage"
	"github.com/google/uuid"
)

type ArtifactService interface {
	UploadFunction(ctx context.Context, name, version string, binary []byte) (*model.Function, error)
	GetFunction(ctx context.Context, id uuid.UUID) (*model.Function, error)
	GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error)
	DeleteFunction(ctx context.Context, id uuid.UUID) error
}

type artifactService struct {
	repo        repository.FunctionRepository
	minioClient *storage.MinIOClient
}

func NewArtifactService(repo repository.FunctionRepository, minioClient *storage.MinIOClient) ArtifactService {
	return &artifactService{
		repo:        repo,
		minioClient: minioClient,
	}
}

func (s *artifactService) UploadFunction(ctx context.Context, name, version string, binary []byte) (*model.Function, error) {
	// Check if exists
	existing, err := s.repo.GetByNameAndVersion(ctx, name, version)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("function %s version %s already exists", name, version)
	}

	// Calculate hash
	hash := sha256.Sum256(binary)
	hashStr := hex.EncodeToString(hash[:])

	// Upload to MinIO
	objectName := fmt.Sprintf("%s/%s/function.wasm", name, version)
	if err := s.minioClient.Upload(ctx, objectName, binary, "application/wasm"); err != nil {
		return nil, err
	}

	// Save to DB
	fn := &model.Function{
		Name:      name,
		Version:   version,
		Hash:      hashStr,
		SizeBytes: int64(len(binary)),
		MinioPath: objectName,
	}

	if err := s.repo.Create(ctx, fn); err != nil {
		return nil, err
	}

	return fn, nil
}

func (s *artifactService) GetFunction(ctx context.Context, id uuid.UUID) (*model.Function, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *artifactService) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	fn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return s.minioClient.GetPresignedURL(ctx, fn.MinioPath, 15*time.Minute)
}

func (s *artifactService) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
