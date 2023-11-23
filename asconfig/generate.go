package asconfig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/go-logr/logr"
)

var namespaceRegex = regexp.MustCompile(fmt.Sprintf(`(^namespaces\%s)(.+?)(\%s.+)`, sep, sep))
var setRegex = regexp.MustCompile(fmt.Sprintf(`(^namespaces\%s.+?\%ssets\%s)(.+?)(\%s.+)`, sep, sep, sep, sep))
var dcRegex = regexp.MustCompile(fmt.Sprintf(`(^xdr.dcs\%s)(.+?)(\%s.+)`, sep, sep))
var dcNamespaceRegex = regexp.MustCompile(fmt.Sprintf(`(^xdr.dcs\%s.+?\%snamespaces\%s)(.+?)(\%s.+)`, sep, sep, sep, sep))
var indexedRegex = regexp.MustCompile(`(.+)\[(\d+)\](.*)`)
var securityRegex = regexp.MustCompile(fmt.Sprintf(`^security\%s+`, sep))

type ConfGetter interface {
	AllConfigs() (Conf, error)
	GetAsInfo(cmdList ...string) (Conf, error)
}

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

func GenerateConf(log logr.Logger, confGetter ConfGetter) (Conf, error) {
	log.V(1).Info("Generating config")
	validConfig := Conf{}

	// Flatten the config returned from the server. Then convert it to a map
	// that valid according the the schema
	p := newPipeline(log, []pipelineStep{
		newGetConfigStep(log, confGetter),
		newServerVersionCheckStep(log, IsSupportedVersion),
		newFlattenConfStep(log),
		newCopyEffectiveRackIDStep(log),
		newConvertMapContextToListStep(log),
		newConvertUndefinedNullToEmptyStringStep(log),
		newConvertIndexedToListStep(log),
		newSplitListFieldStep(log),
		newRemoveSecurityIfDisabledStep(log),
		newAddTypeKeyToContextStep(log),
	})

	err := p.execute(validConfig)

	return validConfig, err
}

type pipelineStep interface {
	execute(conf Conf) error
}

type pipeline struct {
	log   logr.Logger
	steps []pipelineStep
}

func newPipeline(log logr.Logger, steps []pipelineStep) *pipeline {
	return &pipeline{
		log:   log,
		steps: steps,
	}
}

func (p *pipeline) execute(conf Conf) error {
	for _, step := range p.steps {
		err := step.execute(conf)
		if err != nil {
			return err
		}
	}

	return nil
}

type GetConfigStep struct {
	log        logr.Logger
	confGetter ConfGetter
}

func newGetConfigStep(log logr.Logger, confGetter ConfGetter) *GetConfigStep {
	return &GetConfigStep{
		log:        log,
		confGetter: confGetter,
	}
}

func (s *GetConfigStep) execute(conf Conf) error {
	s.log.V(1).Info("Getting configs and metadata")

	configs, err := s.confGetter.AllConfigs()
	if err != nil {
		return err
	}

	conf["config"] = configs["config"]

	metadata, err := s.confGetter.GetAsInfo("metadata")
	if err != nil {
		return err
	}

	conf["metadata"] = metadata["metadata"]
	return nil
}

type ServerVersionCheckStep struct {
	log       logr.Logger
	checkFunc func(string) (bool, error)
}

func newServerVersionCheckStep(log logr.Logger, checkFunc func(string) (bool, error)) *ServerVersionCheckStep {
	return &ServerVersionCheckStep{
		log:       log,
		checkFunc: checkFunc,
	}
}

func (s *ServerVersionCheckStep) execute(conf Conf) error {
	s.log.V(1).Info("Checking server version")
	build := conf["metadata"].(Conf)["build"].(string)
	is_supported, err := s.checkFunc(build)

	if err != nil {
		return fmt.Errorf("error checking for supported server version: %s", err)
	}

	if !is_supported {
		return fmt.Errorf("unsupported version: %s", build)
	}

	return nil
}

type copyEffectiveRackIDStep struct {
	log logr.Logger
}

func newCopyEffectiveRackIDStep(log logr.Logger) *copyEffectiveRackIDStep {
	return &copyEffectiveRackIDStep{
		log: log,
	}
}

var rackRegex = regexp.MustCompile(`rack_(\d+)`)

func (s *copyEffectiveRackIDStep) execute(conf Conf) error {
	s.log.V(1).Info("Copying effective rack-id to rack-id")

	config := conf["flat_config"].(Conf)
	effectiveRacks := conf["config"].(Conf)["racks"].([]Conf)
	nodeID := conf["metadata"].(Conf)["node_id"].(string)

	for _, rackInfo := range effectiveRacks {
		ns := rackInfo["ns"].(string)

		// For this ns find which rack this node belongs to
		for rack, nodesStr := range rackInfo {
			if strings.Contains(nodesStr.(string), nodeID) {
				rackIDStr := rackRegex.FindStringSubmatch(rack)[1]
				if rackIDStr == "" {
					return fmt.Errorf("unable to find rack id for rack %s", rack)
				}

				rackID, err := strconv.Atoi(rackIDStr)

				if err != nil {
					return fmt.Errorf("unable to convert rack id %s to int", rackIDStr)
				}

				// Copy effective rack-id over the ns config
				key := fmt.Sprintf("namespaces.%s.rack-id", ns)
				config[key] = rackID
				break
			}
		}
	}
	// TODO: Consider adding a step that deletes invalid contexts and parameters
	// delete(conf["config"].(Conf), "racks")

	return nil
}

