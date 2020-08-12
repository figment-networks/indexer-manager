all: build

.PHONY: build-manager-migration
build-manager-migration:
	go build -o migration ./cmd/manager-migration

.PHONY: build-cosmos
build-cosmos:
	go build -o worker_cosmos_bin ./cmd/worker_cosmos


.PHONY: build-terra
build-terra:
	go build -o worker_terra_bin ./cmd/worker_terra

.PHONY: build-coda
build-coda:
	go build -o worker_coda_bin ./cmd/worker_coda

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/
