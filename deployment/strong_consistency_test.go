package deployment

import (
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

	asinfo := info.NewAsInfoWithConnFactory(logr.Discard(), aHost, policy, mockConnFact)

	s.host = &host{
		log: logr.Discard(),
		asConnInfo: &asConnInfo{
			aerospikePolicy: policy,
			asInfo:          asinfo,
		},
		id: "h1",
	}
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledTrue() {
	build := testBuild710
	cmd := info.NamespaceConfigCmd(testNS, build)
	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=true"}, nil),
	)

	isSC, err := isNamespaceSCEnabled(s.host, testNS, build)
	s.NoError(err)
	s.True(isSC)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledFalse() {
	build := testBuild710
	cmd := info.NamespaceConfigCmd(testNS, build)
	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=false"}, nil)

	isSC, err := isNamespaceSCEnabled(s.host, testNS, build)
	s.NoError(err)
	s.False(isSC)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledMissingKey() {
	build := testBuild710
	cmd := info.NamespaceConfigCmd(testNS, build)
	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "some-key=value"}, nil)

	_, err := isNamespaceSCEnabled(s.host, testNS, build)
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestIsNamespaceSCEnabledParseError() {
	build := testBuild710
	cmd := info.NamespaceConfigCmd(testNS, build)
	s.mockConn.EXPECT().RequestInfo(cmd).Return(map[string]string{cmd: "strong-consistency=notabool"}, nil)

	_, err := isNamespaceSCEnabled(s.host, testNS, build)
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestGetSCNamespacesReusesBuild() {
	nsCmd := "namespaces"
	buildCmd := "build"
	build := testBuild710
	cmdTest := info.NamespaceConfigCmd(testNS, build)
	cmdBar := info.NamespaceConfigCmd("bar", build)

	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "test;bar"}, nil),
		s.mockConn.EXPECT().RequestInfo(buildCmd).Return(map[string]string{buildCmd: build}, nil),
		s.mockConn.EXPECT().RequestInfo(cmdTest).Return(map[string]string{cmdTest: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(cmdBar).Return(map[string]string{cmdBar: "strong-consistency=false"}, nil),
	)

	res, clusterSC, err := getSCNamespaces([]*host{s.host})
	s.NoError(err)
	s.True(clusterSC)
	s.Equal([]string{"test"}, res[s.host])
}

func (s *StrongConsistencyTestSuite) TestGetSCNamespacesBuildError() {
	nsCmd := "namespaces"
	buildCmd := "build"

	nsCall := s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "test"}, nil)
	s.mockConn.EXPECT().RequestInfo(buildCmd).Return(nil, aero.ErrTimeout).MinTimes(1).After(nsCall)

	_, _, err := getSCNamespaces([]*host{s.host})
	s.Error(err)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_RemovedNamespace() {
	removed := map[string]bool{testNS: true}

	skip, err := s.skipInfoQuiesceCheck(removed)
	s.NoError(err)
	s.True(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCEnabledNotInRoster() {
	build := testBuild710
	nsCmd := info.NamespaceConfigCmd(testNS, build)
	rosterCmd := "roster:namespace=test"

	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(nodeInfoCmd).Return(map[string]string{nodeInfoCmd: "N1"}, nil),
		s.mockConn.EXPECT().RequestInfo(rosterCmd).Return(
			map[string]string{rosterCmd: "roster=N2@1;observed_nodes=N1@1"},
			nil,
		),
	)

	skip, err := s.skipInfoQuiesceCheck(map[string]bool{})
	s.NoError(err)
	s.True(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCEnabledInRoster() {
	build := testBuild710
	nsCmd := info.NamespaceConfigCmd(testNS, build)
	rosterCmd := "roster:namespace=test"

	gomock.InOrder(
		s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=true"}, nil),
		s.mockConn.EXPECT().RequestInfo(nodeInfoCmd).Return(map[string]string{nodeInfoCmd: "N1"}, nil),
		s.mockConn.EXPECT().RequestInfo(rosterCmd).Return(
			map[string]string{rosterCmd: "roster=N1@1;observed_nodes=N1@1"},
			nil,
		),
	)

	skip, err := s.skipInfoQuiesceCheck(map[string]bool{})
	s.NoError(err)
	s.False(skip)
}

func (s *StrongConsistencyTestSuite) TestSkipInfoQuiesceCheck_SCDisabled() {
	build := testBuild710
	nsCmd := info.NamespaceConfigCmd(testNS, build)

	s.mockConn.EXPECT().RequestInfo(nsCmd).Return(map[string]string{nsCmd: "strong-consistency=false"}, nil)

	skip, err := s.skipInfoQuiesceCheck(map[string]bool{})
	s.NoError(err)
	s.False(skip)
}

func (s *StrongConsistencyTestSuite) skipInfoQuiesceCheck(
	removed map[string]bool,
) (bool, error) {
	return (&cluster{log: logr.Discard()}).skipInfoQuiesceCheck(s.host, testNS, removed, testBuild710)
}

func TestStrongConsistencyTestSuite(t *testing.T) {
	suite.Run(t, new(StrongConsistencyTestSuite))
}
