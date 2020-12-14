extern crate argparse;
extern crate reqwest;
#[macro_use]
extern crate prettytable;

pub mod ecs;
pub mod mapping;

use argparse::ArgumentParser;
use mapping::*;
use prettytable::Table;
use std::string::String;
use url::Url;

struct Config {
    // Elasticsearch URL
    es_url: String,

    // Optional API Key
    api_key: Option<String>,

    // Pattern that matches the indexes to test.
    index_pattern: Option<String>,
}

fn parse_args() -> Config {
    let mut es_url = String::from("https://localhost:9200");
    let mut api_key = String::new();
    let mut index_pattern = String::new();
    {
        let mut ap = ArgumentParser::new();
        ap.refer(&mut es_url)
            .add_option(&["--url"], argparse::Store, "Elasticsearch URL");
        ap.refer(&mut api_key)
            .add_option(&["--api-key"], argparse::Store, "API key");
        ap.refer(&mut index_pattern)
            .add_option(&["--index"], argparse::Store, "Index Pattern");
        ap.parse_args_or_exit();
    }

    return Config {
        es_url,
        api_key: if api_key.is_empty() {
            None
        } else {
            Some(api_key)
        },
        index_pattern: if index_pattern.is_empty() {
            None
        } else {
            Some(index_pattern)
        },
    };
}

fn es_request(conf: &Config, path: &str) -> reqwest::Request {
    let cat_url = Url::parse(&conf.es_url).unwrap().join(path).unwrap();

    let mut req = reqwest::Request::new(reqwest::Method::GET, cat_url);

    // Add optional API key header.
    if conf.api_key.is_some() {
        let mut key = String::from("ApiKey ");
        key.push_str(conf.api_key.as_ref().unwrap());
        req.headers_mut()
            .insert("Authorization", key.parse().unwrap());
    }

    return req;
}

async fn cat_indices(client: &reqwest::Client, conf: &Config) -> Vec<String> {
    // Make request to list all indices.
    let mut path = String::from("_cat/indices");
    if conf.index_pattern.is_some() {
        path.push_str("/");
        path.push_str(conf.index_pattern.as_ref().unwrap());
    }
    let cat_req = es_request(&conf, path.as_str());

    let resp = client
        .execute(cat_req)
        .await
        .expect("_cat/indices request failed");

    if !resp.status().is_success() {
        println!("Status: {}", resp.status());
        println!("Headers:\n{:#?}", resp.headers());
        panic!("_cat/indices failed with {}", resp.status());
    }

    // Read the response.
    let body = resp.text().await.expect("failed to read response body");
    return get_indices(&body);
}

async fn get_mapping(client: &reqwest::Client, conf: &Config, index: &str) -> String {
    let mapping_req = es_request(&conf, index);
    println!("URL: {}", mapping_req.url());

    let resp = client
        .execute(mapping_req)
        .await
        .expect("mapping request failed");

    if !resp.status().is_success() {
        println!("Status: {}", resp.status());
        println!("Headers:\n{:#?}", resp.headers());
        panic!("get mapping failed with {}", resp.status());
    }

    // Read the response.
    let body = resp.text().await.expect("failed to read response body");
    return body.to_string();
}

async fn get_ecs(client: &reqwest::Client, version: &str) -> String {
    let resp = client
        .get("https://raw.githubusercontent.com/elastic/ecs/v1.7.0/generated/ecs/ecs_flat.yml")
        .send()
        .await
        .expect("ecs_flag request failed");

    if !resp.status().is_success() {
        println!("Status: {}", resp.status());
        println!("Headers:\n{:#?}", resp.headers());
        panic!("get ecs_flat failed with {}", resp.status());
    }

    // Read the response.
    let body = resp.text().await.expect("failed to read response body");
    return body.to_string();
}

#[tokio::main]
async fn main() {
    let conf = parse_args();

    let client = reqwest::Client::new();

    let ecs_yaml = get_ecs(&client, "v1.7.0").await;
    let ecs_fields = ecs::parse_ecs_flat_yaml(&ecs_yaml);

    let indices = cat_indices(&client, &conf);
    for idx in indices.await {
        let body = get_mapping(&client, &conf, &idx).await;
        let get_index_response = parse_mapping(&body).expect("failed to parse mapping body");
        for idx_mapping in get_index_response.indices.iter() {
            let mut table = Table::new();
            table.set_titles(row!["Status", "Field", "Type", "ECS Type"]);
            for field in idx_mapping.1.mappings.flat_fields() {
                let ecs_field = ecs_fields.fields.get(&field.name);
                if ecs_field.is_none() {
                    continue;
                }
                let ecs_field = ecs_field.unwrap();
                if field.data_type != ecs_field.data_type {
                    table.add_row(row!["🔴", field.name, field.data_type, ecs_field.data_type]);
                }
            }
            table.printstd();
        }
    }
}
