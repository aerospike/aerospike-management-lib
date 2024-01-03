module github.com/aerospike/aerospike-management-lib

go 1.19

require (
	github.com/aerospike/aerospike-client-go/v6 v6.14.0
	github.com/deckarep/golang-set/v2 v2.3.1
	github.com/docker/go-connections v0.4.0
	github.com/go-logr/logr v1.2.3
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/mock v0.3.0
	k8s.io/apimachinery v0.27.2
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)

// Pinned this dependcy to fix vulnerbaility in `golang.org/x/net` pkg
replace golang.org/x/net => golang.org/x/net v0.17.0

// Pinned this dependcy to fix vulnerbaility in `google.golang.org/grpc` pkg
replace google.golang.org/grpc => google.golang.org/grpc v1.56.3

require (
	github.com/docker/docker v24.0.7+incompatible
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/stretchr/testify v1.8.4
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yuin/gopher-lua v0.0.0-20220504180219-658193537a64 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
