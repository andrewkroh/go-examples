package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func main() {
	wasmBytes, err := ioutil.ReadFile("sample.wasm")
	if err != nil {
		panic(err)
	}

	wm, err := newWasmModule(wasmBytes)
	if err != nil {
		log.Fatal("Failed to create module:", err)
	}

	rtn, err := wm.process()
	if err != nil {
		log.Fatal("Failed to execute process().", err)
	}
	log.Println("Done. Return code: ", rtn)
}

type wasmModule struct {
	instance *wasmer.Instance

	mallocFunc  wasmer.NativeFunction
	processFunc wasmer.NativeFunction
}

func newWasmModule(wasmData []byte) (*wasmModule, error) {
	// Create an Engine
	engine := wasmer.NewEngine()

	// Create a Store
	store := wasmer.NewStore(engine)

	log.Println("Compiling module...")
	module, err := wasmer.NewModule(store, wasmData)
	if err != nil {
		return nil, fmt.Errorf("failed to compile module: %w", err)
	}

	wm := &wasmModule{}

	importObject := wasmer.NewImportObject()
	importObject.Register(
		"env",
		map[string]wasmer.IntoExtern{
			"get_field": wasmer.NewFunction(
				store,
				wasmer.NewFunctionType(
					wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
					wasmer.NewValueTypes(wasmer.I32)),
				wm.getField,
			),
			"log_it": wasmer.NewFunction(
				store,
				wasmer.NewFunctionType(
					wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32),
					wasmer.NewValueTypes(wasmer.I32),
				),
				wm.log,
			),
		},
	)

	wm.instance, err = wasmer.NewInstance(module, importObject)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate the module: %w", err)
	}

	wm.mallocFunc, err = wm.instance.Exports.GetFunction("malloc")
	if err != nil {
		return nil, fmt.Errorf("failed to find malloc export: %w", err)
	}

	wm.processFunc, err = wm.instance.Exports.GetFunction("process")
	if err != nil {
		return nil, fmt.Errorf("failed to find process export: %w", err)
	}

	return wm, nil
}

func (m *wasmModule) getField(args []wasmer.Value) ([]wasmer.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("get_field requires 4 arguments, but got %d", len(args))
	}

	dataPtr := args[0].I32()
	dataLen := args[1].I32()
	rtnPtr := args[2].I32()
	rtnLen := args[3].I32()

	memory, err := m.instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, fmt.Errorf("failed to get the `memory` memory: %w", err)
	}

	data := memory.Data()[dataPtr : dataPtr+dataLen]
	log.Println("get_field: ", string(data))

	if string(data) == "foo/bar" {
		value := "hello"
		valueSize := int32(len(value))

		valuePtr, err := m.malloc(valueSize)
		if err != nil {
			return nil, err
		}

		// Copy into allocated memory.
		copy(memory.Data()[valuePtr:valuePtr+valueSize], value)

		binary.LittleEndian.PutUint32(memory.Data()[rtnPtr:rtnPtr+4], uint32(valuePtr))
		binary.LittleEndian.PutUint32(memory.Data()[rtnLen:rtnLen+4], uint32(valueSize))

		return []wasmer.Value{wasmer.NewI32(0)}, nil
	}

	return []wasmer.Value{wasmer.NewI32(0)}, nil
}

func (m *wasmModule) log(args []wasmer.Value) ([]wasmer.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("log requires 3 arguments, but got %d", len(args))
	}

	level := args[0].I32()
	dataPtr := args[1].I32()
	dataLen := args[2].I32()

	memory, err := m.instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, fmt.Errorf("failed to get the `memory` memory: %w", err)
	}

	data := memory.Data()[dataPtr : dataPtr+dataLen]
	log.Printf("log[%d]: %s", level, string(data))
	return []wasmer.Value{wasmer.NewI32(0)}, nil
}

func (m *wasmModule) malloc(size int32) (wasmPointer int32, err error) {
	ptr, err := m.mallocFunc(size)
	if err != nil {
		return 0, err
	}
	return ptr.(int32), nil
}

func (m *wasmModule) process() (int32, error) {
	rtn, err := m.processFunc()
	if err != nil {
		return 0, err
	}
	return rtn.(int32), nil
}
