VERSION := 0.0.1
SOURCE := crowbird.com/andrewkroh/ciscoios
PLUGIN_DIR := ~/.terraform.d/plugins/${SOURCE}/${VERSION}/darwin_amd64

all: install

build: fmt tidy test
	go build

check: fmt tidy
	git diff-index --exit-code HEAD

fmt:
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports
	goimports -w -local github.com/andrewkroh/crowbird-infrastructure .

install: build test
	mkdir -p ${PLUGIN_DIR}
	cp -p ./terraform-provider-ciscoios ${PLUGIN_DIR}/terraform-provider-ciscoios_v${VERSION}
	chmod a+x ./terraform-provider-ciscoios ${PLUGIN_DIR}/terraform-provider-ciscoios_v${VERSION}
	rm testdata/.terraform.lock.hcl || true

test:
	go test ./...

tidy:
	go mod tidy

.PHONY: all build check fmt install test tidy
