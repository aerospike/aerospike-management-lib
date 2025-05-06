package asconfig

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/info"
)

var ErrInvalidFormat = fmt.Errorf("invalid config format")

func NewASConfigFromBytes(log logr.Logger, src []byte, srcFmt Format) (*AsConfig, error) {
	var (
		err error
		cfg *AsConfig
	)

	switch srcFmt {
	case YAML:
		cfg, err = loadYAML(log, src)
	case AeroConfig:
		cfg, err = loadAsConf(log, src)
	case Invalid:
		return nil, fmt.Errorf("%w %s", ErrInvalidFormat, srcFmt)
	default:
		return nil, fmt.Errorf("%w %s", ErrInvalidFormat, srcFmt)
	}

	if err != nil {
		return nil, err
	}

	// recreate the management lib config
	// with a sorted config map so that output
	// is always in the same order
	cmap := cfg.ToMap()

	if err = mutateMap(*cmap, []mapping{
		sortLists,
	}); err != nil {
		return nil, fmt.Errorf("failed to sort config map: %w", err)
	}

	cfg, err = NewMapAsConfig(
		log,
		*cmap,
	)

	return cfg, err
}

func loadYAML(log logr.Logger, src []byte) (*AsConfig, error) {
	var data map[string]any

	err := yaml.Unmarshal(src, &data)
	if err != nil {
		return nil, err
	}

	c, err := NewMapAsConfig(
		log,
		data,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize asconfig from yaml: %w", err)
	}

	return c, nil
}

func loadAsConf(log logr.Logger, src []byte) (*AsConfig, error) {
	reader := bytes.NewReader(src)

	// TODO: Why doesn't the management lib do the map mutation? FromConfFile
	// implies it does.
	c, err := FromConfFile(log, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse asconfig file: %w", err)
	}

	// the aerospike management lib parses asconfig files into
	// a format that its validation rejects
	// this is because the schema files are meant to
	// validate the aerospike kubernetes operator's asconfig yaml format
	// so we modify the map here to match that format
	cmap := *c.ToMap()

	// revert the mutation happened in logging section
	logging := lib.DeepCopy(cmap[info.ConfigLoggingContext])

	if err = mutateMap(cmap, []mapping{
		typedContextsToObject,
		ToPlural,
	}); err != nil {
		return nil, fmt.Errorf("failed to mutate config map: %w", err)
	}

	cmap[info.ConfigLoggingContext] = lib.DeepCopy(logging)
	c, err = NewMapAsConfig(
		log,
		cmap,
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// mapping functions get mapped to each key value pair in a management lib Stats map
// m is the map that k and v came from
type mapping func(k string, v any, m Conf) error

// mutateMap maps functions to each key value pair in the management lib's Stats map
// the functions are applied sequentially to each k,v pair.
func mutateMap(in Conf, funcs []mapping) error {
	var errs []error

	keys := lib.GetKeys(in)
	for idx := range keys {
		v := in[keys[idx]]
		switch v := v.(type) {
		case Conf:
			if err := mutateMap(v, funcs); err != nil {
				errs = append(errs, fmt.Errorf("error in nested map for key %s: %w", keys[idx], err))
			}
		case []Conf:
			for i, lv := range v {
				if err := mutateMap(lv, funcs); err != nil {
					errs = append(errs, fmt.Errorf("error in array element %d for key %s: %w", i, keys[idx], err))
				}
			}
		}

		for _, f := range funcs {
			if err := f(keys[idx], in[keys[idx]], in); err != nil {
				errs = append(errs, fmt.Errorf("error in mapping function for key %s: %w", keys[idx], err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors in mutateMap: %v", errs)
	}

	return nil
}

/*
typedContextsToObject converts config entries that the management lib
parses as literal strings into the objects that the yaml schemas expect.
NOTE: As of server 7.0 a context is required for storage-engine memory
so it will no longer be a string. This is still needed for compatibility
with older servers.
Ex Conf

	Conf{
		"storage-engine": "memory"
	}

->

	Conf{
		"storage-engine": Conf{
			"type": "memory"
		}
	}
*/
func typedContextsToObject(k string, _ any, m Conf) error {
	if isTypedSection(k) {
		v := m[k]
		// if a typed context does not have a map value.
		// then it's value is a string like "memory" or "flash"
		// in order to make valid asconfig yaml we convert this context
		// to a map where "type" maps to the value
		if _, ok := v.(Conf); !ok {
			m[k] = Conf{keyType: v}
		}
	}

	return nil
}

/*
sortLists sorts slices of config sections by the "name" or "type"
key that the management lib adds to config list items
Ex config:
namespace ns2 {}
namespace ns1 {}
->
namespace ns1 {}
namespace ns2 {}

Ex matching Conf

	Conf{
		"namespace": []Conf{
			Conf{
				"name": "ns2",
			},
			Conf{
				"name": "ns1",
			},
		}
	}

->

	Conf{
		"namespace": []Conf{
			Conf{
				"name": "ns1",
			},
			Conf{
				"name": "ns2",
			},
		}
	}
*/
func sortLists(k string, v any, m Conf) error {
	if v, ok := v.([]Conf); ok {
		sort.Slice(v, func(i int, j int) bool {
			iv, iok := v[i]["name"]
			jv, jok := v[j]["name"]

			// sections may also use the "type" field to identify themselves
			if !iok {
				iv, iok = v[i]["type"]
			}

			if !jok {
				jv, jok = v[j]["type"]
			}

			// if i or both don't have id fields, consider them i >= j
			if !iok {
				return false
			}

			// if only j has an id field consider i < j
			if !jok {
				return true
			}

			iname := iv.(string)
			jname := jv.(string)

			gt := strings.Compare(iname, jname)

			switch gt {
			case 1:
				return true
			case -1, 0:
				return false
			default:
				panic("unexpected gt value")
			}
		})

		m[k] = v
	}

	return nil
}
