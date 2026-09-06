package deployment

import (
	"fmt"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v8"
)

// IsClusterAndStable returns true if the cluster formed by the set of hosts is stable.
func IsClusterAndStable(log logr.Logger, policy *aero.ClientPolicy, allHosts []*HostConn) (bool, error) {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return false, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.IsClusterAndStable(getHostIDsFromHostConns(allHosts))
}

// InfoQuiesce quiesce hosts.
func InfoQuiesce(log logr.Logger, policy *aero.ClientPolicy, allHosts, selectedHosts []*HostConn,
	removedNamespaces []string) error {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.InfoQuiesce(getHostIDsFromHostConns(selectedHosts), getHostIDsFromHostConns(allHosts), removedNamespaces)
}

// InfoQuiesceUndo reverts the effects of quiesce on the next recluster event
// for all hosts in allHosts.
func InfoQuiesceUndo(log logr.Logger, policy *aero.ClientPolicy, allHosts []*HostConn) error {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.InfoQuiesceUndo(getHostIDsFromHostConns(allHosts))
}

// InfoQuiesceUndoSubset reverts the effects of quiesce only on undoHosts (a
// subset of the cluster) while using allHosts to build the full cluster view.
//
// The distinction from InfoQuiesceUndo:
//   - pending_quiesce is only checked and cleared on undoHosts — other nodes
//     (e.g. scale-down targets that are intentionally quiesced) are not touched.
//   - The internal InfoRecluster call uses all hosts in the cluster object
//     (built from allHosts) so the principal node is always reachable, even
//     when undoHosts is a subset that does not include it.
//
// Use this when you want selective quiesce-undo without disturbing nodes that
// should remain quiesced.
func InfoQuiesceUndoSubset(
	log logr.Logger, policy *aero.ClientPolicy,
	undoHosts, allHosts []*HostConn,
) error {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	// Only scan and undo quiesce on undoHosts; InfoRecluster inside
	// c.InfoQuiesceUndo will use c.allHostIDs() (= allHosts) automatically.
	return c.InfoQuiesceUndo(getHostIDsFromHostConns(undoHosts))
}

// InfoRecluster recluster hosts.
func InfoRecluster(log logr.Logger, policy *aero.ClientPolicy, allHosts []*HostConn) error {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.InfoRecluster(getHostIDsFromHostConns(allHosts))
}

// GetQuiescedNodes returns a list of node hostIDs of all nodes that are pending_quiesce=true.
func GetQuiescedNodes(log logr.Logger, policy *aero.ClientPolicy, allHosts []*HostConn) ([]string, error) {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return nil, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.getQuiescedNodes(getHostIDsFromHostConns(allHosts))
}

// SetMigrateFillDelay sets the given migrate-fill-delay on all the given cluster nodes
func SetMigrateFillDelay(log logr.Logger, policy *aero.ClientPolicy, allHosts []*HostConn, migrateFillDelay int) error {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.setMigrateFillDelay(migrateFillDelay, allHosts)
}

// SetConfigCommandsOnHosts runs set config command for dynamic config on all the given cluster nodes
func SetConfigCommandsOnHosts(log logr.Logger, policy *aero.ClientPolicy, allHosts, selectedHosts []*HostConn,
	cmds []string) ([]string, error) {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return nil, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.setConfigCommandsOnHosts(cmds, selectedHosts)
}

// GetClusterNamespaces gets the cluster namespaces
func GetClusterNamespaces(log logr.Logger, policy *aero.ClientPolicy,
	allHosts []*HostConn) (map[string][]string, error) {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return nil, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.getClusterNamespaces(getHostIDsFromHostConns(allHosts))
}

func GetInfoOnHosts(log logr.Logger, policy *aero.ClientPolicy,
	allHosts []*HostConn, cmd string) (map[string]InfoResult, error) {
	c, err := newCluster(log, policy, allHosts)
	if err != nil {
		return nil, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	defer c.close()

	return c.infoOnHosts(getHostIDsFromHostConns(allHosts), cmd)
}
