# fleetpkg-indexer

`fleetpkg-indexer` is a tool that reads Fleet package specifications and indexes the information
into Elasticsearch.

You can use the data to answer questions like this and more:

- What ECS fields are declared in a data stream?
- Are the fields with the same name but different data types?

### Usage Example

You must have a local clone of `elastic/integrations`.

```shell
go run github.com/andrewkroh/go-examples/fleetpkg-indexer@main \
  -packages-dir ~/code/elastic/integrations/packages \
  -es-url "https://localhost:9200" \
  -insecure \
  -u elastic \
  -p changeme
```

### Dashboard

You can load the included dashboard by importing the saved objects into Kibana.

![dashboard](dashboard.png)

### Data

In addition to the original attributes found in package-spec, some additional
fields are added to the documents to help with pivoting and correlating.

- `@type` - Package data is separated into different documents. They type of data
  is indicated by `@type`. The values for the field are:
    - build_manifest
    - data_stream_manifest
    - data_stream_variable
    - field
    - input_variable
    - manifest
    - package_variable
    - policy_template
    - policy_template_variable
    - sample_event
    - variable
- `@integration` - Associated integration name.
- `@data_stream` - Associated data stream name.
- `@policy_template` - Associated policy template name.
- `@commit` - elastic/integration git commit ID
- `@timestamp` - Timestamp of the git commit.
- `@url` - URL pointing to source file in GitHub.
- `@input` - Associated input types.

### Known issues

- Annotation with the related `@policy_template` is not fully implemented.
- Fields related to input type packages are not indexed.
