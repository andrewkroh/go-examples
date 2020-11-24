extern crate reqwest;

pub mod mapping;

use mapping::*;

#[tokio::main]
async fn main() -> Result<(), reqwest::Error> {
    let client = reqwest::Client::new();
    let res = client.
        get("https://localhost:9200/_cat/indices").
        // header("Authorization", "ApiKey xxxxxx==").
        send().await?;

    println!("Status: {}", res.status());
    println!("Headers:\n{:#?}", res.headers());

    // Move and borrow value of `res`
    let body = res.text().await?;

    let indices = get_indices(&body);
    for idx in indices {
        println!("{}", idx);
    }

    // Now request a specific index mapping.
    let res = client.
        get("https://localhost:9200/winlogbeat-8.0.0-2020.09.02-000001").
        // header("Authorization", "ApiKey xxxxxx==").
        send().await?;
    println!("Status: {}", res.status());
    println!("Headers:\n{:#?}", res.headers());

    // Move and borrow value of `res`
    let body = res.text().await?;

    let get_index_response = parse_mapping(&body).unwrap();
    for idx_mapping in get_index_response.indices.iter() {
        let mut fields: Vec<Field> = Vec::new();
        flatten_fields("", &idx_mapping.1.mappings.properties, &mut fields);
        fields.sort();
        for field in fields {
            println!("{:?}", field);
        }
    }

    Ok(())
}

