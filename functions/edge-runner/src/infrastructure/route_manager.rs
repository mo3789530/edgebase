use crate::domain::Route;
use std::sync::{Arc, RwLock};

pub struct RouteManager {
    routes: Arc<RwLock<Vec<Route>>>,
}

impl RouteManager {
    pub fn new() -> Self {
        RouteManager {
            routes: Arc::new(RwLock::new(Vec::new())),
        }
    }

    pub fn register(&self, route: Route) -> Result<(), String> {
        let mut routes = self.routes.write().unwrap();
        
        // Check for duplicate
        if routes.iter().any(|r| r.id == route.id) {
            return Err("Route already exists".to_string());
        }
        
        routes.push(route);
        Ok(())
    }

    pub fn unregister(&self, route_id: &str) -> Result<(), String> {
        let mut routes = self.routes.write().unwrap();
        
        if let Some(pos) = routes.iter().position(|r| r.id == route_id) {
            routes.remove(pos);
            Ok(())
        } else {
            Err("Route not found".to_string())
        }
    }

    pub fn lookup(&self, host: &str, path: &str, method: &str) -> Option<String> {
        let routes = self.routes.read().unwrap();
        
        routes
            .iter()
            .filter(|r| r.host == host && r.path == path)
            .filter(|r| r.methods.contains(&method.to_string()) || r.methods.contains(&"*".to_string()))
            .max_by_key(|r| r.priority)
            .map(|r| r.function_id.clone())
    }

    pub fn list(&self) -> Vec<Route> {
        let routes = self.routes.read().unwrap();
        routes.clone()
    }

    pub fn update(&self, route_id: &str, new_function_id: String) -> Result<(), String> {
        let mut routes = self.routes.write().unwrap();
        
        if let Some(route) = routes.iter_mut().find(|r| r.id == route_id) {
            route.function_id = new_function_id;
            Ok(())
        } else {
            Err("Route not found".to_string())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_route_manager_register() {
        let manager = RouteManager::new();
        let route = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        
        assert!(manager.register(route).is_ok());
        assert_eq!(manager.list().len(), 1);
    }

    #[test]
    fn test_route_manager_duplicate() {
        let manager = RouteManager::new();
        let route1 = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        let route2 = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn2".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        
        manager.register(route1).unwrap();
        assert!(manager.register(route2).is_err());
    }

    #[test]
    fn test_route_manager_lookup() {
        let manager = RouteManager::new();
        let route = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        
        manager.register(route).unwrap();
        let found = manager.lookup("localhost", "/api/test", "POST");
        assert_eq!(found, Some("fn1".to_string()));
    }

    #[test]
    fn test_route_manager_lookup_not_found() {
        let manager = RouteManager::new();
        let found = manager.lookup("localhost", "/api/test", "POST");
        assert!(found.is_none());
    }

    #[test]
    fn test_route_manager_unregister() {
        let manager = RouteManager::new();
        let route = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        
        manager.register(route).unwrap();
        assert_eq!(manager.list().len(), 1);
        
        manager.unregister("r1").unwrap();
        assert_eq!(manager.list().len(), 0);
    }

    #[test]
    fn test_route_manager_update() {
        let manager = RouteManager::new();
        let route = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        
        manager.register(route).unwrap();
        manager.update("r1", "fn2".to_string()).unwrap();
        
        let found = manager.lookup("localhost", "/api/test", "POST");
        assert_eq!(found, Some("fn2".to_string()));
    }

    #[test]
    fn test_route_manager_priority() {
        let manager = RouteManager::new();
        let route1 = Route {
            id: "r1".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn1".to_string(),
            methods: vec!["POST".to_string()],
            priority: 0,
        };
        let route2 = Route {
            id: "r2".to_string(),
            host: "localhost".to_string(),
            path: "/api/test".to_string(),
            function_id: "fn2".to_string(),
            methods: vec!["POST".to_string()],
            priority: 10,
        };
        
        manager.register(route1).unwrap();
        manager.register(route2).unwrap();
        
        let found = manager.lookup("localhost", "/api/test", "POST");
        assert_eq!(found, Some("fn2".to_string()));
    }
}

