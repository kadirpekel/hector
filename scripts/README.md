# Scripts

This directory contains utility scripts and tools for development, testing, and maintenance.

## Go Utilities

### `populate-qdrant-test.go`

Utility to populate Qdrant vector database with test documents for development and testing.

**Usage:**
```bash
# Ensure Ollama is running with nomic-embed-text model
# Ensure Qdrant is running on localhost:6334

go run scripts/populate-qdrant-test.go
```

This script:
- Indexes documents from `test-docs/cooking` into `cooking_docs` collection
- Indexes documents from `test-docs/programming` into `programming_docs` collection
- Uses Ollama embedder (nomic-embed-text model)
- Creates embeddings and stores them in Qdrant

### `zk-put.go`

Utility to upload configuration files to ZooKeeper for distributed configuration testing.

**Usage:**
```bash
# Upload a config file to ZooKeeper
cat config.yaml | go run scripts/zk-put.go -path /hector/config -servers 127.0.0.1:2181
```

**Options:**
- `-path`: ZooKeeper path where config will be stored (required)
- `-servers`: ZooKeeper server addresses, comma-separated (default: 127.0.0.1:2181)

**Example:**
```bash
# Upload config to ZooKeeper
cat configs/weather-assistant.yaml | \
  go run scripts/zk-put.go \
    -path /hector/configs/weather-assistant \
    -servers 127.0.0.1:2181

# Then use it with Hector
hector serve --config zookeeper://127.0.0.1:2181/hector/configs/weather-assistant
```

## Notes

- These are development/testing utilities, not part of the main Hector binary
- They can be run directly with `go run` or compiled with `go build`
- Make sure required services (Ollama, Qdrant, ZooKeeper) are running before use

