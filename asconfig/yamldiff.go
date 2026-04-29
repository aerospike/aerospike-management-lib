package asconfig

import (
	"fmt"
	"reflect"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"

	"github.com/aerospike/aerospike-management-lib/info"
)

// ─── public types ─────────────────────────────────────────────────────────────

// ConfigDiff is the result of YAMLDiff. It separates changes into those that
// can be applied to a live cluster (Dynamic) and those that require a rolling
// restart (Static). Both maps use the bracket-free flat key format:
//
//	"namespaces.test.replication-factor"   (not "namespaces.{test}.…")
//	"xdr.dcs.dc1.node-address-ports"       (not "xdr.dcs.{dc1}.…")
//
// Both maps are compatible with CreateSetConfigCmdsFromDiff. Pass
// ConfigDiff.Dynamic for live set-config commands.
type ConfigDiff struct {
	// Changes that can be applied live via set-config / log-set commands.
	Dynamic DynamicConfigMap
	// Changes that require a rolling restart.
	Static DynamicConfigMap
}

// HasChanges returns true if there are any differences between the two configs.
func (d *ConfigDiff) HasChanges() bool {
	return len(d.Dynamic) > 0 || len(d.Static) > 0
}

// HasStaticChanges returns true if a rolling restart is required.
func (d *ConfigDiff) HasStaticChanges() bool {
	return len(d.Static) > 0
}

// HasDynamicChanges returns true if changes can be applied live.
func (d *ConfigDiff) HasDynamicChanges() bool {
	return len(d.Dynamic) > 0
}

// All returns every change (dynamic + static) in a single DynamicConfigMap.
func (d *ConfigDiff) All() DynamicConfigMap {
	all := make(DynamicConfigMap, len(d.Dynamic)+len(d.Static))
	for k, v := range d.Dynamic {
		all[k] = v
	}
	for k, v := range d.Static {
		all[k] = v
	}
	return all
}

// ─── YAMLDiff ─────────────────────────────────────────────────────────────────

// YAMLDiff computes the diff between two Aerospike configs expressed as YAML.
//
// Named sections (namespaces, dcs, sets, tls, logging sinks) may be expressed
// in either form:
//
//	List form (AKO):  namespaces: [{name: test, replication-factor: 2}]
//	Map form:         namespaces: {test: {replication-factor: 2}}
//
// Both forms are normalised before diffing; callers need not know which is in use.
//
// ver is the Aerospike server version used to classify changes as dynamic or
// static and to resolve default values for keys removed from the desired config.
func YAMLDiff(
	log logr.Logger, desiredYAML, currentYAML []byte, ver string,
) (*ConfigDiff, error) {
	desiredMap, err := parseYAML(desiredYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse desired YAML: %w", err)
	}

	currentMap, err := parseYAML(currentYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current YAML: %w", err)
	}

	rawDiff, err := diffConfigMaps(log, "", desiredMap, currentMap, ver)
	if err != nil {
		return nil, fmt.Errorf("failed to compute config diff: %w", err)
	}

	dynSet, err := GetDynamic(ver)
	if err != nil {
		log.V(1).Info("could not load dynamic schema, all changes marked static", "err", err)
		dynSet = sets.NewSet[string]()
	}

	result := &ConfigDiff{
		Dynamic: make(DynamicConfigMap),
		Static:  make(DynamicConfigMap),
	}

	for key, opMap := range rawDiff {
		if diffKeyIsDynamic(dynSet, key, opMap) {
			result.Dynamic[key] = opMap
		} else {
			result.Static[key] = opMap
		}
	}

	return result, nil
}

// ─── YAML parsing ─────────────────────────────────────────────────────────────

// parseYAML unmarshals YAML bytes into a raw map. Named sections may remain in
// either list form or map form; diffConfigMaps handles both inline.
func parseYAML(src []byte) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// ─── recursive diff ───────────────────────────────────────────────────────────

// diffConfigMaps recursively walks two config maps and emits a flat
// DynamicConfigMap using clean keys (no {} brackets).
//
// prefix is the dot-separated key path accumulated by callers ("" at the top
// level, "namespaces.test" when recursing into a namespace).
func diffConfigMaps(
	log logr.Logger, prefix string,
	desired, current map[string]interface{},
	ver string,
) (DynamicConfigMap, error) {
	result := make(DynamicConfigMap)

	// ── keys present in desired ───────────────────────────────────────────
	for k, dv := range desired {
		if isNodeSpecificField(k) {
			continue
		}

		key := joinDiffPath(prefix, k)
		cv := current[k]

		// Nested map — recurse.
		// When the key is entirely absent from current, cv is nil and cMap
		// becomes {}, so every field inside produces an Add/Update op.
		// That is the implicit signal that the whole entry is new.
		if dMap, ok := dv.(map[string]interface{}); ok {
			cMap, _ := cv.(map[string]interface{})
			if cMap == nil {
				cMap = map[string]interface{}{}
			}

			sub, err := diffConfigMaps(log, key, dMap, cMap, ver)
			if err != nil {
				return nil, err
			}

			mergeDiff(result, sub)

			continue
		}

		// String-slice field — compute per-element Add/Remove delta.
		if dSlice, ok := toStringSlice(dv); ok {
			cSlice, _ := toStringSlice(cv)
			if opMap := diffSlice(dSlice, cSlice); opMap != nil {
				result[key] = opMap
			}

			continue
		}

		// Scalar — emit Update if the values differ.
		if _, exists := current[k]; !exists || diffScalar(dv, cv) {
			result[key] = map[OpType]interface{}{Update: dv}
		}
	}

	// ── keys present in current but absent from desired ───────────────────
	// Removed scalar keys are reset to their schema default value.
	// Removed map keys recurse with empty desired so each sub-scalar is reset.
	// Removed slice fields are emitted as Remove of the whole slice.
	defaultMap, err := GetDefault(ver)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema defaults: %w", err)
	}

	for k, cv := range current {
		if _, inDesired := desired[k]; inDesired {
			continue
		}

		if isNodeSpecificField(k) {
			continue
		}

		key := joinDiffPath(prefix, k)

		if cMap, ok := cv.(map[string]interface{}); ok {
			sub, err := diffConfigMaps(log, key, map[string]interface{}{}, cMap, ver)
			if err != nil {
				return nil, err
			}

			mergeDiff(result, sub)

			continue
		}

		if cSlice, ok := toStringSlice(cv); ok {
			result[key] = map[OpType]interface{}{Remove: cSlice}
			continue
		}

		// Scalar removed — reset to schema default.
		result[key] = map[OpType]interface{}{Update: defaultMap[schemaKeyFor(key)]}
	}

	return result, nil
}

