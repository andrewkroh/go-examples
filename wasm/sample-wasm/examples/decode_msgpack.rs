use hex;
use rmp_serde;

use elastic_ingest::hostcalls::*;
use elastic_ingest::types::*;

use chrono::prelude::{DateTime, Utc};
use serde_json::Value;
use std::borrow::Borrow;

#[no_mangle]
pub extern "C" fn process() -> i32 {
    let res = get_field("foo/bar").unwrap().unwrap();

    log(
        LogLevel::Debug,
        format!("get_field returned '{}'", res.as_str()).as_str(),
    )
    .unwrap();
    let v: Value = serde_json::from_str(res.as_str()).unwrap();

    let hello_value = v.as_object().unwrap().get("message").unwrap();

    log(LogLevel::Info, format!("json '{}'", hello_value).as_str()).unwrap();

    let msgpack_bytes = hex::decode(hello_value.as_str().unwrap()).unwrap();
    let d: Value = rmp_serde::from_read(msgpack_bytes.as_slice()).unwrap();

    log(
        LogLevel::Debug,
        format!(
            "time={}, data={}",
            iso8601(get_current_time().unwrap().borrow()).as_str(),
            d,
        )
        .as_str(),
    )
    .unwrap();

    put_field("message", d.to_string().as_str()).unwrap();
    return 0;
}

fn iso8601(st: &std::time::SystemTime) -> String {
    let dt: DateTime<Utc> = st.clone().into();
    format!("{}", dt.format("%+"))
    // formats like "2001-07-08T00:34:60.026490+09:30"
}
