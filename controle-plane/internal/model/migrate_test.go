package model

import (
	"testing"
)

func TestAutoMigrateModelsIncludesFunctionExecutionModels(t *testing.T) {
	models := AutoMigrateModels()

	if !containsModel[*FunctionDefinition](models) {
		t.Fatalf("AutoMigrateModels() missing FunctionDefinition")
	}
	if !containsModel[*FunctionRevision](models) {
		t.Fatalf("AutoMigrateModels() missing FunctionRevision")
	}
	if !containsModel[*FunctionDeploymentTarget](models) {
		t.Fatalf("AutoMigrateModels() missing FunctionDeploymentTarget")
	}
}

func containsModel[T any](models []interface{}) bool {
	for _, model := range models {
		if _, ok := model.(T); ok {
			return true
		}
	}
	return false
}
