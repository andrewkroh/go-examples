use hex;
use rmp_serde;

use elastic_ingest::hostcalls::*;
use elastic_ingest::types::*;

use chrono::prelude::{DateTime, Utc};
use serde_json::Value;
use std::borrow::Borrow;

#[no_mangle]
pub extern "C" fn process() -> Status {
    // Get field returns the raw value as a JSON string.
    let message_value_json = get_field("message").unwrap();
    if message_value_json.is_none() {
        return Status::Ok;
    }
    let message_value_json = message_value_json.unwrap();
    log(
        LogLevel::Debug,
        format!(
            "get_field returned message='{}'",
            message_value_json.as_str()
        )
        .as_str(),
    )
    .unwrap();

    // Decode the JSON.
    let message_value: Value = serde_json::from_str(message_value_json.as_str()).unwrap();

    // The value is a string.
    let msgpack_hex = message_value.as_str().unwrap();
    log(
        LogLevel::Info,
        format!("message is a string of value '{}'.", msgpack_hex).as_str(),
    )
    .unwrap();

    // Decode the hex into a slice of bytes.
    let msgpack_bytes = hex::decode(msgpack_hex).unwrap();

    // Decode bytes as msgpack.
    let msgpack_data: Value = rmp_serde::from_read(msgpack_bytes.as_slice()).unwrap();

    log(
        LogLevel::Debug,
        format!(
            "time={}, data={}",
            iso8601(get_current_time().unwrap().borrow()).as_str(),
            msgpack_data,
        )
        .as_str(),
    )
    .unwrap();

    // Write the decoded object back into the message field.
    put_field("message", msgpack_data.to_string().as_str()).unwrap();

    return Status::Ok;
}

fn iso8601(st: &std::time::SystemTime) -> String {
    let dt: DateTime<Utc> = st.clone().into();
    format!("{}", dt.format("%+"))
    // formats like "2001-07-08T00:34:60.026490+09:30"
}
