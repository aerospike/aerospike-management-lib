package deployment

import (
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/info"
)

// Command strings reused across tests.
const (
	nsCmd        = "namespaces"
	nsNS         = "namespace/" + testNS
	quiesceCmd   = "quiesce-undo:"
	reclusterCmd = "recluster:"
)

// QuiesceUndoTestSuite covers cluster.allHostIDs() and the KO-530 behaviour:
// InfoQuiesceUndo must recluster against ALL hosts in the cluster object, not
// only the scanned subset.
type QuiesceUndoTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func (s *QuiesceUndoTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// mockHost creates a host backed by a mock Aerospike connection.
func (s *QuiesceUndoTestSuite) mockHost(id string) (*host, *info.MockConnection) {
	policy := &aero.ClientPolicy{}
	aHost := &aero.Host{}

	connFact := info.NewMockConnectionFactory(s.ctrl)
	conn := info.NewMockConnection(s.ctrl)

	connFact.EXPECT().NewConnection(policy, aHost).Return(conn, nil).AnyTimes()
	conn.EXPECT().IsConnected().Return(true).AnyTimes()
	conn.EXPECT().Login(policy).Return(nil).AnyTimes()
	conn.EXPECT().SetTimeout(gomock.Any(), time.Second*100).AnyTimes()
	conn.EXPECT().Close().Return().AnyTimes()

	return &host{
		log: logr.Discard(),
		id:  id,
		asConnInfo: &asConnInfo{
			aerospikePolicy: policy,
			asInfo:          info.NewAsInfoWithConnFactory(logr.Discard(), aHost, policy, connFact),
		},
		build: sync.OnceValues(func() (string, error) { return testBuild710, nil }),
	}, conn
}

// clusterOf builds a cluster from the supplied hosts without opening any real connections.
func clusterOf(hosts ...*host) *cluster {
	m := make(map[string]*host, len(hosts))
	for _, h := range hosts {
		m[h.id] = h
	}

	return &cluster{allHosts: m, log: logr.Discard()}
}

// expectQuiesceCheck sets up the two mock calls made by getQuiescedNodes for
// a single host: the "namespaces" lookup followed by the per-namespace stat.
// parseInfo only produces individual k=v pairs when the value contains ";".
func expectQuiesceCheck(conn *info.MockConnection, quiesced bool) {
	pending := "false"
	if quiesced {
		pending = "true"
	}

	conn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: testNS}, nil)
	conn.EXPECT().RequestInfo(nsNS).Return(map[string]string{nsNS: "pending_quiesce=" + pending + ";ns=" + testNS}, nil)
}

// ---------------------------------------------------------------------------
// allHostIDs
// ---------------------------------------------------------------------------

func (s *QuiesceUndoTestSuite) TestAllHostIDs() {
	// Use bare host structs — no connection needed to test ID collection.
	c := clusterOf(&host{id: "h1"}, &host{id: "h2"}, &host{id: "h3"})
	s.ElementsMatch([]string{"h1", "h2", "h3"}, c.allHostIDs())

	s.Empty((&cluster{allHosts: map[string]*host{}}).allHostIDs())
}

// ---------------------------------------------------------------------------
// InfoQuiesceUndo
// ---------------------------------------------------------------------------

func (s *QuiesceUndoTestSuite) TestInfoQuiesceUndoEmptyHostIDsIsNoOp() {
	// No mock expectations — any unexpected call would fail the test.
	h1, _ := s.mockHost("h1")
	c := clusterOf(h1)

	s.NoError(c.InfoQuiesceUndo(nil))
	s.NoError(c.InfoQuiesceUndo([]string{}))
}

func (s *QuiesceUndoTestSuite) TestInfoQuiesceUndoNoQuiescedNodesSkipsRecluster() {
	h1, conn1 := s.mockHost("h1")
	expectQuiesceCheck(conn1, false) // pending_quiesce=false → no undo, no recluster

	s.NoError(clusterOf(h1).InfoQuiesceUndo([]string{"h1"}))
}

// TestInfoQuiesceUndoReclusterUsesAllClusterHosts is the core regression test
// for KO-530: recluster: must reach every host in allHosts, not just the
// scanned subset.
//
//	allHosts  = {h1, h2, h3}
//	undoHosts = {h2}           (simulates InfoQuiesceUndoSubset semantics)
//
// Expected: quiesce-undo only on h2; recluster on h1, h2, AND h3.
func (s *QuiesceUndoTestSuite) TestInfoQuiesceUndoReclusterUsesAllClusterHosts() {
	h1, conn1 := s.mockHost("h1")
	h2, conn2 := s.mockHost("h2")
	h3, conn3 := s.mockHost("h3")

	expectQuiesceCheck(conn2, true) // h2 is quiesced
	conn2.EXPECT().RequestInfo(quiesceCmd).Return(map[string]string{quiesceCmd: "ok"}, nil)

	// Recluster must reach all three hosts; only one principal responds "ok".
	conn1.EXPECT().RequestInfo(reclusterCmd).Return(map[string]string{reclusterCmd: "observer"}, nil)
	conn2.EXPECT().RequestInfo(reclusterCmd).Return(map[string]string{reclusterCmd: "ok"}, nil)
	conn3.EXPECT().RequestInfo(reclusterCmd).Return(map[string]string{reclusterCmd: "observer"}, nil)

	s.NoError(clusterOf(h1, h2, h3).InfoQuiesceUndo([]string{"h2"}))
}

func (s *QuiesceUndoTestSuite) TestInfoQuiesceUndoSubsetNotQuiescedSkipsRecluster() {
	h1, _ := s.mockHost("h1")
	h2, conn2 := s.mockHost("h2")
	h3, _ := s.mockHost("h3")

	expectQuiesceCheck(conn2, false) // h2 not quiesced → no undo, no recluster on any host

	s.NoError(clusterOf(h1, h2, h3).InfoQuiesceUndo([]string{"h2"}))
}

func TestQuiesceUndoTestSuite(t *testing.T) {
	suite.Run(t, new(QuiesceUndoTestSuite))
}
