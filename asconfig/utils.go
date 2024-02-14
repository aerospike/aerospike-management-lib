// Copyright (C) 2018 Aerospike, Inc.
//
// All rights reserved.
//
// THIS IS UNPUBLISHED PROPRIETARY SOURCE CODE. THE COPYRIGHT NOTICE ABOVE DOES
// NOT EVIDENCE ANY ACTUAL OR INTENDED PUBLICATION.

package asconfig

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	lib "github.com/aerospike/aerospike-management-lib"
)

type sysproptype string

// types of system properties
const (
	FSPATH  sysproptype = "FSPATH"
	NETADDR sysproptype = "NETADDR"
	DEVICE  sysproptype = "DEVICE"
	NONE    sysproptype = "NONE"
)

var portRegex = regexp.MustCompile("port")

type humanize func(string) (uint64, error)

func deHumanizeTime(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}

	endswith := val[len(val)-1]

	var multiplier uint64

	switch endswith {
	case 's', 'S':
		multiplier = 1

	case 'm', 'M':
		multiplier = 60

	case 'h', 'H':
		multiplier = 60 * 60

	case 'd', 'D':
		multiplier = 24 * 60 * 60

	default:
		return strconv.ParseUint(val, 10, 64)
	}

	n, err := strconv.ParseUint(val[:len(val)-1], 10, 64)
	if err != nil {
		return n, err
	}

	n *= multiplier

	return n, nil
}

func deHumanizeSize(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}

	endswith := val[len(val)-1]

	var multiplier uint64

	switch endswith {
	case 'K', 'k':
		multiplier = 1024

	case 'M', 'm':
		multiplier = 1024 * 1024

	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024

	case 'T', 't':
		multiplier = 1024 * 1024 * 1024 * 1024

	case 'P', 'p':
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024

	default:
		return strconv.ParseUint(val, 10, 64)
	}

	n, err := strconv.ParseUint(val[:len(val)-1], 10, 64)
	if err != nil {
		return n, err
	}

	n *= multiplier

	return n, nil
}

// expandConf expands map with flat keys (with sep) to Conf
func expandConf(log logr.Logger, input *Conf, sep string) Conf { //nolint:unparam // We should think about removing the arg 'sep'
	m := expandConfMap(log, input, sep)
	return expandConfList(log, m)
}

// expandConfMap expands flat map to Conf by using sep
// it does not check for list sections
func expandConfMap(log logr.Logger, input *Conf, sep string) Conf {
	m := make(Conf)

	// generate.go adds "security": Conf{} to flatMap to ensure that an empty
	// security context is present in the config. This is required to enable
	// security in server >= 5.7. Sorting the keys ensures "security" is process
	// before "security.*" keys.
	keys := sortKeys(*input)

	for _, k := range keys {
		v := (*input)[k]
		switch v := v.(type) {
		case Conf:
			m[k] = expandConfMap(log, &v, sep)

		default:
			expandKey(log, m, SplitKey(log, k, sep), v)
		}
	}

	return m
}

// expandConfList expands expected list sections to list of Conf
func expandConfList(log logr.Logger, input Conf) Conf {
	for k, val := range input {
		v, ok := val.(Conf)
		if ok {
			if isListSection(k) || isSpecialListSection(k) {
				// expected list section
				confList := make([]Conf, len(v))
				found := false

				for k2, v2 := range v {
					v2Conf, ok := v2.(Conf)
					if !ok {
						log.V(-1).Info(
							"Wrong value type for list section",
							"section", k, "key", k2, "key", reflect.TypeOf(v2),
						)

						continue
					}

					// fetch index stored by flattenConf
					index, ok := v2Conf[keyIndex].(int)
					if !ok {
						log.V(-1).Info("Index not available", "section", k, "key", k2)

						continue
					}

					confList[index] = expandConfList(log, v2Conf)

					// index is flattenConf generated field, delete it
					delete(confList[index], keyIndex)

					found = true
				}

				if found {
					input[k] = confList
				}
			} else {
				input[k] = expandConfList(log, v)
			}
		}
	}

	return input
}

func replaceUnderscore(conf Conf) Conf {
	if len(conf) == 0 {
		return conf
	}

	updatedConf := make(Conf, len(conf))

	for k, v := range conf {
		newK := strings.ReplaceAll(k, "_", "-")

		val, ok := v.(Conf)
		if ok {
			updatedConf[newK] = replaceUnderscore(val)
		} else {
			updatedConf[newK] = v
		}
	}

	return updatedConf
}

var namedContextRe = regexp.MustCompile("(namespace|set|dc|tls|datacenter)(=)([^{^}/]*)")
var loggingContextRe = regexp.MustCompile("(logging)(=)([^{^}]*)")

