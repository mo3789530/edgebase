package apply

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestK8sApplier_ApplyDeploymentAndDelete(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	applier := NewK8sApplier(clientset, nil)

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
	applier := NewK8sApplier(clientset, nil)

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
	applier := NewK8sApplier(clientset, nil)

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

func TestK8sApplier_ApplyKService(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	applier := NewK8sApplier(clientset, dynamicClient)

	payload, _ := json.Marshal(map[string]any{
		"apiVersion": "serving.knative.dev/v1",
		"kind":       "Service",
		"metadata": map[string]any{
			"name":      "telemetry-normalizer",
			"namespace": "edge-functions",
			"labels": map[string]string{
				"edgebase.io/function-name": "telemetry-normalizer",
			},
		},
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						"autoscaling.knative.dev/min-scale": "0",
						"autoscaling.knative.dev/max-scale": "20",
					},
				},
				"spec": map[string]any{
					"containerConcurrency": int64(10),
					"timeoutSeconds":       int64(3),
					"containers": []map[string]any{{
						"image": "registry.local/telemetry-normalizer@sha256:abcd",
						"ports": []map[string]any{{
							"containerPort": int64(8080),
						}},
						"env": []map[string]any{{
							"name":  "MODE",
							"value": "prod",
						}},
					}},
				},
			},
			"traffic": []map[string]any{{
				"latestRevision": true,
				"percent":        int64(100),
			}},
		},
	})

	ack, err := applier.Apply(context.Background(), &model.SyncPlan{
		SyncID:  uuid.New(),
		Actions: []model.SyncAction{{Type: model.ActionApplyKService, Payload: payload}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !ack.Success {
		t.Fatalf("ack.Success = false, results=%+v", ack.Results)
	}

	obj, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group: "serving.knative.dev", Version: "v1", Resource: "services",
	}).Namespace("edge-functions").Get(context.Background(), "telemetry-normalizer", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("kservice not created: %v", err)
	}
	if obj.GetName() != "telemetry-normalizer" {
		t.Fatalf("name = %s", obj.GetName())
	}
	if obj.GetLabels()["edgebase.io/managed-by"] != "cluster-agent" {
		t.Fatalf("managed-by label not injected: %+v", obj.GetLabels())
	}

	spec, ok, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !ok {
		t.Fatalf("spec not found: ok=%v err=%v", ok, err)
	}
	if _, ok := spec["traffic"]; !ok {
		t.Fatalf("traffic not found in spec: %+v", spec)
	}
}
