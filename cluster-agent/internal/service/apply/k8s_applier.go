package apply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type K8sApplier struct {
	clientset kubernetes.Interface
}

func NewK8sApplier(clientset kubernetes.Interface) *K8sApplier {
	return &K8sApplier{clientset: clientset}
}

func (a *K8sApplier) Apply(ctx context.Context, plan *model.SyncPlan) (model.SyncAck, error) {
	ack := model.SyncAck{SyncID: plan.SyncID, Success: true, Results: make([]model.SyncAckResource, 0, len(plan.Actions))}

	for _, action := range plan.Actions {
		result := a.applyAction(ctx, action)
		ack.Results = append(ack.Results, result)
		if result.Status == "failed" {
			ack.Success = false
		}
	}

	return ack, nil
}

func (a *K8sApplier) applyAction(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	switch action.Type {
	case model.ActionApplyDeployment:
		return a.applyDeployment(ctx, action)
	case model.ActionApplyService:
		return a.applyService(ctx, action)
	case model.ActionDeleteDeployment:
		return a.deleteDeployment(ctx, action)
	case model.ActionDeleteService:
		return a.deleteService(ctx, action)
	case model.ActionRestartDeployment:
		return a.restartDeployment(ctx, action)
	default:
		return model.SyncAckResource{ResourceType: action.Type, ResourceName: action.Description, Status: "skipped", ErrorMessage: "unsupported action"}
	}
}

type deploymentPayload struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Replicas  *int32            `json:"replicas,omitempty"`
	Port      int32             `json:"port,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type servicePayload struct {
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	Port       int32             `json:"port"`
	TargetPort int32             `json:"target_port,omitempty"`
	Type       string            `json:"type,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
}

type namedPayload struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (a *K8sApplier) applyDeployment(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	payload, err := decode[deploymentPayload](action.Payload)
	if err != nil {
		return failed(action.Type, action.Description, err)
	}
	if payload.Namespace == "" || payload.Name == "" || payload.Image == "" {
		return failed(action.Type, payload.Name, errors.New("namespace, name and image are required"))
	}
	if payload.Replicas == nil {
		defaultReplicas := int32(1)
		payload.Replicas = &defaultReplicas
	}
	if payload.Port == 0 {
		payload.Port = 8080
	}
	if payload.Labels == nil {
		payload.Labels = map[string]string{}
	}
	payload.Labels["edgebase.io/managed-by"] = "cluster-agent"

	depClient := a.clientset.AppsV1().Deployments(payload.Namespace)
	existing, err := depClient.Get(ctx, payload.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return failed(action.Type, payload.Name, err)
	}
	exists := err == nil

	manifest := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: payload.Name, Namespace: payload.Namespace, Labels: payload.Labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: payload.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: payload.Labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: payload.Labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:            payload.Name,
					Image:           payload.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports:           []corev1.ContainerPort{{ContainerPort: payload.Port}},
				}}},
			},
		},
	}

	if !exists {
		if _, err := depClient.Create(ctx, manifest, metav1.CreateOptions{}); err != nil {
			return failed(action.Type, payload.Name, err)
		}
		return applied(action.Type, payload.Name)
	}

	existing.Labels = manifest.Labels
	existing.Spec = manifest.Spec
	if _, err := depClient.Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return failed(action.Type, payload.Name, err)
	}
	return applied(action.Type, payload.Name)
}

