package deployment

import (
	"fmt"
	"strconv"
	"strings"

	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"

	as "github.com/aerospike/aerospike-client-go/v8"
	lib "github.com/aerospike/aerospike-management-lib"
)

const (
	rosterKeyObservedNodes     = "observed_nodes"
	rosterKeyRosterNodes       = "roster"
	nsKeyUnavailablePartitions = "unavailable_partitions"
	nsKeyDeadPartitions        = "dead_partitions"
	nsKeyStrongConsistency     = "strong-consistency"
)

func ManageRoster(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy, rosterNodeBlockList []string,
	ignorableNamespaces, racksBlockedFromRoster sets.Set[string]) error {
	log.Info("Check if we need to Get and Set roster for SC namespaces")

	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return err
	}

	scNamespacesPerHost, isClusterSCEnabled, err := getSCNamespaces(clHosts)
	if err != nil {
		return err
	}

	if !isClusterSCEnabled {
		log.Info("No SC namespace found in the cluster")
		return nil
	}

	// Removed namespaces should not be validated, as it will fail when namespace will be available in nodes
	// fewer than replication-factor
	if err := validateSCClusterNsState(log, scNamespacesPerHost, ignorableNamespaces, racksBlockedFromRoster); err != nil {
		return fmt.Errorf("cluster namespace state not good, can not set roster: %v", err)
	}

	var runReclusterFlag bool

	for clHost, nsList := range scNamespacesPerHost {
		if len(nsList) > 0 {
			clHost.log.Info("Get and set roster", "namespaces", nsList)
		}

		for _, scNs := range nsList {
			if ignorableNamespaces.Contains(scNs) {
				continue
			}

			rosterNodes, err := getRoster(clHost, scNs)
			if err != nil {
				return err
			}

			isSettingRoster, err := setFilteredRosterNodes(clHost, scNs, rosterNodes,
				rosterNodeBlockList, racksBlockedFromRoster)
			if err != nil {
				return err
			}

			if isSettingRoster {
				runReclusterFlag = true
			}
		}
	}

	if runReclusterFlag {
		return runRecluster(clHosts)
	}

	return nil
}

func GetAndSetRoster(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy, rosterNodeBlockList []string,
	ignorableNamespaces sets.Set[string]) error {
	return ManageRoster(log, hostConns, policy, rosterNodeBlockList, ignorableNamespaces, nil)
}

// setFilteredRosterNodes removes the rosterNodeBlockList from observed nodes and sets the roster if needed.
// It also returns true if roster is being set and returns false if roster is already set.
func setFilteredRosterNodes(clHost *host, scNs string, rosterNodes map[string]string,
	rosterNodeBlockList []string, racksBlockedFromRoster sets.Set[string]) (bool, error) {
	observedNodes := rosterNodes[rosterKeyObservedNodes]

	observedNodesList, activeRackPrefix := splitRosterNodes(observedNodes)

	var newObservedNodesList []string

	for _, obn := range observedNodesList {
		splitNode := strings.Split(obn, "@")
		if len(splitNode) != 2 {
			return false, fmt.Errorf("invalid observed node format: %s", obn)
		}
		// nodeRoster: nodeID + "@" + rackID
		obnNodeID := splitNode[0]
		obnRackID := splitNode[1]

		if !lib.ContainsString(rosterNodeBlockList, obnNodeID) && !racksBlockedFromRoster.Contains(obnRackID) {
			newObservedNodesList = append(newObservedNodesList, obn)
		}
	}

	newObservedNodes := strings.Join(newObservedNodesList, ",")
	newObservedNodes = activeRackPrefix + newObservedNodes

	clHost.log.Info("Remove rosterNodeBlockList from observedNodes", "observedNodes", observedNodes,
		"rosterNodeBlockList", rosterNodeBlockList)

	currentRoster := rosterNodes[rosterKeyRosterNodes]
	if newObservedNodes == currentRoster {
		// Setting roster is skipped if roster is already set
		clHost.log.Info("Roster is already set for the node. Skipping", "observedNodes", newObservedNodes,
			"roster", currentRoster)
		return false, nil
	}

	return true, setRoster(clHost, scNs, newObservedNodes)
}

func ValidateSCClusterState(log logr.Logger, hostConns []*HostConn, policy *as.ClientPolicy,
	ignorableNamespaces sets.Set[string]) error {
	clHosts, err := getHostsFromHostConns(hostConns, policy)
	if err != nil {
		return err
	}

	scNamespacesPerHost, isClusterSCEnabled, err := getSCNamespaces(clHosts)
	if err != nil {
		return err
	}

	if !isClusterSCEnabled {
		log.Info("No SC namespace found in the cluster")
		return nil
	}

	return validateSCClusterNsState(log, scNamespacesPerHost, ignorableNamespaces, nil)
}

func getSCNamespaces(clHosts []*host) (scNamespacesPerHost map[*host][]string, isClusterSCEnabled bool, err error) {
	scNamespacesPerHost = map[*host][]string{}

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
		clHosts[i].log.Info("Fetched SC namespaces for host", "namespace", nsList)
	}

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

