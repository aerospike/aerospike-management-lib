package deployment

import (
	"fmt"
	"github.com/go-logr/logr"
	"strconv"
	"strings"

	as "github.com/ashishshinde/aerospike-client-go/v6"
)

var (
	rosterKeyObservedNodes     = "observed_nodes"
	rosterKeyRosterNodes       = "roster"
	nsKeyUnavailablePartitions = "unavailable_partitions"
	nsKeyDeadPartitions        = "dead_partitions"
	nsKeyStrongConsistency     = "strong-consistency"
)

func GetAndSetRoster(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy, rosterBlockList []string) error {
	log.Info("Check if we need to Get and set roster for SC namespaces")

	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return err
	}

	scNamespacesPerHost, isClusterSCEnabled, err := getSCNamespaces(log, clHosts)
	if err != nil {
		return err
	}

	if !isClusterSCEnabled {
		log.Info("No SC namespace found in the cluster")
		return nil
	}

	if err := validateSCClusterNsState(log, scNamespacesPerHost, nil); err != nil {
		return fmt.Errorf("cluster namespace state not good, can not set roster: %v", err)
	}

	for clHost, nsList := range scNamespacesPerHost {
		if len(nsList) > 0 {
			clHost.log.Info("Get and set roster", "namespaces", nsList)
		}

		for _, scNs := range nsList {
			rosterNodes, err := getRoster(clHost, scNs)
			if err != nil {
				return err
			}

			if err := setFilteredRosterNodes(clHost, scNs, rosterNodes, rosterBlockList); err != nil {
				return err
			}
		}
	}

	return runRecluster(clHosts)
}

func setFilteredRosterNodes(clHost *host, scNs string, rosterNodes map[string]string, rosterBlockList []string) error {
	observedNodes := rosterNodes[rosterKeyObservedNodes]

	// Remove blocked node from observed_nodes
	observedNodesList := strings.Split(observedNodes, ",")
	var newObservedNodesList []string
	for _, obn := range observedNodesList {
		// nodeRoster := nodeID + "@" + rackID
		obnNodeID := strings.Split(obn, "@")[0]
		if !containsString(rosterBlockList, obnNodeID) {
			newObservedNodesList = append(newObservedNodesList, obn)
		}
	}
	newObservedNodes := strings.Join(newObservedNodesList, ",")
	clHost.log.Info("Remove rosterBlockList from observedNodes", "observedNodes", observedNodes, "rosterBlockList", rosterBlockList)

	currentRoster := rosterNodes[rosterKeyRosterNodes]
	if newObservedNodes == currentRoster {
		// Setting roster is skipped if roster already set
		clHost.log.Info("Roster already set for the node. Skipping", "observedNodes", newObservedNodes, "roster", currentRoster)
		return nil
	}

	return setRoster(clHost, scNs, newObservedNodes)
}

func ValidateSCClusterState(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy, removedNamespaces []string) error {
	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return err
	}

	scNamespacesPerHost, isClusterSCEnabled, err := getSCNamespaces(log, clHosts)
	if err != nil {
		return err
	}

	if !isClusterSCEnabled {
		log.Info("No SC namespace found in the cluster")
		return nil
	}
	return validateSCClusterNsState(log, scNamespacesPerHost, removedNamespaces)
}

func getSCNamespaces(log logr.Logger, clHosts []*host) (map[*host][]string, bool, error) {
	var isClusterSCEnabled bool

	scNamespacesPerHost := map[*host][]string{}
	for i := range clHosts {
		namespaces, err := getNamespaces(clHosts[i])
		if err != nil {
			return nil, isClusterSCEnabled, err
		}

		var nsList []string
		for _, ns := range namespaces {
			isSC, err := isNamespaceSCEnabled(clHosts[i], ns)
			if err != nil {
				return nil, isClusterSCEnabled, err
			}
			if isSC {
				nsList = append(nsList, ns)
				isClusterSCEnabled = true
			}
		}

		scNamespacesPerHost[clHosts[i]] = nsList
	}
	log.Info("Fetched SC namespaces for hosts", "namespace", scNamespacesPerHost)

	return scNamespacesPerHost, isClusterSCEnabled, nil
}

func runRecluster(clHosts []*host) error {
	for _, clHost := range clHosts {
		if err := recluster(clHost); err != nil {
			return err
		}
	}

	return nil
}

