.PHONY: proto tidy run-grpc run-gw

PROTO_DIR=proto
GEN_DIR=gen/go
THIRD_PARTY=third_party
OPENAPI_DIR=openapi

proto:
	@echo ">> generating protobuf + grpc + gateway + openapi..."
	protoc -I $(PROTO_DIR) -I $(THIRD_PARTY) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GEN_DIR) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=$(OPENAPI_DIR) \
		$(PROTO_DIR)/user/v1/user.proto
	@echo ">> done"

tidy:
	go mod tidy

run-grpc:
	go run ./cmd/grpc-server

run-gw:
	go run ./cmd/gateway