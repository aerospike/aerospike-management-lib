package deployment

import (
	"fmt"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/info"
)

// host is a system on which the aerospike server is running. It provides aerospike
// specific capabilities on the system.
type host struct {
	asConnInfo *asConnInfo
	// build provides access to the Aerospike build version.
	// It is injected for testability.
	build func() (string, error)
	log   logr.Logger
	id    string // host UUID string
}

type asConnInfo struct {
	// aerospike specific details
	aerospikePolicy   *aero.ClientPolicy
	asInfo            *info.AsInfo
	aerospikeHostName string
	aerospikePort     int
}

// newHost creates an aerospike host.
func newHost(
	id string, aerospikePolicy *aero.ClientPolicy, asConn *ASConn,
) (*host, error) {
	nd := &host{
		id: id,
	}

	if asConn != nil {
		nd.log = asConn.Log
		nd.asConnInfo = newASConnInfo(aerospikePolicy, asConn)
		nd.build = nd.asConnInfo.asInfo.Build
	}

	return nd, nil
}

func newASConnInfo(aerospikePolicy *aero.ClientPolicy, asConn *ASConn) *asConnInfo {
	h := aero.Host{
		Name:    asConn.AerospikeHostName,
		Port:    asConn.AerospikePort,
		TLSName: asConn.AerospikeTLSName,
	}
	asInfo := info.NewAsInfo(asConn.Log, &h, aerospikePolicy)

	return &asConnInfo{
		aerospikeHostName: asConn.AerospikeHostName,
		aerospikePort:     asConn.AerospikePort,
		aerospikePolicy:   aerospikePolicy,
		asInfo:            asInfo,
	}
}

func (n *host) String() string {
	return n.id
}

// Build returns the Aerospike build version for this host.
func (n *host) Build() (string, error) {
	if n.build == nil {
		return "", fmt.Errorf("host connection not initialized")
	}

	return n.build()
}

// Close closes all the open connections of the host.
func (n *host) Close() error {
	if n.asConnInfo.asInfo != nil {
		if err := n.asConnInfo.asInfo.Close(); err != nil {
			return fmt.Errorf(
				"failed to close asinfo/system connection for host %s: %v",
				n.asConnInfo.aerospikeHostName, err,
			)
		}
	}

	return nil
}
