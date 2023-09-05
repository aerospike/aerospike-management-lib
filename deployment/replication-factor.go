package deployment

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	"github.com/aerospike/aerospike-management-lib/info"
	as "github.com/ashishshinde/aerospike-client-go/v6"
)

func GetReplicationFactor(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy,
	namespaces []string) (rfNamespacesPerHost map[string]map[string]int64, err error) {
	log.Info("Check if we need to Get and Set replication-factor for non-SC namespaces")

	rfNamespacesPerHost = make(map[string]map[string]int64)

	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return rfNamespacesPerHost, err
	}

	for _, clHost := range clHosts {
		namespaceRFMap := make(map[string]int64)

		for _, ns := range namespaces {
			rf, err := getRF(clHost, ns)
			if err != nil {
				return rfNamespacesPerHost, err
			}

			// Skip invalid replication-factor
			if rf != -1 {
				namespaceRFMap[ns] = rf
			}
		}

		rfNamespacesPerHost[clHost.id] = namespaceRFMap
	}

	return rfNamespacesPerHost, nil
}

func SetReplicationFactor(hostConns []*HostConn, policy *as.ClientPolicy,
	namespaceRFMap map[string]map[string]int64) (err error) {
	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return err
	}

	for _, host := range clHosts {
		if rfMap, ok := namespaceRFMap[host.id]; ok {
			for ns, rf := range rfMap {
				if err := setRF(host, ns, rf); err != nil {
					return err
				}
			}
		}
	}

	return runRecluster(clHosts)
}

func getRF(clHost *host, ns string) (int64, error) {
	cmd := fmt.Sprintf("get-config:context=namespace;id=%s", ns)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return -1, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	if !strings.EqualFold(cmdOutput, "ok") && strings.Contains(cmdOutput, "namespace not found") {
		clHost.log.V(1).Info("Failed to get replication-factor, namespace not found", "namespace", ns)

		return -1, nil
	}

	idxMap := info.ParseIntoMap(cmdOutput, ";", "=")

	return idxMap["replication-factor"].(int64), nil
}

func setRF(clHost *host, ns string, rf int64) error {
	cmd := fmt.Sprintf("set-config:context=namespace;id=%s;replication-factor=%d", ns, rf)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	if !strings.EqualFold(cmdOutput, "ok") {
		return fmt.Errorf("failed to set roster for namespace %s, %v", ns, cmdOutput)
	}

	return nil
}
