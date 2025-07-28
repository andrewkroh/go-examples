package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/andrewkroh/go-ecs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ecsFieldInfoToolDesc is a description of the ecs_field_info MCP tool.
const ecsFieldInfoToolDesc = `
Looks up a field in Elastic Common Schema (ECS) and returns its definition.

Example: To get the definition of the 'process.pid' field, call the tool with 'process.pid'.

Returns: A JSON object with the following keys:
- name: The flattened field name.
- data_type: The Elasticsearch field data type (e.g. keyword, match_only_text).
- array: Indicates if the value type must be an array.
- pattern: Regular expression pattern that can be used to validate the value.
- description: Short description of the field.

Annotations: This tool is readonly and the result is cachable.`

type ECSFieldInfoArgs struct {
	Field   string `json:"field" jsonschema:"The full field name to look up (e.g., 'process.pid'). Only leaf fields are supported. For example, 'process.pid' will work, but 'process' will not."`
	Version string `json:"version,omitempty" jsonschema:"The ECS version to use. If not specified, the latest version will be used."`
}

func ECSFieldInfo(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ECSFieldInfoArgs]) (*mcp.CallToolResultFor[ecs.Field], error) {
	field, err := ecs.Lookup(params.Arguments.Field, params.Arguments.Version)
	if err != nil {
		return &mcp.CallToolResultFor[ecs.Field]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%q is not a valid ECS field", params.Arguments.Field)},
			},
		}, nil
	}

	return &mcp.CallToolResultFor[ecs.Field]{
		StructuredContent: *field,
	}, nil
}

var (
	httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	noLog    = flag.Bool("no-log", false, "if set, disables logging")
)

func main() {
	flag.Parse()

	var logOutput io.Writer = os.Stderr
	if *noLog {
		logOutput = io.Discard
	}
	log.SetOutput(logOutput)

	s := mcp.NewServer(&mcp.Implementation{Name: "ecs"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ecs_field_info",
		Description: ecsFieldInfoToolDesc,
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, ECSFieldInfo)

	log.Println("ecs-mcp is starting...")

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return s
		}, nil)
		log.Printf("ecs-mcp handler listening at %s", *httpAddr)
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), logOutput)
		if err := s.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}

	log.Println("ecs-mcp has shut down")
}
