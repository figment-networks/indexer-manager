all: build-proto build-manager build-manager-migration build-cosmos build-artificial

.PHONY: build-manager
build-manager:
	go build -o manager_bin ./cmd/manager

.PHONY: build-manager-w-scheduler
build-manager-w-scheduler:
	go build -tags scheduler -o manager_bin  ./cmd/manager

.PHONY: build-manager-migration
build-manager-migration:
	go build -o manager_migration_bin ./cmd/manager-migration

.PHONY: build-cosmos
build-cosmos:
	go build -o worker_cosmos_bin ./cmd/worker_cosmos

.PHONY: build-artificial
build-artificial:
	go build -o artificial_source ./cmd/artificial-source

.PHONY: build-scheduler
build-scheduler:
	go build -o scheduler_bin ./cmd/scheduler

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/
