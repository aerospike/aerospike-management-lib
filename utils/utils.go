package utils //nolint:revive // utils is an acceptable package name for utility functions

// TODO derive these from the schema file
func IsStringField(key string) bool {
	switch key {
	// NOTE: before 7.0 "debug-allocations" was a string field. Since it does not except
	// numeric values it is safe to remove from this table so that it functions as a bool
	// when parsing server 7.0+ config files
	case "tls-name", "encryption", "query-user-password-file", "encryption-key-file",
		"tls-authenticate-client", "mode", "auto-pin", "compression", "user-path",
		"auth-user", "user", "cipher-suite", "ca-path", "write-policy", "vault-url",
		"protocols", "bin-policy", "ca-file", "key-file", "pidfile", "cluster-name",
		"auth-mode", "encryption-old-key-file", "group", "work-directory", "write-commit-level-override",
		"vault-ca", "cert-blacklist", "vault-token-file", "query-user-dn", "node-id",
		"conflict-resolution-policy", "server", "query-base-dn", "node-id-interface",
		"auth-password-file", "feature-key-file", "read-consistency-level-override",
		"cert-file", "user-query-pattern", "key-file-password", "protocol", "vault-path",
		"user-dn-pattern", "scheduler-mode", "token-hash-method",
		"remote-namespace", "tls-ca-file", "role-query-base-dn",
		"secrets-tls-context", "secrets-uds-path", "secrets-address-port",
		"default-password-file", "ship-versions-policy":
		return true
	}

	return false
}
