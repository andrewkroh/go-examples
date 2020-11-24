# ecs-fields-yml-gen

Given a file containing a whitespace separated list of field names, this
generates "fields.yml" content for the fields that are part of ECS.

It will also report which fields are not part of ECS so that you can ensure
to define them manually.

* NOTE: `geo_point` fields like `source.geo.location.lat` will not match their
ECS definition since the field is defined as `source.geo.location`.

```
make demo
go run main.go -f testdata/fields.txt
---
# ECS fields
- name: network.iana_number
  type: keyword
  description: IANA Protocol Number.
- name: network.packets
  type: long
  description: Total packets transferred in both directions.
- name: url.original
  type: keyword
  description: Unmodified original url as seen in the event source.

---
# Non-ECS fields
- name: not.in.ecs.foobar
```
