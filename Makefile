all: build


.PHONY: build
build:
	go build ./cmd/worker

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/
