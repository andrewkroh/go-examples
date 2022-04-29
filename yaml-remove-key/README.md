# yaml-remove-key

Remove the specified keys from a YAML document.

## Install

`go install github.com/andrewkroh/go-examples/yaml-remove-key@main`

## Usage

```
Usage of yaml-remove-key:
  -indent int
        YAML indentation (default 2)
  -key value
        Key to filter. May be used more than once and value can be comma separated.
  -w    Write modification to file.
```

## Example

`yaml-remove-key -w -key "level,group,title,example" integrations/packages/*/data_stream/*/fields/*.yml`

Output:
```
2022/04/29 14:58:50 gcp/data_stream/audit/fields/agent.yml: 45 changes
2022/04/29 14:58:50 gcp/data_stream/audit/fields/ecs.yml: 1 changes
2022/04/29 14:58:50 gcp/data_stream/dns/fields/agent.yml: 45 changes
2022/04/29 14:58:50 gcp/data_stream/firewall/fields/agent.yml: 45 changes
2022/04/29 14:58:50 gcp/data_stream/firewall/fields/ecs.yml: 2 changes
2022/04/29 14:58:50 gcp/data_stream/vpcflow/fields/agent.yml: 45 changes
2022/04/29 14:58:50 gcp/data_stream/vpcflow/fields/ecs.yml: 2 changes
```
