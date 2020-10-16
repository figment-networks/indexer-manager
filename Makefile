
LDFLAGS      := -w -s
MODULE       := github.com/figment-networks/indexer-manager
VERSION_FILE ?= ./VERSION


# Git Status
GIT_SHA ?= $(shell git rev-parse --short HEAD)

ifneq (,$(wildcard $(VERSION_FILE)))
VERSION ?= $(shell head -n 1 $(VERSION_FILE))
else
VERSION ?= n/a
endif

all: generate build-proto build-manager build-manager-migration build-scheduler

.PHONY: generate
generate:
	go generate ./...

.PHONY: build-manager
build-manager: LDFLAGS += -X $(MODULE)/cmd/manager/config.Timestamp=$(shell date +%s)
build-manager: LDFLAGS += -X $(MODULE)/cmd/manager/config.Version=$(VERSION)
build-manager: LDFLAGS += -X $(MODULE)/cmd/manager/config.GitSHA=$(GIT_SHA)
build-manager:
	$(info building manager binary as ./manager_bin)
	go build -o manager_bin -ldflags '$(LDFLAGS)' ./cmd/manager

.PHONY: build-manager-w-scheduler
build-manager-w-scheduler: LDFLAGS += -X $(MODULE)/cmd/manager/config.Timestamp=$(shell date +%s)
build-manager-w-scheduler: LDFLAGS += -X $(MODULE)/cmd/manager/config.Version=$(VERSION)
build-manager-w-scheduler: LDFLAGS += -X $(MODULE)/cmd/manager/config.GitSHA=$(GIT_SHA)
build-manager-w-scheduler:
	$(info building manager binary with embedded scheduler as ./manager_bin)
	go build -tags scheduler -o manager_bin  -ldflags '$(LDFLAGS)'  ./cmd/manager

.PHONY: build-manager-migration
build-manager-migration:
	$(info building migration binary as ./migration)
	go build -o migration ./cmd/manager-migration

.PHONY: build-scheduler
build-scheduler:
	$(info building scheduler binary as ./scheduler)
	go build -o scheduler_bin ./cmd/scheduler

.PHONY: build-proto
build-proto:
	@protoc -I ./ --go_opt=paths=source_relative --go_out=plugins=grpc:. ./proto/indexer.proto
	@mkdir -p ./worker/transport/grpc/indexer
	@mkdir -p ./manager/transport/grpc/indexer
	@cp ./proto/indexer.pb.go ./worker/transport/grpc/indexer/
	@cp ./proto/indexer.pb.go ./manager/transport/grpc/indexer/

.PHONY: pack-release
pack-release:
	$(info preparing release)
	@mkdir -p ./release
	@make build-manager-migration
	@mv ./migration ./release/migration
	@make build-manager-w-scheduler
	@mv ./manager_bin ./release/manager
	@cp -R ./cmd/manager-migration/migrations ./release/
	@zip -r indexer-manager ./release
	@rm -rf ./release


