// Copyright (C) 2018 Aerospike, Inc.
//
// All rights reserved.
//
// THIS IS UNPUBLISHED PROPRIETARY SOURCE CODE. THE COPYRIGHT NOTICE ABOVE DOES
// NOT EVIDENCE ANY ACTUAL OR INTENDED PUBLICATION.

package asconfig

import "fmt"

// ErrConfigParse is config parse error
var ErrConfigParse = fmt.Errorf("config parse error")

// ErrConfigSchema is config schema error
var ErrConfigSchema = fmt.Errorf("config schema error")

// ErrConfigVersionUnsupported is unsupported config version
var ErrConfigVersionUnsupported = fmt.Errorf("unsupported config version")

// ErrConfigVersionInvalid is invalid config version
var ErrConfigVersionInvalid = fmt.Errorf("invalid config version")

// ErrConfigTransformUnsupported is unsupported config transform
var ErrConfigTransformUnsupported = fmt.Errorf("unsupported config transform")

// ErrConfigKeyInvalid is invalid config key error
var ErrConfigKeyInvalid = fmt.Errorf("invalid config key")
