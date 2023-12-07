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

const (
	expandConfKey     = "expanded_config"
	flatConfKey       = "flat_config"
	flatSchemaKey     = "flat_schema"
	normFlatSchemaKey = "normalized_flat_schema"
	metadataKey       = "metadata"
	buildKey          = "asd_build"
)

// namespaceRe is a regular expression used to match and extract namespace configurations from the config file.
var namespaceRe = regexp.MustCompile(fmt.Sprintf(`(^namespaces\%s)(.+?)(\%s.+)`, sep, sep))
var indexedRe = regexp.MustCompile(`(.+)\[(\d+)\](.*)`)
var namedRe = regexp.MustCompile("({.+?})")
var securityRe = regexp.MustCompile(fmt.Sprintf(`^security\%s+`, sep))

// ConfGetter is an interface that defines methods for retrieving configurations.
type ConfGetter interface {
	AllConfigs() (Conf, error)
	GetAsInfo(cmdList ...string) (Conf, error)
}

type GenConf struct {
	Conf    Conf
	Version string
}

func newGenConf(conf Conf, version string) *GenConf {
	return &GenConf{
		Conf:    conf,
		Version: version,
	}
}

// GenerateConf generates the config based on the provided log and ConfGetter.
// If removeDefaults is true, it will remove default values from the config.
// Without removeDefaults, the config that is generate will not be valid. Many
// default values are out of the acceptable range required by the server.
func GenerateConf(log logr.Logger, confGetter ConfGetter, removeDefaults bool) (*GenConf, error) {
	log.V(1).Info("Generating config")

	validConfig := Conf{}

	// Flatten the config returned from the server. Then convert it to a map
	// that is valid according to the schema.
	p := newPipeline(log, []pipelineStep{
		newGetConfigStep(log, confGetter),
		newServerVersionCheckStep(log, isSupportedGenerateVersion),
		newGetFlatSchemaStep(log),
		newRenameLoggingContextsStep(log),
		newFlattenConfStep(log),
		newCopyEffectiveRackIDStep(log),
		newRemoveSecurityIfDisabledStep(log),
		newTransformKeyValuesStep(log),
	})

	if removeDefaults {
		p.addStep(newRemoveDefaultsStep(log))
	}

	p.addStep(newExpandConfStep(log))

	err := p.execute(validConfig)
	if err != nil {
		log.Error(err, "Error generating config")
		return nil, err
	}

	return newGenConf(validConfig[expandConfKey].(Conf), validConfig[metadataKey].(Conf)[buildKey].(string)), nil
}

// isSupportedGenerateVersion checks if the provided version is supported for generating the config.
func isSupportedGenerateVersion(version string) (bool, error) {
	s, err := IsSupportedVersion(version)

	if err != nil {
		return false, err
	}

	if !s {
		return false, nil
	}

	cmp, err := lib.CompareVersions(version, "5.0.0")

	return cmp >= 0, err
}

// pipelineStep is an interface that defines a step in the pipeline for generating the config.
type pipelineStep interface {
	execute(conf Conf) error
}

// pipeline represents a pipeline for generating the config.
type pipeline struct {
	log   logr.Logger
	steps []pipelineStep
}

// newPipeline creates a new pipeline with the provided log and steps.
func newPipeline(log logr.Logger, steps []pipelineStep) *pipeline {
	return &pipeline{
		log:   log,
		steps: steps,
	}
}

func (p *pipeline) addStep(step pipelineStep) {
	p.steps = append(p.steps, step)
}

