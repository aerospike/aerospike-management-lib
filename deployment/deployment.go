package deployment

import (
	"net"
	"strconv"

	aero "github.com/ashishshinde/aerospike-client-go"
	"github.com/aerospike/aerospike-management-lib/system"
	log "github.com/inconshreveable/log15"
	"golang.org/x/crypto/ssh"
)

// logger with the package name prefixed
var pkglog = log.New(log.Ctx{"module": "lib.deployment"})

// HostConn has all parameters to connect to an aerospike host and the machine.
type HostConn struct {
	ID      string // host UUID string
	ASConn  *ASConn
	SSHConn *SSHConn
}

type ASConn struct {
	AerospikeHostName string // host name of the machine to connect through aerospike
	AerospikePort     int    // aerospike port to connec to
	AerospikeTLSName  string // tls name of the aerospike connection
}

type SSHConn struct {
	SSHHostName string            // host name of the machine to use in ssh
	SSHPort     int               // port to ssh into
	SSHConfig   *ssh.ClientConfig // ssh config to connect to machine
	Sudo        *system.Sudo      // sudo privileges on the machine
}

// NewHostConn returns a new HostConn
func NewHostConn(id string, asConn *ASConn, sshConn *SSHConn) *HostConn {
	return &HostConn{
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
