use serde::Deserialize;
use std::cmp::Ord;
use std::cmp::Ordering;
use std::collections::HashMap;

#[derive(Deserialize, Debug, Clone)]
pub struct GetIndexResponse {
    #[serde(flatten)]
    pub indices: HashMap<String, IndexMapping>,
}

#[derive(Deserialize, Debug, Clone)]
pub struct IndexMapping {
    pub mappings: Mappings,
}

#[derive(Deserialize, Debug, Clone)]
pub struct Mappings {
    #[serde(rename = "type")]
    data_type: Option<String>,

    ignore_above: Option<u32>,

    #[serde(default)]
    fields: HashMap<String, Mappings>,

    #[serde(default)]
    pub properties: HashMap<String, Mappings>,
}

impl Mappings {
    pub fn flat_fields(&self) -> Vec<Field> {
        let mut fields: Vec<Field> = Vec::new();
        flatten_fields("", &self.properties, &mut fields);
        fields.sort();
        return fields;
    }
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct Field {
    pub name: String,
    pub data_type: String,
}

impl Ord for Field {
    fn cmp(&self, other: &Self) -> Ordering {
        self.name.cmp(&other.name)
    }
}

impl PartialOrd for Field {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

pub fn get_indices(cat_output: &str) -> Vec<String> {
    let mut indices: Vec<String> = Vec::new();
    for line in cat_output.lines() {
        let parts = line.split_whitespace().collect::<Vec<&str>>();
        let index_name = parts.get(2);
        if index_name.is_some() {
            indices.push(index_name.unwrap().to_string());
        }
    }
    indices.sort();
    return indices;
}

pub fn parse_mapping(body: &str) -> serde_json::Result<GetIndexResponse> {
    let result: serde_json::Result<GetIndexResponse> = serde_json::from_str(body);
    result
}

pub fn flatten_fields(parent_key: &str, m: &HashMap<String, Mappings>, fields: &mut Vec<Field>) {
    for itr in m.iter() {
        let full_key: String;
        if !parent_key.is_empty() {
            full_key = [parent_key, itr.0].join(".");
        } else {
            full_key = itr.0.to_owned();
        }

        let mapping = itr.1;
        if mapping.data_type.is_some() {
            let field = Field {
                name: full_key.to_owned(),
                data_type: mapping.data_type.as_ref().unwrap().to_owned(),
            };
            fields.push(field);
        }
        flatten_fields(&full_key, &mapping.properties, fields);
        flatten_fields(&full_key, &mapping.fields, fields);
    }
}

#[cfg(test)]
mod tests {
    use crate::mapping::GetIndexResponse;
    use crate::mapping::{flatten_fields, get_indices, Field};

    #[test]
    fn parse_cat_output() {
        let cat_indices = "\
        green open auditbeat-7.8.0-2020.08.13-000003   L7ybcQriTBOvkUNjy5Bhww 1 1 17472128       0   17.7gb   8.8gb
        green open winlogbeat-8.0.0-2020.10.02-000002  jX21_EYDTX2t7O1GWRMmFQ 1 1   364567       0      1gb   526mb";

        let vec = get_indices(cat_indices);

        assert_eq!(vec.len(), 2);
        assert_eq!(
            vec,
            [
                "auditbeat-7.8.0-2020.08.13-000003",
                "winlogbeat-8.0.0-2020.10.02-000002"
            ]
        );
    }

    #[test]
    fn test_flatten_fields() {
        let mapping = r#"
{
  "winlogbeat-8.0.0-2020.09.02-000001": {
    "mappings": {
      "properties": {
        "@timestamp": {
          "type": "date"
        },
        "process": {
          "properties": {
            "name": {
              "type": "keyword",
              "ignore_above": 1024,
              "fields": {
                "text": {
                  "type": "text",
                  "norms": false
                }
              }
            }
          }
        }
      }
    }
  }
}
"#;

        let v: GetIndexResponse = serde_json::from_str(mapping).unwrap();
        assert!(v.indices.contains_key("winlogbeat-8.0.0-2020.09.02-000001"));
        let index = v.indices.get("winlogbeat-8.0.0-2020.09.02-000001").unwrap();

        let mut fields: Vec<Field> = Vec::new();
        flatten_fields("", &index.mappings.properties, fields.as_mut());
        fields.sort();
        println!("{:?}", fields);
        assert_eq!(fields.len(), 3);
        assert_eq!(
            fields,
            [
                Field {
                    name: "@timestamp".to_owned(),
                    data_type: "date".to_owned()
                },
                Field {
                    name: "process.name".to_owned(),
                    data_type: "keyword".to_owned()
                },
                Field {
                    name: "process.name.text".to_owned(),
                    data_type: "text".to_owned()
                },
            ]
        );
    }
}