func toAsConfigContext(context string) string {
	// Internal asConfig representation has {} parenthesis
	// around names in named contexts. And has . in it.
	if loggingContextRe.MatchString(context) {
		// logging filename can have / - avoid further replacements
		asConfigCtx := loggingContextRe.ReplaceAllString(
			context,
			fmt.Sprintf("$1.%c$3%c", SectionNameStartChar, SectionNameEndChar),
		)

		return asConfigCtx
	}

	asConfigCtx := namedContextRe.ReplaceAllString(
		context,
		fmt.Sprintf("$1.%c$3%c", SectionNameStartChar, SectionNameEndChar),
	)
	asConfigCtx = strings.ReplaceAll(asConfigCtx, "/", sep)

	return asConfigCtx
}

// toAsConfigKey Returns key which can be used by Config key.
func toAsConfigKey(context, name string) string {
	// Internal asConfig keys have dots
	return fmt.Sprintf("%s%s%s", toAsConfigContext(context), sep, name)
}

// getRawName trims parenthesis and return raw value of
// named context
func getRawName(name string) string {
	return strings.Trim(
		name, fmt.Sprintf("%c%c", SectionNameStartChar, SectionNameEndChar),
	)
}

// getContainedName returns config name and true if key is part of the passed in
// context, otherwise empty string and false
func getContainedName(log logr.Logger, fullKey, context string) (
	string, bool,
) {
	ctx := toAsConfigContext(context)

	if strings.Contains(fullKey, ctx) {
		fKs := SplitKey(log, fullKey, sep)
		cKs := SplitKey(log, ctx, sep)

		// Number of keys in fullKey should
		// be 1 more that ctx
		if len(cKs)+1 != len(fKs) {
			return "", false
		}

		return fKs[len(fKs)-1], true
	}

	return "", false
}

func expandKey(
	log logr.Logger, input Conf, keys []string, val interface{},
) bool {
	if len(keys) == 1 {
		return false
	}

	m := input
	i := 0

	for _, k := range keys {
		m = processKey(log, k, keys, m)
		i++

		if i == len(keys)-1 {
			break
		}
	}

	m[keys[len(keys)-1]] = val

	return true
}

func processKey(log logr.Logger, k string, keys []string, m Conf) Conf {
	defer func() {
		if r := recover(); r != nil {
			log.Info("Recovered", "key", k, "keys", keys, "err", r)
		}
	}()

	if v, ok := m[k]; ok {
		m = v.(Conf)
	} else {
		m[k] = make(Conf)
		m = m[k].(Conf)
	}

	return m
}

// flattenConfList flatten list and save index for expandConf
func flattenConfList(log logr.Logger, input []Conf, sep string) Conf {
	res := make(Conf, len(input))

	for i, v := range input {
		if len(v) == 0 {
			continue
		}

		name, ok := v[KeyName].(string)
		if !ok {
			// Some lists like for storage-engine, index-type, and sindex-type use "type" instead
			// of "name" in order to be compatible with the schema files.
			name, ok = v[keyType].(string)
			if !ok {
				log.V(-1).Info(
					"FlattenConfList not possible for ListSection" +
						" without name or type",
				)

				continue
			}
		}

		// create key for this item as {name}
		// while expanding we are ignoring sep inside {...}
		// still its not complete solution, it fails if user has section names with imbalance parenthesis
		// for ex. namespace name -> test}.abcd
		// but this solution will work for most of the cases and reduce most of the failure scenarios
		name = string(SectionNameStartChar) + name + string(SectionNameEndChar)

		for k2, v2 := range flattenConf(log, v, sep) {
			res[name+sep+k2] = v2
		}
		// store index for expanding in correct order
		res[name+sep+keyIndex] = i
	}

	return res
}

// flattenConf flatten Conf by creating new key by using sep
func flattenConf(log logr.Logger, input Conf, sep string) Conf {
	res := make(Conf, len(input))

	for k, v := range input {
		switch v := v.(type) {
		case Conf:
			if len(v) == 0 {
				res[k] = v
			}

			for k2, v2 := range flattenConf(log, v, sep) {
				res[k+sep+k2] = v2
			}

		case []Conf:
			for k2, v2 := range flattenConfList(log, v, sep) {
				res[k+sep+k2] = v2
			}

		default:
			res[k] = v
		}
	}

	return res
}

func BaseKey(k string) (baseKey string) {
	s := strings.Split(k, sep)
	return s[len(s)-1]
}

func ContextKey(k string) string {
	contextKey := k

	s := strings.Split(k, sep)
	if len(s) > 1 {
		// Extract the prefix (before the first dot)
		contextKey = s[0]
	}

	return contextKey
}

