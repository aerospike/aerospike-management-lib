package info

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-management-lib/asconfig"
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

// CreateConfigSetCmdList creates set-config commands for given config.
func CreateConfigSetCmdList(
	configMap map[string]interface{},
) ([]string, error) {
	cmdList := make([]string, 0, len(configMap))

	for c, v := range configMap {
		tokens := strings.Split(c, ".")
		context := tokens[0]
		cmd := ""

		val, err := convertValueToString(v)
		if err != nil {
			return nil, err
		}

		switch context {
		case ConfigServiceContext:
			cmd = cmdSetConfigService + ";"
			for _, token := range tokens[1:] {
				cmd = cmd + token + "."
			}

			cmd = strings.TrimSuffix(cmd, ".")

			for _, v := range val {
				finalCMD := cmd + "=" + v
				cmdList = append(cmdList, finalCMD)
			}

		case ConfigNetworkContext:
			cmd = cmdSetConfigNetwork + ";"
			for _, token := range tokens[1:] {
				cmd = cmd + token + "."
			}

			cmd = strings.TrimSuffix(cmd, ".")

			for _, v := range val {
				finalCMD := cmd + "=" + v
				cmdList = append(cmdList, finalCMD)
			}

		case asconfig.PluralOf(ConfigNamespaceContext):
			cmd = cmdSetConfigNamespace
			prevToken := context

			for _, token := range tokens[1:] {
				if token[0] == '{' && token[len(token)-1] == '}' {
					switch prevToken {
					case asconfig.PluralOf(ConfigSetContext):
						cmd += fmt.Sprintf(";%s=%s", asconfig.SingularOf(prevToken), strings.Trim(token, "{}"))
					case asconfig.PluralOf(ConfigNamespaceContext):
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

		case ConfigXDRContext:
			cmd = cmdSetConfigXDR
			prevToken := ""

			for _, token := range tokens[1:] {
				if token[0] == '{' && token[len(token)-1] == '}' {
					switch prevToken {
					case asconfig.PluralOf(ConfigDCContext), asconfig.PluralOf(ConfigNamespaceContext):
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
			/*	case ConfigLoggingContext:
				logs := parseIntoMap(m[constStatLogIDS], ";", ":")
				for id := range logs {
					cmdList = append(cmdList, cmdConfigLogging+id)
				}
			*/
		case ConfigSecurityContext:
			cmd = cmdSetConfigSecurity + ";"
			for _, token := range tokens[1:] {
				cmd = cmd + token + "."
			}

			cmd = strings.TrimSuffix(cmd, ".")

			for _, v := range val {
				finalCMD := cmd + "=" + v
				cmdList = append(cmdList, finalCMD)
			}
		}
	}

	return cmdList, nil
}
