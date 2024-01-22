package deployment

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/asconfig"
	"github.com/aerospike/aerospike-management-lib/commons"
	"github.com/aerospike/aerospike-management-lib/info"
)

const (
	cmdSetConfigNetwork   = "set-config:context=network"       // ConfigNetwork
	cmdSetConfigService   = "set-config:context=service"       // ConfigService
	cmdSetConfigNamespace = "set-config:context=namespace;id=" // ConfigNamespace
	cmdSetConfigXDR       = "set-config:context=xdr"           // ConfigXDR
	cmdSetConfigSecurity  = "set-config:context=security"      // ConfigSecurity
	cmdSetLogging         = "log-set:id="                      // ConfigLogging

	nodeAddressPorts = "node-address-ports"
)

func convertValueToString(v1 map[string]interface{}) (map[string][]string, error) {
	valueMap := make(map[string][]string)

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

func handleConfigServiceContext(tokens []string, valueMap map[string][]string) []string {
	val := valueMap[commons.UpdateOp]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigService + ";"

	for _, token := range tokens[1:] {
		cmd = cmd + token + "."
	}

	cmd = strings.TrimSuffix(cmd, ".")

	for _, v := range val {
		finalCMD := cmd + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func handleConfigNetworkContext(tokens []string, valueMap map[string][]string) []string {
	val := valueMap[commons.UpdateOp]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNetwork + ";"

	for _, token := range tokens[1:] {
		cmd = cmd + token + "."
	}

	cmd = strings.TrimSuffix(cmd, ".")

	for _, v := range val {
		finalCMD := cmd + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func handleConfigSecurityContext(tokens []string, valueMap map[string][]string) []string {
	cmdList := make([]string, 0, len(valueMap[commons.AddOp])+len(valueMap[commons.RemoveOp])+
		len(valueMap[commons.UpdateOp]))
	cmd := cmdSetConfigSecurity + ";"

	for _, token := range tokens[1 : len(tokens)-1] {
		cmd = cmd + token + "."
	}

	baseKey := tokens[len(tokens)-1]
	switch baseKey {
	case "report-data-op":
		addedValues := valueMap[commons.AddOp]
		for _, v := range addedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, " ")
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + "=" + "true;" + "namespace=" + namespaceAndSet[0] + ";" + "set=" + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + "=" + "true;" + "namespace=" + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

		removedValues := valueMap[commons.RemoveOp]
		for _, v := range removedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, " ")
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + "=" + "false;" + "namespace=" + namespaceAndSet[0] + ";" + "set=" + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + "=" + "false;" + "namespace=" + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

	case "report-data-op-role":
		addedValues := valueMap[commons.AddOp]
		for _, v := range addedValues {
			finalCMD := cmd + "report-data-op" + "=" + "true;" + "role=" + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := valueMap[commons.RemoveOp]
		for _, v := range removedValues {
			finalCMD := cmd + "report-data-op" + "=" + "false;" + "role=" + v
			cmdList = append(cmdList, finalCMD)
		}

	case "report-data-op-user":
		addedValues := valueMap[commons.AddOp]
		for _, v := range addedValues {
			finalCMD := cmd + "report-data-op" + "=" + "true;" + "user=" + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := valueMap[commons.RemoveOp]
		for _, v := range removedValues {
			finalCMD := cmd + "report-data-op" + "=" + "false;" + "user=" + v
			cmdList = append(cmdList, finalCMD)
		}

	default:
		cmd += baseKey
		for _, v := range valueMap[commons.UpdateOp] {
			finalCMD := cmd + "=" + v
			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

func handleConfigNamespaceContext(tokens []string, valueMap map[string][]string) []string {
	val := valueMap[commons.UpdateOp]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNamespace
	prevToken := info.ConfigNamespaceContext

	for _, token := range tokens[1:] {
		if token[0] == '{' && token[len(token)-1] == '}' {
			switch prevToken {
			case info.ConfigSetContext:
				cmd += fmt.Sprintf(";%s=%s", asconfig.SingularOf(prevToken), strings.Trim(token, "{}"))
			case info.ConfigNamespaceContext:
				cmd += strings.Trim(token, "{}")
			}
		} else {
			if prevToken == "index-type" || prevToken == "sindex-type" {
				cmd += fmt.Sprintf(";%s.%s", prevToken, token)
				prevToken = ""
			} else {
				prevToken = token
			}
		}
	}

	for _, v := range val {
		finalCMD := ""
		if prevToken != "" {
			finalCMD = cmd + ";" + asconfig.SingularOf(prevToken) + "=" + v
		} else {
			finalCMD = cmd + "=" + v
		}

		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func handleConfigLoggingContext(tokens []string, valueMap map[string][]string, conn *ASConn,
	aerospikePolicy *aero.ClientPolicy) ([]string, error) {
	val := valueMap[commons.UpdateOp]
	cmdList := make([]string, 0, len(val))

	confs, err := conn.RunInfo(aerospikePolicy, "logs")
	if err != nil {
		return nil, err
	}

	logs := info.ParseIntoMap(confs["logs"], ";", ":")
	cmd := cmdSetLogging

	logName := strings.Trim(tokens[1], "{}")
	if logName == "console" {
		logName = "stderr"
	}

	for id := range logs {
		if logName == logs[id] {
			cmd += id
			break
		}
	}

	for _, v := range val {
		finalCMD := cmd + ";" + tokens[len(tokens)-1] + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList, nil
}

func handleConfigXDRContext(tokens []string, valueMap map[string][]string) []string {
	cmdList := make([]string, 0, len(valueMap[commons.AddOp])+len(valueMap[commons.RemoveOp])+
		len(valueMap[commons.UpdateOp]))
	cmd := cmdSetConfigXDR
	prevToken := ""
	objectAddedOrRemoved := false
	action := commons.AddOp

	for _, token := range tokens[1:] {
		if commons.ReCurlyBraces.MatchString(token) {
			switch prevToken {
			case info.ConfigDCContext, info.ConfigNamespaceContext:
				cmd += fmt.Sprintf(";%s=%s", asconfig.SingularOf(prevToken), strings.Trim(token, "{}"))
			}
		} else {
			// Assuming there are only 2 object types in XDR context (DC and Namespace)
			if token == asconfig.KeyName {
				objectAddedOrRemoved = true
				if prevToken == info.ConfigDCContext {
					if _, ok := valueMap[commons.AddOp]; ok {
						action = "create"
					}
					if _, ok := valueMap[commons.RemoveOp]; ok {
						action = "delete"
					}
				}
				if prevToken == info.ConfigNamespaceContext {
					if _, ok := valueMap[commons.AddOp]; ok {
						action = commons.AddOp
					}
					if _, ok := valueMap[commons.RemoveOp]; ok {
						action = commons.RemoveOp
					}
				}
			}
			prevToken = token
		}
	}

	if objectAddedOrRemoved {
		finalCMD := cmd + ";" + "action=" + action

		return append(cmdList, finalCMD)
	}

	for op, val := range valueMap {
		for _, v := range val {
			var finalCMD string

			if prevToken == nodeAddressPorts {
				ipAndPort := strings.Split(v, " ")
				if len(ipAndPort) >= 2 && len(ipAndPort) <= 3 {
					finalCMD = cmd + ";" + asconfig.SingularOf(prevToken) + "=" + ipAndPort[0] + ":" +
						ipAndPort[1] + ";action=" + op
				}
			} else {
				finalCMD = cmd + ";" + asconfig.SingularOf(prevToken) + "=" + v
			}

			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

// CreateConfigSetCmdList creates set-config commands for given config.
func CreateConfigSetCmdList(
	log logr.Logger, configMap map[string]map[string]interface{}, conn *ASConn, aerospikePolicy *aero.ClientPolicy,
) ([]string, error) {
	cmdList := make([]string, 0, len(configMap))

	orderedConfList := rearrangeConfigMap(log, configMap)
	for _, c := range orderedConfList {
		tokens := strings.Split(c, ".")
		context := tokens[0]

		val, err := convertValueToString(configMap[c])
		if err != nil {
			return nil, err
		}

		switch context {
		case info.ConfigServiceContext:
			cmdList = append(cmdList, handleConfigServiceContext(tokens, val)...)

		case info.ConfigNetworkContext:
			cmdList = append(cmdList, handleConfigNetworkContext(tokens, val)...)

		case info.ConfigNamespaceContext:
			cmdList = append(cmdList, handleConfigNamespaceContext(tokens, val)...)

		case info.ConfigXDRContext:
			cmdList = append(cmdList, handleConfigXDRContext(tokens, val)...)

		case info.ConfigLoggingContext:
			cmds, err := handleConfigLoggingContext(tokens, val, conn, aerospikePolicy)
			if err != nil {
				return nil, err
			}

			cmdList = append(cmdList, cmds...)

		case info.ConfigSecurityContext:
			cmdList = append(cmdList, handleConfigSecurityContext(tokens, val)...)
		}
	}

	return cmdList, nil
}

// Returns a list of config keys in the order in which they should be applied.
// The order is as follows:
// 1. DCs
// 2. Node Address Ports
// 3. Namespaces
// 4. Other keys
func rearrangeConfigMap(log logr.Logger, configMap map[string]map[string]interface{}) []string {
	rearrangedConfigMap := list.New()

	var (
		lastDC  *list.Element
		lastNAP *list.Element
	)

	for k, v := range configMap {
		baseKey := asconfig.BaseKey(k)
		context := asconfig.ContextKey(k)
		_, ok := v[commons.AddOp]

		if context == info.ConfigXDRContext && baseKey == asconfig.KeyName && ok {
			tokens := commons.SplitKey(log, k, ".")
			switch tokens[len(tokens)-3] {
			case info.ConfigDCContext:
				dc := rearrangedConfigMap.PushFront(k)
				if lastDC == nil {
					lastDC = dc
				}
			case info.ConfigNamespaceContext:
				if lastNAP == nil {
					if lastDC != nil {
						rearrangedConfigMap.InsertAfter(k, lastDC)
					} else {
						rearrangedConfigMap.PushFront(k)
					}
				} else {
					rearrangedConfigMap.InsertAfter(k, lastNAP)
				}
			}
		} else {
			if baseKey == nodeAddressPorts {
				if lastDC == nil {
					lastNAP = rearrangedConfigMap.PushFront(k)
				} else {
					nap := rearrangedConfigMap.InsertAfter(k, lastDC)
					if lastNAP == nil {
						lastNAP = nap
					}
				}
			} else {
				rearrangedConfigMap.PushBack(k)
			}
		}
	}

	finalList := make([]string, 0, rearrangedConfigMap.Len())
	for element := rearrangedConfigMap.Front(); element != nil; element = element.Next() {
		finalList = append(finalList, element.Value.(string))
	}

	return finalList
}

// CreateConfigSetCmdsForPatch creates set-config commands for given config.
func CreateConfigSetCmdsForPatch(
	configMap map[string]interface{}, conn *ASConn, aerospikePolicy *aero.ClientPolicy, version string,
) ([]string, error) {
	conf, err := asconfig.NewMapAsConfig(conn.Log, "", configMap)
	if err != nil {
		return nil, err
	}

	flatConf := conf.GetFlatMap()
	asConfChange := make(map[string]map[string]interface{})

	for k, v := range *flatConf {
		valueMap := make(map[string]interface{})
		valueMap["add"] = v
		asConfChange[k] = valueMap
	}

	isDynamic, err := asconfig.IsAllDynamicConfig(conn.Log, asConfChange, version)
	if err != nil {
		return nil, err
	}

	if !isDynamic {
		return nil, fmt.Errorf("static field has been changed, cannot change config dynamically")
	}

	return CreateConfigSetCmdList(conn.Log, asConfChange, conn, aerospikePolicy)
}
