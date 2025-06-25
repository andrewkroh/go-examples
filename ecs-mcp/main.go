package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/andrewkroh/go-ecs"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type GetFieldRequest struct {
	Field   string `json:"field" jsonschema:"required,description=The full field name to look up (e.g., 'process.pid'). Only leaf fields are supported. For example, 'process.pid' will work, but 'process' will not."`
	Version string `json:"version,omitempty" jsonschema:"description=The ECS version to use. If not specified, the latest version will be used."`
}

func GetFieldInfo(req *GetFieldRequest) (resp *mcp_golang.ToolResponse, err error) {
	log.Printf("ecs_field_info called with %+v.", req)
	defer func() {
		log.Printf("ecs_field_info finished: response: %+v", resp)
	}()

	field, err := ecs.Lookup(req.Field, req.Version)
	if err != nil {
		txt := mcp_golang.NewTextContent(fmt.Sprintf("%q is not a valid ECS field", req.Field))
		return mcp_golang.NewToolResponse(txt), nil
	}

	raw, err := json.MarshalIndent(field, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ECS field: %w", err)
	}

	txt := mcp_golang.NewTextContent(string(raw))
	return mcp_golang.NewToolResponse(txt), nil
}

func main() {
	s := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err := s.RegisterTool(
		"ecs_field_info",
		`Looks up a field in Elastic Common Schema (ECS) and returns its definition.

Example: To get the definition of the 'process.pid' field, call the tool with 'process.pid'.

Returns: A JSON object with the following keys:
- name: The flattened field name.
- data_type: The Elasticsearch field data type (e.g. keyword, match_only_text).
- array: Indicates if the value type must be an array.
- pattern: Regular expression pattern that can be used to validate the value.
- description: Short description of the field.

Annotations: This tool is readonly and the result is cachable.`,
		GetFieldInfo,
	)
	if err != nil {
		log.Fatalf("failed to register tool: %v", err)
	}

	if err = s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	log.Println("ecs-mcp is running...")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	<-ctx.Done()
}