func (a *K8sApplier) applyService(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	payload, err := decode[servicePayload](action.Payload)
	if err != nil {
		return failed(action.Type, action.Description, err)
	}
	if payload.Namespace == "" || payload.Name == "" {
		return failed(action.Type, payload.Name, errors.New("namespace and name are required"))
	}
	if payload.Port == 0 {
		payload.Port = 80
	}
	if payload.TargetPort == 0 {
		payload.TargetPort = 8080
	}
	if payload.Type == "" {
		payload.Type = string(corev1.ServiceTypeClusterIP)
	}
	if payload.Selector == nil {
		payload.Selector = map[string]string{"app": payload.Name, "edgebase.io/managed-by": "cluster-agent"}
	}

	svcClient := a.clientset.CoreV1().Services(payload.Namespace)
	existing, err := svcClient.Get(ctx, payload.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return failed(action.Type, payload.Name, err)
	}
	exists := err == nil

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: payload.Name, Namespace: payload.Namespace, Labels: map[string]string{"edgebase.io/managed-by": "cluster-agent"}},
		Spec: corev1.ServiceSpec{
			Selector: payload.Selector,
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       payload.Port,
				TargetPort: intstr.FromInt32(payload.TargetPort),
			}},
			Type: corev1.ServiceType(payload.Type),
		},
	}

	if !exists {
		if _, err := svcClient.Create(ctx, svc, metav1.CreateOptions{}); err != nil {
			return failed(action.Type, payload.Name, err)
		}
		return applied(action.Type, payload.Name)
	}

	clusterIP := existing.Spec.ClusterIP
	existing.Labels = svc.Labels
	existing.Spec = svc.Spec
	existing.Spec.ClusterIP = clusterIP
	if _, err := svcClient.Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return failed(action.Type, payload.Name, err)
	}
	return applied(action.Type, payload.Name)
}

func (a *K8sApplier) deleteDeployment(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	payload, err := decode[namedPayload](action.Payload)
	if err != nil {
		return failed(action.Type, action.Description, err)
	}
	if payload.Namespace == "" || payload.Name == "" {
		return failed(action.Type, payload.Name, errors.New("namespace and name are required"))
	}

	err = a.clientset.AppsV1().Deployments(payload.Namespace).Delete(ctx, payload.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return failed(action.Type, payload.Name, err)
	}
	return deleted(action.Type, payload.Name)
}

func (a *K8sApplier) deleteService(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	payload, err := decode[namedPayload](action.Payload)
	if err != nil {
		return failed(action.Type, action.Description, err)
	}
	if payload.Namespace == "" || payload.Name == "" {
		return failed(action.Type, payload.Name, errors.New("namespace and name are required"))
	}

	err = a.clientset.CoreV1().Services(payload.Namespace).Delete(ctx, payload.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return failed(action.Type, payload.Name, err)
	}
	return deleted(action.Type, payload.Name)
}

func (a *K8sApplier) restartDeployment(ctx context.Context, action model.SyncAction) model.SyncAckResource {
	payload, err := decode[namedPayload](action.Payload)
	if err != nil {
		return failed(action.Type, action.Description, err)
	}
	if payload.Namespace == "" || payload.Name == "" {
		return failed(action.Type, payload.Name, errors.New("namespace and name are required"))
	}

	depClient := a.clientset.AppsV1().Deployments(payload.Namespace)
	existing, err := depClient.Get(ctx, payload.Name, metav1.GetOptions{})
	if err != nil {
		return failed(action.Type, payload.Name, err)
	}
	if existing.Spec.Template.Annotations == nil {
		existing.Spec.Template.Annotations = map[string]string{}
	}
	existing.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().UTC().Format(time.RFC3339)
	if _, err := depClient.Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return failed(action.Type, payload.Name, err)
	}
	return applied(action.Type, payload.Name)
}

func decode[T any](raw json.RawMessage) (T, error) {
	var payload T
	if len(raw) == 0 {
		return payload, errors.New("empty payload")
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}

func failed(resourceType, resourceName string, err error) model.SyncAckResource {
	if resourceName == "" {
		resourceName = "unknown"
	}
	return model.SyncAckResource{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Status:       "failed",
		ErrorMessage: trimErr(err),
	}
}

func applied(resourceType, resourceName string) model.SyncAckResource {
	return model.SyncAckResource{ResourceType: resourceType, ResourceName: resourceName, Status: "applied"}
}

func deleted(resourceType, resourceName string) model.SyncAckResource {
	return model.SyncAckResource{ResourceType: resourceType, ResourceName: resourceName, Status: "deleted"}
}

func trimErr(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 1024 {
		return msg[:1024]
	}
	return msg
}
