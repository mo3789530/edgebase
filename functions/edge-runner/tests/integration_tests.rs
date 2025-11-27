use edge_runner::infrastructure::{
    InMemoryLocalFunctionRepository, InMemoryLocalDeploymentRepository,
    ArtifactDownloader, RouteManager, HeartbeatManager, MetricsCollector,
    SecurityManager, RateLimiter, VersionManager, LocalFunctionRepository,
};
use edge_runner::domain::{Function, Route};

#[test]
fn test_deployment_to_execution_flow() {
    // Setup
    let func_repo = std::sync::Arc::new(InMemoryLocalFunctionRepository::new());
    let dep_repo = std::sync::Arc::new(InMemoryLocalDeploymentRepository::new());
    let downloader = ArtifactDownloader::new(
        "http://localhost:9000".to_string(),
        "minioadmin".to_string(),
        "minioadmin".to_string(),
    );

    // Create function
    let function = Function::new(
        "fn1".to_string(),
        "test_function".to_string(),
        1,
        "main".to_string(),
        256,
        5000,
        "http://example.com/fn.wasm".to_string(),
        "abc123".to_string(),
    ).unwrap();

    // Store function
    func_repo.create(function.clone()).unwrap();

    // Verify function is stored
    let retrieved = func_repo.get("fn1").unwrap();
    assert!(retrieved.is_some());
    assert_eq!(retrieved.unwrap().name, "test_function");
}

#[test]
fn test_routing_and_metrics_flow() {
    // Setup
    let route_manager = RouteManager::new();
    let metrics = MetricsCollector::new();

    // Register route
    let route = Route {
        id: "r1".to_string(),
        host: "localhost".to_string(),
        path: "/api/test".to_string(),
        function_id: "fn1".to_string(),
        methods: vec!["POST".to_string()],
        priority: 0,
    };

    route_manager.register(route).unwrap();

    // Lookup route
    let found = route_manager.lookup("localhost", "/api/test", "POST");
    assert_eq!(found, Some("fn1".to_string()));

    // Record metrics
    metrics.record_invocation(100, true);
    metrics.record_cache_hit();

    assert_eq!(metrics.get_total_invocations(), 1);
    assert_eq!(metrics.get_cache_hits(), 1);
}

#[test]
fn test_security_and_rate_limiting_flow() {
    // Setup
    let security = SecurityManager::new();
    let rate_limiter = RateLimiter::new(60);

    // Generate and validate API key
    let key = security.generate_api_key();
    security.add_api_key(key.clone()).unwrap();
    assert!(security.validate_api_key(&key).unwrap());

    // Register quota
    let quota = edge_runner::infrastructure::ResourceQuota::new(
        "fn1".to_string(),
        512,
        5000,
        10,
    );
    rate_limiter.register_quota(quota).unwrap();

    // Check rate limit
    for _ in 0..5 {
        assert!(rate_limiter.check_rate_limit("fn1", 10).unwrap());
    }

    assert_eq!(rate_limiter.get_request_count("fn1"), 5);
}

#[test]
fn test_version_management_and_rollback_flow() {
    // Setup
    let version_manager = VersionManager::new();

    // Register versions
    let v1 = edge_runner::infrastructure::FunctionVersion::new(
        "fn1".to_string(),
        1,
        "http://example.com/fn1.wasm".to_string(),
        "abc123".to_string(),
    );
    let v2 = edge_runner::infrastructure::FunctionVersion::new(
        "fn1".to_string(),
        2,
        "http://example.com/fn2.wasm".to_string(),
        "def456".to_string(),
    );

    version_manager.register_version(v1).unwrap();
    version_manager.register_version(v2).unwrap();

    // Set v2 as active
    version_manager.set_active_version("fn1", 2).unwrap();
    let active = version_manager.get_active_version("fn1").unwrap();
    assert_eq!(active.version, 2);

    // Rollback to v1
    let rolled_back = version_manager.rollback("fn1").unwrap();
    assert_eq!(rolled_back, 1);

    let active = version_manager.get_active_version("fn1").unwrap();
    assert_eq!(active.version, 1);
}

