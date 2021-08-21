#[cfg(feature = "wee-alloc")]
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

#[cfg_attr(
    all(target_arch = "wasm32", target_os = "unknown"),
    export_name = "malloc"
)]
#[no_mangle]
pub extern "C" fn memory_allocate(size: usize) -> *mut u8 {
    let mut vec: Vec<u8> = Vec::with_capacity(size);
    unsafe {
        vec.set_len(size);
    }
    let slice = vec.into_boxed_slice();
    Box::into_raw(slice) as *mut u8
}

#[no_mangle]
pub extern "C" fn sum(x: i32, y: i32) -> i32 {
    x + y
}

#[no_mangle]
pub extern "C" fn process() -> i32 {
    let res = hostcall_get_field("foo/bar");
    return res.err().unwrap_or(0);
}

extern "C" {
    fn get_field(addr: *const u8, size: usize) -> i32;
}

pub fn hostcall_get_field(field: &str) -> Result<(), i32> {
    unsafe {
        match get_field(field.as_ptr(), field.len()) {
            0 => Ok(()),
            status => panic!("unexpected status: {}", status as i32),
        }
    }
}
