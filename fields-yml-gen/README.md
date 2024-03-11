# fields-yml-gen

Given a file containing a whitespace separated list of field names, this
generates "fields.yml" content for the fields that are part of ECS.

It will also report which fields are not part of ECS so that you can ensure
to define them manually.

* NOTE: `geo_point` fields like `source.geo.location.lat` will not match their
ECS definition since the field is defined as `source.geo.location`.

## Install

`go install github.com/andrewkroh/go-examples/fields-yml-gen@main`

## Usage

If you are running elastic-package, and it fails because of undefined fields,
then you can pipe that through this command.

```sh
cat << EOF | fields-yml-gen
--- Test results for package: crowdstrike - START ---
FAILURE DETAILS:
crowdstrike/fdr logfile:
[0] field "event.action" is undefined
[1] field "event.agent_id_status" is undefined
[2] field "event.category" is undefined
[3] field "event.created" is undefined
[4] field "event.id" is undefined
[5] field "event.ingested" is undefined
[6] field "event.kind" is undefined
[7] field "event.original" is undefined
[8] field "event.outcome" is undefined
[9] field "event.timezone" is undefined
[10] field "event.type" is undefined
[11] field "crowdstrike.foo" is undefined
EOF
```

That will give you output that you can paste into a fields.yml file.

```yaml
EOF
---
# ECS fields
- name: event.action
  external: ecs
- name: event.category
  external: ecs
- name: event.created
  external: ecs
- name: event.id
  external: ecs
- name: event.kind
  external: ecs
- name: event.original
  external: ecs
- name: event.outcome
  external: ecs
- name: event.timezone
  external: ecs
- name: event.type
  external: ecs

---
# Non-ECS fields
- name: crowdstrike.foo
  type: keyword
  description: TODO
```
