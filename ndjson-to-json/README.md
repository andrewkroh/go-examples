# ndjson-to-json

Reads an ndjson file and outputs an array of objects.

```
Usage of ./ndjson-to-json:
  -in string
        input ndjson file
  -key string
        name of json array in output (default "events")
```

Example usage:

```
./ndjson-to-json -in=events.ndjson
```
