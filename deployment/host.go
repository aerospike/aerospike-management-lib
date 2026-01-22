package deployment

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/info"
)

// host is a system on which the aerospike server is running. It provides aerospike
// specific capabilities on the system.
type host struct {
	asConnInfo *asConnInfo
	// build provides cached, thread-safe access to the Aerospike build version.
	// Initialized via sync.OnceValues to ensure the network call happens only once.
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
		// Initialize cached build getter - fetches once on first call, thread-safe
		nd.build = sync.OnceValues(func() (string, error) {
			return nd.asConnInfo.asInfo.Build()
		})
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
// The value is fetched once on first call and cached for the lifetime of the host connection.
// This method is safe for concurrent use.
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
