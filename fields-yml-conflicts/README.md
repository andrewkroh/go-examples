# fields-yml-conflicts

Detect field type conflicts between Elastic Fleet integrations.

## Install

`go install github.com/andrewkroh/go-examples/fields-yml-conflicts@main`

## Usage

```
cd elastic/integrations
fields-yml-conflicts packages/*/data_stream/*/fields/*yml > conflicts.json
```

Example output: https://gist.github.com/andrewkroh/4db68e7edd97a8316bada0877c3da13b
