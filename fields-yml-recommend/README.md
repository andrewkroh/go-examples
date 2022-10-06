# fields-yml-recommend

`fields-yml-recommend` advises you on changes to fields YAML fields. It
is recommended that you point it at all fields files within a single data stream
so that it has the full list of fields. It detects multiple issues:

- Fields that exist in ECS, but are not using an 'external: ecs' definition.
- Fields that are duplicated.

## Install

`go install github.com/andrewkroh/go-examples/fields-yml-recommend@main`

## Usage

```
cd elastic/integrations
fields-yml-recommend packages/aws/data_stream/*/fields/*.yml
```

```json
[
  {
  "name": "host.os.version",
  "type": "keyword",
  "file": "packages/aws/data_stream/transitgateway/fields/agent.yml",
  "line": 169,
  "notes": [
    "Use 'external: ecs' to import the ECS definition."
  ]
},
...
]
```
