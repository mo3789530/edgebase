mod cache;
mod metrics;

use axum::{Router, routing::{any, get}, extract::State, http::{Request, StatusCode}, body::Body, response::IntoResponse};
use std::sync::Arc;
use wasmer::{Store, Module, Instance, imports, Memory, MemoryType, Pages};
use sha2::{Sha256, Digest};

struct AppState {
    store: Arc<tokio::sync::Mutex<Store>>,
    instance: Arc<tokio::sync::Mutex<Option<Instance>>>,
}

#[tokio::main]
async fn main() {
    metrics::init();
    
    let wasm_path = std::env::args().nth(1).expect("Usage: edge-runner <wasm_file>");
    
    let wasm_bytes = std::fs::read(&wasm_path).expect("Failed to read WASM file");
    let mut hasher = Sha256::new();
    hasher.update(&wasm_bytes);
    let hash = hasher.finalize();
    println!("WASM SHA256: {:x}", hash);
    
    let mut store = Store::default();
    let module = Module::new(&store, &wasm_bytes).expect("Failed to compile WASM");
    
    let memory = Memory::new(&mut store, MemoryType::new(Pages(16), Some(Pages(16)), false))
        .expect("Failed to create memory");
    
    let import_object = imports! {
        "env" => {
            "memory" => memory.clone(),
        }
    };
    
    let instance = Instance::new(&mut store, &module, &import_object)
        .expect("Failed to instantiate WASM");
    
    let state = AppState {
        store: Arc::new(tokio::sync::Mutex::new(store)),
        instance: Arc::new(tokio::sync::Mutex::new(Some(instance))),
    };
    
    let app = Router::new()
        .route("/metrics", get(metrics_handler))
        .route("/*path", any(handler))
        .with_state(Arc::new(state));
    
    let listener = tokio::net::TcpListener::bind("0.0.0.0:3000").await.unwrap();
    println!("Edge Runner listening on http://0.0.0.0:3000");
    axum::serve(listener, app).await.unwrap();
}

async fn metrics_handler() -> impl IntoResponse {
    metrics::export_metrics()
}

async fn handler(
    State(state): State<Arc<AppState>>,
    req: Request<Body>,
) -> impl IntoResponse {
    let start = std::time::Instant::now();
    metrics::INVOKE_COUNT.inc();
    
    let method = req.method().as_str();
    let path = req.uri().path();
    
    let mut store = state.store.lock().await;
    let instance_guard = state.instance.lock().await;
    let instance = instance_guard.as_ref().unwrap();
    
    let handle = instance.exports.get_function("handle")
        .expect("handle function not found");
    
    let memory = instance.exports.get_memory("memory")
        .expect("memory not found");
    
    let method_bytes = method.as_bytes();
    let path_bytes = path.as_bytes();
    let response_buf = vec![0u8; 4096];
    
    {
        let mem_view = memory.view(&store);
        mem_view.write(0, method_bytes).unwrap();
        mem_view.write(256, path_bytes).unwrap();
        mem_view.write(1024, &response_buf).unwrap();
    }
    
    let result = handle.call(&mut *store, &[
        0i32.into(), (method_bytes.len() as i32).into(),
        256i32.into(), (path_bytes.len() as i32).into(),
        512i32.into(), 0i32.into(),
        768i32.into(), 0i32.into(),
        1024i32.into(), (response_buf.len() as i32).into(),
    ]);
    
    match result {
        Ok(results) => {
            let len = results[0].i32().unwrap() as usize;
            let mut response_data = vec![0u8; len];
            let mem_view = memory.view(&store);
            mem_view.read(1024, &mut response_data).unwrap();
            
            metrics::INVOKE_LATENCY.observe(start.elapsed().as_secs_f64());
            (StatusCode::OK, response_data)
        }
        Err(e) => {
            metrics::INVOKE_ERRORS.inc();
            metrics::INVOKE_LATENCY.observe(start.elapsed().as_secs_f64());
            (StatusCode::INTERNAL_SERVER_ERROR, format!("WASM error: {}", e).into_bytes())
        }
    }
}
