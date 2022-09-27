package deployment

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aerospike/aerospike-management-lib/asconfig"
	aero "github.com/ashishshinde/aerospike-client-go/v6"
	"github.com/go-logr/logr"
)

// cluster represents an aerospike cluster
type cluster struct {
	allHosts      map[string]*host // all cluster hosts
	selectedHosts map[string]*host // hosts on which script will work

	aerospikeHasTLS      bool // whether aerospike server requires tls authentication
	useServicesAlternate bool // whether aerospike connection uses alternate addresses

	log logr.Logger
}

func getHosts(policy *aero.ClientPolicy, conns []*HostConn) (
	map[string]*host, error,
) {
	var err error
	hosts := make(map[string]*host)
	var nd *host

	for _, conn := range conns {
		nd, err = conn.toHost(policy)
		if err != nil {
			err = fmt.Errorf(
				"failed to create info/conn object for running"+
					" deployment script for host %s: %v",
				conn.ASConn.AerospikeHostName, err,
			)
			break
		}
		hosts[nd.id] = nd
	}
	if err != nil {
		for _, n := range hosts {
			_ = n.Close()
		}
		return nil, err
	}
	return hosts, nil
}

// NewCluster returns a new cluster for the hosts
func newCluster(
	log logr.Logger, policy *aero.ClientPolicy, allConns []*HostConn,
	operableConns []*HostConn, aerospikeHasTLS, useServicesAlternate bool,
) (*cluster, error) {
	allHosts, err := getHosts(policy, allConns)
	if err != nil {
		return nil, err
	}
	selectedHosts, err := getHosts(policy, operableConns)
	if err != nil {
		return nil, err
	}
	c := cluster{
		allHosts:             allHosts,
		selectedHosts:        selectedHosts,
		aerospikeHasTLS:      aerospikeHasTLS,
		useServicesAlternate: useServicesAlternate,
		log:                  log,
	}
	return &c, nil
}

// close closes the aerospike client connections and the ssh connections.
func (c *cluster) close() {
	for _, nd := range c.allHosts {
		if err := nd.Close(); err != nil {
			c.log.V(1).Info(
				"Failed to close node connections", "node", nd, "err", err,
			)
		}
	}
	for _, nd := range c.selectedHosts {
		if err := nd.Close(); err != nil {
			c.log.V(1).Info(
				"Failed to close node connections", "node", nd, "err", err,
			)
		}
	}
}

// IsClusterAndStable returns true if the cluster formed by the set of hosts is stable.
func (c *cluster) IsClusterAndStable(hostIDs []string) (bool, error) {
	lg := c.log.WithValues("nodes", hostIDs)

	if len(hostIDs) == 0 {
		return true, nil
	}

	lg.V(1).Info("Running IsClusterAndStable")

	stats, err := c.infoOnHosts(hostIDs, "statistics")
	if err != nil {
		return false, err
	}
	clusterKeys := make(map[string]bool) // set of all cluster keys
	for id, info := range stats {
		key, err := info.toString("cluster_key")
		if err != nil {
			return false, fmt.Errorf(
				"failed to fetch cluster_key on host %s: %v", id, err,
			)
		}

		clusterKeys[key] = true // add to set key

		size, err := info.toInt("cluster_size")
		if err != nil {
			return false, fmt.Errorf(
				"failed to fetch cluster_size on host %s: %v", id, err,
			)
		}
		if size != len(hostIDs) {
			c.log.V(1).Info(
				"Cluster size not equal", "infoSize", size, "runninSize",
				len(hostIDs),
			)
			return false, nil
		}
		allowed, err := info.toBool("migrate_allowed")
		if err != nil {
			return false, fmt.Errorf(
				"failed to fetch migrate_allowed on host %s: %v", id, err,
			)
		}
		if !allowed {
			c.log.V(1).Info("Cluster not stable, migration not allowed")
			return false, nil
		}

		integrity, err := info.toBool("cluster_integrity")
		if err != nil {
			return false, fmt.Errorf(
				"failed to fetch cluster_integrity on host %s: %v", id, err,
			)
		}
		if !integrity {
			c.log.V(1).Info("Cluster not stable, cluster integrity false")
			return false, nil
		}

		remaining, err := info.toInt("migrate_partitions_remaining")
		if err != nil {
			return false, fmt.Errorf(
				"failed to fetch migrate_partitions_remaining on host %s: %v",
				id, err,
			)
		}
		if remaining > 0 {
			c.log.V(1).Info(
				"Cluster not stable, migrate partitions remaining",
				"remaining", remaining,
			)
			return false, nil
		}
	}
	// it assumes that cluster is running, len(hostIDs) == 0 has bailed out early
	if len(clusterKeys) != 1 { // cluster key not unique
		return false, nil
	}

	lg.V(1).Info("Finished running IsClusterAndStable")
	return true, nil
}

