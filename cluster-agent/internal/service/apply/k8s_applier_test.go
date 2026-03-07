package apply

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sApplier_ApplyDeploymentAndDelete(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	applier := NewK8sApplier(clientset)

	createPayload, _ := json.Marshal(map[string]any{
		"namespace": "edge-functions",
		"name":      "fn-hello-v1",
		"image":     "registry.local/hello:v1",
		"replicas":  1,
		"port":      8080,
		"labels": map[string]string{
			"app": "fn-hello-v1",
		},
	})
	plan := &model.SyncPlan{
		SyncID: uuid.New(),
		Actions: []model.SyncAction{{
			Type:    model.ActionApplyDeployment,
			Order:   1,
			Payload: createPayload,
		}},
	}

	ack, err := applier.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !ack.Success {
		t.Fatalf("ack.Success = false, results=%+v", ack.Results)
	}

	if _, err := clientset.AppsV1().Deployments("edge-functions").Get(context.Background(), "fn-hello-v1", metav1.GetOptions{}); err != nil {
		t.Fatalf("deployment not created: %v", err)
	}

	deletePayload, _ := json.Marshal(map[string]any{"namespace": "edge-functions", "name": "fn-hello-v1"})
	deletePlan := &model.SyncPlan{
		SyncID: uuid.New(),
		Actions: []model.SyncAction{{
			Type:    model.ActionDeleteDeployment,
			Order:   1,
			Payload: deletePayload,
		}},
	}

	ack, err = applier.Apply(context.Background(), deletePlan)
	if err != nil {
		t.Fatalf("Apply(delete) error = %v", err)
	}
	if !ack.Success {
		t.Fatalf("ack.Success(delete) = false, results=%+v", ack.Results)
	}

	if _, err := clientset.AppsV1().Deployments("edge-functions").Get(context.Background(), "fn-hello-v1", metav1.GetOptions{}); err == nil {
		t.Fatal("deployment still exists")
	}
}

func TestK8sApplier_ApplyService(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "fn-hello-v1", Namespace: "edge-functions"}},
	)
	applier := NewK8sApplier(clientset)

	payload, _ := json.Marshal(map[string]any{
		"namespace":   "edge-functions",
		"name":        "fn-hello",
		"port":        80,
		"target_port": 8080,
		"selector":    map[string]string{"app": "fn-hello-v1"},
	})

	plan := &model.SyncPlan{SyncID: uuid.New(), Actions: []model.SyncAction{{Type: model.ActionApplyService, Payload: payload}}}
	ack, err := applier.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !ack.Success {
		t.Fatalf("ack.Success = false, results=%+v", ack.Results)
	}

	svc, err := clientset.CoreV1().Services("edge-functions").Get(context.Background(), "fn-hello", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service not created: %v", err)
	}
	if got := svc.Spec.Ports[0].Port; got != 80 {
		t.Fatalf("service port = %d, want 80", got)
	}
}

func TestK8sApplier_UnsupportedActionSkipped(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "edge-functions"}})
	applier := NewK8sApplier(clientset)

	ack, err := applier.Apply(context.Background(), &model.SyncPlan{
		SyncID:  uuid.New(),
		Actions: []model.SyncAction{{Type: "ADD_FUNCTION", Description: "legacy"}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !ack.Success {
		t.Fatalf("ack.Success = false")
	}
	if len(ack.Results) != 1 || ack.Results[0].Status != "skipped" {
		t.Fatalf("unexpected results: %+v", ack.Results)
	}
}
