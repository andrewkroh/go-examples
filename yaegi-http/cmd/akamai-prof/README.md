# akamai-prof

`akamai-prof` runs the interpreted Akamai program and captures pprof data while it runs.

## How To Run

From `yaegi-http`:

```bash
go run ./cmd/akamai-prof -prog testdata/programs/akamai.go -duration 2m -cpu-profile-duration 30s -out profiles-2m
```

Flags:

- `-prog` (default `testdata/programs/akamai.go`): interpreted Yaegi program path.
- `-duration` (default `2m`): total run time.
- `-cpu-profile-duration` (default `30s`): CPU profile duration (taken at midpoint).
- `-out` (default `profiles`): output directory for pprof files.

## Environment Variables

This runner forwards all `YAEGI_HTTP_*` variables into the interpreted program config.

Required:

- `YAEGI_HTTP_URL` (for example `https://.../siem/v1`)
- `YAEGI_HTTP_CLIENT_TOKEN`
- `YAEGI_HTTP_CLIENT_SECRET`
- `YAEGI_HTTP_ACCESS_TOKEN`

Common optional variables:

- `YAEGI_HTTP_CONFIG_ID` (default `1`)
- `YAEGI_HTTP_LIMIT` (default `1000`)
- `YAEGI_HTTP_INITIAL_INTERVAL` (default `1h`, used to compute initial `from` when `FROM` is unset)
- `YAEGI_HTTP_FROM` (explicit initial unix timestamp override)
- `YAEGI_HTTP_POLL_TIMEOUT` (if unset, runner sets it to `-duration`)
- `YAEGI_HTTP_MAX_REQUESTS`
- `YAEGI_HTTP_POLL_INTERVAL`
- `YAEGI_HTTP_OFFSET_TTL`
- `YAEGI_HTTP_TO_LAG`
- `YAEGI_HTTP_CURSOR`
- `YAEGI_HTTP_HEADERS_TO_SIGN`

## Example Command

```bash
YAEGI_HTTP_URL="https://proteus-akamai-5a50ea16.sit.estc.dev/siem/v1" \
YAEGI_HTTP_CLIENT_TOKEN="your-client-token" \
YAEGI_HTTP_CLIENT_SECRET="your-client-secret" \
YAEGI_HTTP_ACCESS_TOKEN="your-access-token" \
YAEGI_HTTP_LIMIT=50000 \
YAEGI_HTTP_INITIAL_INTERVAL=1h \
go run ./cmd/akamai-prof -duration 2m -cpu-profile-duration 30s -out profiles-2m
```

## What It Measures

- Total events emitted by the interpreted Akamai program callback.
- Elapsed runtime and computed throughput (`events/sec`).
- Heap profile at midpoint.
- CPU profile for the configured CPU profile window, starting at midpoint.
- Heap profile at end.

## Console Output

At completion, it prints one summary log line:

```text
done elapsed=<duration> events=<count> eps=<events_per_second> out=<output_dir>
```

Example:

```text
done elapsed=2m0.001s events=373056 eps=3108.77 out=profiles-2m
```

If execution or profiling fails, it exits non-zero with an error log.

## Profiling Notes (Run: profiles-2m-2)

Run command used (credentials redacted):

```bash
cd /Users/akroh/code/personal/go-examples/yaegi-http && \
YAEGI_HTTP_URL="https://proteus-akamai-5a50ea16.sit.estc.dev/siem/v1" \
YAEGI_HTTP_CLIENT_TOKEN="<redacted>" \
YAEGI_HTTP_CLIENT_SECRET="<redacted>" \
YAEGI_HTTP_ACCESS_TOKEN="<redacted>" \
YAEGI_HTTP_LIMIT="60000" \
YAEGI_HTTP_INITIAL_INTERVAL="5h" \
go run ./cmd/akamai-prof -prog testdata/programs/akamai.go -duration 2m -cpu-profile-duration 30s -out profiles-2m-2
```

Observed run result:

```text
akamai_siem requests=26 events=1560000 ... last_event_unix=1771340890
done elapsed=1m55.658s events=1560000 eps=13488.09 out=profiles-2m-2
```

CPU profile highlights (`cpu_mid.pprof`):

- Dominant cost is I/O + decompression path:
  - `syscall.syscall` about 40% flat
  - `compress/flate` / `compress/gzip` read path about 40% cumulative
- Yaegi interpreter + reflection dispatch is also significant:
  - `github.com/traefik/yaegi/interp.*` call stack up to about 54% cumulative
  - `reflect.Value.Call` path about 47% cumulative
- JSON decode is present but smaller than I/O/interpreter overhead:
  - `encoding/json.Unmarshal` about 5.7% cumulative
- GC work is visible in cumulative runtime stacks (`gcDrain`, `scanobject`, mark workers), consistent with high allocation churn.

Heap profile highlights (`heap_mid.pprof`, `heap_end.pprof`):

- Live heap (in-use) remained low:
  - midpoint: about 8.37 MB
  - end: about 7.18 MB
- No sign of monotonic heap growth in these snapshots.
- Largest in-use contributors are runtime/worker structures and Yaegi interpreter setup state.

