# fields-yml-conflicts

Detect field type conflicts between Elastic Fleet integrations. It detects
two classes of conflicts:

- same field declared with different types (by default text-family conflicts
are ignored)
- scalar field type where there are child fields declared

## Install

`go install github.com/andrewkroh/go-examples/fields-yml-conflicts@main`

## Usage

```
cd elastic/integrations
fields-yml-conflicts packages/*/data_stream/*/fields/*yml
```
