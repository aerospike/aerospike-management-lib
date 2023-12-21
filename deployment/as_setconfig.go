package deployment

import (
	"fmt"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/asconfig"
	"github.com/aerospike/aerospike-management-lib/info"
)

const (
	cmdSetConfigNetwork   = "set-config:context=network"       // ConfigNetwork
	cmdSetConfigService   = "set-config:context=service"       // ConfigService
	cmdSetConfigNamespace = "set-config:context=namespace;id=" // ConfigNamespace
	cmdSetConfigXDR       = "set-config:context=xdr"           // ConfigXDR
	cmdSetConfigSecurity  = "set-config:context=security"      // ConfigSecurity
	cmdSetLogging         = "log-set:id="                      // ConfigLogging
)

func convertValueToString(v1 interface{}) ([]string, error) {
	values := make([]string, 0)

	if v1 == nil {
		return values, nil
	}

	switch val1 := v1.(type) {
	case []string:
		return val1, nil

	case string:
		return append(values, val1), nil

	case bool:
		return append(values, fmt.Sprintf("%t", v1)), nil

	case int, uint64, int64, float64:
		return append(values, fmt.Sprintf("%v", v1)), nil

	default:
		return values, fmt.Errorf("format not supported")
	}
}

func handleConfigServiceContext(tokens, val []string) []string {
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

func handleConfigNetworkContext(tokens, val []string) []string {
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

func handleConfigSecurityContext(tokens, val []string) []string {
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigSecurity + ";"

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

func handleConfigNamespaceContext(tokens, val []string) []string {
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigNamespace
	prevToken := asconfig.PluralOf(info.ConfigNamespaceContext)

	for _, token := range tokens[1:] {
		if token[0] == '{' && token[len(token)-1] == '}' {
			switch prevToken {
			case asconfig.PluralOf(info.ConfigSetContext):
				cmd += fmt.Sprintf(";%s=%s", asconfig.SingularOf(prevToken), strings.Trim(token, "{}"))
			case asconfig.PluralOf(info.ConfigNamespaceContext):
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

func handleConfigLoggingContext(tokens, val []string, conn *ASConn,
	aerospikePolicy *aero.ClientPolicy) ([]string, error) {
	cmdList := make([]string, 0, len(val))

	confs, err := conn.RunInfo(aerospikePolicy, "logs")
	if err != nil {
		return nil, err
	}

	logs := info.ParseIntoMap(confs["logs"], ";", ":")
	cmd := cmdSetLogging

	if len(tokens) < 3 {
		return nil, fmt.Errorf("invalid logging context")
	}

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

func handleConfigXDRContext(tokens, val []string) []string {
	cmdList := make([]string, 0, len(val))
	cmd := cmdSetConfigXDR
	prevToken := ""

	for _, token := range tokens[1:] {
		if token[0] == '{' && token[len(token)-1] == '}' {
			switch prevToken {
			case asconfig.PluralOf(info.ConfigDCContext), asconfig.PluralOf(info.ConfigNamespaceContext):
				cmd += fmt.Sprintf(";%s=%s", asconfig.SingularOf(prevToken), strings.Trim(token, "{}"))
			}
		} else {
			prevToken = token
		}
	}

	for _, v := range val {
		finalCMD := cmd + ";" + asconfig.SingularOf(prevToken) + "=" + v
		cmdList = append(cmdList, finalCMD)
	}

	return cmdList
}

// CreateConfigSetCmdList creates set-config commands for given config.
func CreateConfigSetCmdList(
	configMap map[string]interface{}, conn *ASConn, aerospikePolicy *aero.ClientPolicy,
) ([]string, error) {
	cmdList := make([]string, 0, len(configMap))

	for c, v := range configMap {
		tokens := strings.Split(c, ".")
		context := tokens[0]

		val, err := convertValueToString(v)
		if err != nil {
			return nil, err
		}

		switch context {
		case info.ConfigServiceContext:
			cmdList = append(cmdList, handleConfigServiceContext(tokens, val)...)

		case info.ConfigNetworkContext:
			cmdList = append(cmdList, handleConfigNetworkContext(tokens, val)...)

		case asconfig.PluralOf(info.ConfigNamespaceContext):
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

// CreateConfigSetCmdList creates set-config commands for given config.
func CreateConfigSetCmdsForPatch(
	configMap map[string]interface{}, conn *ASConn, aerospikePolicy *aero.ClientPolicy, version string,
) ([]string, error) {
	conf, err := asconfig.NewMapAsConfig(conn.Log, "", configMap)
	if err != nil {
		return nil, err
	}

	flatConf := conf.GetFlatMap()

	isDynamic := asconfig.IsAllDynamicConfig(*flatConf, version)
	if !isDynamic {
		return nil, fmt.Errorf("static field has been changed, cannot change config dynamically")
	}

	return CreateConfigSetCmdList(*flatConf, conn, aerospikePolicy)
}
