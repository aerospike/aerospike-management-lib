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

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/go-logr/logr"
)

type sysproptype string

// types of system properties
const (
	FSPATH  sysproptype = "FSPATH"
	NETADDR sysproptype = "NETADDR"
	DEVICE  sysproptype = "DEVICE"
	NONE    sysproptype = "NONE"
)

const sep = "."

var portRegex = regexp.MustCompile("port")

type humanize func(string) (uint64, error)

func deHumanizeTime(val string) (uint64, error) {
	if len(val) == 0 {
		return 0, nil
	}

	endswith := val[len(val)-1]

	multiplier := uint64(1)

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
	n = n * multiplier
	return n, nil
}

func deHumanizeSize(val string) (uint64, error) {
	if len(val) == 0 {
		return 0, nil
	}

	endswith := val[len(val)-1]

	multiplier := uint64(1)

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
	n = n * multiplier
	return n, nil
}

// start and end charecter for section names
var sectionNameStartChar = '{'
var sectionNameEndChar = '}'

// expandConf expands map with flat keys (with sep) to Conf
func expandConf(log logr.Logger, input *Conf, sep string) Conf {
	m := expandConfMap(log, input, sep)
	return expandConfList(log, m)
}

// expandConfMap expands flat map to Conf by using sep
// it does not check for list sections
func expandConfMap(log logr.Logger, input *Conf, sep string) Conf {
	m := make(Conf)
	for k, v := range *input {
		switch v := v.(type) {
		case Conf:
			m[k] = expandConfMap(log, &v, sep)
		default:
			expandKey(log, m, splitKey(log, k, sep), v)
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
						log.V(-1).Info("Wrong value type for list section",
							"section", k, "key", k2, "key", reflect.TypeOf(v2))
						continue
					}

					// fetch index stored by flattenConf
					index, ok := v2Conf["index"].(int)
					if !ok {
						log.V(-1).Info("Index not available", "section", k,
							"key", k2)
						continue
					}

					confList[index] = expandConfList(log, v2Conf)

					// index is flattenConf generated field, delete it
					delete(confList[index], "index")
					found = true
				}

				if found {
					input[k] = confList
				}

			} else {
				v = expandConfList(log, v)
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
		newK := strings.Replace(k, "_", "-", -1)
		val, ok := v.(Conf)
		if ok {
			updatedConf[newK] = replaceUnderscore(val)
		} else {
			updatedConf[newK] = v
		}
	}
	return updatedConf
}

var namedContextRe = regexp.MustCompile("(namespace|set|dc|tls|datacenter)(=)([^{^}^/]*)")
var loggingContextRe = regexp.MustCompile("(logging)(=)([^{^}]*)")

func toAsConfigContext(context string) string {
	// Internal asConfig representation has {} parenthesis
	// around names in named contexts. And has . in it.
	if loggingContextRe.MatchString(context) {
		// logging filename can have / - avoid further replacements
		asConfigCtx := loggingContextRe.ReplaceAllString(context, fmt.Sprintf("$1.%c$3%c", sectionNameStartChar, sectionNameEndChar))
		return asConfigCtx
	}

	asConfigCtx := namedContextRe.ReplaceAllString(context, fmt.Sprintf("$1.%c$3%c", sectionNameStartChar, sectionNameEndChar))
	asConfigCtx = strings.Replace(asConfigCtx, "/", sep, -1)
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
	return strings.Trim(name, fmt.Sprintf("%c%c", sectionNameStartChar, sectionNameEndChar))
}

// getContainedName returns config name and true if key is part of the passed in
// context, otherwise empty string and false
func getContainedName(log logr.Logger, fullKey string, context string) (string, bool) {
	ctx := toAsConfigContext(context)

	if strings.Contains(fullKey, ctx) {
		fKs := splitKey(log, fullKey, sep)
		cKs := splitKey(log, ctx, sep)

		// Number of keys in fullKey should
		// be 1 more that ctx
		if len(cKs)+1 != len(fKs) {
			return "", false
		}

		return fKs[len(fKs)-1], true
	}
	return "", false
}

