package accesscontrol

// PrivilegeScope enumerates valid scopes for privileges.
type PrivilegeScope int

const (
	// Global scoped privileges.
	Global PrivilegeScope = iota

	// NamespaceSet is namespace and optional set scoped privilege.
	NamespaceSet
)

// Privileges are all privilege string allowed in the spec and associated scopes.
var Privileges = map[string][]PrivilegeScope{
	"read":           {Global, NamespaceSet},
	"write":          {Global, NamespaceSet},
	"read-write":     {Global, NamespaceSet},
	"read-write-udf": {Global, NamespaceSet},
	"data-admin":     {Global},
	"sys-admin":      {Global},
	"user-admin":     {Global},
	"truncate":       {Global, NamespaceSet},
	"sindex-admin":   {Global},
	"udf-admin":      {Global},
}

// Post6Privileges are post version 6.0 privilege strings allowed in the spec and associated scopes.
var Post6Privileges = map[string][]PrivilegeScope{
	"truncate":     {Global, NamespaceSet},
	"sindex-admin": {Global},
	"udf-admin":    {Global},
}
