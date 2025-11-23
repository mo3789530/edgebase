use std::slice;
use std::str;

#[no_mangle]
pub extern "C" fn handle(
    method_ptr: *const u8, method_len: usize,
    path_ptr: *const u8, path_len: usize,
    _headers_ptr: *const u8, _headers_len: usize,
    _body_ptr: *const u8, _body_len: usize,
    response_ptr: *mut u8, response_cap: usize
) -> i32 {
    let method = unsafe { str::from_utf8_unchecked(slice::from_raw_parts(method_ptr, method_len)) };
    let path = unsafe { str::from_utf8_unchecked(slice::from_raw_parts(path_ptr, path_len)) };
    
    let response = format!("{{\"message\":\"Hello from WASM!\",\"method\":\"{}\",\"path\":\"{}\"}}",
        method, path);
    let bytes = response.as_bytes();
    let len = bytes.len().min(response_cap);
    
    unsafe {
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), response_ptr, len);
    }
    
    len as i32
}