// Conf is of following values types.
// basicValue : int64, boolean, string
// List : list of string
//
//	: empty list of interface{} uninitialized list
func isValueDiff(log logr.Logger, v1, v2 interface{}) bool {
	if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
		return true
	}

	if v1 == nil && v2 == nil {
		return false
	}

	if v1 == nil || v2 == nil {
		return true
	}

	switch val2 := v2.(type) {
	case []string:
		sort.Strings(v1.([]string))
		sort.Strings(val2)

		if !reflect.DeepEqual(v1, v2) {
			return true
		}

	case string:
		if isEmptyField("", v1.(string)) && isEmptyField("", val2) {
			return false
		}

		return v1 != v2

	case bool, int, uint64, int64, float64:
		if v1 != v2 {
			return true
		}

	case lib.Stats:
		// Ignoring changes in map type as each key is being compared separately eg. security {}.
		return false

	default:
		log.V(1).Info(
			"Unhandled value type in config diff", "type",
			reflect.TypeOf(v2),
		)

		return true
	}

	return false
}

// diff find diff between two configs;
//
//	diff = c1 - c2
//
// Generally used to compare config from two different nodes. This ignores
// node specific information like address, device, interface etc..
func diff(
	log logr.Logger, c1, c2 Conf,
	isFlat, c2IsDefault, ignoreInternalFields bool,
) Conf {
	// Flatten if not flattened already.
	if !isFlat {
		c1 = flattenConf(log, c1, sep)
		c2 = flattenConf(log, c2, sep)
	}

	d := make(Conf)

	// For all keys in C1 if it does not exist in C2
	// or if type or value is different add/update it
	for k, v1 := range c1 {
		// Ignore the node specific details
		bN := BaseKey(k)
		if !c2IsDefault && (isNodeSpecificContext(k) || isNodeSpecificField(bN)) {
			// If we need diff with defaults then we need to consider all fields
			// otherwise ignore nodespcific details
			continue
		}

		// Ignore internal fields
		if ignoreInternalFields && isInternalField(k) {
			continue
		}

		_k := k
		if c2IsDefault {
			// NOTE Default map has all the names in named field as _
			// _k is def key if diff is attempted from default
			_k = changeKey(k)
		}

		// Add if not found in C2
		v2, ok := c2[_k]
		if !ok {
			// NOTE it is not possible to find
			// a value in the conf which does
			// exist in default map, unless user
			// is adding some key which system
			// does not know about.
			if c2IsDefault && !isInternalField(k) {
				log.V(1).Info(
					"Key not in default map while performing diff"+
						" from default. Ignoring",
					"key", _k,
				)
				// TODO: How to handle dynamic only configs???
				continue
			}

			d[k] = c1[k]

			continue
		}
		//nolint:gocritic // revisit this later
		/*
			// FIXME FIXME looks hacky
			// Defaults comes with storage-engine type device and
			// not with memory. Have storage-engine always in diff
			// from default
			if bN == "storage-engine" ||
				bNMinusOne == "storage-engine" ||
				isValueDiff(v, defV) {
				diff[k] = v
			}
		*/
		if isValueDiff(log, v1, v2) {
			d[k] = v1
		}
	}

	return d
}

func handleMissingSection(log logr.Logger, k1 string, c1, c2 Conf, d DynamicConfigMap,
	desiredToActual bool) bool {
	tokens := SplitKey(log, k1, ".")
	for idx, token := range tokens {
		nameKeyPath := strings.Join(tokens[:idx+1], sep) + "." + KeyName
		// Whole section which has "name" as key is not present in c2
		// If token is under "{}", then it is a named section
		if _, okay := c2[nameKeyPath]; ReCurlyBraces.MatchString(token) && !okay {
			operationValueMap := make(map[Operation]interface{})

			if desiredToActual {
				if _, updated := d[k1]; !updated {
					// If desired config has this section, then add it to actual config
					// Using AddOp for named section and slice eg. node-address-ports
					if tokens[len(tokens)-1] == KeyName || reflect.ValueOf(c1[k1]).Kind() == reflect.Slice {
						operationValueMap[Add] = c1[k1]
					} else {
						operationValueMap[Update] = c1[k1]
					}

					d[k1] = operationValueMap
				}
			} else if _, updated := d[nameKeyPath]; !updated {
				// If desired config does not have this section, then remove it from actual config
				operationValueMap[Remove] = c1[nameKeyPath]
				d[nameKeyPath] = operationValueMap
			}

			return true
		}
	}

	return false
}

func handlePartialMissingSection(k1, ver string, current Conf, d DynamicConfigMap) (bool, error) {
	diffUpdated := false
	// Check current for any key which starts with k1
	// if found, then add default value to k2 config parameter
	for k2 := range current {
		if !strings.HasPrefix(k2, k1+".") {
			continue
		}

		defaultMap, err := GetDefault(ver)
		if err != nil {
			return false, err
		}

		defaultValue := getDefaultValue(defaultMap, k2)
		operationValueMap := make(map[Operation]interface{})
		operationValueMap[Update] = defaultValue
		d[k2] = operationValueMap
		diffUpdated = true
	}

	return diffUpdated, nil
}

func handleSliceFields(k string, desired Conf, d DynamicConfigMap, desiredToActual bool) {
	operationValueMap := make(map[Operation]interface{})

	if reflect.ValueOf(desired[k]).Kind() == reflect.Slice {
		if desiredToActual {
			operationValueMap[Add] = desired[k].([]string)
		} else {
			operationValueMap[Remove] = desired[k].([]string)
		}
	} else {
		operationValueMap[Update] = desired[k]
	}

	d[k] = operationValueMap
}