// execute executes the pipeline steps on the provided config.
func (p *pipeline) execute(conf Conf) error {
	for _, step := range p.steps {
		err := step.execute(conf)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFlatSchema
type GetFlatSchemaStep struct {
	log logr.Logger
}

func newGetFlatSchemaStep(log logr.Logger) *GetFlatSchemaStep {
	return &GetFlatSchemaStep{
		log: log,
	}
}

func (s *GetFlatSchemaStep) execute(conf Conf) error {
	s.log.V(1).Info("Getting flat schema")

	build := conf[metadataKey].(Conf)[buildKey].(string)
	flatSchema, err := getFlatSchema(build)

	if err != nil {
		s.log.V(-1).Error(err, "Error getting flat schema")
		return err
	}

	conf[flatSchemaKey] = flatSchema
	conf[normFlatSchemaKey] = normalizeFlatSchema(flatSchema)

	return nil
}

// GetConfigStep is a pipeline step that retrieves the configs and metadata.
type GetConfigStep struct {
	confGetter ConfGetter
	log        logr.Logger
}

// newGetConfigStep creates a new GetConfigStep with the provided log and ConfGetter.
func newGetConfigStep(log logr.Logger, confGetter ConfGetter) *GetConfigStep {
	return &GetConfigStep{
		confGetter: confGetter,
		log:        log,
	}
}

// execute retrieves the configs and metadata using the ConfGetter.
func (s *GetConfigStep) execute(conf Conf) error {
	s.log.V(1).Info("Getting configs and metadata")

	configs, err := s.confGetter.AllConfigs()
	if err != nil {
		s.log.V(-1).Error(err, "Error getting configs from node")
		return err
	}

	conf["config"] = configs

	if _, ok := configs["racks"]; ok {
		// We don't need the racks config. flattenConf logs an error when it sees this.
		conf["racks"] = configs["racks"]
		delete(configs, "racks")
	}

	metadata, err := s.confGetter.GetAsInfo(metadataKey)
	if err != nil {
		s.log.V(-1).Error(err, "Error getting metadata from node")
		return err
	}

	conf[metadataKey] = metadata[metadataKey]

	return nil
}

// ServerVersionCheckStep is a pipeline step that checks if the server version is supported.
type ServerVersionCheckStep struct {
	checkFunc func(string) (bool, error)
	log       logr.Logger
}

// newServerVersionCheckStep creates a new ServerVersionCheckStep with the provided log and check function.
func newServerVersionCheckStep(log logr.Logger, checkFunc func(string) (bool, error)) *ServerVersionCheckStep {
	return &ServerVersionCheckStep{
		checkFunc: checkFunc,
		log:       log,
	}
}

// execute checks if the server version is supported using the check function.
func (s *ServerVersionCheckStep) execute(conf Conf) error {
	s.log.V(1).Info("Checking server version")

	build := conf[metadataKey].(Conf)[buildKey].(string)
	isSupported, err := s.checkFunc(build)

	if err != nil {
		s.log.V(-1).Error(err, "Error checking for supported server version")
		return err
	}

	if !isSupported {
		s.log.V(-1).Info("Unsupported server version: %s", build)
		return fmt.Errorf("unsupported version: %s", build)
	}

	return nil
}

// copyEffectiveRackIDStep is a pipeline step that copies the effective rack-id to rack-id.
type copyEffectiveRackIDStep struct {
	log logr.Logger
}

// newCopyEffectiveRackIDStep creates a new copyEffectiveRackIDStep with the provided log.
func newCopyEffectiveRackIDStep(log logr.Logger) *copyEffectiveRackIDStep {
	return &copyEffectiveRackIDStep{
		log: log,
	}
}

// rackRegex is a regular expression used to match and extract rack IDs.
var rackRegex = regexp.MustCompile(`rack_(\d+)`)

// execute copies the effective rack-id to rack-id in the config.
func (s *copyEffectiveRackIDStep) execute(conf Conf) error {
	s.log.V(1).Info("Copying effective rack-id to rack-id")

	if _, ok := conf["racks"]; !ok {
		s.log.V(-1).Info("No racks config found")
		return nil
	}

	flatConfig := conf[flatConfKey].(Conf)
	effectiveRacks := conf["racks"].([]Conf)
	nodeID := conf[metadataKey].(Conf)["node_id"].(string)

	for _, rackInfo := range effectiveRacks {
		ns := rackInfo["ns"].(string)

		// For this ns find which rack this node belongs to
		for rack, nodesStr := range rackInfo {
			if !strings.Contains(nodesStr.(string), nodeID) {
				continue
			}

			rackIDStr := rackRegex.FindStringSubmatch(rack)[1]
			if rackIDStr == "" {
				err := fmt.Errorf("unable to find rack id for rack %s", rack)
				s.log.V(-1).Error(err, "Error copying effective rack-id to rack-id")
				return err
			}

			rackID, err := strconv.ParseInt(rackIDStr, 10, 64) // Matches what is found in info/as_parser.go

			if err != nil {
				err := fmt.Errorf("unable to convert rack id %s to int", rackIDStr)
				s.log.V(-1).Error(err, "Error copying effective rack-id to rack-id")
				return err
			}

			// Copy effective rack-id over the ns config
			key := fmt.Sprintf("namespaces.{%s}.rack-id", ns)
			flatConfig[key] = rackID

			break
		}
	}

	return nil
}

// renameKeysStep is a pipeline step that renames logging contexts in the config.
type renameKeysStep struct {
	log logr.Logger
}

// newRenameLoggingContextsStep creates a new renameKeysStep with the provided log.
func newRenameLoggingContextsStep(log logr.Logger) *renameKeysStep {
	return &renameKeysStep{
		log: log,
	}
}

// execute renames logging contexts in the config.
func (s *renameKeysStep) execute(conf Conf) error {
	s.log.V(1).Info("Renaming keys")

	config := conf["config"].(Conf)
	logging, ok := config["logging"].(Conf)

	if !ok {
		s.log.V(-1).Info("No logging config found")
		return nil
	}

	newLoggingEntries := Conf{}

	for key, value := range logging {
		switch v := value.(type) {
		case Conf:
			if key == "stderr" {
				newLoggingEntries["console"] = value

				delete(logging, key)
			} else if !strings.HasSuffix(key, ".log") {
				newLoggingEntries["syslog"] = v
				syslog := newLoggingEntries["syslog"].(Conf)
				syslog["path"] = key

				delete(logging, key)
			}
		default:
			continue
		}
	}

	for key, value := range newLoggingEntries {
		logging[key] = value
	}

	return nil
}

// flattenConfStep is a pipeline step that flattens the config.
type flattenConfStep struct {
	log logr.Logger
}

// newFlattenConfStep creates a new flattenConfStep with the provided log.
func newFlattenConfStep(log logr.Logger) *flattenConfStep {
	return &flattenConfStep{
		log: log,
	}
}

func sortKeys(config lib.Stats) []string {
	keys := make([]string, len(config))
	idx := 0

	for key := range config {
		keys[idx] = key
		idx++
	}

	sort.Strings(keys)

	return keys
}

// convertDictToList converts a dictionary configuration to a list configuration.
func convertDictToList(config Conf) []Conf {
	list := make([]Conf, len(config))
	count := 0
	keys1 := sortKeys(config)

	for _, key1 := range keys1 {
		config2, ok := config[key1].(Conf)

		if !ok || config2 == nil {
			continue
		}

		config2["name"] = key1
		list[count] = config2
		count++

		keys2 := sortKeys(config2)
		for _, key2 := range keys2 {
			value := config2[key2]

			switch v := value.(type) {
			case Conf:
				config2[key2] = convertDictToList(v)
			default:
				continue
			}
		}
	}

	return list
}

// convertDictRespToConf converts a dictionary response to a Conf.
func convertDictRespToConf(config Conf) {
	if _, ok := config["logging"].(Conf); ok {
		config["logging"] = convertDictToList(config["logging"].(Conf))
	}

	if _, ok := config["namespaces"].(Conf); ok {
		config["namespaces"] = convertDictToList(config["namespaces"].(Conf))
	}

	if xdr, ok := config["xdr"].(Conf); ok {
		for key, value := range xdr {
			switch v := value.(type) {
			case Conf:
				xdr[key] = convertDictToList(v)
			default:
				continue
			}
		}
	}
}

// execute flattens the config.
func (s *flattenConfStep) execute(conf Conf) error {
	s.log.V(1).Info("Flattening config")

	origConfig := conf["config"].(Conf)

	convertDictRespToConf(origConfig)

	conf[flatConfKey] = flattenConf(s.log, conf["config"].(Conf), sep)

	return nil
}

// transformKeyValuesStep is a pipeline step that transforms key values in the config.
type transformKeyValuesStep struct {
	log logr.Logger
}

// newTransformKeyValuesStep creates a new transformKeyValuesStep with the provided log.
func newTransformKeyValuesStep(log logr.Logger) *transformKeyValuesStep {
	return &transformKeyValuesStep{
		log: log,
	}
}

func splitContextBaseKey(key string) (contextKey, bKey string) {
	bKey = baseKey(key)
	contextKey = strings.TrimSuffix(key, bKey)
	contextKey = strings.TrimSuffix(contextKey, sep)

	return contextKey, bKey
}

func getPluralKey(key string) string {
	contextKey, bKey := splitContextBaseKey(key)
	return contextKey + sep + PluralOf(bKey)
}

var serverRespFieldToConfField = map[string]string{
	"shipped-bins": "ship-bins",
	"shipped-sets": "ship-sets",
	"ignored-bins": "ignore-bins",
	"ignored-sets": "ignore-sets",
}

func toConfField(key string) string {
	if val, ok := serverRespFieldToConfField[key]; ok {
		return val
	}

	return key
}

func renameServerResponseKey(key string) string {
	contextKey, bKey := splitContextBaseKey(key)
	bKey = toConfField(bKey)

	if contextKey == "" {
		return bKey
	}

	return contextKey + sep + toConfField(bKey)
}

func disallowedInConfigWhenSC() []string {
	return []string{
		"read-consistency-level-override", "write-commit-level-override",
	}
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys(m Conf) []string {
	keys := make([]string, len(Conf{}))

	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// undefinedOrNull checks if a value is undefined or null.
func undefinedOrNull(val interface{}) bool {
	if str, ok := val.(string); ok {
		lower := strings.ToLower(str)
		return lower == "undefined" || lower == "null"
	}

	return false
}

// convertIndexedToList converts an indexed key to a list key. It returns the
// new key, the index, and the value as a string. If the key is not indexed or the value is
// not a string, it returns empty strings.
func convertIndexedToList(key string, value interface{}) (newKey, strVal string) {
	if newKey, _, _ = parseIndexField(key); newKey != "" {
		if str, ok := value.(string); ok {
			strVal = str
			return newKey, strVal
		}
	}

	return newKey, strVal
}

func parseIndexField(key string) (newKey string, index int, err error) {
	if match := indexedRe.FindStringSubmatch(key); match != nil {
		index, err = strconv.Atoi(match[2])

		if err != nil {
			return "", 0, err
		}

		newKey = match[1]

		return newKey, index, nil
	}

	return newKey, index, nil
}

// execute transforms key values in the config.
func (s *transformKeyValuesStep) execute(conf Conf) error {
	s.log.V(1).Info("Transforming key values")

	origFlatConf := conf[flatConfKey].(Conf)
	newFlatConf := make(Conf, len(origFlatConf)) // we will overwrite flat_config
	sortedKeys := sortedKeys(origFlatConf)
	scNamspaces := []string{}

	for _, key := range sortedKeys {
		// We will mutate the servers key, value response to match the schema
		value := origFlatConf[key]
		key = renameServerResponseKey(key)

		if nsMatch := namespaceRe.FindStringSubmatch(key); nsMatch != nil {
			if nsMatch[3] == sep+"strong-consistency" && value.(bool) {
				ns := nsMatch[2]
				scNamspaces = append(scNamspaces, ns)
			}
		}

		if undefinedOrNull(value) {
			value = ""
		}

		if isTypedSection(key) {
			key = key + sep + keyType
		}

		if newKey, str := convertIndexedToList(key, value); newKey != "" {
			newKey = getPluralKey(newKey)

			if strings.HasSuffix(key, "shadow") {
				_, index, err := parseIndexField(key)
				if err != nil {
					s.log.V(-1).Error(err, "Error parsing index field for shadow device")
					return err
				}

				// This should not happen because we sorted the keys
				if val, ok := newFlatConf[newKey].([]string); !ok || len(val) <= index {
					err := fmt.Errorf("shadow key %s does not have a corresponding device yet", key)
					s.log.V(-1).Error(err, "Error converting shadow device to list")
					return err
				}

				sliceVal := newFlatConf[newKey].([]string)
				sliceVal[index] = sliceVal[index] + " " + str
				value = sliceVal
			} else {
				if _, ok := newFlatConf[newKey]; ok {
					if _, ok := newFlatConf[newKey].([]string); ok {
						// Indexes should come in order because we sorted the
						// keys
						value = append(newFlatConf[newKey].([]string), str)
					}
				} else {
					value = []string{str}
				}
			}

			key = newKey
		}

		if ok, _ := isListField(key); ok {
			key = getPluralKey(key)

			if strVal, ok := value.(string); ok {
				if strVal == "" {
					value = []string{}
				} else {
					value = strings.Split(strVal, ",")
				}
			}
		}

		nFlatSchema := conf[normFlatSchemaKey].(map[string]interface{})
		normalizedKey := namedRe.ReplaceAllString(key, "_")

		if _, ok := nFlatSchema[normalizedKey+sep+"default"]; !ok && !isInternalField(normalizedKey) {
			// Value is not found in schemas. Must be invalid config
			// parameter which the server returns or our own internal
			// (<index>) key.
			continue
		}

		newFlatConf[key] = value
	}

	for _, ns := range scNamspaces {
		for _, disallowed := range disallowedInConfigWhenSC() {
			key := fmt.Sprintf("namespaces%s%s%s%s", sep, ns, sep, disallowed)
			delete(newFlatConf, key)
		}
	}

	conf[flatConfKey] = newFlatConf

	return nil
}

// removeSecurityIfDisabledStep is a pipeline step that removes security configurations if security is disabled.
type removeSecurityIfDisabledStep struct {
	log logr.Logger
}

// newRemoveSecurityIfDisabledStep creates a new removeSecurityIfDisabledStep with the provided log.
func newRemoveSecurityIfDisabledStep(log logr.Logger) *removeSecurityIfDisabledStep {
	return &removeSecurityIfDisabledStep{
		log: log,
	}
}

// execute removes security configurations if security is disabled.
func (s *removeSecurityIfDisabledStep) execute(conf Conf) error {
	s.log.V(1).Info("Removing security configs if security is disabled")

	flatConf := conf[flatConfKey].(Conf)
	build := conf[metadataKey].(Conf)[buildKey].(string)

	if val, ok := flatConf["security.enable-security"]; ok {
		securityEnabled, ok := val.(bool)

		if !ok {
			err := fmt.Errorf("enable-security is not a boolean")
			s.log.V(-1).Error(err, "Error removing security configs")
			return err
		}

		cmp, err := lib.CompareVersions(build, "5.7.0")
		if err != nil {
			s.log.V(-1).Error(err, "Error removing security configs")
			return err
		}

		if securityEnabled {
			if cmp >= 0 {
				delete(flatConf, "security.enable-security")
			}
		} else {
			// 5.7 and newer can't have any security configs. An empty security
			// context will enable-security.
			if cmp >= 0 {
				for key := range flatConf {
					if securityRe.MatchString(key) {
						delete(flatConf, key)
					}
				}
			}
		}
	}

	return nil
}

type removeDefaultsStep struct {
	log logr.Logger
}

func newRemoveDefaultsStep(log logr.Logger) *removeDefaultsStep {
	return &removeDefaultsStep{
		log: log,
	}
}

func compareDefaults(log logr.Logger, defVal, confVal interface{}) bool {
	switch val := defVal.(type) {
	case []interface{}:
		return reflect.DeepEqual(val, confVal)
	case []string:
		return reflect.DeepEqual(val, confVal)
	case string:
		// The schema sometimes has " " and ""
		if val == " " {
			defVal = ""
		}

		// Sometimes what is a slice value in the schema might be
		// allowed to be a string in the config. Also,
		// service.tls-authenticate-client is "oneOf" slice or string where only
		// the string has a "default".
		if sliceVal, ok := confVal.([]string); ok && len(sliceVal) == 1 {
			return sliceVal[0] == val
		}

		return defVal == confVal
	case uint64:
		// Schema deals with uint64 when positive but config deals with int
		switch confVal := confVal.(type) {
		case int64:
			if confVal < 0 {
				return false
			}

			return val == uint64(confVal)
		case uint64:
			return val == confVal
		}
	case int64:
		// Schema deals with int64 when negative but config deals with int
		switch confVal := confVal.(type) {
		case uint64:
			if val < 0 {
				return false
			}

			return val == int64(confVal)
		case int64:
			return val == confVal
		default:
			log.V(-1).Info("Unexpected type when comparing default (%s) to config value (%s)", val, confVal)
		}
	default:
		return val == confVal
	}

	return false
}

func defaultSlice(m map[string][]string, key string) []string {
	if val, ok := m[key]; ok {
		return val
	}

	return []string{}
}

func (s *removeDefaultsStep) execute(conf Conf) error {
	s.log.V(1).Info("Removing default values")

	flatConf := conf[flatConfKey].(Conf)
	flatSchema := conf[flatSchemaKey].(map[string]interface{})
	nFlatSchema := conf[normFlatSchemaKey].(map[string]interface{})
	defaults := getDefaultSchema(flatSchema)

	// "logging.<file>" -> "log-level" -> list of contexts with that level
	// We will use this to find the most common log level in order to replace
	// with "any".
	loggingMap := make(map[string]map[string][]string)
	securityFound := false

	for key, value := range flatConf {
		if strings.HasPrefix(key, "security"+sep) {
			// We expect there to be no security keys if security is disabled in
			// 5.7 or newer
			securityFound = true
		}

		// Handle logging differently
		if strings.HasPrefix(key, "logging"+sep) {
			contextKey, _ := splitContextBaseKey(key)

			if loggingMap[contextKey] == nil {
				loggingMap[contextKey] = make(map[string][]string)
			}

			if strVal, ok := value.(string); ok {
				strVal = strings.ToUpper(strVal)
				switch strVal {
				case "CRITICAL":
					loggingMap[contextKey]["CRITICAL"] = append(defaultSlice(loggingMap[contextKey], "CRITICAL"), key)
				case "WARNING":
					loggingMap[contextKey]["WARNING"] = append(defaultSlice(loggingMap[contextKey], "WARNING"), key)
				case "INFO":
					loggingMap[contextKey]["INFO"] = append(defaultSlice(loggingMap[contextKey], "INFO"), key)
				case "DEBUG":
					loggingMap[contextKey]["DEBUG"] = append(defaultSlice(loggingMap[contextKey], "DEBUG"), key)
				case "DETAIL":
					loggingMap[contextKey]["DETAIL"] = append(defaultSlice(loggingMap[contextKey], "DETAIL"), key)
				default:
					continue
				}
			}

			continue
		}

		normalizedKey := namedRe.ReplaceAllString(key, "_")

		if defVal, ok := defaults[normalizedKey]; ok {
			if compareDefaults(s.log, defVal, value) {
				delete(flatConf, key)
			}
		} else {
			if _, ok := nFlatSchema[normalizedKey+sep+"type"]; !ok && !isInternalField(normalizedKey) {
				// Value is not found in schemas. Must be invalid config
				// parameter which the server returns or our own internal
				// (<index>) key.
				delete(flatConf, key)
			}
		}
	}

	// Replace most common log level with a single "any" context
	for loggingContext, levels := range loggingMap {
		freq := "CRITICAL"
		count := 0

		for level, contexts := range levels {
			if len(contexts) > count {
				freq = level
				count = len(contexts)
			}
		}

		for _, context := range levels[freq] {
			delete(flatConf, context)
		}

		flatConf[loggingContext+sep+"any"] = freq
	}

	if securityFound {
		build := conf[metadataKey].(Conf)[buildKey].(string)
		cmp, err := lib.CompareVersions(build, "5.7.0")

		if err != nil {
			s.log.V(-1).Error(err, "Error removing default values")
			return err
		}

		if cmp >= 0 {
			// Security is enabled and we are 5.7 or newer. This ensures there
			// is at least an empry security context.
			flatConf["security"] = make(Conf)
		}
	}

	return nil
}

// expandConfStep is a pipeline step that expands the config.
type expandConfStep struct {
	log logr.Logger
}

// newExpandConfStep creates a new expandConfStep with the provided log.
func newExpandConfStep(log logr.Logger) *expandConfStep {
	return &expandConfStep{
		log: log,
	}
}

// execute expands the config.
func (s *expandConfStep) execute(conf Conf) error {
	s.log.V(1).Info("Expanding config")

	flatConf := conf[flatConfKey].(Conf)
	expandedConf := expandConf(s.log, &flatConf, sep)

	conf[expandConfKey] = expandedConf

	return nil
}
