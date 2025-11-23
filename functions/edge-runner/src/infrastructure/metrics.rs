use lazy_static::lazy_static;
use prometheus::{Counter, HistogramOpts, HistogramVec};

lazy_static! {
    pub static ref INVOKE_COUNT: Counter = Counter::new("wasm_invoke_count_total", "Total WASM invocations").unwrap();
    pub static ref INVOKE_LATENCY: HistogramVec = HistogramVec::new(
        HistogramOpts::new("wasm_invoke_latency_seconds", "WASM invocation latency"),
        &["function"]
    ).unwrap();
    pub static ref INVOKE_ERRORS: Counter = Counter::new("wasm_invoke_errors_total", "Total WASM errors").unwrap();
}