// InfoQuiesce quiesces host.
func (c *cluster) InfoQuiesce(hostID string, hostIDs []string, removedNamespaces []string) error {
	lg := c.log.WithValues("node", hostID)

	if len(hostIDs) < 2 {
		lg.V(1).Info(
			fmt.Sprintf(
				"Skipping quiesce: cluster size %d", len(hostIDs),
			),
		)
		return nil
	}
	lg.V(1).Info("Running InfoQuiesce")

	n, err := c.findHost(hostID)
	if err != nil {
		return err
	}

	lg.V(1).Info("Finding aerospike version")

	build, err := n.asConnInfo.asinfo.RequestInfo("build")
	if err != nil {
		return err
	}

	r, err := asconfig.CompareVersions(build["build"], "4.3.1")
	if err != nil {
		return fmt.Errorf(
			"failed to compare aerospike version on node %s: %v"+
				"", hostID, err,
		)
	}

	if r < 0 {
		// aerospike server version < 4.3.1 does not support quiesce
		lg.V(1).Info(
			fmt.Sprintf(
				"Skipping quiesce: server version (%s) < 4.3.1", build["build"],
			),
		)
		return nil
	}

	lg.V(1).Info("Executing cluster-stable command")

	cmd := fmt.Sprintf(
		"cluster-stable:size=%d;ignore-migrations=no", len(hostIDs),
	)
	infoResults, err := c.infoOnHosts(hostIDs, cmd)
	if err != nil {
		return err
	}

	clusterKey := ""
	for id, info := range infoResults {
		ck, err := info.toString(cmd)
		if err != nil {
			return fmt.Errorf(
				"failed to execute cluster-stable command on"+
					" node %s: %v", id, err,
			)
		}

		if strings.Contains(strings.ToLower(ck), "error") {
			return fmt.Errorf(
				"failed to execute cluster-stable command on node %s: %v", id,
				ck,
			)
		}

		if len(clusterKey) == 0 {
			clusterKey = ck
			continue
		}

		if ck != clusterKey {
			return fmt.Errorf("node %s not part of the cluster", id)
		}
	}

	lg.V(1).Info("Issuing quiesce command `quiesce:`")

	res, err := n.asConnInfo.asinfo.RequestInfo("quiesce:")
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(res["quiesce:"]), "error") {
		return fmt.Errorf("issuing quiesce command failed: %v", res["quiesce:"])
	}

	lg.V(1).Info("Fetching namespace name")

	info, err := n.asConnInfo.asinfo.RequestInfo("namespaces")
	if err != nil {
		return err
	}

	var namespaces []string
	if len(info["namespaces"]) > 0 {
		namespaces = strings.Split(info["namespaces"], ";")
	}

	removedNamespaceMap := make(map[string]bool)
	for _, namespace  := range removedNamespaces {
		removedNamespaceMap[namespace] = true
	}

	for index := range namespaces {
		var passed bool

		if removedNamespaceMap[namespaces[index]] {
			continue
		}

		for i := 0; i < 30; i++ {
			lg.V(1).Info(
				"Verifying execution of quiesce by using namespace", "ns", namespaces[index],
			)

			cmd = fmt.Sprintf("namespace/%s", namespaces[index])
			info, err = c.infoCmd(hostID, cmd)
			if err != nil {
				return err
			}
			key := "pending_quiesce"
			pendingQuiesce, ok := info[key]
			if !ok {
				return fmt.Errorf(
					"field %s missing on node %s, "+
						"namespace %s", key, hostID, namespaces[index],
				)
			}

			if pendingQuiesce != "true" {
				lg.V(1).Info(
					"Verifying pending_quiesce failed on node, "+
						"should be true",
					"pending_quiesce", pendingQuiesce, "host", hostID, "ns", namespaces[index],
				)
				time.Sleep(2 * time.Second)
				continue
			}

			passed = true
			break
		}
		if !passed {
			return fmt.Errorf(
				"pending_quiesce verification failed on node %s, namespace %s",
				hostID, namespaces[index],
			)
		}
	}

	// TODO: skip recluster if the node is already effectively_quesced.
	lg.V(1).Info("Issuing recluster command")

	cmd = "recluster:"
	infoResults, err = c.infoOnHosts(hostIDs, cmd)
	if err != nil {
		return err
	}

	found := false
	for _, info := range infoResults {
		r, _ := info.toString(cmd)
		if r == "ok" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("failed to execute recluster command: no response from principle node")
	}

	for index := range namespaces {
		var passed bool

		if removedNamespaceMap[namespaces[index]] {
			continue
		}

		for i := 0; i < 30; i++ {
			lg.V(1).Info(
				"Verifying execution of recluster by using namespace", "ns", namespaces[index],
			)

			cmd = fmt.Sprintf("namespace/%s", namespaces[index])
			info, err = c.infoCmd(hostID, cmd)
			if err != nil {
				return err
			}

			key := "effective_is_quiesced"
			effectiveIsQuiesced, ok := info[key]
			if !ok {
				return fmt.Errorf(
					"field %s missing on node %s, "+
						"namespace %s", key, hostID, namespaces[index],
				)
			}

			if effectiveIsQuiesced != "true" {
				lg.V(1).Info(
					"Verifying effective_is_quiesced failed on node,"+
						" should be true",
					"effective_is_quiesced", effectiveIsQuiesced, "host",
					hostID, "ns", namespaces[index],
				)
				time.Sleep(2 * time.Second)
				continue
			}

			key = "nodes_quiesced"
			nodesQuiescedStr, ok := info[key]
			if !ok {
				return fmt.Errorf(
					"field %s missing on node %s, "+
						"namespace %s", key, hostID, namespaces[index],
				)
			}

			nodesQuiesced, err := strconv.Atoi(nodesQuiescedStr)
			if err != nil {
				return fmt.Errorf(
					"failed to convert key %q to int: %v", key, err,
				)
			}

			if nodesQuiesced <= 0 {
				lg.V(1).Info(
					"Verifying nodes_quiesced failed on node, "+
						"should be >= 1",
					"nodes_quiesced", nodesQuiesced, "host", hostID, "ns", namespaces[index],
				)
				time.Sleep(2 * time.Second)
				continue
			}

			passed = true
			break
		}
		if !passed {
			return fmt.Errorf(
				"effective_is_quiesced or nodes_quiesced verification failed on node %s, namespace %s",
				hostID, namespaces[index],
			)
		}
	}

	// TODO: Check if we need to add proxy checks.
	lg.V(1).Info("Verifying throughput on the node")

	// client refresh interval is 1 second
	// need to wait till client refreshes cluster and gets new partition table
	sleepSeconds := 2
	succeed := false

	// testing for last 60 seconds transaction
	// so retry loop for 30
	for i := 0; i < 30; i++ {
		lg.V(1).Info("Will try after time", "Seconds", sleepSeconds)
		time.Sleep(time.Duration(sleepSeconds) * time.Second)

		cmd = "throughput:back=10;duration=10;slice=10"
		throughputStr, err := c.infoCmd(hostID, cmd)

		// {test}-read:06:50:24-GMT,ops/sec;06:50:34,4864.8;{test}-write:06:50:24-GMT,ops/sec;06:50:34,4863.9;error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small;error-no-data-yet-or-back-too-small
		if err == nil {
			allList := strings.Split(throughputStr[cmd], ";")
			if len(allList) > 0 {
				nodeInUse := false

				for _, histInfo := range allList {
					fields := strings.Split(histInfo, ",")
					if len(fields) == 2 && fields[1] != "ops/sec" {
						throughputVal, err := strconv.ParseFloat(fields[1], 64)
						if err == nil && throughputVal > 0 {
							nodeInUse = true
							break
						}
					}
				}

				if !nodeInUse {
					succeed = true
					break
				}
			}
		}
	}

	if !succeed {
		return fmt.Errorf("node %s still in use", hostID)
	}

	lg.V(1).Info("Finished running InfoQuiesce")
	return nil
}

