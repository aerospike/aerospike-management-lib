package test

import (
	"log"
	"os"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/asconfig"
	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
)

type GenerateE2eTestSuite struct {
	suite.Suite
}

func (suite *GenerateE2eTestSuite) SetupSuite() {
	err := Start(1)

	if err != nil {
		suite.T().Fatal(err)
	}

	schemaDir := os.Getenv("TEST_SCHEMA_DIR")
	if schemaDir == "" {
		log.Printf("Env var TEST_SCHEMA_DIR must be set.")
		suite.T().Fail()
	}

	asconfig.Init(logr.Discard(), schemaDir)
}

// Uncomment this function to check server logs after failure
func (suite *GenerateE2eTestSuite) TearDownSuite() {
	err := Stop()

	if err != nil {
		suite.T().Fatal(err)
	}

}

func (suite *GenerateE2eTestSuite) SetupTest() {
}

func (suite *GenerateE2eTestSuite) TestGenerate() {
	asPolicy := aero.NewClientPolicy()
	host := aero.NewHost(IP, PORT_START)
	asPolicy.User = "admin"
	asPolicy.Password = "admin"

	asinfo := info.NewAsInfo(logr.Discard(), host, asPolicy)

	genConf, err := asconfig.GenerateConf(logr.Discard(), asinfo, true)
	suite.Assert().Nil(err)
	genConfWithDefaults, err := asconfig.GenerateConf(logr.Discard(), asinfo, false)
	suite.Assert().Nil(err)

	asconf, err := asconfig.NewMapAsConfig(logr.Discard(), genConf.Version, genConf.Conf)
	suite.Assert().Nil(err)
	asconfWithDefaults, err := asconfig.NewMapAsConfig(logr.Discard(), genConfWithDefaults.Version, genConfWithDefaults.Conf)
	suite.Assert().Nil(err)

	RestartAerospikeContainer(GetAerospikeContainerName(0), asconf.ToConfFile())

	asinfo2 := info.NewAsInfo(logr.Discard(), host, asPolicy)

	genConf2, err := asconfig.GenerateConf(logr.Discard(), asinfo2, true)
	suite.Assert().Nil(err)
	genConfWithDefaults2, err := asconfig.GenerateConf(logr.Discard(), asinfo2, false)
	suite.Assert().Nil(err)

	asconf2, err := asconfig.NewMapAsConfig(logr.Discard(), genConf2.Version, genConf2.Conf)
	suite.Assert().Nil(err)
	asconfWithDefaults2, err := asconfig.NewMapAsConfig(logr.Discard(), genConfWithDefaults2.Version, genConfWithDefaults2.Conf)
	suite.Assert().Nil(err)

	suite.Assert().Equal(asconf, asconf2)
	suite.Assert().Equal(asconfWithDefaults, asconfWithDefaults2)
}

func TestGenerateE2ETestSuiteSuite(t *testing.T) {
	suite.Run(t, new(GenerateE2eTestSuite))
}
