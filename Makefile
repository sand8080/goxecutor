GOPACKAGES?=$(shell find . -name '*.go' -not -path "./vendor/*" -exec dirname {} \;| sort | uniq)
GOFLAGS?="-count=1" # Disabling test cache
GO111MODULE=on
GOLANGCI=$(shell which golangci-lint)
COVER_REPORTS?="reports"

all: help

.PHONY: help clean fmt check test coverage

help:
	@echo "clean          - remove artifacts"
	@echo "fmt            - format application sources"
	@echo "check          - check code style"
	@echo "test           - run tests"
	@echo "coverage       - run tests with coverage, generate coverage reports"

clean:
	@go clean ./...

fmt:
	@goimports -local "goxecutor" -w $(GOPACKAGES) || go fmt $(GOPACKAGES)

check:
	@echo "Performing code check"
	@if [ -z "$(GOLANGCI)" ]; then \
		docker run --rm -v "$(PWD):/go/src/goxecutor" \
			-w /go/src/goxecutor golangci/golangci-lint:v1.43 golangci-lint run --config .golangci.yml; \
	else \
		golangci-lint run --config .golangci.yml; \
	fi

test: clean
	@GOFLAGS=$(GOFLAGS) go test -race -tags=integration $(TEST_PARAMS) $(GOPACKAGES)

coverage: TEST_PARAMS=-coverprofile=$(COVER_REPORTS)/coverage.out
coverage: test
coverage:
	@go tool cover -html=reports/coverage.out