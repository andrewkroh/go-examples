use serde::Deserialize;
use std::collections::HashMap;

pub const ALLOWED_ECS_VALUES_QUERY: &str = r#"
{
  "size": 0,
  "aggs": {
    "network_direction": {
      "terms": {
        "field": "network.direction",
        "size": 10
      }
    },
    "event_kind": {
      "terms": {
        "field": "event.kind",
        "size": 10
      }
    },
    "event_category": {
      "terms": {
        "field": "event.category",
        "size": 10
      }
    },
    "event_type": {
      "terms": {
        "field": "event.type",
        "size": 10
      }
    },
    "event_outcome": {
      "terms": {
        "field": "event.outcome",
        "size": 10
      }
    }
  }
}
"#;

#[derive(Debug, Clone, Deserialize)]
pub struct ECSFields {
    #[serde(flatten)]
    pub fields: HashMap<String, Field>,
}

#[derive(Debug, Clone, Deserialize, Eq, PartialEq)]
pub struct Field {
    #[serde(rename = "flat_name")]
    pub name: String,

    #[serde(rename = "type")]
    pub data_type: String,
}

pub fn parse_ecs_flat_yaml(data: &str) -> ECSFields {
    let fields: ECSFields = serde_yaml::from_str(&data).unwrap();
    return fields;
}

#[cfg(test)]
mod tests {
    use crate::ecs::parse_ecs_flat_yaml;

    #[test]
    fn test_flatten_fields() {
        let mapping = r#"
client.as.organization.name:
  dashed_name: client-as-organization-name
  description: Organization name.
  example: Google LLC
  flat_name: client.as.organization.name
  ignore_above: 1024
  level: extended
  multi_fields:
  - flat_name: client.as.organization.name.text
    name: text
    norms: false
    type: text
  name: organization.name
  normalize: []
  original_fieldset: as
  short: Organization name.
  type: keyword
"#;

        let ecs_data = parse_ecs_flat_yaml(&mapping);
        assert_eq!(ecs_data.fields.len(), 1);
        for f in ecs_data.fields.iter() {
            assert_eq!(f.1.name, "client.as.organization.name")
        }
    }
}
