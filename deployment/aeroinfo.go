package deployment

import (
	"fmt"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/citrusleaf/aerospike-management-lib/info"
	log "github.com/inconshreveable/log15"
)

// IsClusterAndStable returns true if the cluster formed by the set of hosts is stable.
func IsClusterAndStable(policy *aero.ClientPolicy, allHosts []*HostConn) (bool, error) {
	c, err := newCluster(policy, allHosts, allHosts, false, false)
	if err != nil {
		return false, fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}
	return c.IsClusterAndStable(getHostIDsFromHostConns(allHosts))
}

// InfoQuiesce quiesces host.
func InfoQuiesce(policy *aero.ClientPolicy, allHosts []*HostConn, selectedHost *HostConn) error {
	c, err := newCluster(policy, allHosts, []*HostConn{selectedHost}, false, false)
	if err != nil {
		return fmt.Errorf("unable to create a cluster copy for running aeroinfo: %v", err)
	}

	return c.InfoQuiesce(selectedHost.ID, getHostIDsFromHostConns(allHosts))
}

// TipClearHostname runs tip clear
func TipClearHostname(aerospikePolicy *aero.ClientPolicy, asConn *ASConn, address string, heartbeatPort int) error {
	res, err := RunInfo(aerospikePolicy, asConn, fmt.Sprintf("tip-clear:host-port-list=%s:%d", address, heartbeatPort))
	pkglog.Info("TipClearHostname", log.Ctx{"res": res})
	return err
}

// TipHostname runs tip clear
func TipHostname(aerospikePolicy *aero.ClientPolicy, asConn *ASConn, address string, heartbeatPort int) error {
	res, err := RunInfo(aerospikePolicy, asConn, fmt.Sprintf("tip:host=%s;port=%d", address, heartbeatPort))
	pkglog.Info("TipHostname", log.Ctx{"res": res})
	return err
}

// AlumniReset runs tip clear
func AlumniReset(aerospikePolicy *aero.ClientPolicy, asConn *ASConn) error {
	res, err := RunInfo(aerospikePolicy, asConn, "services-alumni-reset")
	pkglog.Info("TipClearHostname", log.Ctx{"res": res})
	return err
}

// RunInfo runs info command on given host
func RunInfo(aerospikePolicy *aero.ClientPolicy, asConn *ASConn, command string) (map[string]string, error) {
	h := aero.Host{
		Name:    asConn.AerospikeHostName,
		Port:    asConn.AerospikePort,
		TLSName: asConn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(&h, aerospikePolicy)
	return asinfo.RequestInfo(command)
}