// ─── dynamic classification ───────────────────────────────────────────────────

// diffKeyIsDynamic reports whether a change at the given clean flat key can be
// applied to a live cluster without a rolling restart.
//
// This mirrors the logic of IsDynamicConfig but works on clean keys (no {})
// using schemaKeyFor for schema lookups.
func diffKeyIsDynamic(
	dynSet sets.Set[string],
	key string,
	opMap map[OpType]interface{},
) bool {
	tokens := strings.Split(key, sep)
	baseKey := tokens[len(tokens)-1]
	context := tokens[0]

	if baseKey == "replication-factor" || baseKey == keyNodeAddressPorts {
		return true
	}

	// XDR DCs and their namespaces can be added/removed dynamically.
	if context == info.ConfigXDRContext && baseKey == KeyName {
		return true
	}

	// rack-id changes always require a restart.
	if baseKey == "rack-id" {
		return false
	}

	// Removing entries from these slice fields is not dynamically supported.
	conditionalStatic := sets.NewSet("ignore-bins", "ignore-sets", "ship-bins", "ship-sets")
	if conditionalStatic.Contains(baseKey) {
		if _, ok := opMap[Remove]; ok {
			return false
		}
	}

	// All logging severity changes are dynamic (schema $ref chains make the
	// dynamic flag unreachable via the flat schema walk).
	if context == info.ConfigLoggingContext && baseKey != KeyName {
		return true
	}

	return dynSet.Contains(schemaKeyFor(key))
}

// schemaKeyFor converts a clean flat key to the schema lookup key used by the
// dynamic/default maps, replacing instance-name tokens with "_".
//
// Instance names are identified positionally: the token immediately following a
// list-section keyword (namespaces, dcs, sets, tls, logging) is an instance
// name and is replaced with "_".
//
// Examples:
//
//	"namespaces.test.replication-factor"      → "namespaces._.replication-factor"
//	"xdr.dcs.dc1.namespaces.ns1.bin-policy"   → "xdr.dcs._.namespaces._.bin-policy"
//	"logging.console.any"                     → "logging._.any"
//	"service.proto-fd-max"                    → "service.proto-fd-max"
func schemaKeyFor(key string) string {
	tokens := strings.Split(key, sep)
	result := make([]string, 0, len(tokens))
	nextIsInstance := false

	for _, token := range tokens {
		if nextIsInstance {
			result = append(result, "_")
			// After consuming the instance name, check if this token itself
			// is a list section (it won't be, but be defensive).
			nextIsInstance = isListSection(token) || isSpecialListSection(token)
			continue
		}

		result = append(result, token)
		nextIsInstance = isListSection(token) || isSpecialListSection(token)
	}

	return strings.Join(result, sep)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// joinDiffPath joins path segments with sep, skipping empty parts.
func joinDiffPath(parts ...string) string {
	var b strings.Builder

	for _, p := range parts {
		if p == "" {
			continue
		}

		if b.Len() > 0 {
			b.WriteByte('.')
		}

		b.WriteString(p)
	}

	return b.String()
}

// mergeDiff copies all entries from src into dst.
func mergeDiff(dst, src DynamicConfigMap) {
	for k, v := range src {
		dst[k] = v
	}
}

// toStringSlice converts []string or []interface{}{string…} to []string.
// Returns nil, false if the value is not a string slice.
func toStringSlice(v interface{}) ([]string, bool) {
	switch v := v.(type) {
	case []string:
		return v, true

	case []interface{}:
		result := make([]string, 0, len(v))

		for _, elem := range v {
			s, ok := elem.(string)
			if !ok {
				return nil, false
			}

			result = append(result, s)
		}

		return result, true
	}

	return nil, false
}

// diffSlice computes the per-element Add/Remove delta between two string slices.
// Returns nil when there is no change.
func diffSlice(desired, current []string) map[OpType]interface{} {
	dSet := sets.NewSet(desired...)
	cSet := sets.NewSet(current...)

	opMap := make(map[OpType]interface{})

	if added := dSet.Difference(cSet); added.Cardinality() > 0 {
		opMap[Add] = added.ToSlice()
	}

	if removed := cSet.Difference(dSet); removed.Cardinality() > 0 {
		opMap[Remove] = removed.ToSlice()
	}

	if len(opMap) == 0 {
		return nil
	}

	return opMap
}

// diffScalar returns true if two scalar config values differ.
// Empty string and "null" are treated as equivalent (matching the existing
// isEmptyField convention used elsewhere in the library).
func diffScalar(a, b interface{}) bool {
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			if isEmptyField("", sa) && isEmptyField("", sb) {
				return false
			}

			return sa != sb
		}
	}

	return !reflect.DeepEqual(a, b)
}
