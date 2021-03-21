package asconfig

// Maps singular array names to plurals

var singularToPlural = map[string]string{
	"access-address":               "access-addresses",
	"address":                      "addresses",
	"alternate-access-address":     "alternate-access-addresses",
	"datacenter":                   "datacenters",
	"dc":                           "dcs",
	"dc-int-ext-ipmap":             "dc-int-ext-ipmap",
	"dc-node-address-port":         "dc-node-address-ports",
	"device":                       "devices",
	"file":                         "files",
	"mount":                        "mounts",
	"http-url":                     "http-urls",
	"ignore-bin":                   "ignore-bins",
	"ignore-set":                   "ignore-sets",
	"logging":                      "logging",
	"mesh-seed-address-port":       "mesh-seed-address-ports",
	"multicast-group":              "multicast-groups",
	"namespace":                    "namespaces",
	"node-address-port":            "node-address-ports",
	"report-data-op":               "report-data-op",
	"role-query-pattern":           "role-query-patterns",
	"set":                          "sets",
	"ship-bin":                     "ship-bins",
	"ship-set":                     "ship-sets",
	"tls":                          "tls",
	"tls-access-address":           "tls-access-addresses",
	"tls-address":                  "tls-addresses",
	"tls-alternate-access-address": "tls-alternate-access-addresses",
	"tls-mesh-seed-address-port":   "tls-mesh-seed-address-ports",
	"tls-node":                     "tls-nodes",
	"xdr-remote-datacenter":        "xdr-remote-datacenters",
}

var pluralToSingular map[string]string = map[string]string{}

func init() {
	// Create the reverse mapping.
	for k, v := range singularToPlural {
		pluralToSingular[v] = k
	}
}

// PluralOf returns the plural for of the input noun.
func PluralOf(noun string) string {
	plural, ok := singularToPlural[noun]

	if !ok {
		return noun
	}

	return plural
}

// SingularOf returns the singular for of the input noun.
func SingularOf(noun string) string {
	singular, ok := pluralToSingular[noun]

	if !ok {
		return noun
	}

	return singular
}
