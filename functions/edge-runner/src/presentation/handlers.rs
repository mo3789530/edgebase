use axum::{http::{Request, StatusCode}, body::Body, response::IntoResponse};
use std::sync::Arc;
use crate::application::InvocationService;
use crate::infrastructure::INVOKE_COUNT;
use prometheus::Encoder;

pub struct HttpHandler {
    invocation_service: Arc<InvocationService>,
}

impl HttpHandler {
    pub fn new(invocation_service: Arc<InvocationService>) -> Self {
        Self { invocation_service }
    }
    
    pub async fn handle_request(&self, req: Request<Body>) -> impl IntoResponse {
        let start = std::time::Instant::now();
        INVOKE_COUNT.inc();
        
        let method = req.method().as_str().to_string();
        let path = req.uri().path().to_string();
        let host = req.headers()
            .get("host")
            .and_then(|h| h.to_str().ok())
            .unwrap_or("localhost")
            .to_string();
        
        match self.invocation_service.invoke(&host, &path, &method).await {
            Ok(response) => {
                crate::infrastructure::INVOKE_LATENCY.with_label_values(&[&path]).observe(start.elapsed().as_secs_f64());
                (StatusCode::OK, response).into_response()
            }
            Err(e) => {
                crate::infrastructure::INVOKE_ERRORS.inc();
                crate::infrastructure::INVOKE_LATENCY.with_label_values(&[&path]).observe(start.elapsed().as_secs_f64());
                
                let status = if e.contains("Route not found") {
                    StatusCode::NOT_FOUND
                } else if e.contains("method not allowed") {
                    StatusCode::METHOD_NOT_ALLOWED
                } else {
                    StatusCode::INTERNAL_SERVER_ERROR
                };
                
                (status, e).into_response()
            }
        }
    }
}

pub async fn metrics_handler() -> impl IntoResponse {
    let encoder = prometheus::TextEncoder::new();
    let metric_families = prometheus::gather();
    let mut buffer = Vec::new();
    let _ = encoder.encode(&metric_families, &mut buffer);
    String::from_utf8(buffer).unwrap_or_default()
}
