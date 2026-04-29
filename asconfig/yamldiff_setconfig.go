package asconfig

// CreateSetConfigCmdsFromDiff generates live Aerospike set-config / log-set
// commands from the Dynamic portion of a ConfigDiff produced by YAMLDiff.
//
// The keys in ConfigDiff.Dynamic use the bracket-free format emitted by
// YAMLDiff (e.g. "namespaces.test.replication-factor" rather than the legacy
// "namespaces.{test}.replication-factor").  This function is the intended
// replacement for CreateSetConfigCmdList when the diff originates from
// YAMLDiff.
//
// buildVersion is the Aerospike server build string (e.g. "8.1.1.1"). When
// empty the command format is resolved via conn.RunInfo when conn is non-nil.
//
// Only dynamic changes are processed; static changes in ConfigDiff.Static
// require a rolling restart and cannot be expressed as set-config commands.
// The caller should check HasStaticChanges() and decide how to handle them.

import (
	"container/list"
	"fmt"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/deployment"
	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/go-logr/logr"
)

// CreateSetConfigCmdsFromDiff generates live set-config commands from the
// dynamic changes in diff.
func CreateSetConfigCmdsFromDiff(
	log logr.Logger,
	diff *ConfigDiff,
	conn deployment.ASConnInterface,
	aerospikePolicy *aero.ClientPolicy,
	buildVersion string,
) ([]string, error) {
	if len(diff.Dynamic) == 0 {
		return nil, nil
	}

	build := buildVersion
	if build == "" && conn != nil && aerospikePolicy != nil {
		if m, err := conn.RunInfo(aerospikePolicy, "build"); err == nil {
			build = m["build"]
		}
	}

	cmdList := make([]string, 0, len(diff.Dynamic))

	orderedKeys := orderDiffKeys(diff.Dynamic)

	for _, key := range orderedKeys {
		opMap := diff.Dynamic[key]

		val, err := convertValueToString(opMap)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key, err)
		}

		// Split on "." — no bracket-aware splitting needed in the new format.
		tokens := strings.Split(key, sep)
		context := tokens[0]

		switch context {
		case info.ConfigServiceContext:
			cmdList = append(cmdList, diffGenServiceCmds(tokens, val)...)

		case info.ConfigNetworkContext:
			cmdList = append(cmdList, diffGenNetworkCmds(tokens, val)...)

		case info.ConfigNamespaceContext:
			cmdList = append(cmdList, diffGenNamespaceCmds(tokens, val, build)...)

		case info.ConfigXDRContext:
			cmdList = append(cmdList, diffGenXDRCmds(tokens, val)...)

		case info.ConfigLoggingContext:
			cmds, err := diffGenLoggingCmds(tokens, val, conn, aerospikePolicy)
			if err != nil {
				return nil, err
			}

			cmdList = append(cmdList, cmds...)

		case info.ConfigSecurityContext:
			cmdList = append(cmdList, diffGenSecurityCmds(tokens, val)...)
		}
	}

	return cmdList, nil
}

// ─── service ─────────────────────────────────────────────────────────────────

// diffGenServiceCmds builds set-config:context=service commands.
//
// Key format:  service.<key>[.<subkey>…]
// Command:     set-config:context=service;<key>[.<subkey>…]=<value>
func diffGenServiceCmds(tokens []string, val map[OpType][]string) []string {
	return diffGenSimpleCmds(cmdSetConfigService, tokens[1:], val[Update])
}

// ─── network ─────────────────────────────────────────────────────────────────

// diffGenNetworkCmds builds set-config:context=network commands.
//
// Key format:  network.<sub>.<key>
// Command:     set-config:context=network;<sub>.<key>=<value>
func diffGenNetworkCmds(tokens []string, val map[OpType][]string) []string {
	return diffGenSimpleCmds(cmdSetConfigNetwork, tokens[1:], val[Update])
}