func handleValueDiff(k string, v1, v2 interface{}, d DynamicConfigMap) {
	operationValueMap := make(map[Operation]interface{})

	if reflect.ValueOf(v1).Kind() == reflect.Slice {
		currentSet := sets.NewSet[string]()
		currentSet.Append(v2.([]string)...)

		desiredSet := sets.NewSet[string]()
		desiredSet.Append(v1.([]string)...)

		removedValues := currentSet.Difference(desiredSet)
		if removedValues.Cardinality() > 0 {
			operationValueMap[Remove] = removedValues.ToSlice()
			d[k] = operationValueMap
		}

		addedValues := desiredSet.Difference(currentSet)
		if addedValues.Cardinality() > 0 {
			operationValueMap[Add] = addedValues.ToSlice()
			d[k] = operationValueMap
		}
	} else {
		operationValueMap[Update] = v1
		d[k] = operationValueMap
	}
}

// detailedDiff find diff between two configs;
//
//	detailedDiff = desired - current
//
// Generally used to compare current and desired config. This ignores
// node specific information like address, device, interface etc.
func detailedDiff(log logr.Logger, desired, current Conf, isFlat,
	desiredToActual bool, ver string) (DynamicConfigMap, error) {
	// Flatten if not flattened already.
	if !isFlat {
		desired = flattenConf(log, desired, sep)
		current = flattenConf(log, current, sep)
	}

	d := make(DynamicConfigMap)

	// For all keys in desired if it does not exist in current
	// or if type or value is different add/update/remove it
	for k1, v1 := range desired {
		bN := BaseKey(k1)
		if isNodeSpecificField(bN) || bN == keyIndex {
			// Ignore node specific details and ordering
			continue
		}

		// Add if not found in current
		v2, ok := current[k1]
		if !ok {
			diffUpdated := false
			if diffUpdated = handleMissingSection(log, k1, desired, current, d, desiredToActual); !diffUpdated {
				var err error
				// Add default values to config parameter if available in schema.
				// If k1 is not present in current, then check if any key which starts with k1 is present in current
				// eg. desired has security: {} current has security.log.report-sys-admin: true
				// final diff should be map[security.log.report-sys-admin] = <default value>
				diffUpdated, err = handlePartialMissingSection(k1, ver, current, d)
				if err != nil {
					return nil, err
				}
			}

			if !diffUpdated {
				handleSliceFields(k1, desired, d, desiredToActual)
			}

			continue
		}

		log.V(1).Info(
			"compare", "key",
			k1, "v1", v1, "v2", v2,
		)

		if isValueDiff(log, v1, v2) {
			handleValueDiff(k1, v1, v2, d)
		}
	}

	return d, nil
}

// ConfDiff find diff between two configs;
//
//		diff = desired - current
//	 if any config parameter is present in current but not in desired,
//	 result map will contain the corresponding default value for
//	 that config parameter.
//
// It returns a map of flatten conf key and value(which is another map of added and removed fields, mostly helps in the
// case of list of string fields)
func ConfDiff(
	log logr.Logger, desiredConf, currentConf Conf, isFlat bool, ver string,
) (DynamicConfigMap, error) {
	// Comparing desired and current config
	diffs, err := detailedDiff(log, desiredConf, currentConf, isFlat, true, ver)
	if err != nil {
		return nil, err
	}

	// Comparing current and desired config
	// If any config parameter is present in current but not in desired.
	removedConfigs, err := detailedDiff(log, currentConf, desiredConf, isFlat, false, ver)
	if err != nil {
		return nil, err
	}

	for removedConfigKey := range removedConfigs {
		// If any key difference is already captured while comparing desired and current config in detailedDiff,
		// then ignore it while comparing current and desired config.
		if _, ok := diffs[removedConfigKey]; ok {
			continue
		}

		// If whole string array or map type config is not present in desired config.
		// No default values are available for these configs.
		if _, removed := removedConfigs[removedConfigKey][Remove]; removed {
			diffs[removedConfigKey] = removedConfigs[removedConfigKey]
			continue
		}

		// Setting defaults for atomic keys which are not present in desired config
		defaultMap, err := GetDefault(ver)
		if err != nil {
			return nil, err
		}

		valueMap := make(map[Operation]interface{})
		valueMap[Update] = getDefaultValue(defaultMap, removedConfigKey)
		diffs[removedConfigKey] = valueMap
	}

	return diffs, nil
}

// defaultDiff returns the values different from the default.
// This ignores the node specific value. i
// For all Keys conf
//
//	diff = flatConf - flatDefConf
func defaultDiff(
	log logr.Logger, flatConf Conf, flatDefConf Conf,
) map[string]interface{} {
	return diff(log, flatConf, flatDefConf, true, true, false)
}

