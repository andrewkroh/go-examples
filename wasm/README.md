# WebAssembly Runtime Example

This example executes a Rust program compiled to WebAssembly (`sample.wasm`)
inside of a Go binary. The Go binary uses a Wasmer as its WebAssembly runtime.
go-wasmer provides bindings that use cgo. The wasmer libary is
bundled into go-wasmer (for certain architectures).

![diagram](wasm-diagram.png)

To execute:

`go run wasm.go`
