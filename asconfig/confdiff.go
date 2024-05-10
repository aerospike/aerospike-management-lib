package asconfig

import (
	"reflect"
	"sort"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	lib "github.com/aerospike/aerospike-management-lib"
)

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

func handleMissingSection(log logr.Logger, key string, desired, current Conf, d DynamicConfigMap,
	desiredToActual bool) bool {
	tokens := SplitKey(log, key, sep)
	for idx, token := range tokens {
		nameKeyPath := strings.Join(tokens[:idx+1], sep) + sep + KeyName
		// Whole section which has "name" as key is not present in current
		// If token is under "{}", then it is a named section
		if _, okay := current[nameKeyPath]; ReCurlyBraces.MatchString(token) && !okay {
			operationValueMap := make(map[Operation]interface{})

			if desiredToActual {
				if _, updated := d[key]; !updated {
					// If desired config has this section, then add it to actual config
					// Using AddOp for named section and slice eg. node-address-ports
					if tokens[len(tokens)-1] == KeyName || reflect.ValueOf(desired[key]).Kind() == reflect.Slice {
						operationValueMap[Add] = desired[key]
					} else {
						operationValueMap[Update] = desired[key]
					}

					d[key] = operationValueMap
				}
			} else if _, updated := d[nameKeyPath]; !updated {
				// If desired config does not have this section, then remove it from actual config
				operationValueMap[Remove] = desired[nameKeyPath]
				d[nameKeyPath] = operationValueMap
			}

			return true
		}
	}

	return false
}

func handlePartialMissingSection(desiredKey, ver string, current Conf, d DynamicConfigMap) (bool, error) {
	diffUpdated := false
	// Check current conf for any key which starts with desiredKey
	// if found, then add default value to currentKey config parameter
	for currentKey := range current {
		if !strings.HasPrefix(currentKey, desiredKey+sep) {
			continue
		}

		operationValueMap := make(map[Operation]interface{})
		// If removed subsection is of type slice, then there is no default values to be set.
		// eg. current = security.log.report-data-op: []string{test}
		// desired = security: {}
		if reflect.ValueOf(current[currentKey]).Kind() == reflect.Slice {
			operationValueMap[Remove] = current[currentKey].([]string)
		} else {
			defaultMap, err := GetDefault(ver)
			if err != nil {
				return false, err
			}

			defaultValue := getDefaultValue(defaultMap, currentKey)

			operationValueMap[Update] = defaultValue
		}

		d[currentKey] = operationValueMap
		diffUpdated = true
	}

	return diffUpdated, nil
}

func handleSliceFields(key string, desired Conf, d DynamicConfigMap, desiredToActual bool) {
	operationValueMap := make(map[Operation]interface{})

	if reflect.ValueOf(desired[key]).Kind() == reflect.Slice {
		if desiredToActual {
			operationValueMap[Add] = desired[key].([]string)
		} else {
			operationValueMap[Remove] = desired[key].([]string)
		}
	} else {
		operationValueMap[Update] = desired[key]
	}

	d[key] = operationValueMap
}

func handleValueDiff(key string, desiredValue, currentValue interface{}, d DynamicConfigMap) {
	operationValueMap := make(map[Operation]interface{})

	if reflect.ValueOf(desiredValue).Kind() == reflect.Slice {
		currentSet := sets.NewSet[string]()
		currentSet.Append(currentValue.([]string)...)

		desiredSet := sets.NewSet[string]()
		desiredSet.Append(desiredValue.([]string)...)

		removedValues := currentSet.Difference(desiredSet)
		if removedValues.Cardinality() > 0 {
			operationValueMap[Remove] = removedValues.ToSlice()
			d[key] = operationValueMap
		}

		addedValues := desiredSet.Difference(currentSet)
		if addedValues.Cardinality() > 0 {
			operationValueMap[Add] = addedValues.ToSlice()
			d[key] = operationValueMap
		}
	} else {
		operationValueMap[Update] = desiredValue
		d[key] = operationValueMap
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
	for key, desiredValue := range desired {
		bN := BaseKey(key)
		//if isNodeSpecificField(bN) || bN == keyIndex {
		if isNodeSpecificField(bN) {
			// Ignore node specific details and ordering
			continue
		}

		// Add if not found in current
		currentValue, ok := current[key]
		if !ok {
			diffUpdated := false
			if diffUpdated = handleMissingSection(log, key, desired, current, d, desiredToActual); !diffUpdated {
				var err error
				// Add default values to config parameter if available in schema.
				// If key is not present in current, then check if any key in desired which starts with key is present in current
				// eg. desired has security: {} current has security.log.report-sys-admin: true
				// final diff should be map[security.log.report-sys-admin] = <default value>
				diffUpdated, err = handlePartialMissingSection(key, ver, current, d)
				if err != nil {
					return nil, err
				}
			}

			if !diffUpdated {
				handleSliceFields(key, desired, d, desiredToActual)
			}

			continue
		}

		log.V(1).Info(
			"compare", "key",
			key, "desiredValue", desiredValue, "currentValue", currentValue,
		)

		if isValueDiff(log, desiredValue, currentValue) {
			handleValueDiff(key, desiredValue, currentValue, d)
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