func validateSCClusterNsState(log logr.Logger, scNamespacesPerHost map[*host][]string,
	ignorableNamespaces, racksBlockedFromRoster sets.Set[string]) error {
	var errMsgs = sets.NewSet[string]()

	for clHost, nsList := range scNamespacesPerHost {
		clHost.log.Info("Validate SC enabled Cluster namespace State. Looking for unavailable or dead partitions",
			"namespaces", nsList)

		for _, ns := range nsList {
			// NS is getting removed from nodes. This may lead to unavailable partitions. Therefore, skip the check for this NS
			if ignorableNamespaces.Contains(ns) {
				continue
			}

			// If rack that needs to be blocked is part of roster, then ignore partition errors
			// as some partitions would be unavailable until all nodes in the blocked rack are removed from roster
			// and recluster is run.
			ignorePartitionErrors, err := shouldIgnorePartitions(clHost, ns, racksBlockedFromRoster)
			if err != nil {
				return err
			}

			kvMap, err := getNamespaceStats(clHost, ns)
			if err != nil {
				return err
			}

			// https://aerospike.com/docs/database/reference/metrics#namespace__unavailable_partitions
			// This is the number of partitions that are unavailable when roster nodes are missing.
			// Will turn into dead_partitions if still unavailable when all roster nodes are present.
			// Some partitions would typically be unavailable under some cluster split situations or
			// when removing more than replication-factor number of nodes from a strong-consistency enabled namespace
			// Partition validation
			if errMsg := validateNamespacePartitions(ns, kvMap); errMsg != "" {
				if !ignorePartitionErrors {
					return fmt.Errorf("%s", errMsg)
				}

				errMsgs.Add(errMsg)
			}
		}
	}

	if errMsgs.Cardinality() > 0 {
		allErrMsgs := strings.Join(errMsgs.ToSlice(), "\n- ")

		log.Info("Ignoring partition error for namespace as some racks are blocked from roster", "errors", allErrMsgs)
	}

	return nil
}

func shouldIgnorePartitions(clHost *host, ns string, racksBlockedFromRoster sets.Set[string]) (bool, error) {
	if racksBlockedFromRoster.Cardinality() == 0 {
		return false, nil
	}

	rosterNodes, err := getRoster(clHost, ns)
	if err != nil {
		return false, err
	}

	if rosterNodes[rosterKeyRosterNodes] == "null" {
		return false, nil
	}

	nodes, _ := splitRosterNodes(rosterNodes[rosterKeyRosterNodes])
	for _, node := range nodes {
		parts := strings.Split(node, "@")
		if len(parts) != 2 {
			// treat malformed input as fatal
			return false, fmt.Errorf("invalid roster node format: %s", node)
		}
		// parts[0] = nodeID, parts[1] = rackID
		if racksBlockedFromRoster.Contains(parts[1]) {
			return true, nil
		}
	}

	return false, nil
}

func validateNamespacePartitions(ns string, stats map[string]string) string {
	if stats[nsKeyUnavailablePartitions] != "0" {
		return fmt.Sprintf(
			"namespace %q has non-zero unavailable_partitions: %v",
			ns, stats[nsKeyUnavailablePartitions],
		)
	}

	if stats[nsKeyDeadPartitions] != "0" {
		return fmt.Sprintf(
			"namespace %q has non-zero dead_partitions: %v",
			ns, stats[nsKeyDeadPartitions],
		)
	}

	return ""
}

func splitRosterNodes(rosterNodes string) (nodeIDs []string, activeRackPrefix string) {
	// In active-rack mode,
	// observedNodes: "M" + activeRackID + "|" + nodeID + "@" + rackID + "," + nodeID + "@" + rackID
	if idx := strings.IndexRune(rosterNodes, '|'); idx != -1 {
		activeRackPrefix = rosterNodes[:idx+1]
		rosterNodes = rosterNodes[idx+1:]
	}

	return strings.Split(rosterNodes, ","), activeRackPrefix
}

func isNamespaceSCEnabled(clHost *host, ns string) (bool, error) {
	cmd := fmt.Sprintf("get-config:context=namespace;id=%s", ns)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
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

func recluster(clHost *host) error {
	cmd := "recluster:"

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	if !strings.EqualFold(cmdOutput, "ok") && !strings.EqualFold(cmdOutput, "ignored-by-non-principal") {
		return fmt.Errorf("failed to run `%s` for cluster, %v", cmd, cmdOutput)
	}

	return nil
}

func getNamespaceStats(clHost *host, namespace string) (map[string]string, error) {
	cmd := fmt.Sprintf("namespace/%s", namespace)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd)

	return ParseInfoIntoMap(cmdOutput, ";", "=")
}

func setRoster(clHost *host, namespace, observedNodes string) error {
	cmd := fmt.Sprintf("roster-set:namespace=%s;nodes=%s", namespace, observedNodes)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	if !strings.EqualFold(cmdOutput, "ok") {
		return fmt.Errorf("failed to set roster for namespace %s, %v", namespace, cmdOutput)
	}

	return nil
}

func getRoster(clHost *host, namespace string) (map[string]string, error) {
	cmd := fmt.Sprintf("roster:namespace=%s", namespace)

	res, err := clHost.asConnInfo.asInfo.RequestInfo(cmd)
	if err != nil {
		return nil, err
	}

	cmdOutput := res[cmd]

	clHost.log.V(1).Info("Run info command", "cmd", cmd, "output", cmdOutput)

	return ParseInfoIntoMap(cmdOutput, ":", "=")
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
