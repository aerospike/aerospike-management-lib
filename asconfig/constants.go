package asconfig

type (
	OpType string
	Format string
)

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
	keyNodeID                    = "node-id"

	keyReportDataOp           = "report-data-op"
	keyReportDataOpUser       = "report-data-op-user"
	keyReportDataOpRole       = "report-data-op-role"
	keyNamespace              = "namespace"
	keySet                    = "set"
	keyLogs                   = "logs"
	keyTLS                    = "tls"
	keyMeshSeedAddressPort    = "mesh-seed-address-port"
	keyTLSMeshSeedAddressPort = "tls-mesh-seed-address-port"

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

	Invalid    Format = ""
	YAML       Format = "yaml"
	AeroConfig Format = "asconfig"
)
