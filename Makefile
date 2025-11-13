# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

MOCKGEN ?= $(LOCALBIN)/mockgen
MOCKGEN_VERSION ?= v0.3.0
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.6.1

.PHONY: golanci-lint
golanci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

go-lint: golanci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: get-mockgen
get-mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)

.PHONY: mocks
mocks: get-mockgen
	$(MOCKGEN) --source info/as_parser.go --destination info/as_parser_mock.go --package info
	$(MOCKGEN) --source asconfig/generate.go --destination asconfig/generate_mock.go --package asconfig
	$(MOCKGEN) --source deployment/deployment.go --destination deployment/deployment_mock.go --package deployment

.PHONY: test
test: mocks
	go test -v ./...

.PHONY: coverage
coverage: mocks
	go test ./... -coverprofile coverage.cov -coverpkg ./...
	grep -v "_mock.go" coverage.cov > coverage_no_mocks.cov && mv coverage_no_mocks.cov coverage.cov
	grep -v "test/" coverage.cov > coverage_no_mocks.cov && mv coverage_no_mocks.cov coverage.cov
	go tool cover -func coverage.cov

.PHONY: clean-mocks
clean-mocks:
	rm info/as_parser_mock.go
	rm asconfig/generate_mock.go
	rm deployment/deployment_mock.go

.PHONY: clean
clean: clean-mocks
	rm coverage.cov

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef