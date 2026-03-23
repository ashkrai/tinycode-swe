.PHONY: run test generate install-tools build

# Run the demo (requires go mod tidy first)
run:
	go run ./cmd/demo

# Run all tests
test:
	go test ./... -v -count=1

# Build binary
build:
	go build -o bin/demo ./cmd/demo

# Install protoc-gen-go (needed only if regenerating from .proto files)
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
	@echo "Also install protoc: brew install protobuf  (macOS)"

# Regenerate Go code from proto/user_v2.proto
# Run 'make install-tools' first, then 'make generate'
generate:
	mkdir -p internal/user
	protoc \
	  --go_out=internal/user \
	  --go_opt=paths=source_relative \
	  --go_opt=Mproto/user_v2.proto=github.com/ashkrai/protobuf-user/internal/user \
	  --proto_path=proto \
	  proto/user_v2.proto
	@echo "Generated: internal/user/user_v2.pb.go"
	@echo "Note: The pre-written internal/user/user.go implements the same types."
	@echo "      Delete user.go and use user_v2.pb.go if you want the official generated code."

# Download dependencies
deps:
	go mod tidy
