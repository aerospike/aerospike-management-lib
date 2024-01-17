package asconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/xeipuuv/gojsonschema"

	lib "github.com/aerospike/aerospike-management-lib"
)

// Conf is format for configs
// It has list for named sections like namespace, set, dc, tls, logging file
type Conf = lib.Stats

// DotConf is string of aerospike.conf content
type DotConf = string

// CfgValue is config details
type CfgValue struct {
	Value   interface{}
	Context string
	Name    string
}

// confIsValid checks if passed conf is valid. If it is not valid
// then returns json validation error string. String is nil in case of other
// error conditions.
func confIsValid(log logr.Logger, flatConf *Conf, ver string) (bool, []*ValidationErr, error) {
	confJSON, err := json.Marshal(expandConf(log, flatConf, sep))
	if err != nil {
		return false, nil, fmt.Errorf("failed to do json.Marshal for flatten aerospike conf: %v", err)
	}

	confLoader := gojsonschema.NewStringLoader(string(confJSON))

	schema, err := getSchema(ver)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get aerospike config schema for version %s: %v", ver, err)
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)

	result, err := gojsonschema.Validate(schemaLoader, confLoader)
	if err != nil {
		return false, nil, err
	}

	if result.Valid() {
		return true, nil, nil
	}

	vErrs := make([]*ValidationErr, 0)

	for _, desc := range result.Errors() {
		vErr := &ValidationErr{
			ErrType:     desc.Type(),
			Context:     desc.Context().String(),
			Description: desc.Description(),
			Field:       desc.Field(),
			Value:       desc.Value(),
		}
		vErrs = append(vErrs, vErr)
	}

	return false, vErrs, ErrConfigSchema
}

func ConfValuesValid(flatConf *Conf) []*ValidationErr {
	vErrs := make([]*ValidationErr, 0)

	var vErr *ValidationErr

	for key, value := range *flatConf {
		baseKey := BaseKey(key)

		switch val := value.(type) {
		case []string:
			vErrs = append(vErrs, validateSlice(baseKey, val)...)

		case string:
			vErrs = append(vErrs, validateString(baseKey, val))

		case bool, int, uint64, int64, float64:
			continue

		case lib.Stats:
			// Ignoring changes in map type as each key is being compared separately eg. security {}.
			continue

		default:
			vErr = &ValidationErr{
				Description: "Unhandled value type in config",
				Field:       key,
				Value:       val,
			}
			vErrs = append(vErrs, vErr)
		}
	}

	return vErrs
}

func validateSlice(baseKey string, val []string) []*ValidationErr {
	vErrs := make([]*ValidationErr, 0)
	for _, v := range val {
		vErrs = append(vErrs, validateString(baseKey, v))
	}

	return vErrs
}

func validateString(baseKey, v string) *ValidationErr {
	literals := strings.Split(v, " ")

	switch baseKey {
	case "node-address-ports":
		if len(literals) > 3 {
			return &ValidationErr{
				Description: "Invalid node-address-ports",
				Field:       baseKey,
				Value:       v,
			}
		}

	case "report-data-op":
		if len(literals) > 2 {
			return &ValidationErr{
				Description: "Invalid report-data-op",
				Field:       baseKey,
				Value:       v,
			}
		}

	default:
		if len(literals) > 1 {
			return &ValidationErr{
				Description: "Invalid value",
				Field:       baseKey,
				Value:       v,
			}
		}
	}

	return nil
}

// confToDotConf takes Conf as parameter and returns server
// aerospike.conf file. Returns error in case the Conf does
// not adhere to standards.
func confToDotConf(log logr.Logger, flatConf *Conf) DotConf {
	var buf bytes.Buffer

	writeDotConf(log, &buf, expandConf(log, flatConf, sep), 0, nil)

	return buf.String()
}
