package asconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// map of version to schema
var schemas map[string]string

var validVersionRe = regexp.MustCompile(`(\d+\.){2,2}\d+`)
var defRegex = regexp.MustCompile("(.*).default$")
var dynRegex = regexp.MustCompile("(.*).dynamic$")

// var storageRegex = regexp.MustCompile("(.*).storage-engine$")

// Init initializes aerospike schemas.
// Init needs to be called before using this package.
//
// schemaDir is the path to directory having the aerospike config schemas.
func Init(schemaDir string) error {
	pkglog.V(4).Info("Config schema dir", "dir", schemaDir)
	schemas = make(map[string]string)

	fileInfo, err := ioutil.ReadDir(schemaDir)
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

		schema, err := ioutil.ReadFile(filepath.Join(schemaDir, file.Name()))
		if err != nil {
			return fmt.Errorf("wrong config schema file %s: %v", file.Name(), err)
		}

		schemas[versionFormat(file.Name())] = string(schema)
		pkglog.V(4).Info("Config schema added", "version", versionFormat(file.Name()))
	}

	return nil
}

// InitFromMap init schema map from a map.
// Map key format -> 4.1.0
// Map value format -> string of json schema
func InitFromMap(schemaMap map[string]string) {
	schemas = make(map[string]string)
	for name, schema := range schemaMap {
		pkglog.V(4).Info("Config schema added", "version", name)
		schemas[name] = schema
	}
}

func versionFormat(filename string) string {
	if len(filename) == 0 {
		return ""
	}

	fields := strings.Split(filename, ".")
	name := fields[0]

	if len(fields) > 1 {
		name = strings.Join(fields[:len(fields)-1], ".")
	}

	return strings.Replace(name, "_", ".", -1)
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
func getDynamic(ver string) (map[string]bool, error) {

	flatSchema, err := getFlatSchema(ver)
	if err != nil {
		return nil, err
	}

	dynMap := make(map[string]bool)

	for k, v := range flatSchema {
		switch {
		case dynRegex.MatchString(k):
			key := removeJSONSpecKeywords(k)
			key = dynRegex.ReplaceAllString(key, "${1}")
			if dyn, ok := v.(bool); ok {
				dynMap[key] = dyn
			} else {
				dynMap[key] = false
			}
		}
	}

	return dynMap, nil
}

// getDefault return the map of values which are dynamic
// values.
func getDefault(ver string) (map[string]interface{}, error) {

	flatSchema, err := getFlatSchema(ver)
	if err != nil {
		return nil, err
	}

	defMap := make(map[string]interface{})

	for k, v := range flatSchema {
		switch {
		case defRegex.MatchString(k):
			key := removeJSONSpecKeywords(k)
			key = defRegex.ReplaceAllString(key, "${1}")
			// if storageRegex.MatchString(key) {
			// 	// NOTE Skip .*storage-engine:memory in default
			// 	continue
			// }

			defMap[key] = eval(v)
		}
	}

	return defMap, nil
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
				res[k], err = v.Float64()
			}

		default:
			res[k] = v
		}
	}
	return res
}

func removeJSONSpecKeywords(key string) string {
	// Cleanup json schema strings
	key = strings.Replace(key, "items", "_", -1)
	key = strings.Replace(key, "properties.", "", -1)
	key = strings.Replace(key, ".oneOf.1", "", -1)
	key = strings.Replace(key, ".oneOf.0", "", -1)
	return key
}

// eval casts value v to correct data type
func eval(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		strList := make([]string, len(v))
		for i := range v {
			strList = append(strList, v[i].(string))
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

func isPost315(ver string) bool {
	return gtEqVersion(ver, "3.15.0.0")
}

func isPre315(ver string) bool {
	ver, err := baseVersion(ver)
	if err != nil {
		// FIXME
	}

	gt := gtEqVersion("3.14.0.0", ver)
	return gt
}

func gtEqVersion(v1 string, v2 string) bool {
	// TODO string.Split is allocation can it be avoided
	s1 := strings.Split(v1, sep)
	s2 := strings.Split(v2, sep)

	for len(s1) > len(s2) {
		s2 = append(s2, "0")
	}
	for len(s2) > len(s1) {
		s1 = append(s1, "0")
	}
	loop := len(s1)

	for i := 0; i < loop; i++ {
		if s1[i] > s2[i] {
			return true
		}

		if s1[i] < s2[i] {
			return false
		}
	}
	// Equal
	return true
}
