.PHONY: build run test docker-up docker-down clean help

# Build the webhook handler binary
build:
	go build -o webhook-handler .

# Run with passthrough handler (requires Kafka on localhost:9092)
run: build
	./webhook-handler -handler=handlers/passthrough.go

# Run with enrichment handler
run-enrichment: build
	WEBHOOK_SOURCE=local ./webhook-handler -handler=handlers/json_enrichment.go

# Run with array splitter handler
run-splitter: build
	./webhook-handler -handler=handlers/json_array_splitter.go

# Run with filtering handler
run-filter: build
	WEBHOOK_ALLOWED_LEVELS=error,warning ./webhook-handler -handler=handlers/filtering.go

# Start Docker Compose environment
docker-up:
	docker-compose up --build -d

# Stop Docker Compose environment
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f webhook-handler

# Run tests against running service
test:
	./test.sh

# Clean build artifacts
clean:
	rm -f webhook-handler
	rm -f go.sum
	docker-compose down -v

# Show help
help:
	@echo "Webhook Handler - Makefile Commands"
	@echo "===================================="
	@echo ""
	@echo "build              - Build the webhook-handler binary"
	@echo "run                - Run with passthrough handler"
	@echo "run-enrichment     - Run with JSON enrichment handler"
	@echo "run-splitter       - Run with array splitter handler"
	@echo "run-filter         - Run with filtering handler"
	@echo "docker-up          - Start Docker Compose environment"
	@echo "docker-down        - Stop Docker Compose environment"
	@echo "docker-logs        - View webhook-handler logs"
	@echo "test               - Run test script"
	@echo "clean              - Clean build artifacts and Docker volumes"
	@echo "help               - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make docker-up"
	@echo "  make test"
	@echo "  make docker-logs"