func statsToMap(stats lib.Stats) map[string]interface{} {
	m := make(map[string]interface{})

	for key, value := range stats {
		switch v := value.(type) {
		case lib.Stats:
			m[key] = statsToMap(v)
		case []interface{}:
			listVal := make([]interface{}, len(v))
			for i, val := range v {
				switch val.(type) {
				case lib.Stats:
					listVal[i] = statsToMap(val.(lib.Stats))
				case string:
					listVal[i] = val.(string)
				case int:
					listVal[i] = val.(int)
				case float64:
					listVal[i] = val.(float64)
				case bool:
					listVal[i] = val.(bool)
				default:
					listVal[i] = val
				}
			}
			m[key] = listVal
		case string:
			m[key] = value.(string)
		case int:
			m[key] = value.(int)
		case float64:
			m[key] = value.(float64)
		case bool:
			m[key] = value.(bool)
		default:
			m[key] = value
		}

	}

	return m
}

type flattenConfStep struct {
	log logr.Logger
}

func newFlattenConfStep(log logr.Logger) *flattenConfStep {
	return &flattenConfStep{
		log: log,
	}
}

func convertConfigResponseToConf(config lib.Stats) {
	namespacesConfig := config["namespaces"].(lib.Stats)
	nsList := make([]lib.Stats, len(namespacesConfig))
	nsCount := 0

	for ns, c := range namespacesConfig {
		nsConfig := c.(lib.Stats)
		nsConfig["name"] = ns
		nsList[nsCount] = nsConfig
		nsCount++
		setConfig := nsConfig["sets"].(lib.Stats)
		setList := make([]lib.Stats, len(setConfig))
		setCount := 0

		for set, c := range setConfig {
			setConfig := c.(lib.Stats)
			setConfig["name"] = set
			setList[setCount] = setConfig
			setCount++
		}

		nsConfig["sets"] = setList
	}

	config["namespaces"] = nsList

}

func (s *flattenConfStep) execute(conf Conf) error {
	s.log.V(1).Info("Flattening config")
	origConfig := conf["config"].(lib.Stats)
	convertConfigResponseToConf(origConfig)

	// We want flattenConf to be able to flatten the response dict. To do that
	// we must format the response dict to look like an unmarshalled yaml
	conf["flat_config"] = flattenConf(s.log, conf["config"].(lib.Stats), sep)
	return nil
}

type convertMapContextToListStep struct {
	log logr.Logger
}

func newConvertMapContextToListStep(log logr.Logger) *convertMapContextToListStep {
	return &convertMapContextToListStep{
		log: log,
	}
}

func (s *convertMapContextToListStep) execute(conf Conf) error {
	s.log.V(1).Info("Converting map context to list")
	contexts := conf["flat_config"].(Conf)
	newEntries := Conf{}
	nsCount := map[string]int{}
	setCount := map[string]map[string]int{}
	dcCount := map[string]int{}
	dcNSCount := map[string]map[string]int{}

	for key, value := range contexts {
		if match := namespaceRegex.FindStringSubmatch(key); match != nil {
			if _, ok := nsCount[match[2]]; !ok {
				nsCount[match[2]] = len(nsCount)
				nameKey := fmt.Sprintf("%s%d.name", match[1], nsCount[match[2]])
				newEntries[nameKey] = match[2]
			}
			newNSKey := fmt.Sprintf("%s%d%s", match[1], nsCount[match[2]], match[3])
			delete(contexts, key)

			if match := setRegex.FindStringSubmatch(newNSKey); match != nil {
				if _, ok := setCount[match[1]]; !ok {
					setCount[match[1]] = map[string]int{}
				}
				if _, ok := setCount[match[1]][match[2]]; !ok {
					setCount[match[1]][match[2]] = len(setCount[match[1]])
					nameKey := fmt.Sprintf("%s%d.name", match[1], setCount[match[1]][match[2]])
					newEntries[nameKey] = match[2]
				}
				newSetKey := fmt.Sprintf("%s%d%s", match[1], setCount[match[1]][match[2]], match[3])
				newEntries[newSetKey] = value
			} else {
				newEntries[newNSKey] = value
			}
		} else if match := dcRegex.FindStringSubmatch(key); match != nil {
			if _, ok := dcCount[match[2]]; !ok {
				dcCount[match[2]] = len(dcCount)
				nameKey := fmt.Sprintf("%s%d.name", match[1], dcCount[match[2]])
				newEntries[nameKey] = match[2]
			}
			newDCKey := fmt.Sprintf("%s%d%s", match[1], dcCount[match[2]], match[3])
			delete(contexts, key)

			// TODO - create a function for replicated code. call it ... makeNewEntries
			if match := dcNamespaceRegex.FindStringSubmatch(newDCKey); match != nil {
				if _, ok := dcNSCount[match[1]]; !ok {
					dcNSCount[match[1]] = map[string]int{}
				}
				if _, ok := dcNSCount[match[1]][match[2]]; !ok {
					dcNSCount[match[1]][match[2]] = len(dcNSCount[match[1]])
					nameKey := fmt.Sprintf("%s%d.name", match[1], dcNSCount[match[1]][match[2]])
					newEntries[nameKey] = match[2]
				}
				newNSKey := fmt.Sprintf("%s%d%s", match[1], dcNSCount[match[1]][match[2]], match[3])
				newEntries[newNSKey] = value
			} else {
				newEntries[newDCKey] = value
			}
		}

	}

	for key, value := range newEntries {
		contexts[key] = value
	}

	return nil
}

