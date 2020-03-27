package main

import (
	"fmt"
	"io/ioutil"

	"github.com/perlin-network/life/exec"
)

func main() {
	wasm, err := ioutil.ReadFile("sample.wasm")
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(wasm, exec.VMConfig{}, &exec.NopResolver{}, nil)
	if err != nil {
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport("_start") // can be changed to your own exported function
	if !ok {
		panic("_start entry function not found")
	}

	fmt.Println("Globals: ", vm.Module.Base.Export.Names)

	ret, err := vm.Run(entryID)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}
	fmt.Printf("return value = %d\n", ret)
}
