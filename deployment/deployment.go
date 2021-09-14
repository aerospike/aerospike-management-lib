package deployment

import (
	"fmt"
	"net"
	"strconv"

	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/aerospike/aerospike-management-lib/system"
	aero "github.com/ashishshinde/aerospike-client-go/v5"
	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
)

// HostConn has all parameters to connect to an aerospike host and the machine.
type HostConn struct {
	Log     *logr.Logger
	ID      string // host UUID string
	ASConn  *ASConn
	SSHConn *SSHConn
}

type ASConn struct {
	Log               *logr.Logger
	AerospikeHostName string // host name of the machine to connect through aerospike
	AerospikePort     int    // aerospike port to connec to
	AerospikeTLSName  string // tls name of the aerospike connection
}

// RunInfo runs info command on given host
func (asc *ASConn) RunInfo(aerospikePolicy *aero.ClientPolicy, command ...string) (map[string]string, error) {
	h := aero.Host{
		Name:    asc.AerospikeHostName,
		Port:    asc.AerospikePort,
		TLSName: asc.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(asc.Log, &h, aerospikePolicy)
	return asinfo.RequestInfo(command...)
}

// AlumniReset runs tip clear
func (asc *ASConn) AlumniReset(aerospikePolicy *aero.ClientPolicy) error {
	res, err := asc.RunInfo(aerospikePolicy, "services-alumni-reset")
	asc.Log.Info("TipClearHostname", "res", res)
	return err
}

// TipClearHostname runs tip clear
func (asc *ASConn) TipClearHostname(aerospikePolicy *aero.ClientPolicy, address string, heartbeatPort int) error {
	res, err := asc.RunInfo(aerospikePolicy, fmt.Sprintf("tip-clear:host-port-list=%s:%d", address, heartbeatPort))
	asc.Log.Info("TipClearHostname", "res", res)
	return err
}

// TipHostname runs tip clear
func (asc *ASConn) TipHostname(aerospikePolicy *aero.ClientPolicy, address string, heartbeatPort int) error {
	res, err := asc.RunInfo(aerospikePolicy, fmt.Sprintf("tip:host=%s;port=%d", address, heartbeatPort))
	asc.Log.Info("TipHostname", "res", res)
	return err
}

type SSHConn struct {
	SSHHostName string            // host name of the machine to use in ssh
	SSHPort     int               // port to ssh into
	SSHConfig   *ssh.ClientConfig // ssh config to connect to machine
	Sudo        *system.Sudo      // sudo privileges on the machine
	Log         *logr.Logger
}

// NewHostConn returns a new HostConn
func NewHostConn(log *logr.Logger, id string, asConn *ASConn, sshConn *SSHConn) *HostConn {
	return &HostConn{
		Log:     log,
		ID:      id,
		ASConn:  asConn,
		SSHConn: sshConn,
	}
}

// ToHost returns a host object
func (n *HostConn) toHost(policy *aero.ClientPolicy) (*host, error) {
	return newHost(n.ID, policy, n.ASConn, n.SSHConn)
}

// Implements stringer interface
func (n *HostConn) String() string {
	return net.JoinHostPort(n.ASConn.AerospikeHostName, strconv.Itoa(n.ASConn.AerospikePort))
}
