# fields-yml-gen

Given a file containing a whitespace separated list of field names, this
generates "fields.yml" content for the fields that are part of ECS.

It will also report which fields are not part of ECS so that you can ensure
to define them manually.

* NOTE: `geo_point` fields like `source.geo.location.lat` will not match their
ECS definition since the field is defined as `source.geo.location`.

## Install

`go install github.com/andrewkroh/go-examples/fields-yml-gen@main`
