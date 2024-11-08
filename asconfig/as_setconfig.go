package asconfig

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v7"
	"github.com/aerospike/aerospike-management-lib/deployment"
	"github.com/aerospike/aerospike-management-lib/info"
)

const (
	cmdSetConfigNetwork   = "set-config:context=network;"      // ConfigNetwork
	cmdSetConfigService   = "set-config:context=service;"      // ConfigService
	cmdSetConfigNamespace = "set-config:context=namespace;id=" // ConfigNamespace
	cmdSetConfigXDR       = "set-config:context=xdr"           // ConfigXDR
	cmdSetConfigSecurity  = "set-config:context=security;"     // ConfigSecurity
	cmdSetLogging         = "log-set:id="                      // ConfigLogging
)

// convertValueToString converts the value of a config to a string.
// only string type can be used to populate set-config commands with values.
func convertValueToString(v1 map[OpType]interface{}) (map[OpType][]string, error) {
	valueMap := make(map[OpType][]string)

	for k, v := range v1 {
		values := make([]string, 0)

		switch val1 := v.(type) {
		case []string:
			valueMap[k] = val1

		case string:
			if val1 == "" {
				val1 = "null"
			}

			valueMap[k] = append(values, val1)

		case bool:
			valueMap[k] = append(values, fmt.Sprintf("%t", v))

		case int, uint64, int64, float64:
			valueMap[k] = append(values, fmt.Sprintf("%v", v))

		default:
			return valueMap, fmt.Errorf("format not supported")
		}
	}

	return valueMap, nil
}

