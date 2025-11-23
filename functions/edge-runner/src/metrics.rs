use prometheus::{Counter, Histogram, HistogramOpts, Registry, TextEncoder, Encoder};
use lazy_static::lazy_static;

lazy_static! {
    pub static ref REGISTRY: Registry = Registry::new();
    
    pub static ref INVOKE_COUNT: Counter = Counter::new(
        "wasm_invoke_count_total",
        "Total number of WASM function invocations"
    ).unwrap();
    
    pub static ref INVOKE_LATENCY: Histogram = Histogram::with_opts(
        HistogramOpts::new(
            "wasm_invoke_latency_seconds",
            "WASM function invocation latency"
        )
    ).unwrap();
    
    pub static ref INVOKE_ERRORS: Counter = Counter::new(
        "wasm_invoke_errors_total",
        "Total number of WASM function errors"
    ).unwrap();
}

pub fn init() {
    REGISTRY.register(Box::new(INVOKE_COUNT.clone())).unwrap();
    REGISTRY.register(Box::new(INVOKE_LATENCY.clone())).unwrap();
    REGISTRY.register(Box::new(INVOKE_ERRORS.clone())).unwrap();
}

pub fn export_metrics() -> String {
    let encoder = TextEncoder::new();
    let metric_families = REGISTRY.gather();
    let mut buffer = vec![];
    encoder.encode(&metric_families, &mut buffer).unwrap();
    String::from_utf8(buffer).unwrap()
}