var nsRe = regexp.MustCompile(`namespace\.({[^.]+})\.(.+)`)
var setRe = regexp.MustCompile(`namespace\.({[^.]+})\.set\.({[^.]+})\.([^.]+)`)
var dcRe = regexp.MustCompile(`xdr\.datacenter\.({[^.]+})\.(.+)`)
var tlsRe = regexp.MustCompile(`network\.tls\.([^.]+)\.(.+)`)
var logRe = regexp.MustCompile(`logging\.({.+})\.(.+)`)

func changeKey(key string) string {
	if setRe.MatchString(key) {
		key = setRe.ReplaceAllString(key, "namespace._.set._.${3}")
	} else {
		key = nsRe.ReplaceAllString(key, "namespace._.${2}")
	}

	key = dcRe.ReplaceAllString(key, "xdr.datacenter._.${2}")
	key = tlsRe.ReplaceAllString(key, "network.tls._.${2}")
	key = logRe.ReplaceAllString(key, "logging._.${2}")

	return key
}

// getSystemProperty return property type and their stringified
// values
func getSystemProperty(log logr.Logger, c Conf, key string) (
	stype sysproptype, value []string,
) {
	baseKey := BaseKey(key)
	baseKey = SingularOf(baseKey)
	value = make([]string, 0)

	// Catch all exception for type cast.
	defer func() {
		if r := recover(); r != nil {
			log.V(1).Info(
				"Unexpected type", "type", reflect.TypeOf(c[key]),
				"key", baseKey,
			)

			stype = NONE
		}
	}()

	switch baseKey {
	// device <deviceName>:<shadowDeviceName>
	case keyDevice:
		for _, d := range c[key].([]interface{}) {
			value = append(value, strings.Split(d.(string), colon)...)
		}

		return DEVICE, value

	// file <filename>
	// feature-key-file <filename>
	// work-directory <direname>
	// FIXME FIXME add logging file ...
	case keyFile, keyFeatureKeyFile, "work-directory", "system-path", "user-path":
		v := c[key]
		switch v := v.(type) {
		case string:
			value = append(value, v)
			return FSPATH, value

		case []interface{}:
			for _, f := range v {
				value = append(value, f.(string))
			}

			return FSPATH, value

		case []string:
			value = append(value, v...)
			return FSPATH, value
		}

		return NONE, value

	case "xdr-digestlog-path":
		value = append(value, strings.Split(c[key].(string), " ")[0])
		return FSPATH, value

	case keyAddress, keyTLSAddress, keyAccessAddress,
		keyTLSAccessAddress, keyAlternateAccessAddress,
		keyTLSAlternateAccessAddress:
		v := c[key]
		switch v := v.(type) {
		case []interface{}:
			for _, f := range v {
				value = append(value, f.(string))
			}

			return NETADDR, value

		case []string:
			value = append(value, v...)
			return NETADDR, value
		}

		return NONE, value

	default:
		return NONE, value
	}
}

// isListField return true if passed in key representing
// aerospike config is of type List that is can have multiple
// entries for same config key. The separator is the secondary delimiter
// used in the .yml config and in the response returned from the server.
// As opposed to the aerospike.conf file which uses space delimiters.
// Example of different formats:
//
//	 server response:
//			node-address-port=1.1.1.1:3000;2.2.2.2:3000
//	 yaml config:
//			node-address-ports:
//				- 1.1.1.1:3000
//				- 2.2.2.2:3000
//	 aerospike.conf:
//			node-address-port 1.1.1.1 3000
//			node-address-port 2.2.2.2 3000
func isListField(key string) (exists bool, separator string) {
	bKey := BaseKey(key)
	bKey = SingularOf(bKey)

	switch bKey {
	case "dc-node-address-port", "tls-node", "dc-int-ext-ipmap":
		return true, "+"

	// TODO: Device with shadow device is not reported by server
	// yet in runtime making it colon separated for now.
	case "mesh-seed-address-port", "tls-mesh-seed-address-port",
		keyDevice, keyReportDataOp, "node-address-port", keyFeatureKeyFile:
		return true, colon

	case keyFile, keyAddress, keyTLSAddress, keyAccessAddress, "mount",
		keyTLSAccessAddress, keyAlternateAccessAddress,
		keyTLSAlternateAccessAddress, "role-query-pattern",
		"xdr-remote-datacenter", "multicast-group",
		keyTLSAuthenticateClient, "http-url", "report-data-op-user",
		"report-data-op-role":
		return true, ""

	default:
		// TODO: This should use the configuration schema instead.
		// If this field is in singularToPlural or pluralToSingular it is a list field.
		if _, ok := singularToPlural[bKey]; ok && !strings.HasPrefix(key, "logging.") {
			return true, ""
		}

		return false, ""
	}
}

