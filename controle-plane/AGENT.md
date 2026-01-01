# Edgebase Control Plane Coding Conventions

## 1. Architecture
- **Layered Structure**: Follow the Handler -> Service -> Repository pattern.
- **Handler Layer (`internal/handler`)**: Responsible for HTTP routing (Fiber), request parsing, validation, and error response mapping.
- **Service Layer (`internal/service`)**: Contains core business logic. Defines interfaces to facilitate testing/mocking.
- **Repository Layer (`internal/repository`)**: Handles data persistence (GORM).
- **Models (`internal/model`)**: Defines database schemas and common data structures.

## 2. Frameworks & Libraries
- **Web Framework**: [Fiber v2](https://gofiber.io/)
- **ORM**: [GORM](https://gorm.io/) with PostgreSQL.
- **Validation**: Custom validator in `internal/validator`.
- **Testing**: [Testify](https://github.com/stretchr/testify) (assert/require).
- **MQTT**: Eclipse Paho MQTT.
- **Storage**: MinIO Go SDK.

## 3. Error Handling
- Use `internal/errors` for consistent API responses.
- Services should return standard Go `error`.
- Handlers are responsible for mapping service errors or validation errors to HTTP status codes using helper functions:
  - `errors.BadRequest(c, message, details)`
  - `errors.Unauthorized(c, message)`
  - `errors.NotFound(c, message)`
  - `errors.InternalError(c, message)`

## 4. Logging
- Use `internal/logger` for JSON structured logging.
- Always pass `requestID` to logging functions.
- `requestID` should be retrieved from Fiber context (`logger.GetRequestID(c)` or `c.Locals("request_id")`).
- Methods: `Debug`, `Info`, `Warn`, `Error`.

## 5. Naming Conventions
- **Go Symbols**: Standard Go CamelCase (e.g., `RegisterNode`, `nodeService`).
- **JSON Tags**: snake_case (e.g., `json:"auth_token_hash"`).
- **Database Columns**: GORM default (snake_case).
- **Log Keys**: snake_case for consistency with JSON.

## 6. Dependency Injection
- Use constructor functions (e.g., `NewXxxService`, `NewXxxRepository`).
- Pass dependencies as interfaces to allow mocking in tests.
- Handlers are grouped under a single `Handler` struct that holds all service dependencies.

## 7. Context Usage
- Always propagate `context.Context` from the HTTP handler down to the repository layer.
- Use `c.Context()` from Fiber to get the request context.

## 8. Testing
- Place tests in the same package as the code (e.g., `xxx_test.go`).
- Use mocks for dependencies (Service/Repository interfaces).
- Focus on unit tests for services and integration tests for handlers.

## 9. Verification
- Before completing any task, always ensure that the following commands pass:
  - `make build`: Verifies the project compiles.
  - `make lint`: Runs `go vet` and other linting checks.
  - `make test`: Executes all unit and integration tests.
