GOPACKAGES?=$(shell find . -name '*.go' -not -path "./vendor/*" -exec dirname {} \;| sort | uniq)
GOFLAGS?="-count=1" # Disabling test cache
GO111MODULE=on
GOLANGCI=$(shell which golangci-lint)
COVER_REPORTS=reports

all: help

.PHONY: help run fmt vendor clean test coverage check vet lint

help:
	@echo "fmt            - format application sources"
	@echo "check          - check code style"
	@echo "test           - run tests"
	@echo "cover          - run tests with coverage, generate coverage reports"
	@echo "clean          - remove artifacts"

fmt:
	@goimports -local "goxecutor" -w $(GOPACKAGES)  || go fmt $(GOPACKAGES)

check:
	@echo "Performing code check"
	@if [ -z "$(GOLANGCI)" ]; then \
		docker run --rm -v "$(PWD):/go/src/goxecutor" \
			-w /go/src/goxecutor golangci/golangci-lint golangci-lint run --config .golangci.yml; \
	else \
		golangci-lint run --config .golangci.yml; \
	fi

clean:
	@go clean ./...

test: clean
	@GOFLAGS=$(GOFLAGS) go test $(TEST_PARAMS) $(GOPACKAGES) ./...

coverage: TEST_PARAMS=-coverprofile=$(COVER_REPORTS)/coverage.out
coverage: test