// isIncompleteSetSectionFields returns true if passed in key
// representing aerospike set config which is incomplete and needs
// 'set-' prefix
func isIncompleteSetSectionFields(key string) bool {
	key = BaseKey(key)
	switch key {
	case "disable-eviction", "enable-xdr", "stop-writes-count":
		return true

	default:
		return false
	}
}

func isInternalField(key string) bool {
	key = BaseKey(key)
	switch key {
	case keyIndex, KeyName:
		return true

	default:
		return false
	}
}

func isListSection(section string) bool {
	section = BaseKey(section)
	section = SingularOf(section)

	switch section {
	case keyNamespace, "datacenter", "dc", keySet, "tls", keyFile:
		return true

	default:
		return false
	}
}

// section without name but should consider as list
// for ex. logging
func isSpecialListSection(section string) bool {
	section = BaseKey(section)
	section = SingularOf(section)

	switch section {
	case "logging":
		return true

	default:
		return false
	}
}

// isFormField return true if the field in aerospike Conf is
// not aerospike config value but is present in Conf file by
// virtue of it generated from the config form. Forms are the
// JSON schema for nice form layout in UI.
func isFormField(key string) bool {
	key = BaseKey(key)
	// "name" is id for named sections
	// "storage-engine-type" is type of storage engine.
	switch key {
	case KeyName, "storage-engine-type":
		return true

	default:
		return false
	}
}

// isEmptyField return true if value is either NULL or "". Also,
// for the cases where port number is 0
func isEmptyField(key, value string) bool {
	// "name" is id for named sections
	// "storage-engine-type" is type of storage engine.
	switch {
	case strings.EqualFold(value, "NULL"), strings.EqualFold(value, ""):
		return true

	case portRegex.MatchString(key):
		if value == "0" {
			return true
		}

	default:
		return false
	}

	return false
}

// isSpecialOrNormalBoolField returns true if the passed key
// in aerospike config is boolean type field which can have
// a true/false value in the config or, its mere presence
// indicates a true/false value
// e.g. run-as-daemon fields
func isSpecialOrNormalBoolField(key string) bool {
	return key == "run-as-daemon"
}

// isSpecialBoolField returns true if the passed key
// in aerospike config is boolean type field but does not
// need true or false in config file. Their mere presence
// config file is true/false.
// e.g. namespace and storage level benchmark fields
func isSpecialBoolField(key string) bool {
	switch key {
	case "enable-benchmarks-batch-sub", "enable-benchmarks-read",
		"enable-benchmarks-udf", "enable-benchmarks-write",
		"enable-benchmarks-udf-sub", "enable-benchmarks-storage":
		return true

	default:
		return false
	}
}

// isSpecialStringField returns true if the passed key
// in aerospike config is string type field but can have
// bool value also
// e.g. tls-authenticate-client
func isSpecialStringField(key string) bool {
	key = BaseKey(key)
	switch key {
	case keyTLSAuthenticateClient:
		return true

	default:
		return false
	}
}

// isNodeSpecificField returns true if the passed key
// in aerospike config is Node specific field.
func isNodeSpecificField(key string) bool {
	key = SingularOf(key)
	switch key {
	case keyFile, keyDevice, "pidfile",
		"node-id", keyAddress, "port", keyAccessAddress, "access-port",
		"external-address", "interface-address", keyAlternateAccessAddress,
		keyTLSAddress, "tls-port", keyTLSAccessAddress, "tls-access-port",
		keyTLSAlternateAccessAddress, "tls-alternate-access-port", "alternate-access-port",
		"xdr-info-port", "service-threads", "batch-index-threads",
		"mesh-seed-address-port", "mtu":
		return true
	}

	return false
}

// isNodeSpecificContext returns true if the passed key
// in aerospike config is from Node specific context like logging.
func isNodeSpecificContext(key string) bool {
	if key == "" || strings.Contains(key, "logging.") {
		return true
	}

	return false
}

func isSizeOrTime(key string) (bool, humanize) {
	switch key {
	case "default-ttl", "max-ttl", "tomb-raider-eligible-age",
		"tomb-raider-period", "nsup-period", "migrate-fill-delay":
		return true, deHumanizeTime

	case "memory-size", "filesize", "write-block-size",
		"partition-tree-sprigs", "max-write-cache",
		"mounts-size-limit", "index-stage-size",
		"stop-writes-count", "stop-writes-size",
		"mounts-budget", "data-size",
		"quarantine-allocations":
		return true, deHumanizeSize

	default:
		return false, nil
	}
}

func isStorageEngineKey(key string) bool {
	if key == keyStorageEngine || strings.Contains(key, keyStorageEngine+".") {
		return true
	}

	return false
}

func isTypedSection(key string) bool {
	baseKey := BaseKey(key)
	baseKey = SingularOf(baseKey)

	// TODO: This should be derived from the configuration schema
	switch baseKey {
	case keyStorageEngine, "index-type", "sindex-type":
		return true
	default:
		return false
	}
}

