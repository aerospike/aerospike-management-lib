package asconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	"github.com/aerospike/aerospike-management-lib/info"
)

// map of version to schema
var schemas map[string]string

var validVersionRe = regexp.MustCompile(`(\d+\.){2}\d+`)
var defRegex = regexp.MustCompile("(.*).default$")
var dynRegex = regexp.MustCompile("(.*).dynamic$")
var reqRegex = regexp.MustCompile(`(.*?)\.?required$`)

// var storageRegex = regexp.MustCompile("(.*).storage-engine$")

// Init initializes aerospike schemas.
// Init needs to be called before using this package.
//
// schemaDir is the path to directory having the aerospike config schemas.
func Init(log logr.Logger, schemaDir string) error {
	log.V(1).Info("Config schema dir", "dir", schemaDir)

	schemas = make(map[string]string)

	fileInfo, err := os.ReadDir(schemaDir)
	if err != nil {
		return err
	}

	if len(fileInfo) == 0 {
		return fmt.Errorf("no config schema file available in %s", schemaDir)
	}

	for _, file := range fileInfo {
		if file.IsDir() {
			// no need to check recursively
			continue
		}

		schema, err := os.ReadFile(filepath.Clean(filepath.Join(schemaDir, file.Name())))
		if err != nil {
			return fmt.Errorf("wrong config schema file %s: %v", file.Name(), err)
		}

		schemas[versionFormat(file.Name())] = string(schema)

		log.V(1).Info("Config schema added", "version", versionFormat(file.Name()))
	}

	return nil
}

// InitFromMap init schema map from a map.
// Map key format -> 4.1.0
// Map value format -> string of json schema
func InitFromMap(log logr.Logger, schemaMap map[string]string) {
	schemas = make(map[string]string)

	for name, schema := range schemaMap {
		log.V(1).Info("Config schema added", "version", name)

		schemas[name] = schema
	}
}

func versionFormat(filename string) string {
	if filename == "" {
		return ""
	}

	fields := strings.Split(filename, ".")
	name := fields[0]

	if len(fields) > 1 {
		name = strings.Join(fields[:len(fields)-1], ".")
	}

	return strings.ReplaceAll(name, "_", ".")
}

// isSupportedVersion returns true if server version supported by ACC
func isSupportedVersion(ver string) (bool, error) {
	baseVersion, err := baseVersion(ver)

	if err != nil {
		return false, err
	}

	_, ok := schemas[baseVersion]
	if ok {
		return true, nil
	}

	return false, nil
}

// getSchema returns JSON schema string based on the passed in version.
// return nil if not supported.
func getSchema(ver string) (string, error) {
	baseVersion, err := baseVersion(ver)
	if err != nil {
		return baseVersion, err
	}

	schema, ok := schemas[baseVersion]
	if ok {
		return schema, nil
	}

	return "", fmt.Errorf("unsupported version")
}

// BaseVersion returns baseVersion for ver
func baseVersion(ver string) (string, error) {
	baseVersion := validVersionRe.FindString(ver)
	if baseVersion == "" {
		return baseVersion, fmt.Errorf("invalid version")
	}

	return baseVersion, nil
}

// getDynamic return the map of values which are dynamic
// values.
func getDynamic(ver string) (sets.Set[string], error) {
	flatSchema, err := getFlatSchema(ver)
	if err != nil {
		return nil, err
	}

	return getDynamicSchema(flatSchema), nil
}

func normalizeFlatSchema(flatSchema map[string]interface{}) map[string]interface{} {
	normMap := make(map[string]interface{})

	keys := sortKeys(flatSchema)
	for _, k := range keys {
		v := flatSchema[k]
		key := removeJSONSpecKeywords(k)
		normMap[key] = eval(v)
	}

	return normMap
}

// getDynamicSchema return the map of values which are dynamic
// values.
func getDynamicSchema(flatSchema map[string]interface{}) sets.Set[string] {
	dynSet := sets.NewSet[string]()

	for k, v := range flatSchema {
		if dynRegex.MatchString(k) {
			key := removeJSONSpecKeywords(k)
			key = dynRegex.ReplaceAllString(key, "${1}")

			if dyn, ok := v.(bool); ok && dyn {
				dynSet.Add(key)
			}
		}
	}

	return dynSet
}

// IsAllDynamicConfig returns true if all the fields in the given configMap are dynamically configured.
func IsAllDynamicConfig(log logr.Logger, configMap DynamicConfigMap, version string) (bool, error) {
	dynamic, err := getDynamic(version)
	if err != nil {
		// retry error fall back to rolling restart.
		return false, err
	}

	for confKey := range configMap {
		if !isDynamicConfig(log, dynamic, confKey, configMap[confKey]) {
			return false, nil
		}
	}

	return true, nil
}

// isDynamicConfig returns true if the given field is dynamically configured.
func isDynamicConfig(log logr.Logger, dynamic sets.Set[string], conf string,
	valueMap map[Operation]interface{}) bool {
	tokens := SplitKey(log, conf, sep)
	baseKey := tokens[len(tokens)-1]
	context := tokens[0]

	if context == info.ConfigXDRContext {
		// XDR context is always considered static.
		return false
	}

	if baseKey == "replication-factor" {
		return true
	}

	// Marking these fields as static as corresponding set-config commands are not straight forward.
	staticFieldSet := sets.NewSet("rack-id")
	if staticFieldSet.Contains(baseKey) {
		return false
	}

	// Marking these fields as static as removing an entry from these slices is not supported dynamically.
	conditionalStaticFieldSet := sets.NewSet("ignore-bins", "ignore-sets", "ship-bins", "ship-sets")
	if conditionalStaticFieldSet.Contains(baseKey) {
		if _, ok := valueMap[Remove]; ok {
			return false
		}
	}

	return dynamic.Contains(getFlatKey(tokens))
}

