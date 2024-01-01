package deployment

import (
	"fmt"
	"net"
	"strconv"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/info"
)

// HostConn has all parameters to connect to an aerospike host and the machine.
type HostConn struct {
	Log    logr.Logger
	ASConn *ASConn
	ID     string // host UUID string
}

type ASConn struct {
	Log               logr.Logger
	AerospikeHostName string // host name of the machine to connect through aerospike
	AerospikeTLSName  string // tls name of the aerospike connection
	AerospikePort     int    // aerospike port to connect to
}

// NewHostConn returns a new HostConn
func NewHostConn(log logr.Logger, id string, asConn *ASConn) *HostConn {
	return &HostConn{
		Log:    log,
		ID:     id,
		ASConn: asConn,
	}
}

// RunInfo runs info command on given host
func (asc *ASConn) RunInfo(
	aerospikePolicy *aero.ClientPolicy, command ...string,
) (map[string]string, error) {
	h := aero.Host{
		Name:    asc.AerospikeHostName,
		Port:    asc.AerospikePort,
		TLSName: asc.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(asc.Log, &h, aerospikePolicy)

	return asinfo.RequestInfo(command...)
}

// AlumniReset runs services alumni reset
func (asc *ASConn) AlumniReset(aerospikePolicy *aero.ClientPolicy) error {
	res, err := asc.RunInfo(aerospikePolicy, "services-alumni-reset")
	asc.Log.Info("AlumniReset", "res", res)

	return err
}

// TipClearHostname runs tip clear
func (asc *ASConn) TipClearHostname(
	aerospikePolicy *aero.ClientPolicy, address string, heartbeatPort int,
) error {
	res, err := asc.RunInfo(
		aerospikePolicy,
		fmt.Sprintf("tip-clear:host-port-list=%s:%d", address, heartbeatPort),
	)
	asc.Log.Info("TipClearHostname", "res", res)

	return err
}

// TipHostname runs tip clear
func (asc *ASConn) TipHostname(
	aerospikePolicy *aero.ClientPolicy, address string, heartbeatPort int,
) error {
	res, err := asc.RunInfo(
		aerospikePolicy,
		fmt.Sprintf("tip:host=%s;port=%d", address, heartbeatPort),
	)
	asc.Log.Info("TipHostname", "res", res)

	return err
}

// ToHost returns a host object
func (n *HostConn) toHost(policy *aero.ClientPolicy) (*host, error) {
	return newHost(n.ID, policy, n.ASConn)
}

// Implements stringer interface
func (n *HostConn) String() string {
	return net.JoinHostPort(
		n.ASConn.AerospikeHostName, strconv.Itoa(n.ASConn.AerospikePort),
	)
}
