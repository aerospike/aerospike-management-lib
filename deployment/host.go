package deployment

import (
	"fmt"

	"github.com/aerospike/aerospike-management-lib/info"
	aero "github.com/ashishshinde/aerospike-client-go/v6"
	"github.com/go-logr/logr"
)

// host is a system on which the aerospike server is running. It provides aerospike
// specific capabilities on the system.
type host struct {
	id         string // host UUID string
	asConnInfo *asConnInfo
	log        logr.Logger
}

type asConnInfo struct {
	// aerospike specific details
	aerospikeHostName string
	aerospikePort     int
	aerospikePolicy   *aero.ClientPolicy
	asinfo            *info.AsInfo
}

// newHost creates an aerospike host.
func newHost(
	id string, aerospikePolicy *aero.ClientPolicy, asConn *ASConn,
) (*host, error) {
	nd := host{
		id: id,
	}

	if asConn != nil {
		nd.log = asConn.Log
		nd.asConnInfo = newASConn(aerospikePolicy, asConn)
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
	return fmt.Errorf(
		"failed to close asinfo/system connection for host %s: %v",
		n.asConnInfo.aerospikeHostName, err,
	)
}
