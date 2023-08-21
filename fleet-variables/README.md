# fleet-variables

This tool generates CSV containing all variables defined in Fleet packages.
It excludes variables that are marked as `secret: true`. You may optionally
specify `-owner <github team>` to output data from packages owned by that team.

Data is sorted by variable name.

`go run . -integ-dir ./elastic/integrations > vars.csv`

## Example output

```csv
name,type,description,integration,path,line,url,owner
__ui,yaml,,synthetics,packages/synthetics/data_stream/tcp/manifest.yml,23,https://github.com/elastic/integrations/blob/1007ccd783ba329719df61be833613159a9089be/packages/synthetics/data_stream/tcp/manifest.yml#L23,elastic/uptime
```
