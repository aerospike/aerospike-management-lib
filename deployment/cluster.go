package deployment

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/citrusleaf/aerospike-management-lib/asconfig"
	log "github.com/inconshreveable/log15"

	aero "github.com/aerospike/aerospike-client-go"
)

// cluster represents an aerospike cluster
type cluster struct {
	allHosts      map[string]*host // all cluster hosts
	selectedHosts map[string]*host // hosts on which script will work

	aerospikeHasTLS      bool // whether aerospike server requires tls authentication
	useServicesAlternate bool // whether aerospike connection uses alternate addresses

	log log.Logger
}

func getHosts(policy *aero.ClientPolicy, conns []*HostConn) (map[string]*host, error) {
	var err error
	hosts := make(map[string]*host)
	var nd *host

	for _, conn := range conns {
		nd, err = conn.toHost(policy)
		if err != nil {
			err = fmt.Errorf("Failed to create info/conn object for running deployment script for host %s: %v", conn.ASConn.AerospikeHostName, err)
			break
		}
		hosts[nd.id] = nd
	}
	if err != nil {
		for _, n := range hosts {
			n.Close()
		}
		return nil, err
	}
	return hosts, nil
}

// NewCluster returns a new cluster for the hosts
func newCluster(policy *aero.ClientPolicy, allConns []*HostConn, operableConns []*HostConn, aerospikeHasTLS, useServicesAlternate bool) (*cluster, error) {
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
		log:                  pkglog.New(), // TODO add appropriate identifier for cluster
	}
	return &c, nil
}

// close closes the aerospike client connections and the ssh connections.
func (c *cluster) close() {
	for _, nd := range c.allHosts {
		if err := nd.Close(); err != nil {
			c.log.Debug("Failed to close node connections", log.Ctx{"node": nd, "err": err})
		}
	}
	for _, nd := range c.selectedHosts {
		if err := nd.Close(); err != nil {
			c.log.Debug("Failed to close node connections", log.Ctx{"node": nd, "err": err})
		}
	}
}

// IsClusterAndStable returns true if the cluster formed by the set of hosts is stable.
func (c *cluster) IsClusterAndStable(hostIDs []string) (bool, error) {
	lg := c.log.New(log.Ctx{"nodes": hostIDs})

	if len(hostIDs) == 0 {
		return true, nil
	}

	lg.Debug("Running IsClusterAndStable")

	stats, err := c.infoOnHosts(hostIDs, "statistics")
	if err != nil {
		return false, err
	}
	clusterKeys := make(map[string]bool) // set of all cluster keys
	for id, info := range stats {
		key, err := info.toString("cluster_key")
		if err != nil {
			return false, fmt.Errorf("Failed to fetch cluster_key on host %s: %v", id, err)
		}

		clusterKeys[key] = true // add to set key

		size, err := info.toInt("cluster_size")
		if err != nil {
			return false, fmt.Errorf("Failed to fetch cluster_size on host %s: %v", id, err)
		}
		if size != len(hostIDs) {
			c.log.Debug("Cluster size not equal", log.Ctx{"infoSize": size, "runninSize": len(hostIDs)})
			return false, nil
		}

		allowed, err := info.toBool("migrate_allowed")
		if err != nil {
			return false, fmt.Errorf("Failed to fetch migrate_allowed on host %s: %v", id, err)
		}
		if !allowed {
			c.log.Debug("Cluster not stable, migration not allowed")
			return false, nil
		}

		integrity, err := info.toBool("cluster_integrity")
		if err != nil {
			return false, fmt.Errorf("Failed to fetch cluster_integrity on host %s: %v", id, err)
		}
		if !integrity {
			c.log.Debug("Cluster not stable, cluster integrity false")
			return false, nil
		}

		remaining, err := info.toInt("migrate_partitions_remaining")
		if err != nil {
			return false, fmt.Errorf("Failed to fetch migrate_partitions_remaining on host %s: %v", id, err)
		}
		if remaining > 0 {
			c.log.Debug("Cluster not stable, migrate partitions remaining", log.Ctx{"remaining": remaining})
			return false, nil
		}
	}
	// it assumes that cluster is running, len(hostIDs) == 0 has bailed out early
	if len(clusterKeys) != 1 { // cluster key not unique
		return false, nil
	}

	lg.Debug("Finished running IsClusterAndStable")
	return true, nil
}

