package asconfig

type OpType string

// All the aerospike config related keys
const (
	keyFeatureKeyFile = "feature-key-file"
	keyDevice         = "device"
	keyFile           = "file"
	KeyName           = "name"
	keyType           = "type"
	keyIndex          = "<index>"

	keyStorageEngine             = "storage-engine"
	keyAddress                   = "address"
	keyTLSAddress                = "tls-address"
	keyAccessAddress             = "access-address"
	keyTLSAccessAddress          = "tls-access-address"
	keyAlternateAccessAddress    = "alternate-access-address"
	keyTLSAlternateAccessAddress = "tls-alternate-access-address"
	keyTLSAuthenticateClient     = "tls-authenticate-client"
	keyNodeAddressPorts          = "node-address-ports"

	keyReportDataOp = "report-data-op"
	keyNamespace    = "namespace"
	keySet          = "set"
	keyLogs         = "logs"

	sep                  = "."
	SectionNameStartChar = '{'
	SectionNameEndChar   = '}'
	semicolon            = ";"
	equal                = "="
	colon                = ":"

	// Enum values for OpType
	Add    OpType = "add"
	Remove OpType = "remove"
	Update OpType = "update"
)
