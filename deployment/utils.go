package deployment

import (
	"fmt"
	as "github.com/ashishshinde/aerospike-client-go/v6"
	"strconv"
	"strings"
)

type infoResult map[string]string

func (ir infoResult) toInt(key string) (int, error) {
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

func (ir infoResult) toString(key string) (string, error) {
	val, ok := ir[key]
	if !ok {
		return "", fmt.Errorf("field %s missing", key)
	}
	return val, nil
}

func (ir infoResult) toBool(key string) (bool, error) {
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
	var hostIDs []string
	for _, hc := range hostConns {
		hostIDs = append(hostIDs, hc.ID)
	}
	return hostIDs
}

func getHostsFromHostConns(hostConns []*HostConn, policy *as.ClientPolicy) ([]*host, error) {
	var hosts []*host
	for i := range hostConns {
		host, err := hostConns[i].toHost(policy)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}