func (c *cluster) getQuiescedNodes(hostIDs []string) ([]string, error) {
	var quiescedNodes []string

	namespaces, err := c.getClusterNamespaces(hostIDs)
	if err != nil {
		return nil, err
	}

	hostIDCmdMap := map[string]string{}

	for _, hostID := range hostIDs {
		cmd := fmt.Sprintf("namespace/%s", namespaces[hostID][0])
		hostIDCmdMap[hostID] = cmd
	}

	infoResults, err := c.infoCmdsOnHosts(hostIDCmdMap)
	if err != nil {
		return quiescedNodes, err
	}

	pendingQuiesceKey := "pending_quiesce"

	for hostID, info := range infoResults {
		nodesQuiesced, err := info.toString(pendingQuiesceKey)
		if err != nil {
			return quiescedNodes, fmt.Errorf(
				"failed to get %s on node %s: %v", pendingQuiesceKey, hostID,
				err,
			)
		}

		if nodesQuiesced == "true" {
			quiescedNodes = append(quiescedNodes, hostID)
		}
	}

	return quiescedNodes, nil
}

func (c *cluster) getClusterNamespaces(hostIDs []string) (
	map[string][]string, error,
) {
	cmd := "namespaces"
	infoResults, err := c.infoOnHosts(hostIDs, cmd)
	if err != nil {
		return nil, err
	}

	namespaces := map[string][]string{}
	for hostID, info := range infoResults {
		if len(info["namespaces"]) > 0 {
			namespaces[hostID] = strings.Split(info["namespaces"], ";")
		} else {
			return nil, fmt.Errorf(
				"failed to get namespaces for node %v", hostID,
			)
		}
	}

	return namespaces, nil
}

