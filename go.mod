module github.com/aerospike/aerospike-management-lib

go 1.19

require (
	github.com/aerospike/aerospike-client-go/v6 v6.14.0
	github.com/deckarep/golang-set/v2 v2.3.1
	github.com/go-logr/logr v1.2.4
	github.com/xeipuuv/gojsonschema v1.2.0
)

// Pinned this dependcy to fix vulnerbaility in `golang.org/x/net` pkg
replace golang.org/x/net => golang.org/x/net v0.17.0

// Pinned this dependcy to fix vulnerbaility in `google.golang.org/grpc` pkg
replace google.golang.org/grpc => google.golang.org/grpc v1.56.3

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/onsi/ginkgo/v2 v2.9.4 // indirect
	github.com/onsi/gomega v1.27.6 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yuin/gopher-lua v0.0.0-20220504180219-658193537a64 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
