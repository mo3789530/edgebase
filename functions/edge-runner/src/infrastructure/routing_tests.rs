#[cfg(test)]
mod tests {
    use crate::domain::*;
    use crate::infrastructure::*;
    use std::sync::Arc;

    #[tokio::test]
    async fn test_exact_path_match() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/api/users", "GET").await;
        assert!(result.is_some());
        assert_eq!(result.unwrap().function_id, "func1");
    }

    #[tokio::test]
    async fn test_path_parameter_extraction() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users/:id".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/api/users/123", "GET").await;
        assert!(result.is_some());
        let route_match = result.unwrap();
        assert_eq!(route_match.function_id, "func1");
        assert_eq!(route_match.path_params.get("id"), Some(&"123".to_string()));
    }

    #[tokio::test]
    async fn test_multiple_path_parameters() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users/:user_id/posts/:post_id".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/api/users/123/posts/456", "GET").await;
        assert!(result.is_some());
        let route_match = result.unwrap();
        assert_eq!(route_match.path_params.get("user_id"), Some(&"123".to_string()));
        assert_eq!(route_match.path_params.get("post_id"), Some(&"456".to_string()));
    }

    #[tokio::test]
    async fn test_prefix_wildcard_match() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/*".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result1 = repo.match_route("localhost", "/api/users", "GET").await;
        let result2 = repo.match_route("localhost", "/api/posts", "GET").await;
        
        assert!(result1.is_some());
        assert!(result2.is_some());
    }

    #[tokio::test]
    async fn test_root_wildcard_match() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/*".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/any/path", "GET").await;
        assert!(result.is_some());
    }

    #[tokio::test]
    async fn test_method_matching() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string(), "POST".to_string()],
            priority: 100,
        }).await;

        let get_result = repo.match_route("localhost", "/api/users", "GET").await;
        let post_result = repo.match_route("localhost", "/api/users", "POST").await;
        let put_result = repo.match_route("localhost", "/api/users", "PUT").await;

        assert!(get_result.is_some());
        assert!(post_result.is_some());
        assert!(put_result.is_none());
    }

    #[tokio::test]
    async fn test_wildcard_method() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["*".to_string()],
            priority: 100,
        }).await;

        let get_result = repo.match_route("localhost", "/api/users", "GET").await;
        let post_result = repo.match_route("localhost", "/api/users", "POST").await;
        let delete_result = repo.match_route("localhost", "/api/users", "DELETE").await;

        assert!(get_result.is_some());
        assert!(post_result.is_some());
        assert!(delete_result.is_some());
    }

    #[tokio::test]
    async fn test_host_matching() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "example.com".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let match_result = repo.match_route("example.com", "/api/users", "GET").await;
        let no_match_result = repo.match_route("other.com", "/api/users", "GET").await;

        assert!(match_result.is_some());
        assert!(no_match_result.is_none());
    }

    #[tokio::test]
    async fn test_priority_ordering() {
        let repo = InMemoryRouteRepository::new();
        
        // 低優先度ルート
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/*".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 10,
        }).await;

        // 高優先度ルート
        repo.add_route(Route {
            id: "r2".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func2".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/api/users", "GET").await;
        assert!(result.is_some());
        assert_eq!(result.unwrap().function_id, "func2");
    }

    #[tokio::test]
    async fn test_no_match() {
        let repo = InMemoryRouteRepository::new();
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        let result = repo.match_route("localhost", "/api/posts", "GET").await;
        assert!(result.is_none());
    }

    #[tokio::test]
    async fn test_list_routes() {
        let repo = InMemoryRouteRepository::new();
        
        repo.add_route(Route {
            id: "r1".to_string(),
            host: "*".to_string(),
            path: "/api/users".to_string(),
            function_id: "func1".to_string(),
            methods: vec!["GET".to_string()],
            priority: 100,
        }).await;

        repo.add_route(Route {
            id: "r2".to_string(),
            host: "*".to_string(),
            path: "/api/posts".to_string(),
            function_id: "func2".to_string(),
            methods: vec!["GET".to_string()],
            priority: 50,
        }).await;

        let routes = repo.list_routes().await;
        assert_eq!(routes.len(), 2);
        // 優先度順にソートされている
        assert_eq!(routes[0].function_id, "func1");
        assert_eq!(routes[1].function_id, "func2");
    }
}