// InfoQuiesce quiesces host.
func (c *cluster) InfoQuiesce(hostID string, hostIDs []string) error {
	lg := c.log.New(log.Ctx{"node": hostID})

	if len(hostIDs) < 2 {
		lg.Debug(fmt.Sprintf("Skipping quiesce: cluster size %d", len(hostIDs)))
		return nil
	}
	lg.Debug("Running InfoQuiesce")

	n, err := c.findHost(hostID)
	if err != nil {
		return err
	}

	lg.Debug("Finding aerospike version")

	build, err := n.asConnInfo.asinfo.RequestInfo("build")
	if err != nil {
		return err
	}

	r, err := asconfig.CompareVersions(build["build"], "4.3.1")
	if err != nil {
		return fmt.Errorf("Failed to compare aerospike version on node %s: %v", hostID, err)
	}

	if r < 0 {
		// aerospike server version < 4.3.1 does not support quiesce
		lg.Debug(fmt.Sprintf("Skipping quiesce: server version (%s) < 4.3.1", build["build"]))
		return nil
	}

	lg.Debug("Executing cluster-stable command")

	cmd := fmt.Sprintf("cluster-stable:size=%d;ignore-migrations=no", len(hostIDs))
	infoResults, err := c.infoOnHosts(hostIDs, cmd)
	if err != nil {
		return err
	}

	clusterKey := ""
	for id, info := range infoResults {
		ck, err := info.toString(cmd)
		if err != nil {
			return fmt.Errorf("Failed to execute cluster-stable command on node %s: %v", id, err)
		}

		if strings.Contains(strings.ToLower(ck), "error") {
			return fmt.Errorf("Failed to execute cluster-stable command on node %s: %v", id, ck)
		}

		if len(clusterKey) == 0 {
			clusterKey = ck
			continue
		}

		if ck != clusterKey {
			return fmt.Errorf("Node %s not part of the cluster", id)
		}
	}

	lg.Debug("Issuing quiesce command")

	_, err = n.asConnInfo.asinfo.RequestInfo("quiesce:")
	if err != nil {
		return err
	}

	lg.Debug("Fetching namespace name")

	info, err := n.asConnInfo.asinfo.RequestInfo("namespaces")
	if err != nil {
		return err
	}

	ns := ""
	if len(info["namespaces"]) > 0 {
		ns = strings.Split(info["namespaces"], ";")[0]
	}

	// Retry 3 times, but why...?
	if len(ns) > 0 {
		var passed bool

		for i := 0; i < 3; i++ {
			lg.Debug("Verifying execution of quiesce by using namespace", log.Ctx{"ns": ns})

			cmd = fmt.Sprintf("namespace/%s", ns)
			info, err = c.infoCmd(hostID, cmd)
			if err != nil {
				return err
			}

			key := "pending_quiesce"
			pendingQuiesce, ok := info[key]
			if !ok {
				return fmt.Errorf("Field %s missing on node %s, namespace %s", key, hostID, ns)
			}

			if pendingQuiesce != "true" {
				lg.Debug("pending_quiesce verification failed on node, should be true", log.Ctx{"pending_quiesce": pendingQuiesce, "host": hostID, "ns": ns})
				time.Sleep(1 * time.Second)
				continue
			}

			passed = true
			break
		}
		if !passed {
			return fmt.Errorf("pending_quiesce verification failed on node %s, namespace %s", hostID, ns)
		}
	}

	lg.Debug("Issuing recluster command")

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
		return fmt.Errorf("Failed to execute recluster command: no response from principle node")
	}

	if len(ns) > 0 {
		var passed bool
		// Retry 3 times
		for i := 0; i < 3; i++ {
			lg.Debug("Verifying execution of recluster by using namespace", log.Ctx{"ns": ns})

			cmd = fmt.Sprintf("namespace/%s", ns)
			info, err = c.infoCmd(hostID, cmd)
			if err != nil {
				return err
			}

			key := "effective_is_quiesced"
			effectiveIsQuiesced, ok := info[key]
			if !ok {
				return fmt.Errorf("Field %s missing on node %s, namespace %s", key, hostID, ns)
			}

			if effectiveIsQuiesced != "true" {
				lg.Debug("effective_is_quiesced failed on node, should be true", log.Ctx{"effective_is_quiesced": effectiveIsQuiesced, "host": hostID, "ns": ns})
				time.Sleep(1 * time.Second)
				continue
			}

			key = "nodes_quiesced"
			nodesQuiescedStr, ok := info[key]
			if !ok {
				return fmt.Errorf("Field %s missing on node %s, namespace %s", key, hostID, ns)
			}

			nodesQuiesced, err := strconv.Atoi(nodesQuiescedStr)
			if err != nil {
				return fmt.Errorf("Failed to convert key %q to int: %v", key, err)
			}

			if nodesQuiesced != 1 {
				lg.Debug("nodes_quiesced verification failed on node, should be 1", log.Ctx{"nodes_quiesced": nodesQuiesced, "host": hostID, "ns": ns})
				time.Sleep(1 * time.Second)
				continue
			}

			passed = true
			break
		}
		if !passed {
			return fmt.Errorf("effective_is_quiesced or nodes_quiesced verification failed on node %s, namespace %s", hostID, ns)
		}
	}

	lg.Debug("Verifying throughput on the node")

	// client refresh interval is 1 second
	// need to wait till client refreshes cluster and gets new partition table
	sleepSeconds := 1
	succeed := false

	// testing for last 10 seconds transaction
	// so retry loop for 10
	for i := 0; i < 10; i++ {
		lg.Debug("Will try after time", log.Ctx{"Seconds": sleepSeconds})
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
		return fmt.Errorf("Node %s still in use", hostID)
	}

	lg.Debug("Finished running InfoQuiesce")
	return nil
}

// infoCmd runs info cmd on the host
func (c *cluster) infoCmd(hostID, cmd string) (map[string]string, error) {
	lg := c.log.New(log.Ctx{"node": hostID, "cmd": cmd})
	lg.Debug("running aerospike InfoCmd")

	n, err := c.findHost(hostID)
	if err != nil {
		return nil, err
	}

	info, err := n.asConnInfo.asinfo.RequestInfo(cmd)
	lg.Debug("finished running InfoCmd", log.Ctx{"err": err})
	if err != nil {
		return nil, err
	}
	return parseInfo(info), nil
}

// infoOnHosts returns the result of running the info command on the hosts.
func (c *cluster) infoOnHosts(hostIDs []string, cmd string) (map[string]infoResult, error) {
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
		return nil, fmt.Errorf("failed to fetch aerospike info `%s` for all hosts %v", cmd, hostIDs)
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