// diffGenSimpleCmds is shared by service and network: the command prefix is
// already set; the remaining tokens are joined with "." and appended as
// <key>=<value>.
func diffGenSimpleCmds(prefix string, keyTokens []string, values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cmd := prefix + strings.Join(keyTokens, sep)
	cmds := make([]string, 0, len(values))

	for _, v := range values {
		cmds = append(cmds, cmd+equal+v)
	}

	return cmds
}

// ─── namespace ────────────────────────────────────────────────────────────────

// diffGenNamespaceCmds builds set-config:context=namespace commands.
//
// Key format (examples):
//
//	namespaces.<ns>.<key>
//	namespaces.<ns>.sets.<set>.<key>
//	namespaces.<ns>.index-type.<key>
//	namespaces.<ns>.geo2dsphere-within.<key>
//	namespaces.<ns>.name                  ← namespace add/remove
//
// Command examples:
//
//	set-config:context=namespace;id=<ns>;<key>=<value>     (server < 7.2)
//	set-config:context=namespace;namespace=<ns>;<key>=<value>  (server ≥ 7.2)
//	set-config:context=namespace;id=<ns>;set=<set>;<key>=<value>
func diffGenNamespaceCmds(tokens []string, val map[OpType][]string, build string) []string {
	if len(tokens) < 2 {
		return nil
	}

	cmd := namespaceSetConfigCmd(build) // "set-config:context=namespace;id=" or ";namespace="
	nsName := tokens[1]
	cmd += nsName // append the namespace name directly (no {} to strip)

	prevToken := info.ConfigNamespaceContext
	cmds := make([]string, 0)

	for _, token := range tokens[2:] {
		if isListSection(prevToken) {
			// token is an instance name following a list-section keyword
			// (e.g. "sets" → next token is set name)
			cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), token)
			prevToken = token
			continue
		}

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

	// Namespace lifecycle (new key is namespaces.<ns> with Add/Remove) is
	// handled at the caller level; no set-config command is emitted.
	if len(tokens) == 2 {
		return nil
	}

	for _, v := range val[Update] {
		var finalCmd string
		if prevToken != "" && prevToken != nsName {
			finalCmd = cmd + semicolon + SingularOf(prevToken) + equal + v
		} else {
			finalCmd = cmd + equal + v
		}

		cmds = append(cmds, finalCmd)
	}

	return cmds
}

// ─── XDR ─────────────────────────────────────────────────────────────────────

// diffGenXDRCmds builds set-config:context=xdr commands.
//
// Key format (examples):
//
//	xdr.dcs.<dc>                             ← DC add/remove (Add/Remove on entry key)
//	xdr.dcs.<dc>.namespaces.<ns>             ← namespace add/remove within DC
//	xdr.dcs.<dc>.node-address-ports          ← add/remove node-address-port entries
//	xdr.dcs.<dc>.<key>                       ← update DC field
//	xdr.dcs.<dc>.namespaces.<ns>.<key>       ← update namespace field within DC
func diffGenXDRCmds(tokens []string, val map[OpType][]string) []string {
	cmd := cmdSetConfigXDR
	prevToken := ""

	for _, token := range tokens[1:] {
		if isListSection(prevToken) {
			// token is the instance name following a list-section keyword
			// (e.g. "dcs" → DC name, "namespaces" → namespace name within DC)
			switch prevToken {
			case info.ConfigDCContext, info.ConfigNamespaceContext:
				cmd += fmt.Sprintf(";%s=%s", SingularOf(prevToken), token)
			}

			prevToken = token
			continue
		}

		prevToken = token
	}

	// Detect section-level add/remove: the last consumed token is an instance
	// name (i.e. its parent was a list-section keyword).
	// tokens layout for DC add/remove:
	//   xdr . dcs . <dc>                  → tokens[len-2] = "dcs"
	// tokens layout for NS add/remove:
	//   xdr . dcs . <dc> . namespaces . <ns>  → tokens[len-2] = "namespaces"
	if len(tokens) >= 3 {
		parentSection := tokens[len(tokens)-2]
		if isListSection(parentSection) {
			if _, ok := val[Add]; ok {
				switch parentSection {
				case info.ConfigDCContext:
					return []string{cmd + semicolon + "action" + equal + "create"}
				case info.ConfigNamespaceContext:
					return []string{cmd + semicolon + "action" + equal + string(Add)}
				}
			}

			if _, ok := val[Remove]; ok {
				switch parentSection {
				case info.ConfigDCContext:
					return []string{cmd + semicolon + "action" + equal + "delete"}
				case info.ConfigNamespaceContext:
					return []string{cmd + semicolon + "action" + equal + string(Remove)}
				}
			}
		}
	}

	cmds := make([]string, 0)

	for op, values := range val {
		for _, v := range values {
			var finalCmd string

			if prevToken == keyNodeAddressPorts {
				// node-address-port uses action=add/remove
				finalCmd = cmd + semicolon + SingularOf(prevToken) + equal + v +
					semicolon + "action" + equal + string(op)
			} else {
				finalCmd = cmd + semicolon + SingularOf(prevToken) + equal + v
			}

			cmds = append(cmds, finalCmd)
		}
	}

	return cmds
}