func addStorageEngineConfig(
	log logr.Logger, key string, v interface{}, conf Conf,
) {
	if !isStorageEngineKey(key) {
		return
	}

	storageKey := keyStorageEngine

	switch v := v.(type) {
	case map[string]interface{}:
		conf[storageKey] = toConf(log, v)

	case lib.Stats:
		conf[storageKey] = toConf(log, v)

	default:
		vStr, ok := v.(string)
		if key == storageKey {
			if !ok {
				log.V(1).Info(
					"Wrong value type",
					"key", key, "valueType", reflect.TypeOf(v),
				)

				return
			}

			if vStr == "memory" {
				// in-memory storage-engine
				conf[storageKey] = v
			}

			return
		}

		seConf, ok := conf[storageKey].(Conf)
		if !ok {
			conf[storageKey] = make(Conf)
			seConf = conf[storageKey].(Conf)
		}

		key = strings.TrimPrefix(key, keyStorageEngine+".")

		seConf[key] = v
	}
}

// TODO derive these from the schema file
func isStringField(key string) bool {
	switch key {
	// NOTE: before 7.0 "debug-allocations" was a string field. Since it does not except
	// numeric values it is safe to remove from this table so that it functions as a bool
	// when parsing server 7.0+ config files
	case "tls-name", "encryption", "query-user-password-file", "encryption-key-file",
		keyTLSAuthenticateClient, "mode", "auto-pin", "compression", "user-path",
		"auth-user", "user", "cipher-suite", "ca-path", "write-policy", "vault-url",
		"protocols", "bin-policy", "ca-file", "key-file", "pidfile", "cluster-name",
		"auth-mode", "encryption-old-key-file", "group", "work-directory", "write-commit-level-override",
		"vault-ca", "cert-blacklist", "vault-token-file", "query-user-dn", "node-id",
		"conflict-resolution-policy", "server", "query-base-dn", "node-id-interface",
		"auth-password-file", keyFeatureKeyFile, "read-consistency-level-override",
		"cert-file", "user-query-pattern", "key-file-password", "protocol", "vault-path",
		"user-dn-pattern", "scheduler-mode", "token-hash-method",
		"remote-namespace", "tls-ca-file", "role-query-base-dn", "set-enable-xdr",
		"secrets-tls-context", "secrets-uds-path", "secrets-address-port":
		return true
	}

	return false
}

// isDelimitedStringField returns true for configuration fields that
// are delimited strings, but not members of a list section. The separator
// represents the delimiter used in the .yml config as opposed to the
// aerospike.conf file which normally uses spaces.
// EX: secrets-address-port: 127.0.0.1:3000
func isDelimitedStringField(key string) (exists bool, separator string) {
	if key == "secrets-address-port" {
		return true, colon
	}

	return false, ""
}

// toConf does deep conversion of map[string]interface{}
// into Conf objects. Also converts the list form in conf
// into map form, if required. This is needed when converting a unmarshalled
// yaml file into Conf object.
func toConf(log logr.Logger, input map[string]interface{}) Conf {
	result := make(Conf)

	if len(input) == 0 {
		return result
	}

	for k, v := range input {
		if isStorageEngineKey(k) {
			addStorageEngineConfig(log, k, v, result)
			continue
		}

		handleValueType(log, k, v, result)
	}

	return result
}

func handleValueType(log logr.Logger, key string, value interface{}, result Conf) {
	switch v := value.(type) {
	case lib.Stats:
		result[key] = toConf(log, v)

	case map[string]interface{}:
		result[key] = toConf(log, v)

	case []map[string]interface{}:
		result[key] = convertMapSlice(log, v)

	case []interface{}:
		result[key] = convertInterfaceSlice(log, key, v)

	case string:
		result[key] = convertString(key, v)

	case bool:
		result[key] = convertBool(key, v)

	case int64:
		if v < 0 {
			result[key] = v
		} else {
			result[key] = uint64(v)
		}

	case uint64:
		result[key] = v

	case float64:
		// security.syslog.local can be -1
		if v < 0 {
			result[key] = int64(v)
		} else {
			result[key] = uint64(v)
		}

	default:
		result[key] = value
	}
}

// Add other helper functions here...

func convertMapSlice(log logr.Logger, v []map[string]interface{}) (result []Conf) {
	if len(v) == 0 {
		result = make([]Conf, 0)
	} else {
		temp := make([]Conf, len(v))
		for i, m := range v {
			temp[i] = toConf(log, m)
		}

		result = temp
	}

	return result
}