type convertUndefinedNullToEmptyStringStep struct {
	log logr.Logger
}

func newConvertUndefinedNullToEmptyStringStep(log logr.Logger) *convertUndefinedNullToEmptyStringStep {
	return &convertUndefinedNullToEmptyStringStep{
		log: log,
	}
}

func (s *convertUndefinedNullToEmptyStringStep) execute(conf Conf) error {
	s.log.V(1).Info("Converting undefined and null values to empty string")
	contexts := conf["flat_config"].(Conf)

	for key, value := range contexts {
		if strVal, ok := value.(string); ok {
			lower := strings.ToLower(strVal)
			if lower == "undefined" || lower == "null" {
				contexts[key] = ""
			}
		}
	}

	return nil
}

type convertIndexedToListStep struct {
	log logr.Logger
}

func newConvertIndexedToListStep(log logr.Logger) *convertIndexedToListStep {
	return &convertIndexedToListStep{
		log: log,
	}
}

func (s *convertIndexedToListStep) execute(conf Conf) error {
	s.log.V(1).Info("Converting indexed values to list")
	contexts := conf["flat_config"].(Conf)
	newEntries := Conf{}

	for key, value := range contexts {
		if match := indexedRegex.FindStringSubmatch(key); match != nil {
			newKey := fmt.Sprintf("%s.%s%s", match[1], match[2], match[3])
			newEntries[newKey] = value
			delete(contexts, key)
		}
	}

	for key, value := range newEntries {
		contexts[key] = value
	}

	return nil
}

type splitCommaSepToListStep struct {
	log logr.Logger
}

func newSplitListFieldStep(log logr.Logger) *splitCommaSepToListStep {
	return &splitCommaSepToListStep{
		log: log,
	}
}

func (s *splitCommaSepToListStep) execute(conf Conf) error {
	s.log.V(1).Info("Splitting comma separated values")
	contexts := conf["flat_config"].(Conf)
	newEntries := Conf{}

	for key, value := range contexts {
		if ok, delim := isServerRespDelimitedListField(key); ok {
			if strVal, ok := value.(string); ok {
				splitStr := strings.Split(strVal, delim)

				for idx, str := range splitStr {
					if str == "" {
						continue
					}
					newKey := fmt.Sprintf("%s%s%d%s%s", key, sep, idx, sep, str)
					newEntries[newKey] = str
				}
				delete(contexts, key)

			}
		}
	}

	for key, value := range newEntries {
		contexts[key] = value
	}

	return nil
}

type removeSecurityIfDisabledStep struct {
	log logr.Logger
}

func newRemoveSecurityIfDisabledStep(log logr.Logger) *removeSecurityIfDisabledStep {
	return &removeSecurityIfDisabledStep{
		log: log,
	}
}

func (s *removeSecurityIfDisabledStep) execute(conf Conf) error {
	s.log.V(1).Info("Removing security configs if security is disabled")
	contexts := conf["flat_config"].(Conf)
	build := conf["metadata"].(Conf)["build"].(string)

	if val, ok := contexts["security.enable-security"]; ok {
		securityEnabled, ok := val.(bool)

		if !ok {
			return fmt.Errorf("enable-security is not a boolean")
		}

		if !securityEnabled {
			for key := range contexts {
				if securityRegex.MatchString(key) {
					delete(contexts, key)
				}
			}
		} else {
			if cmp, err := lib.CompareVersions(build, "5.7.0"); err != nil {
				return err
			} else if cmp >= 0 {
				delete(contexts, "security.enable-security")
			}
		}
	}

	return nil
}

type addTypeKeyToContextStep struct {
	log logr.Logger
}

func newAddTypeKeyToContextStep(log logr.Logger) *addTypeKeyToContextStep {
	return &addTypeKeyToContextStep{
		log: log,
	}
}

func (s *addTypeKeyToContextStep) execute(conf Conf) error {
	s.log.V(1).Info("Adding type key to context")
	contexts := conf["flat_config"].(Conf)
	newEntries := Conf{}

	for key, value := range contexts {
		if isTypedSection(key) {
			newEntries[key+sep+keyType] = value
		}
	}

	for key, value := range newEntries {
		contexts[key] = value
	}

	return nil
}
