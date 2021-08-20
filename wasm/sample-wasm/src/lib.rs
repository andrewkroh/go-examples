#[no_mangle]
pub extern "C" fn sum(x: i32, y: i32) -> i32 {
    x + y
}

#[no_mangle]
pub extern "C" fn _start() -> i32 {
    hostcall_host_function().unwrap();
    return sum(2, 2);
}

extern "C" {
    fn host_function() -> i32;
}

pub fn hostcall_host_function() -> Result<(), i32> {
    unsafe {
        match host_function() {
            0 => Ok(()),
            status => panic!("unexpected status: {}", status as i32),
        }
    }
}
