#[repr(i32)]
#[derive(Debug)]
pub enum LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3,
    Critical = 4,
}

#[repr(i32)]
#[derive(Debug)]
pub enum Status {
    Ok = 0,
    InternalFailure = 1,
    InvalidArgument = 2,
    NotFound = 3,
}
