package deployment

import (
	"fmt"

	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/aerospike/aerospike-management-lib/system"
	aero "github.com/ashishshinde/aerospike-client-go/v5"
	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
)

// host is a system on which the aerospike server is running. It provides aerospike
// specific capabilities on the system.
type host struct {
	id          string // host UUID string
	asConnInfo  *asConnInfo
	sshConnInfo *sshConnInfo
	log         logr.Logger
}

type asConnInfo struct {
	// aerospike specific details
	aerospikeHostName string
	aerospikePort     int
	aerospikePolicy   *aero.ClientPolicy
	asinfo            *info.AsInfo
}

type sshConnInfo struct {
	// ssh specific details
	sshHostName string
	sshPort     int
	sshConfig   *ssh.ClientConfig
	sudo        *system.Sudo
	*system.System
}

// newHost creates an aerospike host.
func newHost(id string, aerospikePolicy *aero.ClientPolicy, asConn *ASConn, sshConn *SSHConn) (*host, error) {
	nd := host{
		id:  id,
		log: asConn.Log,
	}

	if asConn != nil {
		nd.asConnInfo = newASConn(aerospikePolicy, asConn)
	}

	var err error
	if sshConn != nil {
		nd.sshConnInfo, err = newSSHConn(sshConn)
		if err != nil {
			return nil, err
		}
	}

	return &nd, nil
}

func newASConn(aerospikePolicy *aero.ClientPolicy, asConn *ASConn) *asConnInfo {
	h := aero.Host{
		Name:    asConn.AerospikeHostName,
		Port:    asConn.AerospikePort,
		TLSName: asConn.AerospikeTLSName,
	}
	asinfo := info.NewAsInfo(asConn.Log, &h, aerospikePolicy)

	return &asConnInfo{
		aerospikeHostName: asConn.AerospikeHostName,
		aerospikePort:     asConn.AerospikePort,
		aerospikePolicy:   aerospikePolicy,
		asinfo:            asinfo,
	}
}

func newSSHConn(sshConn *SSHConn) (*sshConnInfo, error) {
	s, err := system.New(sshConn.Log, sshConn.SSHHostName, sshConn.SSHPort, sshConn.Sudo, sshConn.SSHConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host[%s %d]: %v", sshConn.SSHHostName, sshConn.SSHPort, err)
	}

	// supported versions as of now
	if pm := s.PackageManager(); pm != system.DpkgPkgMgr && pm != system.RpmPkgMgr {
		s.Close()
		return nil, fmt.Errorf("unsupported package manager, has to be one of dpkg or rpm")
	}
	if sm := s.InitManager(); sm == system.UknownServiceSystem {
		s.Close()
		return nil, fmt.Errorf("failed to recognize init systems on the host")
	}

	return &sshConnInfo{
		sshHostName: sshConn.SSHHostName,
		sshPort:     sshConn.SSHPort,
		sshConfig:   sshConn.SSHConfig,
		sudo:        sshConn.Sudo,
		System:      s,
	}, nil
}

func (n *host) String() string {
	return n.id
}

// Close closes all the open connections of the host.
func (n *host) Close() error {
	var err error
	if n.asConnInfo.asinfo != nil {
		if e := n.asConnInfo.asinfo.Close(); e != nil {
			err = e
		}
	}
	if e := n.sshConnInfo.System.Close(); e != nil {
		err = e
	}

	return fmt.Errorf("failed to close asinfo/system connection for host %s: %v", n.asConnInfo.aerospikeHostName, err)
}
