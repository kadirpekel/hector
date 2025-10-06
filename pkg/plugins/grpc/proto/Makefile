.PHONY: all clean

# Generate Go code from protocol buffers
all:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		common.proto llm.proto database.proto embedder.proto

clean:
	rm -f *.pb.go

