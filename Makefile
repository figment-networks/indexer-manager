all: build


.PHONY: build
build:
	go build ./cmd/worker
