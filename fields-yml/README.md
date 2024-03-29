# fields-yml

This tool takes a list of "fields.yml" files found in Elastic Fleet Integrations
and outputs a sorted and flattened list of field names. Or if specified via
`-f=json` it can output a JSON array of objects.

NOTE: This tool does not consult the ECS version defined in build.yml file.
Instead, it uses the latest ECS version built into
https://github.com/andrewkroh/go-ecs.

## Installation

`go install github.com/andrewkroh/go-examples/fields-yml@main`

## Example

List format.

```
$ fields-yml integrations/packages/netflow/data_stream/*/fields/*.yml
@timestamp
agent.ephemeral_id
agent.id
agent.name
agent.type
agent.version
as.number
as.organization.name
client.address
...
```

JSON format (with resolved ECS field references).

```
$ fields-yml -f=json integrations/packages/netflow/data_stream/*/fields/*.yml
[
  {
    "name": "@timestamp",
    "type": "date",
    "description": "Date/time when the event originated.\nThis is the date/time extracted from the event, typically representing when the event was generated by the source.\nIf the event source has no original timestamp, this value is typically populated by the first time the event was received by the pipeline.\nRequired field for all events.",
    "external": "ecs",
    "source": {
      "path": "/Users/akroh/code/elastic/integrations/packages/netflow/data_stream/log/fields/ecs.yml",
      "line": 1,
      "column": 3
    }
  },
  {
    "name": "agent.ephemeral_id",
    "type": "keyword",
    "description": "Ephemeral identifier of this agent (if one exists).\nThis id normally changes across restarts, but `agent.id` does not.",
    "external": "ecs",
    "source": {
      "path": "/Users/akroh/code/elastic/integrations/packages/netflow/data_stream/log/fields/ecs.yml",
      "line": 3,
      "column": 3
    }
  },
  ...
]
```
