GOPACKAGES?=$(shell find . -name '*.go' -not -path "./vendor/*" -exec dirname {} \;| sort | uniq)
GOFILES?=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
CWD?=$(shell pwd)
COVER_REPORTS=$(CWD)/reports

all: help

.PHONY: help build run fmt vendor clean test coverage check vet lint

help:
	@echo "fmt            - format application sources"
	@echo "clean          - remove artifacts"
	@echo "test           - run tests"
	@echo "cover          - run tests with coverage, generates coverage reports"
	@echo "check          - check code style"
	@echo "vet            - run go vet"
	@echo "lint           - run golint"
	@echo "vendor         - install the project's dependencies"

fmt:
	go fmt $(GOPACKAGES)

build: clean
	go build -o bin/service-entrypoint ./cmd/service

vendor:
	dep ensure

clean:
	go clean

test: clean
	go test $(GOPACKAGES)

cover: clean
	@mkdir -p $(COVER_REPORTS)
	@for PKG in $(GOPACKAGES) ; do \
		REPORT=$${PKG//[\/.]/_}.out; \
		CONFIG_PATH=$(CONFIG_PATH) go test -coverprofile=$(COVER_REPORTS)/$$REPORT $$PKG; \
	done

check: vet lint

vet:
	go vet $(GOPACKAGES)

lint:
	ls $(GOFILES) | xargs -L1 golint
