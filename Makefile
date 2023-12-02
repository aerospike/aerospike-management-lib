# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

MOCKGEN = $(GOBIN)/mockgen
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.50.1

.PHONY: golanci-lint
golanci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(GOBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)

go-lint: golanci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: mocks
mocks: $(MOCKGEN)
	$(MOCKGEN) --source info/as_parser.go --destination info/as_parser_mock.go --package info
	$(MOCKGEN) --source asconfig/generate.go --destination asconfig/generate_mock.go --package asconfig

.PHONY: test
test: mocks
	go test -v ./...

.PHONY: coverage
coverage: 
	go test ./... -coverprofile coverage.cov -coverpkg ./...
	grep -v "_mock.go" coverage.cov > coverage_no_mocks.cov
	mv coverage_no_mocks.cov coverage.cov
	go tool cover -func coverage.cov

.PHONY: clean-mocks
clean-mocks:
	rm info/as_parser_mock.go
	rm asconfig/generate_mock.go

.PHONY: clean
clean: clean-mocks
	rm coverage.cov