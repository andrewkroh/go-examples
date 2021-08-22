use serde_json::Value;
use std::ptr::null_mut;

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
    let res = hostcall_get_field("foo/bar").unwrap().unwrap();

    hostcall_log(1, format!("get_field returned '{}'", res.as_str()).as_str()).unwrap();
    let v: Value = serde_json::from_str(res.as_str()).unwrap();

    let hello_value = v.as_object().unwrap().get("hello").unwrap();

    hostcall_log(1, format!("json '{}'", hello_value).as_str()).unwrap();

    return 0;
}

extern "C" {
    fn get_field(
        addr: *const u8,
        size: usize,
        return_buffer_data: *mut *mut u8,
        return_buffer_size: *mut usize,
    ) -> i32;
}

pub fn hostcall_get_field(field: &str) -> Result<Option<String>, i32> {
    let mut return_data: *mut u8 = null_mut();
    let mut return_size: usize = 0;
    unsafe {
        match get_field(
            field.as_ptr(),
            field.len(),
            &mut return_data,
            &mut return_size,
        ) {
            0 => {
                if !return_data.is_null() {
                    // This vector will now own the return data memory and deallocate it.
                    let field_value = String::from_utf8(Vec::from_raw_parts(
                        return_data,
                        return_size,
                        return_size,
                    ))
                    .unwrap();

                    Ok(Some(field_value))
                } else {
                    Ok(None)
                }
            }
            1 => Ok(None),
            status => panic!("unexpected status: {}", status as i32),
        }
    }
}

extern "C" {
    fn log_it(level: i32, message_data: *const u8, message_size: usize) -> i32;
}

pub fn hostcall_log(level: i32, message: &str) -> Result<(), i32> {
    unsafe {
        match log_it(level, message.as_ptr(), message.len()) {
            0 => Ok(()),
            status => panic!("unexpected status: {}", status),
        }
    }
}
