package deployment

import (
	"fmt"
	"strconv"
	"strings"

	as "github.com/aerospike/aerospike-client-go/v7"
)

type InfoResult map[string]string

func (ir InfoResult) toInt(key string) (int, error) {
	val, ok := ir[key]
	if !ok {
		return 0, fmt.Errorf("field %s missing", key)
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("failed to convert key %q to int: %v", key, err)
	}

	return n, nil
}

func (ir InfoResult) toString(key string) (string, error) {
	val, ok := ir[key]
	if !ok {
		return "", fmt.Errorf("field %s missing", key)
	}

	return val, nil
}

func (ir InfoResult) toBool(key string) (bool, error) {
	val, ok := ir[key]
	if !ok {
		return false, fmt.Errorf("field %s missing", key)
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("failed to convert key %q to bool: %v", key, err)
	}

	return b, nil
}

// parseInfo parses the output of an info command
func parseInfo(info map[string]string) map[string]string {
	m := make(map[string]string)

	for k, v := range info {
		if strings.Contains(v, ";") {
			all := strings.Split(v, ";")
			for _, s := range all {
				// TODO: Is it correct, it was crashing in parsing below string
				// error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small;
				if strings.Contains(s, "=") {
					ss := strings.Split(s, "=")
					kk, vv := ss[0], ss[1]
					m[kk] = vv
				} else {
					m[k] = v
				}
			}
		} else {
			m[k] = v
		}
	}

	return m
}

func getHostIDsFromHostConns(hostConns []*HostConn) []string {
	hostIDs := make([]string, 0, len(hostConns))

	for _, hc := range hostConns {
		hostIDs = append(hostIDs, hc.ID)
	}

	return hostIDs
}

func getHostsFromHostConns(hostConns []*HostConn, policy *as.ClientPolicy) ([]*host, error) {
	hosts := make([]*host, len(hostConns))

	for i := range hostConns {
		host, err := hostConns[i].toHost(policy)
		if err != nil {
			return nil, err
		}

		hosts[i] = host
	}

	return hosts, nil
}

func getNamespaceStats(hosts []*host, namespace string) (map[string]InfoResult, error) {
	stats := make(map[string]InfoResult, len(hosts))
	for _, host := range hosts {
		ir, err := getNamespaceStatsPerHost(host, namespace)
		if err != nil {
			return nil, err
		}

		stats[host.id] = ir
	}

	return stats, nil
}

func getNamespaceStatsPerHost(clHost *host, namespace string) (map[string]string, error) {
	cmd := fmt.Sprintf("namespace/%s", namespace)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd)

	return ParseInfoIntoMap(cmdOutput, ";", "=")
}

func getNodeID(clHost *host) (string, error) {
	cmd := "node"

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return "", err
	}

	return res[cmd], nil
}

// ParseInfoIntoMap parses info string into a map.
func ParseInfoIntoMap(str, del, sep string) (map[string]string, error) {
	m := map[string]string{}
	if str == "" {
		return m, nil
	}

	items := strings.Split(str, del)

	for _, item := range items {
		if item == "" {
			continue
		}

		kv := strings.Split(item, sep)
		if len(kv) < 2 {
			return nil, fmt.Errorf("error parsing info item %s", item)
		}

		m[kv[0]] = strings.Join(kv[1:], sep)
	}

	return m, nil
}

// ContainsString check whether list contains given string
func containsString(list []string, ele string) bool {
	for _, listEle := range list {
		if strings.EqualFold(ele, listEle) {
			return true
		}
	}

	return false
}

func getNamespaces(clHost *host) ([]string, error) {
	cmd := CmdNamespaces

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	if cmdOutput != "" {
		return strings.Split(cmdOutput, ";"), nil
	}

	return nil, nil
}