// splitKey splits key by using sep
// it ignore sep inside sectionNameStartChar and sectionNameEndChar
func splitKey(log logr.Logger, key, sep string) []string {
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
		if c == sectionNameStartChar {
			openBracket++
		} else if c == sectionNameEndChar {
			openBracket--
		}
		return false
	}

	return strings.FieldsFunc(key, f)
}

func expandKey(log logr.Logger, input Conf, keys []string, val interface{}) bool {
	if len(keys) == 1 {
		return false
	}

	m := input
	i := 0
	for _, k := range keys {
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
		i++
		if i == len(keys)-1 {
			break
		}
	}
	m[keys[len(keys)-1]] = val
	return true
}

// flattenConfList flatten list and save index for expandConf
func flattenConfList(log logr.Logger, input []Conf, sep string) Conf {
	res := make(Conf, len(input))

	for i, v := range input {
		if len(v) == 0 {
			continue
		}
		name, ok := v["name"].(string)
		if !ok {
			log.V(-1).Info("FlattenConfList not possible for ListSection" +
				" without name")
			continue
		}

		// create key for this item as {name}
		// while expanding we are ignoring sep inside {...}
		// still its not complete solution, it fails if user has section names with imbalance paranthesis
		// for ex. namespace name -> test}.abcd
		// but this solution will work for most of the cases and reduce most of the failure scenarios
		name = string(sectionNameStartChar) + name + string(sectionNameEndChar)

		for k2, v2 := range flattenConf(log, v, sep) {
			res[name+sep+k2] = v2
		}
		// store index for expanding in correct order
		res[name+sep+"index"] = i
	}

	return res
}

