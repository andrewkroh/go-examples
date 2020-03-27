#[no_mangle]
pub extern fn sum(x: i32, y: i32) -> i32 {
    x + y
}

#[no_mangle]
pub extern fn _start() -> i32 {
    return sum(2, 2)
}
