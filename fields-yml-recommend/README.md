# fields-yml-recommend

Recommendations for improving fields.yml files in Elastic Fleet packages.

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
