// Copyright (C) 2018 Aerospike, Inc.
//
// All rights reserved.
//
// THIS IS UNPUBLISHED PROPRIETARY SOURCE CODE. THE COPYRIGHT NOTICE ABOVE DOES
// NOT EVIDENCE ANY ACTUAL OR INTENDED PUBLICATION.

package asconfig

import "fmt"

// ConfigParseError is config parse error
var ConfigParseError = fmt.Errorf("config parse error")

// ConfigSchemaError is config schema error
var ConfigSchemaError = fmt.Errorf("config schema error")

// ConfigVersionUnsupported is unsupported config version
var ConfigVersionUnsupported = fmt.Errorf("unsupported config version")

// ConfigVersionInvalid is invalid config version
var ConfigVersionInvalid = fmt.Errorf("invalid config version")

// ConfigTransformUnsupported is unsupported config transform
var ConfigTransformUnsupported = fmt.Errorf("unsupported config transform")

// ConfigKeyInvalid is invalid config key error
var ConfigKeyInvalid = fmt.Errorf("invalid config key")
