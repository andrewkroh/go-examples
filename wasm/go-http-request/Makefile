build:
	cp "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" .
	GOOS=js GOARCH=wasm go build -o xhr.wasm

run: build
	go get -u github.com/shurcooL/goexec
	@echo "Visit http://localhost:8080/"
	goexec 'http.ListenAndServe(`127.0.0.1:8080`, http.FileServer(http.Dir(`.`)))'
