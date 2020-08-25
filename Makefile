all: build-proto build-manager build-manager-migration build-cosmos build-terra build-coda build-artificial

.PHONY: build-manager
build-manager:
	go build -o manager_bin ./cmd/manager

.PHONY: build-manager-migration
build-manager-migration:
	go build -o manager_migration_bin ./cmd/manager-migration

.PHONY: build-cosmos
build-cosmos:
	go build -o worker_cosmos_bin ./cmd/worker_cosmos

.PHONY: build-terra
build-terra:
	go build -o worker_terra_bin ./cmd/worker_terra

.PHONY: build-coda
build-coda:
	go build -o worker_coda_bin ./cmd/worker_coda

.PHONY: build-artificial
build-artificial:
	go build -o artificial_source ./cmd/artificial-source

.PHONY: build-scheduler
build-scheduler:
	go build -o scheduler_bin ./cmd/scheduler

.PHONY: build-scheduler-migration
build-scheduler-migration:
	go build -o scheduler_migration_bin ./cmd/scheduler-migration

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/