// ─── logging ─────────────────────────────────────────────────────────────────

// diffGenLoggingCmds builds log-set commands.
//
// Key format:  logging.<sink>.<severity>
// Command:     log-set:id=<id>;<severity>=<value>
//
// The sink name in the key is the file path or "console". The server uses
// "stderr" internally for the console sink.
func diffGenLoggingCmds(
	tokens []string,
	val map[OpType][]string,
	conn deployment.ASConnInterface,
	aerospikePolicy *aero.ClientPolicy,
) ([]string, error) {
	if len(tokens) < 3 {
		return nil, nil
	}

	values := val[Update]
	if len(values) == 0 {
		return nil, nil
	}

	logName := tokens[1]
	if logName == constLoggingConsole {
		logName = constLoggingStderr
	}

	cmd := cmdSetLogging

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

	severity := tokens[len(tokens)-1]
	cmds := make([]string, 0, len(values))

	for _, v := range values {
		cmds = append(cmds, cmd+semicolon+severity+equal+v)
	}

	return cmds, nil
}

// ─── security ────────────────────────────────────────────────────────────────

// diffGenSecurityCmds builds set-config:context=security commands.
//
// Key format:  security[.<sub>].<key>
// Command:     set-config:context=security;<sub>.<key>=<value>
//
// Special keys (report-data-op, report-data-op-role, report-data-op-user)
// are handled individually.
func diffGenSecurityCmds(tokens []string, val map[OpType][]string) []string {
	cmd := cmdSetConfigSecurity

	for _, token := range tokens[1 : len(tokens)-1] {
		cmd += token + sep
	}

	baseKey := tokens[len(tokens)-1]
	cmds := make([]string, 0)

	switch baseKey {
	case keyReportDataOp:
		for _, v := range val[Add] {
			parts := strings.Split(v, colon)
			var finalCmd string

			switch len(parts) {
			case 2:
				finalCmd = cmd + baseKey + equal + "true" +
					semicolon + keyNamespace + equal + parts[0] +
					semicolon + keySet + equal + parts[1]
			case 1:
				finalCmd = cmd + baseKey + equal + "true" +
					semicolon + keyNamespace + equal + parts[0]
			}

			cmds = append(cmds, finalCmd)
		}

		for _, v := range val[Remove] {
			parts := strings.Split(v, colon)
			var finalCmd string

			switch len(parts) {
			case 2:
				finalCmd = cmd + baseKey + equal + "false" +
					semicolon + keyNamespace + equal + parts[0] +
					semicolon + keySet + equal + parts[1]
			case 1:
				finalCmd = cmd + baseKey + equal + "false" +
					semicolon + keyNamespace + equal + parts[0]
			}

			cmds = append(cmds, finalCmd)
		}

	case "report-data-op-role":
		for _, v := range val[Add] {
			cmds = append(cmds, cmd+keyReportDataOp+equal+"true"+semicolon+"role"+equal+v)
		}

		for _, v := range val[Remove] {
			cmds = append(cmds, cmd+keyReportDataOp+equal+"false"+semicolon+"role"+equal+v)
		}

	case "report-data-op-user":
		for _, v := range val[Add] {
			cmds = append(cmds, cmd+keyReportDataOp+equal+"true"+semicolon+"user"+equal+v)
		}

		for _, v := range val[Remove] {
			cmds = append(cmds, cmd+keyReportDataOp+equal+"false"+semicolon+"user"+equal+v)
		}

	default:
		cmd += baseKey
		for _, v := range val[Update] {
			cmds = append(cmds, cmd+equal+v)
		}
	}

	return cmds
}

