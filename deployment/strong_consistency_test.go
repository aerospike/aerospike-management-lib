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

const (
	testBuild710 = "7.1.0.0"
	nodeInfoCmd  = "node"
	testNS       = "test"
)

type StrongConsistencyTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	mockConn *info.MockConnection
	host     *host
	asinfo   *info.AsInfo
}

func (s *StrongConsistencyTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	mockConnFact := info.NewMockConnectionFactory(s.ctrl)
	s.mockConn = info.NewMockConnection(s.ctrl)

	policy := &aero.ClientPolicy{}
	aHost := &aero.Host{}

	mockConnFact.EXPECT().NewConnection(policy, aHost).Return(s.mockConn, nil).AnyTimes()
	s.mockConn.EXPECT().IsConnected().Return(true).AnyTimes()
	s.mockConn.EXPECT().Login(policy).Return(nil).AnyTimes()
	s.mockConn.EXPECT().SetTimeout(gomock.Any(), time.Second*100).AnyTimes()
	s.mockConn.EXPECT().Close().Return().AnyTimes()

	s.asinfo = info.NewAsInfoWithConnFactory(logr.Discard(), aHost, policy, mockConnFact)

	s.host = &host{
		log: logr.Discard(),
		asConnInfo: &asConnInfo{
			aerospikePolicy: policy,
			asInfo:          s.asinfo,
		},
		id: "h1",
	}
}

// newTestHostWithBuild creates a test host with a pre-cached build value.
// This avoids the need to mock the build call in tests that don't care about it.
func (s *StrongConsistencyTestSuite) newTestHostWithBuild(build string) *host {
	h := &host{
		log: logr.Discard(),
		asConnInfo: &asConnInfo{
			aerospikePolicy: &aero.ClientPolicy{},
			asInfo:          s.asinfo,
		},
		id:    "h1",
		build: sync.OnceValues(func() (string, error) { return build, nil }),
	}

	return h
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledTrue() {
	h := s.newTestHostWithBuild(testBuild710)
	cmd := info.NamespaceConfigCmd(testNS, testBuild710)

	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=true"}, nil)

	isSC, err := isNamespaceSCEnabled(h, testNS)
	s.NoError(err)
	s.True(isSC)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledFalse() {
	h := s.newTestHostWithBuild(testBuild710)
	cmd := info.NamespaceConfigCmd(testNS, testBuild710)

	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=false"}, nil)

	isSC, err := isNamespaceSCEnabled(h, testNS)
	s.NoError(err)
	s.False(isSC)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledMissingKey() {
	h := s.newTestHostWithBuild(testBuild710)
	cmd := info.NamespaceConfigCmd(testNS, testBuild710)

	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "some-key=value"}, nil)

	_, err := isNamespaceSCEnabled(h, testNS)
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledParseError() {
	h := s.newTestHostWithBuild(testBuild710)
	cmd := info.NamespaceConfigCmd(testNS, testBuild710)

	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=notabool"}, nil)

	_, err := isNamespaceSCEnabled(h, testNS)
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestGetSCNamespacesCachesBuild() {
	h := s.newTestHostWithBuild(testBuild710)
	nsCmd := "namespaces"
	cmdTest := info.NamespaceConfigCmd(testNS, testBuild710)
	cmdBar := info.NamespaceConfigCmd("bar", testBuild710)

	// Build is cached via sync.OnceValues, so no "build" command expected here
	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "test;bar"}, nil),
		s.mockConn.EXPECT().RequestInfo(cmdTest).Return(map[string]string{cmdTest: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(cmdBar).Return(map[string]string{cmdBar: "strong-consistency=false"}, nil),
	)

	res, clusterSC, err := getSCNamespaces([]*host{h})
	s.NoError(err)
	s.True(clusterSC)
	s.Equal([]string{"test"}, res[h])
}

func (s *StrongConsistencyTestSuite) TestGetSCNamespacesBuildError() {
	// Create host with build func that will actually call the mock
	h := &host{
		log: logr.Discard(),
		asConnInfo: &asConnInfo{
			aerospikePolicy: &aero.ClientPolicy{},
			asInfo:          s.asinfo,
		},
		id:    "h1",
		build: sync.OnceValues(func() (string, error) { return s.asinfo.Build() }),
	}

	nsCmd := "namespaces"
	buildCmd := "build"

	nsCall := s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "test"}, nil)
	s.mockConn.EXPECT().RequestInfo(buildCmd).Return(nil, aero.ErrTimeout).MinTimes(1).After(nsCall)

	_, _, err := getSCNamespaces([]*host{h})
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_RemovedNamespace() {
	h := s.newTestHostWithBuild(testBuild710)
	removed := map[string]bool{testNS: true}

	skip, err := (&cluster{log: logr.Discard()}).skipInfoQuiesceCheck(h, testNS, removed)
	s.NoError(err)
	s.True(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCEnabledNotInRoster() {
	h := s.newTestHostWithBuild(testBuild710)
	nsCmd := info.NamespaceConfigCmd(testNS, testBuild710)
	rosterCmd := "roster:namespace=test"

	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(nodeInfoCmd).Return(map[string]string{nodeInfoCmd: "N1"}, nil),
		s.mockConn.EXPECT().RequestInfo(rosterCmd).Return(
			map[string]string{rosterCmd: "roster=N2@1;observed_nodes=N1@1"},
			nil,
		),
	)

	skip, err := (&cluster{log: logr.Discard()}).skipInfoQuiesceCheck(h, testNS, map[string]bool{})
	s.NoError(err)
	s.True(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCEnabledInRoster() {
	h := s.newTestHostWithBuild(testBuild710)
	nsCmd := info.NamespaceConfigCmd(testNS, testBuild710)
	rosterCmd := "roster:namespace=test"

	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(nodeInfoCmd).Return(map[string]string{nodeInfoCmd: "N1"}, nil),
		s.mockConn.EXPECT().RequestInfo(rosterCmd).Return(
			map[string]string{rosterCmd: "roster=N1@1;observed_nodes=N1@1"},
			nil,
		),
	)

	skip, err := (&cluster{log: logr.Discard()}).skipInfoQuiesceCheck(h, testNS, map[string]bool{})
	s.NoError(err)
	s.False(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCDisabled() {
	h := s.newTestHostWithBuild(testBuild710)
	nsCmd := info.NamespaceConfigCmd(testNS, testBuild710)

	s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=false"}, nil)

	skip, err := (&cluster{log: logr.Discard()}).skipInfoQuiesceCheck(h, testNS, map[string]bool{})
	s.NoError(err)
	s.False(skip)
}

func TestStrongConsistencyTestSuite(t *testing.T) {
	suite.Run(t, new(StrongConsistencyTestSuite))
}
