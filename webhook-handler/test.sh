#!/bin/bash

set -e

echo "Webhook Handler Test Script"
echo "============================"
echo

# Check if webhook handler is running
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Error: Webhook handler is not running on port 8080"
    echo "Please start the webhook handler first:"
    echo "  ./webhook-handler -handler=handlers/passthrough.go"
    exit 1
fi

echo "✓ Webhook handler is running"
echo

# Test 1: Passthrough handler
echo "Test 1: Simple passthrough"
echo "--------------------------"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data", "timestamp": "2024-01-15T10:00:00Z"}'
echo
echo

# Test 2: JSON enrichment
echo "Test 2: JSON with metadata"
echo "--------------------------"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"event": "user.login", "user_id": "12345"}'
echo
echo

# Test 3: Array of events
echo "Test 3: Array of events"
echo "-----------------------"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '[{"event": "login", "user": "alice"}, {"event": "logout", "user": "bob"}]'
echo
echo

# Test 4: Filtered event (error level)
echo "Test 4: Filtered event (error level)"
echo "------------------------------------"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"level": "error", "message": "Database connection failed"}'
echo
echo

# Test 5: Filtered event (info level - should be filtered out if only error/warning allowed)
echo "Test 5: Filtered event (info level)"
echo "-----------------------------------"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"level": "info", "message": "User logged in"}'
echo
echo

# Test 6: Payload too large (if max is 10MB, send > 10MB)
echo "Test 6: Payload size limit"
echo "-------------------------"
# Generate a large payload (1KB repeated)
LARGE_PAYLOAD=$(printf '{"data":"%s"}' "$(head -c 1000 /dev/zero | tr '\0' 'x')")
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d "$LARGE_PAYLOAD"
echo
echo

echo "✓ All tests completed"
echo
echo "To view events in Kafka:"
echo "  docker exec -it kafka kafka-console-consumer.sh \\"
echo "    --bootstrap-server localhost:9092 \\"
echo "    --topic webhooks \\"
echo "    --from-beginning"
