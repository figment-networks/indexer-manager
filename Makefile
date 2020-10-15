all: generate build-proto build-manager build-manager-migration build-scheduler

.PHONY: generate
generate:
	go generate ./...

.PHONY: build-manager
build-manager:
	go build -o manager_bin ./cmd/manager

.PHONY: build-manager-w-scheduler
build-manager-w-scheduler:
	go build -tags scheduler -o manager_bin  ./cmd/manager

.PHONY: build-manager-migration
build-manager-migration:
	go build -o migration ./cmd/manager-migration

.PHONY: build-scheduler
build-scheduler:
	go build -o scheduler ./cmd/scheduler

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/

.PHONY: pack-release
pack-release:
	@mkdir -p ./release
	@make build-manager-migration
	@mv ./migration ./release/migration
	@make build-manager-w-scheduler
	@mv ./manager_bin ./release/manager
	@cp -R ./cmd/manager-migration/migrations ./release/
	@zip -r indexer-manager ./release
	@rm -rf ./release


