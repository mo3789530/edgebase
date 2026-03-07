package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionControllerService interface {
	GetClusterSyncPlan(ctx context.Context, clusterID uuid.UUID) (*SyncPlan, error)
	AcknowledgeClusterSync(ctx context.Context, clusterID, syncID uuid.UUID, result SyncResult) error
}

type functionControllerService struct {
	targetRepo     repository.FunctionDeploymentTargetRepository
	definitionRepo repository.FunctionDefinitionRepository
	revisionRepo   repository.FunctionRevisionRepository
	inventoryRepo  repository.InventoryRepository
}

const (
	clusterActionApplyDeployment  SyncActionType = "APPLY_DEPLOYMENT"
	clusterActionApplyService     SyncActionType = "APPLY_SERVICE"
	clusterActionDeleteDeployment SyncActionType = "DELETE_DEPLOYMENT"
	clusterActionDeleteService    SyncActionType = "DELETE_SERVICE"
)

func NewFunctionControllerService(
	targetRepo repository.FunctionDeploymentTargetRepository,
	definitionRepo repository.FunctionDefinitionRepository,
	revisionRepo repository.FunctionRevisionRepository,
	inventoryRepo repository.InventoryRepository,
) FunctionControllerService {
	return &functionControllerService{
		targetRepo:     targetRepo,
		definitionRepo: definitionRepo,
		revisionRepo:   revisionRepo,
		inventoryRepo:  inventoryRepo,
	}
}

func (s *functionControllerService) GetClusterSyncPlan(ctx context.Context, clusterID uuid.UUID) (*SyncPlan, error) {
	targets, err := s.targetRepo.ListByClusterID(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	observed, err := s.getObservedResources(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	actions := make([]SyncAction, 0, len(targets)*2)
	desiredDeployments := make(map[string]string, len(targets))
	desiredServices := make(map[string]string, len(targets))
	order := 1

	for _, target := range targets {
		definition, err := s.definitionRepo.GetByID(ctx, target.FunctionDefinitionID)
		if err != nil {
			return nil, err
		}
		revision, err := s.revisionRepo.GetByID(ctx, target.DesiredRevisionID)
		if err != nil {
			return nil, err
		}

		deploymentName := deploymentName(definition.Name, revision.Version)
		serviceName := serviceName(definition.Name)
		desiredDeployments[key(target.Namespace, deploymentName)] = deploymentName
		desiredServices[key(target.Namespace, serviceName)] = serviceName

		actions = append(actions, SyncAction{
			Type:  clusterActionApplyDeployment,
			Order: order,
			Payload: map[string]interface{}{
				"namespace": target.Namespace,
				"name":      deploymentName,
				"image":     buildImageReference(revision.Image, revision.ImageDigest),
				"replicas":  target.Replicas,
				"port":      revision.Port,
				"labels": map[string]string{
					"app":                    deploymentName,
					"edgebase.io/managed-by": "cluster-agent",
					"edgebase.io/function":   definition.Name,
					"edgebase.io/revision":   revision.Version,
				},
			},
			Description: fmt.Sprintf("apply deployment %s", deploymentName),
		})
		order++

		actions = append(actions, SyncAction{
			Type:  clusterActionApplyService,
			Order: order,
			Payload: map[string]interface{}{
				"namespace":   target.Namespace,
				"name":        serviceName,
				"port":        int32(80),
				"target_port": revision.Port,
				"type":        "ClusterIP",
				"selector": map[string]string{
					"app":                    deploymentName,
					"edgebase.io/managed-by": "cluster-agent",
				},
			},
			Description: fmt.Sprintf("apply service %s", serviceName),
		})
		order++
	}

	for resourceKey, deployment := range observed.deployments {
		if !strings.HasPrefix(deployment.Name, "fn-") {
			continue
		}
		if _, ok := desiredDeployments[resourceKey]; ok {
			continue
		}
		actions = append(actions, SyncAction{
			Type:  clusterActionDeleteDeployment,
			Order: order,
			Payload: map[string]interface{}{
				"namespace": deployment.Namespace,
				"name":      deployment.Name,
			},
			Description: fmt.Sprintf("delete deployment %s", deployment.Name),
		})
		order++
	}

	for resourceKey, service := range observed.services {
		if !strings.HasPrefix(service.Name, "fn-") {
			continue
		}
		if _, ok := desiredServices[resourceKey]; ok {
			continue
		}
		actions = append(actions, SyncAction{
			Type:  clusterActionDeleteService,
			Order: order,
			Payload: map[string]interface{}{
				"namespace": service.Namespace,
				"name":      service.Name,
			},
			Description: fmt.Sprintf("delete service %s", service.Name),
		})
		order++
	}

	return &SyncPlan{
		SyncID:     uuid.New(),
		Generation: time.Now().UTC().UnixMilli(),
		Actions:    actions,
	}, nil
}

func (s *functionControllerService) AcknowledgeClusterSync(ctx context.Context, clusterID, syncID uuid.UUID, result SyncResult) error {
	status := "applied"
	if !result.Success {
		status = "failed"
	}
	return s.targetRepo.UpdateStatusByClusterID(ctx, clusterID, status)
}

type observedResources struct {
	deployments map[string]inventoryDeployment
	services    map[string]observedInventoryService
}

type inventoryPayload struct {
	Deployments []inventoryDeployment      `json:"deployments"`
	Services    []observedInventoryService `json:"services"`
}

type inventoryDeployment struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type observedInventoryService struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (s *functionControllerService) getObservedResources(ctx context.Context, clusterID uuid.UUID) (observedResources, error) {
	snapshot, err := s.inventoryRepo.GetLatestSnapshot(ctx, clusterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return observedResources{
				deployments: map[string]inventoryDeployment{},
				services:    map[string]observedInventoryService{},
			}, nil
		}
		return observedResources{}, err
	}

	var payload inventoryPayload
	if err := json.Unmarshal([]byte(snapshot.Payload), &payload); err != nil {
		return observedResources{}, fmt.Errorf("unmarshal inventory payload: %w", err)
	}

	result := observedResources{
		deployments: make(map[string]inventoryDeployment, len(payload.Deployments)),
		services:    make(map[string]observedInventoryService, len(payload.Services)),
	}
	for _, deployment := range payload.Deployments {
		result.deployments[key(deployment.Namespace, deployment.Name)] = deployment
	}
	for _, service := range payload.Services {
		result.services[key(service.Namespace, service.Name)] = service
	}
	return result, nil
}

func deploymentName(functionName, version string) string {
	return sanitizeK8sName("fn-" + functionName + "-" + version)
}

func serviceName(functionName string) string {
	return sanitizeK8sName("fn-" + functionName)
}

func sanitizeK8sName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

func buildImageReference(image, digest string) string {
	if digest == "" || strings.Contains(image, "@") {
		return image
	}
	return image + "@" + digest
}

func key(namespace, name string) string {
	return namespace + "/" + name
}
