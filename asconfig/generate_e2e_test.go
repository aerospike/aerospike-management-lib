package asconfig

import (
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/aerospike/aerospike-management-lib/test"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
)

type GenerateE2eTestSuite struct {
	suite.Suite
}

func (suite *GenerateE2eTestSuite) SetupSuite() {
	err := test.Start(1)

	if err != nil {
		suite.T().Fatal(err)
	}
}

// Uncomment this function to check server logs after failure
func (suite *GenerateE2eTestSuite) TearDownSuite() {
	err := test.Stop()

	if err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *GenerateE2eTestSuite) SetupTest() {
}

func (suite *GenerateE2eTestSuite) TestGenerate() {
	Init(logr.Discard(), "/Users/jesseschmidt/Developer/aerospike-admin/lib/live_cluster/client/config-schemas")
	asPolicy := aero.NewClientPolicy()
	host := aero.NewHost(test.IP, test.PORT_START)
	asPolicy.User = "admin"
	asPolicy.Password = "admin"
	asinfo := info.NewAsInfo(logr.Discard(), host, asPolicy)
	genConf, err := GenerateConf(logr.Discard(), asinfo, true)

	suite.Assert().Nil(err)

	genConfWithDefaults, err := GenerateConf(logr.Discard(), asinfo, false)

	suite.Assert().Nil(err)

	asconf, err := NewMapAsConfig(logr.Discard(), genConf.version, genConf.conf)

	suite.Assert().Nil(err)

	asconfWithDefaults, err := NewMapAsConfig(logr.Discard(), genConfWithDefaults.version, genConfWithDefaults.conf)

	suite.Assert().Nil(err)

	test.RestartAerospikeContainer(test.GetAerospikeContainerName(0), asconf.ToConfFile())

	asinfo2 := info.NewAsInfo(logr.Discard(), host, asPolicy)
	genConf2, err := GenerateConf(logr.Discard(), asinfo2, true)

	suite.Assert().Nil(err)

	genConfWithDefaults2, err := GenerateConf(logr.Discard(), asinfo2, false)

	suite.Assert().Nil(err)

	asconf2, err := NewMapAsConfig(logr.Discard(), genConf2.version, genConf2.conf)

	suite.Assert().Nil(err)

	asconfWithDefaults2, err := NewMapAsConfig(logr.Discard(), genConfWithDefaults2.version, genConfWithDefaults2.conf)

	suite.Assert().Nil(err)
	suite.Assert().Equal(asconf, asconf2)
	suite.Assert().Equal(asconfWithDefaults, asconfWithDefaults2)
}

func TestGenerateE2ETestSuiteSuite(t *testing.T) {
	suite.Run(t, new(GenerateE2eTestSuite))
}