// createSetConfigServiceCmdList creates set-config commands for service context.
func createSetConfigServiceCmdList(tokens []string, operationValueMap map[OpType][]string) []string {
	val := operationValueMap[Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigService

	for _, token := range tokens[1:] {
		cmd = cmd + token + sep
	}

	cmd = strings.TrimSuffix(cmd, sep)

	for _, v := range val {
		finalCMD := cmd + equal + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

// createSetConfigNetworkCmdList creates set-config commands for network context.
func createSetConfigNetworkCmdList(tokens []string, operationValueMap map[OpType][]string) []string {
	val := operationValueMap[Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNetwork

	for _, token := range tokens[1:] {
		cmd = cmd + token + sep
	}

	cmd = strings.TrimSuffix(cmd, sep)

	for _, v := range val {
		finalCMD := cmd + equal + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

// createSetConfigSecurityCmdList creates set-config commands for security context.
func createSetConfigSecurityCmdList(tokens []string, operationValueMap map[OpType][]string) []string {
	cmdList := make([]string, 0, len(operationValueMap))
	cmd := cmdSetConfigSecurity

	for _, token := range tokens[1 : len(tokens)-1] {
		cmd = cmd + token + sep
	}

	baseKey := tokens[len(tokens)-1]
	switch baseKey {
	// example of a command: set-config:context=security;log.report-data-op=true;namespace=test;set=setA
	case keyReportDataOp:
		addedValues := operationValueMap[Add]
		for _, v := range addedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, colon)
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + equal + "true" + semicolon + keyNamespace + equal + namespaceAndSet[0] + semicolon +
					keySet + equal + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + equal + "true" + semicolon + keyNamespace + equal + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[Remove]
		for _, v := range removedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, colon)
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + equal + "false" + semicolon + keyNamespace + equal + namespaceAndSet[0] + semicolon +
					keySet + equal + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + equal + "false" + semicolon + keyNamespace + equal + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

	// example of a command: set-config:context=security;log.report-data-op=false;role=billing
	case "report-data-op-role":
		addedValues := operationValueMap[Add]
		for _, v := range addedValues {
			finalCMD := cmd + keyReportDataOp + equal + "true" + semicolon + "role" + equal + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[Remove]
		for _, v := range removedValues {
			finalCMD := cmd + keyReportDataOp + equal + "false" + semicolon + "role" + equal + v
			cmdList = append(cmdList, finalCMD)
		}

	// example of a command: set-config:context=security;log.report-data-op=true;user=fred
	case "report-data-op-user":
		addedValues := operationValueMap[Add]
		for _, v := range addedValues {
			finalCMD := cmd + keyReportDataOp + equal + "true" + semicolon + "user" + equal + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[Remove]
		for _, v := range removedValues {
			finalCMD := cmd + keyReportDataOp + equal + "false" + semicolon + "user" + equal + v
			cmdList = append(cmdList, finalCMD)
		}

	default:
		cmd += baseKey
		for _, v := range operationValueMap[Update] {
			finalCMD := cmd + equal + v
			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

// createSetConfigNamespaceCmdList creates set-config commands for namespace context.
func createSetConfigNamespaceCmdList(tokens []string, operationValueMap map[OpType][]string) []string {
	val := operationValueMap[Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNamespace
	prevToken := info.ConfigNamespaceContext

	for _, token := range tokens[1:] {
		if token[0] == SectionNameStartChar && token[len(token)-1] == SectionNameEndChar {
			switch prevToken {
			case info.ConfigSetContext:
				cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), strings.Trim(token, "{}"))
			case info.ConfigNamespaceContext:
				cmd += strings.Trim(token, "{}")
			}
		} else {
			switch prevToken {
			case "index-type", "sindex-type":
				cmd += fmt.Sprintf(";%s.%s", prevToken, token)
				prevToken = ""
			case "geo2dsphere-within":
				cmd += fmt.Sprintf(";%s-%s", prevToken, token)
				prevToken = ""
			default:
				prevToken = token
			}
		}
	}

	for _, v := range val {
		finalCMD := ""
		if prevToken != "" {
			finalCMD = cmd + semicolon + SingularOf(prevToken) + equal + v
		} else {
			finalCMD = cmd + equal + v
		}

		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

// createLogSetCmdList creates log-set commands for logging context.
func createLogSetCmdList(tokens []string, operationValueMap map[OpType][]string,
	conn deployment.ASConnInterface, aerospikePolicy *aero.ClientPolicy) ([]string, error) {
	val := operationValueMap[Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetLogging

	logName := strings.Trim(tokens[1], "{}")
	if logName == constLoggingConsole {
		logName = constLoggingStderr
	}

	confs, err := conn.RunInfo(aerospikePolicy, keyLogs)
	if err != nil {
		return nil, err
	}

	loggings := info.ParseIntoMap(confs[keyLogs], semicolon, colon)
	for id, name := range loggings {
		if logName == name {
			cmd += id
			break
		}
	}

	for _, v := range val {
		finalCMD := cmd + semicolon + tokens[len(tokens)-1] + equal + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList, nil
}

// createSetConfigXDRCmdList creates set-config commands for XDR context.
func createSetConfigXDRCmdList(tokens []string, operationValueMap map[OpType][]string) []string {
	cmdList := make([]string, 0, len(operationValueMap))
	cmd := cmdSetConfigXDR
	prevToken := ""
	objectAddedOrRemoved := false
	action := Add

	for _, token := range tokens[1:] {
		if ReCurlyBraces.MatchString(token) {
			switch prevToken {
			case info.ConfigDCContext, info.ConfigNamespaceContext:
				cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), strings.Trim(token, "{}"))
			}
		} else {
			// Assuming there are only 2 section types in XDR context (DC and Namespace)
			if token == KeyName {
				objectAddedOrRemoved = true

				if prevToken == info.ConfigDCContext {
					// example of a command: set-config:context=xdr;dc=dc1;action=create
					if _, ok := operationValueMap[Add]; ok {
						action = "create"
					}

					if _, ok := operationValueMap[Remove]; ok {
						action = "delete"
					}
				}

				if prevToken == info.ConfigNamespaceContext {
					// example of a command: set-config:context=xdr;dc=dc1;namespace=test;action=add
					if _, ok := operationValueMap[Add]; ok {
						action = Add
					}

					if _, ok := operationValueMap[Remove]; ok {
						action = Remove
					}
				}
			}

			prevToken = token
		}
	}

	if objectAddedOrRemoved {
		finalCMD := cmd + semicolon + "action" + equal + string(action)

		return append(cmdList, finalCMD)
	}

	for op, val := range operationValueMap {
		for _, v := range val {
			var finalCMD string

			// example of a command: "set-config:context=xdr;dc=dc1;node-address-port=192.168.55.210:3000;action=add
			if prevToken == keyNodeAddressPorts {
				finalCMD = cmd + semicolon + SingularOf(prevToken) + equal + v + semicolon + "action" + equal + string(op)
			} else {
				finalCMD = cmd + semicolon + SingularOf(prevToken) + equal + v
			}

			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

// CreateSetConfigCmdList creates set-config commands for given config.
func CreateSetConfigCmdList(log logr.Logger, configMap DynamicConfigMap, conn deployment.ASConnInterface,
	aerospikePolicy *aero.ClientPolicy,
) ([]string, error) {
	cmdList := make([]string, 0, len(configMap))

	orderedConfList := rearrangeConfigMap(log, configMap)
	for _, c := range orderedConfList {
		tokens := strings.Split(c, sep)
		context := tokens[0]

		val, err := convertValueToString(configMap[c])
		if err != nil {
			return nil, err
		}

		switch context {
		case info.ConfigServiceContext:
			cmdList = append(cmdList, createSetConfigServiceCmdList(tokens, val)...)

		case info.ConfigNetworkContext:
			cmdList = append(cmdList, createSetConfigNetworkCmdList(tokens, val)...)

		case info.ConfigNamespaceContext:
			cmdList = append(cmdList, createSetConfigNamespaceCmdList(tokens, val)...)

		case info.ConfigXDRContext:
			cmdList = append(cmdList, createSetConfigXDRCmdList(tokens, val)...)

		case info.ConfigLoggingContext:
			cmds, err := createLogSetCmdList(tokens, val, conn, aerospikePolicy)
			if err != nil {
				return nil, err
			}

			cmdList = append(cmdList, cmds...)

		case info.ConfigSecurityContext:
			cmdList = append(cmdList, createSetConfigSecurityCmdList(tokens, val)...)
		}
	}

	return cmdList, nil
}

// Returns a list of config keys in the order in which they should be applied.
// The order is as follows:
// 1. Removed Namespaces -- If user has to change some of the DC direct fields, they will have to remove the namespace
// 2. Added/Removed DCs
// 3. Added/Updated DC direct fields
// 4. Added Namespaces
// 5. Other keys
func rearrangeConfigMap(log logr.Logger, configMap DynamicConfigMap) []string {
	rearrangedConfigMap := list.New()
	finalList := make([]string, 0, len(configMap))

	var (
		lastDC       *list.Element // Last DC name
		lastDCConfig *list.Element // Last DC config eg. node-address-ports
	)

	for k, v := range configMap {
		baseKey := BaseKey(k)
		context := ContextKey(k)
		tokens := SplitKey(log, k, sep)

		if context == info.ConfigXDRContext && baseKey == KeyName {
			switch tokens[len(tokens)-3] {
			// Handle DCs added/removed
			case info.ConfigDCContext:
				dc := rearrangedConfigMap.PushFront(k)
				if lastDC == nil {
					lastDC = dc
				}
			// Handle Namespaces added/removed
			case info.ConfigNamespaceContext:
				if _, ok := v[Remove]; ok {
					// If namespace is removed, directly add it to the final list
					finalList = append(finalList, k)
				} else {
					// If namespace is added, add it after all DCs and their direct configs
					if lastDCConfig == nil {
						if lastDC != nil {
							rearrangedConfigMap.InsertAfter(k, lastDC)
						} else {
							rearrangedConfigMap.PushFront(k)
						}
					} else {
						rearrangedConfigMap.InsertAfter(k, lastDCConfig)
					}
				}
			}
		} else {
			if len(tokens) < 3 {
				rearrangedConfigMap.PushBack(k)
				continue
			}

			// Handle DC direct fields
			if tokens[len(tokens)-3] == info.ConfigDCContext {
				var nap *list.Element
				// Check if the key is 'node-address-ports'
				if strings.HasSuffix(k, sep+keyNodeAddressPorts) {
					if _, ok := v[Remove]; ok {
						dc := rearrangedConfigMap.PushFront(k)

						if lastDC == nil {
							lastDC = dc
						}

						continue
					} else if lastDCConfig != nil {
						// Add 'node-address-ports' after all DC direct fields
						// There are certain fields that must be set before 'node-address-ports', for example, 'tls-name'.
						lastDCConfig = rearrangedConfigMap.InsertAfter(k, lastDCConfig)
						continue
					}
				}

				if lastDC == nil {
					nap = rearrangedConfigMap.PushFront(k)
				} else {
					// Add modified DC direct fields after the DC names and before the namespaces
					nap = rearrangedConfigMap.InsertAfter(k, lastDC)
				}

				if lastDCConfig == nil {
					lastDCConfig = nap
				}
			} else {
				rearrangedConfigMap.PushBack(k)
			}
		}
	}

	for element := rearrangedConfigMap.Front(); element != nil; element = element.Next() {
		finalList = append(finalList, element.Value.(string))
	}

	return finalList
}

func ValidConfigOperations() []OpType {
	return []OpType{Add, Update, Remove}
}

func (o OpType) Validate() error {
	switch o {
	case Add, Update, Remove:
		return nil
	}

	return fmt.Errorf("invalid operation type: %s, Valid operations: %v", o, ValidConfigOperations())
}

type ConfigOperation struct {
	Operation OpType `json:"op"`
	Context   string `json:"context"`
	Config    string `json:"config"`
	Value     string `json:"value,omitempty"`
}

func (p ConfigOperation) Validate() error {
	return p.Operation.Validate()
}

func CreateConfigSetCmdsUsingPatch(
	configMap map[string]interface{}, conn *deployment.ASConn, aerospikePolicy *aero.ClientPolicy, version string,
) ([]string, error) {
	conf, err := NewMapAsConfig(conn.Log, configMap)
	if err != nil {
		return nil, err
	}

	flatConf := conf.GetFlatMap()
	asConfChange := make(DynamicConfigMap)

	for k, v := range *flatConf {
		if strings.HasSuffix(k, sep+KeyName) || strings.HasSuffix(k, sep+keyIndex) {
			// skip namespace, dc, etc names
			continue
		}

		if ok, _ := isListField(k); ok {
			// Ignore these fields as these operations are not update operations
			//TODO: Should we through an error if these fields are present in the configMap patch?
			continue
		}

		valueMap := make(map[OpType]interface{})
		valueMap[Update] = v
		asConfChange[k] = valueMap
	}

	isDynamic, err := IsAllDynamicConfig(conn.Log, asConfChange, version)
	if err != nil {
		return nil, err
	}

	if !isDynamic {
		return nil, fmt.Errorf("static field has been changed, cannot change config dynamically")
	}

	return CreateSetConfigCmdList(logr.Logger{}, asConfChange, conn, aerospikePolicy)
}

func CreateConfigSetCmdsUsingOperation(
	confOp ConfigOperation, conn *deployment.ASConn, aerospikePolicy *aero.ClientPolicy, version string,
) ([]string, error) {
	if err := confOp.Validate(); err != nil {
		return nil, err
	}
	// Context: security.log, Config:report-data-op, Value:test set
	// Map: security.log.report-data-op:map[remove:"test set"]
	path := confOp.Context + sep + confOp.Config
	value := confOp.Value

	if confOp.Config == KeyName {
		if confOp.Operation == Update {
			return nil, fmt.Errorf("cannot update name field")
		}
		// Context: namespaces, Config: name, value: ns1
		// Map: namespaces.{testMem}.name:map[remove:testMem]
		path = confOp.Context + sep + string(SectionNameStartChar) + value + string(SectionNameEndChar) + sep + KeyName
	}

	asConfChange := make(DynamicConfigMap)
	valueMap := make(map[OpType]interface{})
	valueMap[confOp.Operation] = value
	asConfChange[path] = valueMap

	isDynamic, err := IsAllDynamicConfig(conn.Log, asConfChange, version)
	if err != nil {
		return nil, err
	}

	if !isDynamic {
		return nil, fmt.Errorf("static field has been changed, cannot change config dynamically")
	}

	return CreateSetConfigCmdList(logr.Logger{}, asConfChange, conn, aerospikePolicy)
}
