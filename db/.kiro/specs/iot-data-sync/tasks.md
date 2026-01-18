# Tasks: Local DB OTA Implementation

## Phase 1: Control Plane Implementation

- [ ] 1. Define SchemaMigration model & repository
    - Create `SchemaMigration` struct in `internal/model`.
    - Create `SchemaRepository` interface and implementation.
    - _Requirements: REQ-1.1, REQ-1.3_

- [ ] 2. Implement Schema Management API
    - Implement `POST /api/v1/schemas` handler.
    - Implement `GET /api/v1/schemas` handler.
    - Add validation logic (version sequence check).
    - _Requirements: REQ-1.1, REQ-1.2, REQ-1.3_

- [ ] 3. Implement Update Notification
    - Integrate with MQTT service to publish `sys/schema/update` events.
    - Trigger notification upon schema registration.
    - _Requirements: REQ-2.1_

## Phase 2: Edge Agent Implementation (Rust)

- [ ] 4. Define Local Migration Manager
    - Create `MigrationManager` struct in `db/edge-agent`.
    - Implement `_schema_migrations` table initialization.
    - _Requirements: REQ-3.3_

- [ ] 5. Implement Schema Download Client
    - Implement HTTP client to fetch SQL from Control Plane.
    - Add authentication headers.
    - _Requirements: REQ-2.2, REQ-2.3_

- [ ] 6. Implement Migration Execution Logic
    - Implement transactional SQL execution.
    - Implement version tracking and sequential application.
    - Implement rollback handling.
    - _Requirements: REQ-3.1, REQ-3.2_

## Phase 3: Integration & Monitoring

- [ ] 7. Implement Status Reporting
    - Add logic to Edge Agent to send status (version/error) via MQTT/HTTP.
    - Implement status receiving handler in Control Plane.
    - _Requirements: REQ-4.1, REQ-4.2_

- [ ] 8. Integration Testing
    - Verify complete flow: Upload -> Notify -> Download -> Apply -> Report.
    - Test failure scenarios (invalid SQL, network error).
    - _Requirements: All_