// flattenConf flatten Conf by creating new key by using sep
func flattenConf(log logr.Logger, input Conf, sep string) Conf {
	res := make(Conf, len(input))
	for k, v := range input {
		switch v := v.(type) {
		case Conf:
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

func baseKey(k string, sep string) (string, []string) {
	s := strings.Split(k, sep)
	return s[len(s)-1], s
}

// Conf is of following values types.
// basicValue : int64, boolean, string
// List : list of string
//      : empty list of interface{} uninitialized list
func isValueDiff(log logr.Logger, v1 interface{}, v2 interface{}) bool {

	if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
		return true
	}

	if v1 == nil && v2 == nil {
		return false
	}

	if v1 == nil || v2 == nil {
		return true
	}

	switch v2.(type) {
	case []string:
		sort.Strings(v1.([]string))
		sort.Strings(v2.([]string))
		if !reflect.DeepEqual(v1, v2) {
			return true
		}

	case string:
		if isEmptyField("", v1.(string)) && isEmptyField("", v2.(string)) {
			return false
		}
		return v1 != v2

	case bool, int, uint64, int64, float64:
		if v1 != v2 {
			return true
		}
	default:
		log.V(1).Info("Unhandled value type in config diff", "type",
			reflect.TypeOf(v2))
		return true
	}
	return false
}

// diff find diff between two configs;
//      diff = c1 - c2
//
// Generally used to compare config from two different nodes. This ignores
// node specific information like address, device, interface etc..
func diff(log logr.Logger, c1, c2 Conf, isFlat, c2IsDefault, ignoreInternalFields bool) Conf {

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
		bN, _ := baseKey(k, sep)
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
				log.V(1).Info("Key not in default map while performing diff"+
					" from default. Ignoring",
					"key", _k)
				// TODO: How to handle dynamic only configs???
				continue
			}
			d[k] = c1[k]
			continue
		}
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

// confDiff find diff between two configs;
//      diff = c1 - c2
func confDiff(log logr.Logger, c1 Conf, c2 Conf, isFlat, ignoreInternalFields bool) map[string]interface{} {
	return diff(log, c1, c2, isFlat, false, ignoreInternalFields)
}

// defaultDiff returns the values different from the default.
// This ignore the node specific value. i
// For all Keys conf
//    diff = flatConf - flatDefConf
func defaultDiff(log logr.Logger, flatConf Conf, flatDefConf Conf) map[string]interface{} {
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
func getSystemProperty(log logr.Logger, c Conf, key string) (stype sysproptype, value []string) {

	baseKey, _ := baseKey(key, sep)
	baseKey = SingularOf(baseKey)
	value = make([]string, 0)

	// Catch all exception for type cast.
	defer func() {
		if r := recover(); r != nil {
			log.V(1).Info("Unexpected type", "type", reflect.TypeOf(c[key]),
				"key", baseKey)
			stype = NONE
		}
	}()

	switch baseKey {

	// device <deviceName>:<shadowDeviceName>
	case "device":
		for _, d := range c[key].([]interface{}) {
			value = append(value, strings.Split(d.(string), ":")...)
		}
		return DEVICE, value

	// file <filename>
	// feature-key-file <filename>
	// work-directory <direname>
	// FIXME FIXME add logging file ...
	case "file", "feature-key-file", "work-directory", "system-path", "user-path":
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
			for _, f := range v {
				value = append(value, f)
			}
			return FSPATH, value
		}
		return NONE, value

	case "xdr-digestlog-path":
		value = append(value, strings.Split(c[key].(string), " ")[0])
		return FSPATH, value

	case "address", "tls-address", "access-address",
		"tls-access-address", "alternate-access-address",
		"tls-alternate-access-address":
		v := c[key]
		switch v := v.(type) {
		case []interface{}:
			for _, f := range v {
				value = append(value, f.(string))
			}
			return NETADDR, value
		case []string:
			for _, f := range v {
				value = append(value, f)
			}
			return NETADDR, value
		}
		return NONE, value

	default:
		return NONE, value
	}
}

// isListField return true if passed in key representing
// aerospike config is of type List that is can have multiple
// entries for same config key.
func isListField(key string) (bool, string) {

	key, _ = baseKey(key, sep)
	key = SingularOf(key)

	switch key {
	case "dc-node-address-port", "tls-node", "dc-int-ext-ipmap":
		return true, "+"

	// TODO: Device with shadow device is not reported by server
	// yet in runtime making it colon separated for now.
	case "mesh-seed-address-port", "tls-mesh-seed-address-port",
		"device", "report-data-op", "node-address-port", "feature-key-file":
		return true, ":"

	case "file", "address", "tls-address", "access-address", "mount",
		"tls-access-address", "alternate-access-address",
		"tls-alternate-access-address", "role-query-pattern",
		"xdr-remote-datacenter", "multicast-group",
		"tls-authenticate-client", "http-url":
		return true, ""
	default:
		return false, ""
	}
}

// isIncompleteSetSectionFields returns true if passed in key
// representing aerospike set config which is incomplete and needs
// 'set-' prefix
func isIncompleteSetSectionFields(key string) bool {
	key, _ = baseKey(key, sep)
	switch key {
	case "disable-eviction", "enable-xdr", "stop-writes-count":
		return true

	default:
		return false
	}
}

func isInternalField(key string) bool {
	key, _ = baseKey(key, sep)
	switch key {
	case "index", "name":
		return true

	default:
		return false
	}
}

func isListSection(section string) bool {

	section, _ = baseKey(section, sep)
	section = SingularOf(section)
	switch section {
	case "namespace", "datacenter", "dc", "set", "tls", "file":
		return true
	default:
		return false
	}
}

// section without name but should consider as list
// for ex. logging
func isSpecialListSection(section string) bool {

	section, _ = baseKey(section, sep)
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

	key, _ = baseKey(key, sep)
	// "name" is id for named sections
	// "storage-engine-type" is type of storage engine.
	switch key {
	case "name", "storage-engine-type":
		return true
	default:
		return false
	}
}

// isEmptyField return true if value is either NULL or "". Also
// for the cases where port number is 0
func isEmptyField(key string, value string) bool {

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

// isSpecialBoolField returns true if the passed key
// in aerospike config is boolean type field but does not
// need true or false in config file. Their mere presence
// config file is true/false.
//    e.g namespace and storage level benchmark fields
func isSpecialBoolField(key string) bool {
	switch key {
	case "enable-benchmark-batch-sub", "enable-benchmarks-read",
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
//    e.g tls-authenticate-client
func isSpecialStringField(key string) bool {
	key, _ = baseKey(key, sep)
	switch key {
	case "tls-authenticate-client":
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
	case "file", "device", "pidfile",
		"node-id", "address", "port", "access-address", "access-port",
		"external-address", "interface-address", "alternate-access-address",
		"tls-address", "tls-port", "tls-access-address", "tls-access-port",
		"tls-alternate-access-address", "tls-alternate-access-port",
		"xdr-info-port", "service-threads", "batch-index-threads",
		"mesh-seed-address-port", "mtu":
		return true
	}
	return false
}

// isNodeSpecificContext returns true if the passed key
// in aerospike config is from Node specific context like logging.
func isNodeSpecificContext(key string) bool {
	if len(key) == 0 || strings.Contains(key, "logging.") {
		return true
	}
	return false
}

func isSizeOrTime(key string) (bool, humanize) {
	switch key {
	case "default-ttl", "max-ttl", "tomb-raider-eligible-age",
		"tomb-raider-period":
		return true, deHumanizeTime

	case "memory-size", "filesize":
		return true, deHumanizeSize

	default:
		return false, nil
	}
}

func isStorageEngineKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	if key == "storage-engine" || strings.HasPrefix(key, "storage-engine.") {
		return true
	}
	return false
}

func addStorageEngineConfig(log logr.Logger, key string, v interface{}, conf Conf) {
	if !isStorageEngineKey(key) {
		return
	}

	storageKey := "storage-engine"

	switch v := v.(type) {

	case map[string]interface{}:
		conf[storageKey] = toConf(log, v)
	case lib.Stats:
		conf[storageKey] = toConf(log, v)
	default:
		vStr, ok := v.(string)
		if key == storageKey {
			if !ok {
				log.V(1).Info("Wrong value type",
					"key", key, "valueType", reflect.TypeOf(v))
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

		key = strings.TrimPrefix(key, "storage-engine.")

		seConf[key] = v
	}
}

// toConf does deep conversion of map[string]interface{}
// into Conf objects. Also converts the list form in conf
// into map form, if required.
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

		switch v := v.(type) {
		case lib.Stats:
			result[k] = toConf(log, v)

		case map[string]interface{}:
			result[k] = toConf(log, v)

		case []map[string]interface{}:
			if len(v) == 0 {
				result[k] = make([]Conf, 0)
			} else {
				temp := make([]Conf, len(v))
				for i, m := range v {
					temp[i] = toConf(log, m)
				}
				result[k] = temp
			}

		case []interface{}:
			if len(v) == 0 {
				if isListSection(k) || isSpecialListSection(k) {
					result[k] = make([]Conf, 0)
				} else if ok, _ := isListField(k); ok {
					result[k] = make([]string, 0)
				} else {
					log.V(1).Info("[]interface neither list field or list section",
						"key", k)
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
					result[k] = temp
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
					result[k] = temp
				default:
					log.V(1).Info("Unexpected value",
						"type", reflect.TypeOf(v), "key", k, "value", v)
				}
			}
		case string:
			if ok, _ := isListField(k); ok && k != "feature-key-file" {
				if k == "tls-authenticate-client" && (v == "any" || v == "false") {
					result[k] = v
				} else {
					result[k] = []string{v}
				}
			} else {
				result[k] = v
			}

		case bool:
			if isSpecialStringField(k) {
				if ok, _ := isListField(k); ok {
					if k == "tls-authenticate-client" && v == false {
						result[k] = strconv.FormatBool(v)
					} else {
						result[k] = []string{strconv.FormatBool(v)}
					}

				} else {
					result[k] = strconv.FormatBool(v)
				}
			} else {
				result[k] = v
			}

		case int64:
			if v < 0 {
				result[k] = v
			} else {
				result[k] = uint64(v)
			}

		case uint64:
			result[k] = v

		case float64:
			// security.syslog.local can be -1
			if v < 0 {
				result[k] = int64(v)
			} else {
				result[k] = uint64(v)
			}

		default:
			result[k] = v
		}
	}
	return result
}

func getCfgValue(log logr.Logger, diffKeys []string, flatConf Conf) []CfgValue {
	var diffValues []CfgValue
	for _, k := range diffKeys {
		context, name := getContextAndName(log, k, "/")
		diffValues = append(diffValues, CfgValue{
			Context: context,
			Name:    name,
			Value:   flatConf[k],
		})
	}
	return diffValues
}

func getContextAndName(log logr.Logger, key, ctxSep string) (string, string) {
	keys := splitKey(log, key, sep)
	if len(keys) == 1 {
		//panic
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

// CompareVersions compares Aerospike Server versions
// if version1 == version2 returns 0
// else if version1 < version2 returns -1
// else returns 1
func CompareVersions(version1, version2 string) (int, error) {
	if len(version1) == 0 || len(version2) == 0 {
		return 0, fmt.Errorf("wrong versions to compare")
	}

	if version1 == version2 {
		return 0, nil
	}

	// Ignoring extra comment tag... found in git source code build
	v1 := strings.Split(version1, "-")[0]
	v2 := strings.Split(version2, "-")[0]

	if v1 == v2 {
		return 0, nil
	}

	verElems1 := strings.Split(v1, ".")
	verElems2 := strings.Split(v2, ".")

	minLen := len(verElems1)
	if len(verElems2) < minLen {
		minLen = len(verElems2)
	}

	for i := 0; i < minLen; i++ {
		ve1, err := strconv.Atoi(verElems1[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}
		ve2, err := strconv.Atoi(verElems2[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		if ve1 > ve2 {
			return 1, nil
		} else if ve1 < ve2 {
			return -1, nil
		}
	}

	if len(verElems1) > len(verElems2) {
		return 1, nil
	}

	if len(verElems1) < len(verElems2) {
		return -1, nil
	}

	return 0, nil
}

// CompareVersionsIgnoreRevision compares Aerospike Server versions ignoring
// revisions and builds.
// if version1 == version2 returns 0
// else if version1 < version2 returns -1
// else returns 1
func CompareVersionsIgnoreRevision(version1, version2 string) (int, error) {
	if len(version1) == 0 || len(version2) == 0 {
		return 0, fmt.Errorf("wrong versions to compare")
	}

	if version1 == version2 {
		return 0, nil
	}

	// Ignoring extra comment tag... found in git source code build
	v1 := strings.Split(version1, "-")[0]
	v2 := strings.Split(version2, "-")[0]

	if v1 == v2 {
		return 0, nil
	}

	verElems1 := strings.Split(v1, ".")
	verElems2 := strings.Split(v2, ".")

	minLen := len(verElems1)
	if len(verElems2) < minLen {
		minLen = len(verElems2)
	}

	if minLen > 2 {
		// Force comparison of only major and minor version.
		minLen = 2
	}

	for i := 0; i < minLen; i++ {
		ve1, err := strconv.Atoi(verElems1[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}
		ve2, err := strconv.Atoi(verElems2[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		if ve1 > ve2 {
			return 1, nil
		} else if ve1 < ve2 {
			return -1, nil
		}
	}

	return 0, nil
}
