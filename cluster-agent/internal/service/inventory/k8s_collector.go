package inventory

import (
	"context"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sCollector struct {
	clientset  kubernetes.Interface
	namespaces []string
}

func NewK8sCollector(clientset kubernetes.Interface, namespaces []string) *K8sCollector {
	return &K8sCollector{clientset: clientset, namespaces: namespaces}
}

func (c *K8sCollector) Collect(ctx context.Context, clusterID uuid.UUID) (model.ClusterInventory, error) {
	inventory := model.ClusterInventory{
		ClusterID:  clusterID,
		ObservedAt: time.Now().UTC(),
	}

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.ClusterInventory{}, err
	}
	inventory.Nodes = toNodeInfo(nodes.Items)

	for _, namespace := range c.namespaces {
		deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return model.ClusterInventory{}, err
		}
		inventory.Deployments = append(inventory.Deployments, toDeployments(deployments.Items)...)

		services, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return model.ClusterInventory{}, err
		}
		inventory.Services = append(inventory.Services, toServices(services.Items)...)

		pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return model.ClusterInventory{}, err
		}
		inventory.Pods = append(inventory.Pods, toPods(pods.Items)...)
	}

	if version, err := c.clientset.Discovery().ServerVersion(); err == nil {
		inventory.KubernetesVersion = version.String()
	}

	return inventory, nil
}

func toNodeInfo(nodes []corev1.Node) []model.NodeInfo {
	result := make([]model.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, model.NodeInfo{
			Name:       node.Name,
			Role:       nodeRole(node),
			InternalIP: nodeInternalIP(node),
			Status:     nodeStatus(node),
		})
	}
	return result
}

func toDeployments(deployments []appsv1.Deployment) []model.Deployment {
	result := make([]model.Deployment, 0, len(deployments))
	for _, deployment := range deployments {
		image := ""
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			image = deployment.Spec.Template.Spec.Containers[0].Image
		}
		result = append(result, model.Deployment{
			Namespace:         deployment.Namespace,
			Name:              deployment.Name,
			Image:             image,
			ReadyReplicas:     deployment.Status.ReadyReplicas,
			AvailableReplicas: deployment.Status.AvailableReplicas,
		})
	}
	return result
}

func toServices(services []corev1.Service) []model.Service {
	result := make([]model.Service, 0, len(services))
	for _, service := range services {
		result = append(result, model.Service{
			Namespace: service.Namespace,
			Name:      service.Name,
			Selector:  service.Spec.Selector,
		})
	}
	return result
}

func toPods(pods []corev1.Pod) []model.PodInfo {
	result := make([]model.PodInfo, 0, len(pods))
	for _, pod := range pods {
		result = append(result, model.PodInfo{
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Status:    string(pod.Status.Phase),
			NodeName:  pod.Spec.NodeName,
		})
	}
	return result
}

func nodeRole(node corev1.Node) string {
	if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
		return "control-plane"
	}
	if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
		return "control-plane"
	}
	return "worker"
}

func nodeInternalIP(node corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func nodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}
