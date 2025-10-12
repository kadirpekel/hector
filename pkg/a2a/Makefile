# A2A Protocol - Code Generation
# This Makefile is self-contained within the pkg/a2a package

.PHONY: generate
generate:
	@echo "🔧 Generating Go code from official A2A proto..."
	@PATH="$(shell go env GOPATH)/bin:$$PATH" protoc --proto_path=proto \
		--go_out=pb \
		--go-grpc_out=pb \
		--grpc-gateway_out=pb \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_opt=paths=source_relative \
		--grpc-gateway_opt=generate_unbound_methods=true \
		proto/a2a.proto
	@echo "✅ Generation complete (gRPC + REST gateway)"

.PHONY: clean
clean:
	@echo "🧹 Cleaning generated files..."
	rm -f pb/*.go
	@echo "✅ Cleaned"

.PHONY: update-proto
update-proto:
	@echo "📥 Downloading latest official A2A proto..."
	curl -s -o proto/a2a.proto https://raw.githubusercontent.com/a2aproject/A2A/refs/heads/main/specification/grpc/a2a.proto
	@echo "✅ Proto updated"

