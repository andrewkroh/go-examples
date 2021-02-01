ECS_VERSION := 1.7.0

demo: download-flat
	go run main.go -f testdata/fields.txt

download-flat:
ifeq (,$(wildcard ./ecs_flat.yml))
	curl -O https://raw.githubusercontent.com/elastic/ecs/v${ECS_VERSION}/generated/ecs/ecs_flat.yml
endif
	@true

build:
	go build

.PHONY: demo download-flat build