// getDefaultValue returns the default value of a particular config
func getDefaultValue(defaultMap map[string]interface{}, conf string) interface{} {
	tokens := strings.Split(conf, ".")

	return defaultMap[getFlatKey(tokens)]
}

// GetDefault return the map of default values.
func GetDefault(ver string) (map[string]interface{}, error) {
	flatSchema, err := getFlatSchema(ver)
	if err != nil {
		return nil, err
	}

	return getDefaultSchema(flatSchema), nil
}

// getDefaultSchema return the map of values which are default
// values.
func getDefaultSchema(flatSchema map[string]interface{}) map[string]interface{} {
	defMap := make(map[string]interface{})
	removedKeys := map[string]struct{}{}

	for k, v := range flatSchema {
		if defRegex.MatchString(k) {
			key := removeJSONSpecKeywords(k)
			key = defRegex.ReplaceAllString(key, "${1}")

			// If the key is already in the map then we might want to remove it.
			// If the default is always the same then we can remove it. This is
			// helpful for many of the "type" keys which are under a "oneOf" or
			// "anyOf" which means the default is only meaningful to a specific
			// configuration.

			if _, removed := removedKeys[key]; !removed {
				if val, ok := defMap[key]; ok {
					switch val := val.(type) {
					case []string:
						if !reflect.DeepEqual(val, eval(v).([]string)) {
							removedKeys[key] = struct{}{}

							delete(defMap, key)
						}
					default:
						if eval(v) != val {
							removedKeys[key] = struct{}{}

							delete(defMap, key)
						}
					}
				} else {
					defMap[key] = eval(v)
				}
			}
		}
	}

	return defMap
}

// getRequiredSchema returns a map of string to slice of slices of required keys for a given context.
// Multiple slices are required because the required keys can be different
// depending on the "type" of the context.
func getRequiredSchema(flatSchema map[string]interface{}) map[string][][]string {
	keys := sortKeys(flatSchema)
	reqMap := make(map[string][][]string) // We end up with 8 keys with a 6.4 schema.

	for _, k := range keys {
		v := flatSchema[k]

		if reqRegex.MatchString(k) {
			key := removeJSONSpecKeywords(k)
			key = reqRegex.ReplaceAllString(key, "${1}")
			requiredKeys := eval(v)

			if _, ok := reqMap[key]; !ok {
				reqMap[key] = [][]string{}
			}

			// There a multiple "required" keys for a given context. Likely
			// caused by "oneOf" or "anyOf" in the schema.
			reqMap[key] = append(reqMap[key], requiredKeys.([]string))
		}
	}

	return reqMap
}

func flattenSchema(input map[string]interface{}, sep string) map[string]interface{} {
	res := make(map[string]interface{}, len(input))

	for k, v := range input {
		switch v := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range flattenSchema(v, sep) {
				res[k+sep+k2] = v2
			}

		case []interface{}:
			if len(v) == 0 {
				res[k] = v
			} else {
				for i, v2 := range v {
					switch v2 := v2.(type) {
					case map[string]interface{}:
						for k3, v3 := range flattenSchema(v2, sep) {
							res[k+sep+fmt.Sprintf("%d", i)+sep+k3] = v3
						}

					default:
						res[k] = v
					}
				}
			}

		case json.Number:
			if val, err := strconv.ParseUint(v.String(), 10, 64); err == nil {
				res[k] = val
			} else if val, err := v.Int64(); err == nil {
				if val >= 0 {
					// all configs are uint64, cast to uint64
					res[k] = uint64(val)
				} else {
					// negative value, keep it as int64
					res[k] = val
				}
			} else {
				res[k], _ = v.Float64()
			}

		default:
			res[k] = v
		}
	}

	return res
}

var oneOfRegex = regexp.MustCompile(`\.oneOf\.\d+`)
var anyOfRegex = regexp.MustCompile(`\.anyOf\.\d+`)

func removeJSONSpecKeywords(key string) string {
	// Cleanup json schema strings
	key = strings.ReplaceAll(key, "items", "_")
	key = strings.ReplaceAll(key, "properties.", "")
	key = oneOfRegex.ReplaceAllString(key, "")
	key = anyOfRegex.ReplaceAllString(key, "")

	return key
}

// eval casts value v to correct data type
func eval(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		strList := make([]string, len(v))
		for i := range v {
			strList[i] = v[i].(string)
		}

		return strList
	default:
		return v
	}
}

func getFlatSchema(ver string) (map[string]interface{}, error) {
	schemaJSON, err := getSchema(ver)
	if err != nil {
		return nil, err
	}

	schema := make(map[string]interface{})
	d := json.NewDecoder(strings.NewReader(schemaJSON))
	d.UseNumber()

	if err := d.Decode(&schema); err != nil {
		return nil, err
	}

	return flattenSchema(schema, sep), nil
}
