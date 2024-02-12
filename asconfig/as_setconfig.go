package asconfig

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/commons"
	"github.com/aerospike/aerospike-management-lib/deployment"
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
	reportDataOp     = "report-data-op"
	namespace        = "namespace"
	set              = "set"
	logs             = "logs"
)

func convertValueToString(v1 map[commons.Operation]interface{}) (map[commons.Operation][]string, error) {
	valueMap := make(map[commons.Operation][]string)

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

func createSetConfigServiceCmdList(tokens []string, operationValueMap map[commons.Operation][]string) []string {
	val := operationValueMap[commons.Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigService + ";"

	for _, token := range tokens[1:] {
		cmd = cmd + token + sep
	}

	cmd = strings.TrimSuffix(cmd, sep)

	for _, v := range val {
		finalCMD := cmd + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func createSetConfigNetworkCmdList(tokens []string, operationValueMap map[commons.Operation][]string) []string {
	val := operationValueMap[commons.Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNetwork + ";"

	for _, token := range tokens[1:] {
		cmd = cmd + token + sep
	}

	cmd = strings.TrimSuffix(cmd, sep)

	for _, v := range val {
		finalCMD := cmd + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func createSetConfigSecurityCmdList(tokens []string, operationValueMap map[commons.Operation][]string) []string {
	cmdList := make([]string, 0, len(operationValueMap))
	cmd := cmdSetConfigSecurity + ";"

	for _, token := range tokens[1 : len(tokens)-1] {
		cmd = cmd + token + sep
	}

	baseKey := tokens[len(tokens)-1]
	switch baseKey {
	case reportDataOp:
		addedValues := operationValueMap[commons.Add]
		for _, v := range addedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, ":")
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + "=" + "true;" + namespace + "=" + namespaceAndSet[0] + ";" +
					set + "=" + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + "=" + "true;" + namespace + "=" + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[commons.Remove]
		for _, v := range removedValues {
			var finalCMD string

			namespaceAndSet := strings.Split(v, ":")
			switch len(namespaceAndSet) {
			case 2:
				finalCMD = cmd + baseKey + "=" + "false;" + namespace + "=" + namespaceAndSet[0] + ";" +
					set + "=" + namespaceAndSet[1]
			case 1:
				finalCMD = cmd + baseKey + "=" + "false;" + namespace + "=" + namespaceAndSet[0]
			}

			cmdList = append(cmdList, finalCMD)
		}

	case "report-data-op-role":
		addedValues := operationValueMap[commons.Add]
		for _, v := range addedValues {
			finalCMD := cmd + reportDataOp + "=" + "true;" + "role=" + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[commons.Remove]
		for _, v := range removedValues {
			finalCMD := cmd + reportDataOp + "=" + "false;" + "role=" + v
			cmdList = append(cmdList, finalCMD)
		}

	case "report-data-op-user":
		addedValues := operationValueMap[commons.Add]
		for _, v := range addedValues {
			finalCMD := cmd + reportDataOp + "=" + "true;" + "user=" + v
			cmdList = append(cmdList, finalCMD)
		}

		removedValues := operationValueMap[commons.Remove]
		for _, v := range removedValues {
			finalCMD := cmd + reportDataOp + "=" + "false;" + "user=" + v
			cmdList = append(cmdList, finalCMD)
		}

	default:
		cmd += baseKey
		for _, v := range operationValueMap[commons.Update] {
			finalCMD := cmd + "=" + v
			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

func createSetConfigNamespaceCmdList(tokens []string, operationValueMap map[commons.Operation][]string) []string {
	val := operationValueMap[commons.Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNamespace
	prevToken := info.ConfigNamespaceContext

	for _, token := range tokens[1:] {
		if token[0] == commons.SectionNameStartChar && token[len(token)-1] == commons.SectionNameEndChar {
			switch prevToken {
			case info.ConfigSetContext:
				cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), strings.Trim(token, "{}"))
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
			finalCMD = cmd + ";" + SingularOf(prevToken) + "=" + v
		} else {
			finalCMD = cmd + "=" + v
		}

		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

func createLogSetCmdList(tokens []string, operationValueMap map[commons.Operation][]string,
	conn deployment.ASConnInterface, aerospikePolicy *aero.ClientPolicy) ([]string, error) {
	val := operationValueMap[commons.Update]
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetLogging

	logName := strings.Trim(tokens[1], "{}")
	if logName == constLoggingConsole {
		logName = constLoggingStderr
	}

	confs, err := conn.RunInfo(aerospikePolicy, logs)
	if err != nil {
		return nil, err
	}

	loggings := info.ParseIntoMap(confs[logs], ";", ":")
	for id, name := range loggings {
		if logName == name {
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

func createSetConfigXDRCmdList(tokens []string, operationValueMap map[commons.Operation][]string) []string {
	cmdList := make([]string, 0, len(operationValueMap))
	cmd := cmdSetConfigXDR
	prevToken := ""
	objectAddedOrRemoved := false
	action := commons.Add

	for _, token := range tokens[1:] {
		if commons.ReCurlyBraces.MatchString(token) {
			switch prevToken {
			case info.ConfigDCContext, info.ConfigNamespaceContext:
				cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), strings.Trim(token, "{}"))
			}
		} else {
			// Assuming there are only 2 section types in XDR context (DC and Namespace)
			if token == KeyName {
				objectAddedOrRemoved = true
				if prevToken == info.ConfigDCContext {
					if _, ok := operationValueMap[commons.Add]; ok {
						action = "create"
					}
					if _, ok := operationValueMap[commons.Remove]; ok {
						action = "delete"
					}
				}
				if prevToken == info.ConfigNamespaceContext {
					if _, ok := operationValueMap[commons.Add]; ok {
						action = commons.Add
					}
					if _, ok := operationValueMap[commons.Remove]; ok {
						action = commons.Remove
					}
				}
			}
			prevToken = token
		}
	}

	if objectAddedOrRemoved {
		finalCMD := cmd + ";" + "action=" + string(action)

		return append(cmdList, finalCMD)
	}

	for op, val := range operationValueMap {
		for _, v := range val {
			var finalCMD string

			if prevToken == nodeAddressPorts {
				val := v

				tokens := strings.Split(v, ":")
				if len(tokens) >= 2 {
					val = tokens[0] + ":" + tokens[1]
				}

				finalCMD = cmd + ";" + SingularOf(prevToken) + "=" + val + ";action=" + string(op)
			} else {
				finalCMD = cmd + ";" + SingularOf(prevToken) + "=" + v
			}

			cmdList = append(cmdList, finalCMD)
		}
	}

	return cmdList
}

// CreateSetConfigCmdList creates set-config commands for given config.
func CreateSetConfigCmdList(
	log logr.Logger, configMap commons.DynamicConfigMap, conn deployment.ASConnInterface,
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
func rearrangeConfigMap(log logr.Logger, configMap commons.DynamicConfigMap) []string {
	rearrangedConfigMap := list.New()
	finalList := make([]string, 0, len(configMap))

	var (
		lastDC  *list.Element // Last DC name
		lastNAP *list.Element // Last DC direct field eg. node-address-ports
	)

	for k, v := range configMap {
		baseKey := BaseKey(k)
		context := ContextKey(k)
		_, removeOP := v[commons.Remove]
		tokens := commons.SplitKey(log, k, sep)

		if context == info.ConfigXDRContext && baseKey == KeyName {
			switch tokens[len(tokens)-3] {
			case info.ConfigDCContext:
				dc := rearrangedConfigMap.PushFront(k)
				if lastDC == nil {
					lastDC = dc
				}
			case info.ConfigNamespaceContext:
				if removeOP {
					finalList = append(finalList, k)
				} else {
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
			}
		} else {
			if len(tokens) < 3 {
				rearrangedConfigMap.PushBack(k)
				continue
			}

			if tokens[len(tokens)-3] == info.ConfigDCContext {
				var nap *list.Element
				if lastDC == nil {
					nap = rearrangedConfigMap.PushFront(k)
				} else {
					nap = rearrangedConfigMap.InsertAfter(k, lastDC)
				}
				if lastNAP == nil {
					lastNAP = nap
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