func validateSCClusterNsState(log logr.Logger, scNamespacesPerHost map[*host][]string, removedNamespaces []string) error {
	log.Info("Validate SC enabled Cluster namespace State. Looking for unavailable or dead partitions", "namespaces", scNamespacesPerHost)

	for clHost, nsList := range scNamespacesPerHost {
		for _, ns := range nsList {

			// NS is getting removed from nodes. This may lead to unavailable partitions. Therefor skip the check for this NS
			if containsString(removedNamespaces, ns) {
				continue
			}

			kvMap, err := getNamespaceStats(clHost, ns)
			if err != nil {
				return err
			}

			// https://docs.aerospike.com/reference/metrics#unavailable_partitions
			// This is the number of partitions that are unavailable when roster nodes are missing.
			// Will turn into dead_partitions if still unavailable when all roster nodes are present.
			// Some partitions would typically be unavailable under some cluster split situations or
			// when removing more than replication-factor number of nodes from a strong-consistency enabled namespace
			if kvMap[nsKeyUnavailablePartitions] != "0" {
				return fmt.Errorf("cluster namespace %s has non-zero unavailable_partitions %v", ns, kvMap[nsKeyUnavailablePartitions])
			}

			// https://docs.aerospike.com/reference/metrics#dead_partitions
			if kvMap[nsKeyDeadPartitions] != "0" {
				return fmt.Errorf("cluster namespace %s has non-zero dead_partitions %v", ns, kvMap[nsKeyDeadPartitions])
			}
		}
	}
	return nil
}

func isNamespaceSCEnabled(clHost *host, ns string) (bool, error) {

	cmd := fmt.Sprintf("get-config:context=namespace;id=%s", ns)

	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return false, err
	}

	configs, err := ParseInfoIntoMap(res[cmd], ";", "=")
	if err != nil {
		return false, err
	}
	scStr, ok := configs[nsKeyStrongConsistency]
	if !ok {
		return false, fmt.Errorf("strong-consistency config not found, config %v", res)
	}
	scBool, err := strconv.ParseBool(scStr)
	if err != nil {
		return false, err
	}
	clHost.log.Info("Check if namespace is SC enabled", "ns", ns, nsKeyStrongConsistency, scBool)

	return scBool, nil
}

// Info calls

func recluster(clHost *host) error {
	cmd := "recluster:"
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "host", clHost.String(), "cmd", cmd, "output", cmdOutput)

	if strings.ToLower(cmdOutput) != "ok" && strings.ToLower(cmdOutput) != "ignored-by-non-principal" {
		return fmt.Errorf("failed to run `%s` for cluster, %v", cmd, cmdOutput)
	}
	return nil
}

func getNamespaceStats(clHost *host, namespace string) (map[string]string, error) {
	cmd := fmt.Sprintf("namespace/%s", namespace)
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "host", clHost.String(), "cmd", cmd)

	return ParseInfoIntoMap(cmdOutput, ";", "=")
}

func setRoster(clHost *host, namespace, observedNodes string) error {
	cmd := fmt.Sprintf("roster-set:namespace=%s;nodes=%s", namespace, observedNodes)
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "host", clHost.String(), "cmd", cmd, "output", cmdOutput)

	if strings.ToLower(cmdOutput) != "ok" {
		return fmt.Errorf("failed to set roster for namespace %s, %v", namespace, cmdOutput)
	}

	return nil
}

func getRoster(clHost *host, namespace string) (map[string]string, error) {
	cmd := fmt.Sprintf("roster:namespace=%s", namespace)
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}
	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "host", clHost.String(), "cmd", cmd, "output", cmdOutput)

	return ParseInfoIntoMap(cmdOutput, ":", "=")
}

func getNamespaces(clHost *host) ([]string, error) {
	cmd := "namespaces"
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}
	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "host", clHost.String(), "cmd", cmd, "output", cmdOutput)

	if len(cmdOutput) > 0 {
		return strings.Split(cmdOutput, ";"), nil
	}
	return nil, nil
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

//----------------------------------------------------------------

func isNodeInRoster(clHost *host, ns string) (bool, error) {
	nodeID, err := getNodeID(clHost)
	if err != nil {
		return false, err
	}

	rosterNodesMap, err := getRoster(clHost, ns)
	if err != nil {
		return false, err
	}

	clHost.log.Info("Check if node is in roster or not", "nodeID", nodeID, "roster", rosterNodesMap)

	rosterStr := rosterNodesMap[rosterKeyRosterNodes]
	rosterList := strings.Split(rosterStr, ",")

	for _, roster := range rosterList {
		rosterNodeID := strings.Split(roster, "@")[0]
		if nodeID == rosterNodeID {
			return true, nil
		}
	}
	return false, nil
}

func getNodeID(clHost *host) (string, error) {
	cmd := "node"
	res, err := clHost.asConnInfo.asinfo.RequestInfo(cmd)
	if err != nil {
		return "", err
	}

	return res[cmd], nil
}

// ParseInfoIntoMap parses info string into a map.
// TODO adapted from management lib. Should be made public there.
func ParseInfoIntoMap(str string, del string, sep string) (map[string]string, error) {
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
