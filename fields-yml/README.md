# fields-yml

This tool takes a list of "fields.yml" files found in Elastic Fleet Integrations
and outputs a sorted and flattened list of field names. Or if specified via
`-f=json` it can output a JSON array of objects.

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

JSON format. It does not resolve the external definitions.

```
$ fields-yml -f=json integrations/packages/netflow/data_stream/*/fields/*.yml
[
  {
    "name": "@timestamp",
    "type": "date",
    "description": "Event timestamp."
  },
  ...
  {
    "name": "user_agent.version",
    "external": "ecs"
  }
]
```