Allocation profile highlights (`alloc_space`):

- Total allocated bytes are large despite low live heap:
  - midpoint cumulative alloc: about 13.31 GB
  - end cumulative alloc: about 20.16 GB
- Main alloc sources:
  - `yaegi/interp.newFrame`
  - `reflect.New`
  - `encoding/json.(*decodeState).objectInterface`
  - scanner string materialization (`bufio.Scanner.Text`) and JSON unquote/string conversion.

Interpretation:

- Throughput near 13.5k eps is currently bounded more by compressed HTTP read + interpreter/reflect overhead and allocation churn than by peak resident memory.

## Yaegi vs Direct Comparison (2-minute runs)

This section compares:

- Interpreted run: `profiles-2m-2` (`akamai-prof` + Yaegi program)
- Native run: `profiles-direct-2m` (`akamai-direct-prof` with equivalent Akamai logic)

### Throughput

- Yaegi interpreted: `13488.09 eps`
- Direct native: `13555.26 eps`

The delta is small (~0.5%), so interpreter overhead is not the dominant throughput limiter in this workload.

### CPU profile comparison

Interpreted (`profiles-2m-2/cpu_mid.pprof`):

- Yaegi/reflect dispatch is significant:
  - `github.com/traefik/yaegi/interp.*` up to ~53.68% cumulative
  - `reflect.Value.Call` path ~47.35% cumulative
- I/O + decompression path is also large:
  - syscall/TLS/http2/gzip stack roughly ~40%
- JSON decode appears but is secondary:
  - `encoding/json.Unmarshal` ~5.73% cumulative

Direct (`profiles-direct-2m/cpu_mid.pprof`):

- No Yaegi/reflect-call wall.
- Dominated by I/O + decompression:
  - `syscall.syscall` ~59.61% flat
  - gzip/flate read path ~60% cumulative
- JSON decode cost is more visible:
  - `encoding/json.Unmarshal` ~11.03% cumulative

Summary: removing Yaegi shifts CPU from interpreter/reflect overhead to I/O/decompression and JSON decode, but overall eps remains similar because transport/decompression is the main limiter.

### Live heap comparison (`inuse_space`)

- Yaegi midpoint: ~8.37 MB
- Yaegi end: ~7.18 MB
- Direct midpoint: ~4.80 MB
- Direct end: ~4.28 MB

Direct uses less live heap (~40% lower), and both runs appear stable (no sign of monotonic growth in the sampled snapshots).

### Allocation churn (`alloc_space`)

Interpreted cumulative allocations:

- midpoint: ~13.31 GB
- end: ~20.16 GB

Direct cumulative allocations:

- midpoint: ~5.26 GB
- end: ~8.00 GB

Direct reduces allocation churn by roughly 2.5x. The interpreted run spends more allocations in Yaegi frame/reflect internals (`yaegi/interp.newFrame`, `reflect.New`) in addition to JSON decode and scanner string materialization.

## Reduced Parsing Experiment (2-minute runs)

Goal: reduce JSON allocation by avoiding full event decoding. Instead of decoding each event into `map[string]any`, forward raw NDJSON as `{"message":"<raw-line>"}` and parse only control fields needed by the client (`offset`, `total`, `limit`, and minimal timestamp fields for fallback state).

Compared profiles:

- Yaegi baseline: `profiles-2m-2`
- Yaegi reduced-parse: `profiles-yaegi-2m-minparse`
- Direct baseline: `profiles-direct-2m`
- Direct reduced-parse: `profiles-direct-2m-minparse`

### Throughput (full 2-minute)

- Yaegi baseline: `13488.09 eps`
- Yaegi reduced-parse: `13478.38 eps`
- Direct baseline: `13555.26 eps`
- Direct reduced-parse: `13465.12 eps`

Result: throughput remained in the same band (~13.4k eps). Reduced parsing did not materially increase eps.

### Allocation churn (`alloc_space`, end snapshot)

- Yaegi baseline: ~20.16 GB
- Yaegi reduced-parse: ~25.80 GB
- Direct baseline: ~8.00 GB
- Direct reduced-parse: ~3.50 GB

Result:

- Direct path improved significantly in allocation churn (~56% reduction).
- Yaegi path regressed in allocation churn (~28% increase), likely because the reduced-parse path still performs per-line decoding and the interpreted execution path amplifies per-call/per-type overhead.

### Interpretation

- For direct/native execution, reduced parsing is still useful for memory-pressure and GC-effort reduction.
- For Yaegi execution, reduced parsing as currently implemented does not reduce allocator pressure and should not be considered a performance win yet.
- In both paths, eps stayed near ~13.5k because transport/decompression and single-stream polling are still the dominant bottlenecks.

## Updated Conclusion

If the target is substantially higher throughput (for example ~30k eps), the most likely path is **parallelism across independent streams**, not just further single-stream micro-optimizations.

Practical direction:

- Run multiple workers in parallel across multiple `configId` values.
- Maintain cursor/state independently per `configId`.
- Aggregate outputs downstream (for example to Elasticsearch) while preserving per-stream ordering.

Single-stream optimization still matters for efficiency, but current evidence indicates it is unlikely to double throughput on its own.