func convertInterfaceSlice(log logr.Logger, k string, v []interface{}) (result interface{}) {
	if len(v) == 0 {
		if isListSection(k) || isSpecialListSection(k) {
			result = make([]Conf, 0)
		} else if ok, _ := isListField(k); ok {
			result = make([]string, 0)
		} else {
			log.V(1).Info(
				"[]interface neither list field or list section",
				"key", k,
			)
		}
	} else {
		v1 := v[0]
		switch v1.(type) {
		case string:
			temp := make([]string, len(v))
			for i, s := range v {
				if boolVal, isBool := s.(bool); isBool && isSpecialStringField(k) {
					temp[i] = strconv.FormatBool(boolVal)
				} else {
					temp[i] = s.(string)
				}
			}

			result = temp

		case map[string]interface{}, lib.Stats:
			temp := make([]Conf, len(v))
			for i, m := range v {
				m1, ok := m.(map[string]interface{})
				if !ok {
					m1, ok = m.(lib.Stats)
				}
				if ok {
					temp[i] = toConf(log, m1)
				} else {
					log.V(1).Info("[]Conf does not have map[string]interface{}")
					break
				}
			}

			result = temp

		default:
			log.V(1).Info(
				"Unexpected value",
				"type", reflect.TypeOf(v), "key", k, "value", v,
			)
		}
	}

	return result
}

func convertString(k, v string) (result interface{}) {
	if ok, _ := isListField(k); ok && k != keyFeatureKeyFile {
		if k == keyTLSAuthenticateClient && (v == "any" || v == "false") {
			result = v
		} else {
			result = []string{v}
		}
	} else {
		result = v
	}

	return result
}

func convertBool(k string, v bool) (result interface{}) {
	if isSpecialStringField(k) {
		if ok, _ := isListField(k); ok {
			if k == keyTLSAuthenticateClient && !v {
				result = strconv.FormatBool(v)
			} else {
				result = []string{strconv.FormatBool(v)}
			}
		} else {
			result = strconv.FormatBool(v)
		}
	} else {
		result = v
	}

	return result
}

func getCfgValue(log logr.Logger, diffKeys []string, flatConf Conf) []CfgValue {
	diffValues := make([]CfgValue, 0, len(diffKeys))

	for _, k := range diffKeys {
		context, name := getContextAndName(log, k, "/")

		diffValues = append(
			diffValues, CfgValue{
				Context: context,
				Name:    name,
				Value:   flatConf[k],
			},
		)
	}

	return diffValues
}

func getContextAndName(log logr.Logger, key, _ string) (context, name string) {
	keys := SplitKey(log, key, sep)
	if len(keys) == 1 {
		return "", ""
	}

	ctx := ""

	for i := 0; i < len(keys)-1; i++ {
		if isListSection(keys[i]) || isSpecialListSection(keys[i]) {
			ctx = fmt.Sprintf("%s/%s=%s", ctx, keys[i], getRawName(keys[i+1]))
			i++
		} else {
			ctx = fmt.Sprintf("%s/%s", ctx, keys[i])
		}
	}

	return strings.Trim(ctx, "/"), keys[len(keys)-1]
}

func getFlatKey(tokens []string) string {
	var key string

	for _, token := range tokens {
		if ReCurlyBraces.MatchString(token) {
			key += "_."
		} else {
			key = key + token + "."
		}
	}

	return strings.TrimSuffix(key, ".")
}

func ToPlural(k string, v any, m Conf) {
	// convert asconfig fields/contexts that need to be plural
	// in order to create valid asconfig yaml.
	if plural := PluralOf(k); plural != k {
		// if the config item can be plural or singular and is not a slice
		// then the item should not be converted to the plural form.
		// If the management lib ever parses list entries as anything other
		// than []string this might have to change.
		if isListOrString(k) {
			if _, ok := v.([]string); !ok {
				return
			}

			if len(v.([]string)) == 1 {
				// the management lib parses all config fields
				// that are in singularToPlural as lists. If these
				// fields are actually scalars then overwrite the list
				// with the single value
				m[k] = v.([]string)[0]
				return
			}
		}

		delete(m, k)
		m[plural] = v
	}
}

// isListOrString returns true for special config fields that may be a
// single string value or a list with multiple strings in the schema files
// NOTE: any time the schema changes to make a value
// a string or a list (array) that value needs to be added here
func isListOrString(name string) bool {
	switch name {
	case keyFeatureKeyFile, keyTLSAuthenticateClient:
		return true
	default:
		return false
	}
}

var ReCurlyBraces = regexp.MustCompile(`^\{.*\}$`)

// DynamicConfigMap is a map of config flatten keys and their operations and values
// for eg: "xdr.dcs.{DC3}.node-address-ports": {Remove: []string{"1.1.2.1 3000"}}
type DynamicConfigMap map[string]map[Operation]interface{}

// SplitKey splits key by using sep
// it ignores sep inside sectionNameStartChar and sectionNameEndChar
func SplitKey(log logr.Logger, key, sep string) []string {
	sepRunes := []rune(sep)
	if len(sepRunes) > 1 {
		log.Info("Split expects single char as separator")
		return nil
	}

	openBracket := 0
	f := func(c rune) bool {
		if c == sepRunes[0] && openBracket == 0 {
			return true
		}

		if c == SectionNameStartChar {
			openBracket++
		} else if c == SectionNameEndChar {
			openBracket--
		}

		return false
	}

	return strings.FieldsFunc(key, f)
}
