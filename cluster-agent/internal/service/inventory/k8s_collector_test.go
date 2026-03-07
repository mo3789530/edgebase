package inventory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sCollector_Collect(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}, Status: corev1.NodeStatus{
			Addresses:  []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "10.0.0.11"}},
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
		}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "fn-a", Namespace: "edge-functions"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "svc-a", Namespace: "edge-functions"}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "fn-a"}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "edge-functions"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	)

	collector := NewK8sCollector(clientset, []string{"edge-functions"})
	inventory, err := collector.Collect(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(inventory.Nodes) != 1 {
		t.Fatalf("nodes len = %d", len(inventory.Nodes))
	}
	if len(inventory.Deployments) != 1 {
		t.Fatalf("deployments len = %d", len(inventory.Deployments))
	}
	if len(inventory.Services) != 1 {
		t.Fatalf("services len = %d", len(inventory.Services))
	}
	if len(inventory.Pods) != 1 {
		t.Fatalf("pods len = %d", len(inventory.Pods))
	}
}
