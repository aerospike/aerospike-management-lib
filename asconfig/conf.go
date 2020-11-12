package asconfig

import (
	"bytes"
	"encoding/json"
	"fmt"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/xeipuuv/gojsonschema"
)

// Conf is format for configs
// It has list for named sections like namespace, set, dc, tls, logging file
type Conf = lib.Stats

// DotConf is string of aerospike.conf content
type DotConf = string

// CfgValue is config details
type CfgValue struct {
	Context string
	Name    string
	Value   interface{}
}

// confIsValid checks if passed conf is valid. If it is not valid
// then returns json validation error string. String is nil in case of other
// error conditions.
func confIsValid(flatConf Conf, ver string) (bool, []*ValidationErr, error) {
	confJSON, err := json.Marshal(expandConf(flatConf, sep))
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
	return false, vErrs, ConfigSchemaError
}

// confToDotConf takes Conf as parameter and returns server
// aerospike.conf file. Returns error in case the Conf does
// not adhere to standards.
func confToDotConf(flatConf Conf) DotConf {
	var buf bytes.Buffer

	writeDotConf(&buf, expandConf(flatConf, sep), 0, nil)

	return buf.String()
}
