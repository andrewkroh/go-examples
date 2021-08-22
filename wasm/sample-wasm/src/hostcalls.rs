use crate::types::*;
use std::ptr::null_mut;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

#[link(wasm_import_module = "elastic")]
extern "C" {
    fn elastic_get_field(
        addr: *const u8,
        size: usize,
        return_buffer_data: *mut *mut u8,
        return_buffer_size: *mut usize,
    ) -> i32;
}

pub fn get_field(field: &str) -> Result<Option<String>, i32> {
    let mut return_data: *mut u8 = null_mut();
    let mut return_size: usize = 0;
    unsafe {
        match elastic_get_field(
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

#[link(wasm_import_module = "elastic")]
extern "C" {
    fn elastic_log(level: i32, message_data: *const u8, message_size: usize) -> i32;
}

pub fn log(level: LogLevel, message: &str) -> Result<(), i32> {
    unsafe {
        match elastic_log(level as i32, message.as_ptr(), message.len()) {
            0 => Ok(()),
            status => panic!("unexpected status: {}", status),
        }
    }
}

#[link(wasm_import_module = "elastic")]
extern "C" {
    fn elastic_get_current_time_nanoseconds(return_time: *mut u64) -> i32;
}

pub fn get_current_time() -> Result<SystemTime, i32> {
    let mut return_time: u64 = 0;
    unsafe {
        match elastic_get_current_time_nanoseconds(&mut return_time) {
            0 => Ok(UNIX_EPOCH + Duration::from_nanos(return_time)),
            status => panic!("unexpected status: {}", status as i32),
        }
    }
}
