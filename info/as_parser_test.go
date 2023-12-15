package info

import (
	"time"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AsParserTestSuite struct {
	suite.Suite
	asinfo   *AsInfo
	ctrl     *gomock.Controller
	mockConn *MockConnection
}

func (s *AsParserTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	mockConnFact := NewMockConnectionFactory(s.ctrl)
	s.mockConn = NewMockConnection(s.ctrl)
	policy := &aero.ClientPolicy{}
	host := &aero.Host{}
	mockConnFact.EXPECT().NewConnection(policy, host).Return(s.mockConn, nil).AnyTimes()
	s.mockConn.EXPECT().IsConnected().Return(true).AnyTimes()
	s.mockConn.EXPECT().Login(policy).Return(nil).AnyTimes()
	s.mockConn.EXPECT().SetTimeout(gomock.Any(), time.Second*100).AnyTimes()
	s.mockConn.EXPECT().Close().Return().AnyTimes()
	s.asinfo = NewAsInfoWithConnFactory(logr.Discard(), host, policy, mockConnFact)
}
