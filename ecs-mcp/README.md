# ECS MCP Server

This is a [Model Context Protocol (MCP)][mcp]
server that provides tools for working with [Elastic Common Schema (ECS)][ecs].

This server uses [github.com/andrewkroh/go-ecs][go-ecs] lookup ECS field
definitions.

[go-ecs]: https://github.com/andrewkroh/go-ecs
[ecs]: https://www.elastic.co/guide/en/ecs/current/index.html
[mcp]: https://github.com/metoro-io/model-context-protocol

## Tools

### `ecs_field_info`

Looks up a field in Elastic Common Schema (ECS) and returns its definition.

**Parameters:**

* `field` (string, required): The full field name to look up (e.g., '
  process.pid'). Only leaf fields are supported. For example, 'process.pid' will
  work, but 'process' will not.
* `version` (string, optional): The ECS version to use. If not specified, the
  latest version will be used.

**Returns:**

A JSON object with the ECS field definition containing keys like `name`,
`data_type`, `description`, etc.