// orderDiffKeys returns the keys of configMap in the order required by
// Aerospike set-config commands for the new bracket-free key format emitted
// by YAMLDiff.
//
// XDR commands must arrive in a specific sequence:
//  1. DC lifecycle (Add/Remove on xdr.dcs.<dc>)              → front
//  2. DC direct fields (xdr.dcs.<dc>.<field>)                → after DC lifecycle
//  3. XDR namespace lifecycle (xdr.dcs.<dc>.namespaces.<ns>) → after DC fields
//  4. Everything else                                         → back
//
// In the new format section lifecycle is signalled by an Add or Remove op on
// the entry key itself (e.g. xdr.dcs.dc1: {Add: "dc1"}) rather than on a
// "name" sub-key as in the legacy format.
func orderDiffKeys(configMap DynamicConfigMap) []string {
	ordered := list.New()
	finalList := make([]string, 0, len(configMap))

	var (
		lastDC       *list.Element
		lastDCConfig *list.Element
	)

	for k, v := range configMap {
		tokens := strings.Split(k, sep)
		context := tokens[0]

		if context != info.ConfigXDRContext {
			ordered.PushBack(k)
			continue
		}

		// Detect XDR section lifecycle in the new format:
		// xdr.dcs.<dc>              → tokens[len-2] == "dcs"
		// xdr.dcs.<dc>.namespaces.<ns> → tokens[len-2] == "namespaces"
		if len(tokens) >= 2 && isListSection(tokens[len(tokens)-2]) {
			_, hasAdd := v[Add]
			_, hasRemove := v[Remove]

			if hasAdd || hasRemove {
				switch tokens[len(tokens)-2] {
				case info.ConfigDCContext:
					dc := ordered.PushFront(k)
					if lastDC == nil {
						lastDC = dc
					}
				case info.ConfigNamespaceContext:
					if hasRemove {
						finalList = append(finalList, k)
					} else {
						if lastDCConfig == nil {
							if lastDC != nil {
								ordered.InsertAfter(k, lastDC)
							} else {
								ordered.PushFront(k)
							}
						} else {
							ordered.InsertAfter(k, lastDCConfig)
						}
					}
				}

				continue
			}
		}

		// DC direct fields: xdr.dcs.<dc>.<field>  → tokens[len-3] == "dcs"
		if len(tokens) >= 3 && tokens[len(tokens)-3] == info.ConfigDCContext {
			_, hasRemove := v[Remove]

			if strings.HasSuffix(k, sep+keyNodeAddressPorts) {
				if hasRemove {
					dc := ordered.PushFront(k)
					if lastDC == nil {
						lastDC = dc
					}

					continue
				} else if lastDCConfig != nil {
					lastDCConfig = ordered.InsertAfter(k, lastDCConfig)
					continue
				}
			}

			var nap *list.Element
			if lastDC == nil {
				nap = ordered.PushFront(k)
			} else {
				nap = ordered.InsertAfter(k, lastDC)
			}

			if lastDCConfig == nil {
				lastDCConfig = nap
			}

			continue
		}

		ordered.PushBack(k)
	}

	for el := ordered.Front(); el != nil; el = el.Next() {
		finalList = append(finalList, el.Value.(string))
	}

	return finalList
}