#[test]
fn test_heartbeat_and_metrics_collection_flow() {
    // Setup
    let heartbeat_manager = HeartbeatManager::new("node1".to_string(), "pop1".to_string());
    let metrics = MetricsCollector::new();

    // Create heartbeat
    let mut payload = heartbeat_manager.create_heartbeat();

    // Add function info
    payload.add_cached_function("fn1".to_string(), 1, "cached".to_string());

    // Set metrics
    let node_metrics = edge_runner::infrastructure::NodeMetrics {
        memory_usage_mb: 512,
        cpu_usage_percent: 25.5,
        active_instances: 5,
        total_invocations: 1000,
        error_count: 10,
    };
    payload.set_metrics(node_metrics);

    // Verify heartbeat
    assert_eq!(payload.function_count, 1);
    assert_eq!(payload.metrics.memory_usage_mb, 512);

    // Record metrics
    metrics.record_invocation(100, true);
    metrics.record_invocation(150, false);

    assert_eq!(metrics.get_total_invocations(), 2);
    assert_eq!(metrics.get_total_errors(), 1);
}

#[test]
fn test_error_handling_and_fallback_flow() {
    // Setup
    let fallback_manager = edge_runner::infrastructure::FallbackManager::new();
    let circuit_breaker = edge_runner::infrastructure::CircuitBreaker::new(3);

    // Register fallback
    fallback_manager.register_fallback("fn1".to_string(), "v1".to_string()).unwrap();
    assert_eq!(fallback_manager.get_fallback("fn1"), Some("v1".to_string()));

    // Test circuit breaker
    assert!(!circuit_breaker.is_open());

    circuit_breaker.record_failure();
    circuit_breaker.record_failure();
    circuit_breaker.record_failure();

    assert!(circuit_breaker.is_open());

    circuit_breaker.attempt_reset();
    assert_eq!(
        circuit_breaker.get_state(),
        edge_runner::infrastructure::CircuitBreakerState::HalfOpen
    );
}

#[test]
fn test_complete_function_lifecycle() {
    // Setup all components
    let func_repo = std::sync::Arc::new(InMemoryLocalFunctionRepository::new());
    let route_manager = RouteManager::new();
    let version_manager = VersionManager::new();
    let metrics = MetricsCollector::new();
    let security = SecurityManager::new();

    // 1. Register function
    let function = Function::new(
        "fn1".to_string(),
        "my_function".to_string(),
        1,
        "main".to_string(),
        256,
        5000,
        "http://example.com/fn.wasm".to_string(),
        "abc123".to_string(),
    ).unwrap();

    func_repo.create(function).unwrap();

    // 2. Register route
    let route = Route {
        id: "r1".to_string(),
        host: "localhost".to_string(),
        path: "/api/my-function".to_string(),
        function_id: "fn1".to_string(),
        methods: vec!["POST".to_string()],
        priority: 0,
    };

    route_manager.register(route).unwrap();

    // 3. Register version
    let version = edge_runner::infrastructure::FunctionVersion::new(
        "fn1".to_string(),
        1,
        "http://example.com/fn.wasm".to_string(),
        "abc123".to_string(),
    );

    version_manager.register_version(version).unwrap();

    // 4. Verify all components
    assert!(func_repo.get("fn1").unwrap().is_some());
    assert_eq!(
        route_manager.lookup("localhost", "/api/my-function", "POST"),
        Some("fn1".to_string())
    );
    assert!(version_manager.get_active_version("fn1").is_some());

    // 5. Record metrics
    metrics.record_invocation(100, true);
    metrics.record_cache_hit();

    assert_eq!(metrics.get_total_invocations(), 1);
    assert_eq!(metrics.get_cache_hits(), 1);

    // 6. Verify security
    let key = security.generate_api_key();
    security.add_api_key(key.clone()).unwrap();
    assert!(security.validate_api_key(&key).unwrap());
}