// InfoQuiesceUndo revert the effects of the quiesce command on the next recluster event.
func (c *cluster) InfoQuiesceUndo(hostIDs []string) error {
	lg := c.log.WithValues("nodes", hostIDs)

	lg.V(1).Info("Running InfoQuiesceUndo")

	if len(hostIDs) == 0 {
		return nil
	}

	// Fetching quiesced Nodes
	quiescedNodes, err := c.getQuiescedNodes(hostIDs)
	if err != nil {
		return err
	}

	// No Node to undo quiesce
	if len(quiescedNodes) == 0 {
		return nil
	}

	lg.V(-1).Info(
		"Found few nodes in quiesced state. Running `quiesce-undo:` for them",
		"nodes", quiescedNodes,
	)

	for _, hostID := range quiescedNodes {
		nodelg := c.log.WithValues("node", hostID)

		n, err := c.findHost(hostID)
		if err != nil {
			return err
		}

		nodelg.V(-1).Info("Issuing undo quiesce command `quiesce-undo:`")

		res, err := n.asConnInfo.asinfo.RequestInfo("quiesce-undo:")
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(res["quiesce-undo:"]), "error") {
			return fmt.Errorf(
				"issuing quiesce command failed: %v",
				res["quiesce-undo:"],
			)
		}
		// TODO: Do we need to check any stats to verify undo?
	}

	return c.infoRecluster(hostIDs)
}

func (c *cluster) infoRecluster(hostIDs []string) error {
	lg := c.log.WithValues("nodes", hostIDs)

	lg.V(1).Info("Issuing recluster command")

	cmd := "recluster:"
	infoResults, err := c.infoOnHosts(hostIDs, cmd)
	if err != nil {
		return err
	}

	found := false
	for _, info := range infoResults {
		r, _ := info.toString(cmd)
		if r == "ok" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("failed to execute recluster command: no response from principle node")
	}

	lg.V(1).Info("Finished running InfoQuiesceUndo")
	return nil
}

// infoCmd runs info cmd on the host
func (c *cluster) infoCmd(hostID, cmd string) (map[string]string, error) {
	lg := c.log.WithValues("node", hostID, "cmd", cmd)
	lg.V(1).Info("Running aerospike InfoCmd")

	n, err := c.findHost(hostID)
	if err != nil {
		return nil, err
	}

	info, err := n.asConnInfo.asinfo.RequestInfo(cmd)
	lg.V(1).Info("Finished running InfoCmd", "err", err)

	if err != nil {
		return nil, err
	}
	return parseInfo(info), nil
}

// infoOnHosts returns the result of running the info command on the hosts.
func (c *cluster) infoOnHosts(
	hostIDs []string, cmd string,
) (map[string]infoResult, error) {
	infos := make(map[string]infoResult) // host id to info output

	var mut sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(hostIDs))
	for _, id := range hostIDs {
		go func(hostID string, wg *sync.WaitGroup) {
			defer wg.Done()
			if info, err := c.infoCmd(hostID, cmd); err == nil {
				mut.Lock()
				defer mut.Unlock()
				infos[hostID] = info
			}
		}(id, &wg)
	}
	wg.Wait()

	if len(infos) != len(hostIDs) {
		return nil, fmt.Errorf(
			"failed to fetch aerospike info `%s` for all hosts %v", cmd,
			hostIDs,
		)
	}
	return infos, nil
}

// infoCmdsOnHosts returns the result of running the info command on the hosts.
func (c *cluster) infoCmdsOnHosts(hostIDCmdMap map[string]string) (
	map[string]infoResult, error,
) {
	infos := make(map[string]infoResult) // host id to info output

	var mut sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(hostIDCmdMap))
	for hostID, cmd := range hostIDCmdMap {
		go func(hostID string, cmd string, wg *sync.WaitGroup) {
			defer wg.Done()
			if info, err := c.infoCmd(hostID, cmd); err == nil {
				mut.Lock()
				defer mut.Unlock()
				infos[hostID] = info
			}
		}(hostID, cmd, &wg)
	}
	wg.Wait()

	if len(infos) != len(hostIDCmdMap) {
		return nil, fmt.Errorf(
			"failed to fetch aerospike info for all hosts %v", hostIDCmdMap,
		)
	}
	return infos, nil
}

func (c *cluster) findHost(hostID string) (*host, error) {
	n, ok := c.allHosts[hostID]
	if !ok {
		return nil, fmt.Errorf("failed to find host %s", hostID)
	}
	return n, nil
}
