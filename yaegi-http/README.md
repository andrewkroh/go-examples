# Yaegi HTTP Demo

This demo application demonstrates
using [Yaegi](https://github.com/traefik/yaegi) as a Go interpreter to execute
dynamic HTTP processing scripts, similar to concepts found in Elastic's CEL or
httpjson processors.

## Overview

The application executes a Go program (e.g. `testdata/prorams/ipify.go`) that
performs HTTP requests and processes responses. Instead of compiling these
programs, they are interpreted at runtime using Yaegi, allowing for dynamic
execution of Go code without the need to compile.

## How It Works

1. **Program Loading**: The main application embeds a Go program using `//go:embed`
2. **Interpretation**: Yaegi interprets the embedded Go code at runtime
3. **Execution**: The interpreted program executes an `Execute` function that:
   - Makes HTTP requests using the provided HTTP client
   - Processes responses
   - Calls a callback function with event data
4. **Output**: Events are collected and output as JSON

> [!NOTE]
> The demo application does not include any state management.

I intentionally used a function signature that does not rely on any custom types
to not require implementors to have a dependency on external packages.

## Example Program

The included test program (`testdata/job.go`) demonstrates a simple HTTP request
to get public IP information:

```go
func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
    r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org?format=json", nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := c.Do(r)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }

    var data map[string]any
    if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return fmt.Errorf("failed to decode json: %w", err)
    }

    callback(data)
    return nil
}
```

The application will execute the embedded program and output any events as JSON
to stdout.

## Comparison to Other Solutions

### vs Elastic CEL

- Pros: Uses full Go language syntax that developers already know
- Cons: Go is Turing complete, unlike CEL, so programs may not terminate (
  requires execution timeouts)

### vs httpjson

- Pros: More flexible programming model with full Go capabilities
- Cons: More complex (depending on audience) than configuration-based approaches

## Pros and Cons

### Advantages

- Familiar Language: Uses Go syntax that developers already know
- Full Language Features: Access to Go's complete feature set (with some Yaegi
  limitations)
- Type Safety: Maintains Go's type system benefits
- Standard Library: Can use most of Go's standard library
- **Supports stream processing**: CEL and httpjson process the full HTTP
  response into memory before delivery events. This is a problem for large
  responses such as the 100+MB Recorded Future CSV files.

### Disadvantages

- Sandboxing Concerns: Sandboxing may not be as robust as purpose-built
  solutions like CEL
- Execution Control: Requires timeouts since programs can run indefinitely
- Library Dependency: Depends on Yaegi, which has limited active development
- Performance: Interpretation overhead compared to compiled code

## Security Considerations

Since the sandboxing capabilities may not be as strong as other solutions,
consider additional isolation mechanisms:

- Process Isolation: Run in separate processes
- seccomp Filters: Restrict system calls (requires Linux)
- Landlock LSM: Restrict file system and tcp network access (requires Linux)
- Linux Namespaces: Isolate resources
- Execution Timeouts: Prevent infinite loops

_Seccomp and Landlock are demonstrated in the example._

## Yaegi Limitations

Yaegi is a Go interpreter with some limitations compared to compiled Go:

- Some reflection capabilities may be restricted
- Performance overhead of interpretation
- Not all Go features may be supported

See the [Yaegi documentation](https://github.com/traefik/yaegi) for complete
details on limitations and supported features.

> [!IMPORTANT]
> Note that Yaegi has limited active development. The original authors no longer
> work at Traefik, and the project is mostly community-supported. Consider this
> when evaluating for production use.
