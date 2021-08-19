package main

import (
	"fmt"
	"io/ioutil"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func main() {
	wasmBytes, err := ioutil.ReadFile("sample.wasm")
	if err != nil {
		panic(err)
	}

	// Create an Engine
	engine := wasmer.NewEngine()

	// Create a Store
	store := wasmer.NewStore(engine)

	fmt.Println("Compiling module...")
	module, err := wasmer.NewModule(store, wasmBytes)

	if err != nil {
		fmt.Println("Failed to compile module:", err)
	}

	// Create an empty import object.
	importObject := wasmer.NewImportObject()

	fmt.Println("Instantiating module...")
	// Let's instantiate the Wasm module.
	instance, err := wasmer.NewInstance(module, importObject)

	if err != nil {
		panic(fmt.Sprintln("Failed to instantiate the module:", err))
	}

	// We now have an instance ready to be used.
	//
	// From an `Instance` we can fetch any exported entities from the Wasm module.
	// Each of these entities is covered in others examples.
	//
	// Here we are fetching an exported function. We won't go into details here
	// as the main focus of this example is to show how to create an instance out
	// of a Wasm module and have basic interactions with it.
	startFunc, err := instance.Exports.GetFunction("_start")

	if err != nil {
		panic(fmt.Sprintln("Failed to get the `_start` function:", err))
	}

	fmt.Println("Calling `_start` function...")
	result, err := startFunc()
	if err != nil {
		panic(fmt.Sprintln("Failed to call the `_start` function:", err))
	}

	fmt.Println("Results of `_start_start`:", result)
}
