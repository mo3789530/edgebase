package model

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&Node{},
		&Function{},
		&FunctionDefinition{},
		&FunctionRevision{},
		&FunctionDeploymentTarget{},
		&RouteDefinition{},
		&SchemaMigration{},
		&NodeFunctionDeployment{},
		&SyncRecord{},
		&Device{},
		&TelemetryData{},
		&Command{},
		&SyncStatus{},
		&NodeSchemaStatus{},
		&AuditLog{},
		&ClusterInventorySnapshot{},
	}
}
