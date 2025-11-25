package test

import (
	"log"
	"os"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-management-lib/asconfig"
	"github.com/aerospike/aerospike-management-lib/info"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
)

type GenerateE2eTestSuite struct {
	suite.Suite
}

func (s *GenerateE2eTestSuite) SetupSuite() {
	err := Start(1)
	if err != nil {
		s.T().Fatal(err)
	}

	schemaDir := os.Getenv("TEST_SCHEMA_DIR")
	if schemaDir == "" {
		log.Printf("Env var TEST_SCHEMA_DIR must be set.")
		s.T().Fail()
	}

	err = asconfig.Init(logr.Discard(), schemaDir)
	if err != nil {
		s.T().Fatal(err)
	}
}

// Uncomment this function to check server logs after failure
func (s *GenerateE2eTestSuite) TearDownSuite() {
	err := Stop()
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *GenerateE2eTestSuite) TestGenerate() {
	asPolicy := aero.NewClientPolicy()
	host := aero.NewHost(IP, PortStart)
	asPolicy.User = "admin"
	asPolicy.Password = "admin"

	asinfo := info.NewAsInfo(logr.Discard(), host, asPolicy)

	genConf, err := asconfig.GenerateConf(logr.Discard(), asinfo, true)
	s.Assert().Nil(err)
	genConfWithDefaults, err := asconfig.GenerateConf(logr.Discard(), asinfo, false)
	s.Assert().Nil(err)

	asconf, err := asconfig.NewMapAsConfig(logr.Discard(), genConf.Conf)
	s.Assert().Nil(err)
	asconfWithDefaults, err := asconfig.NewMapAsConfig(
		logr.Discard(),
		genConfWithDefaults.Conf,
	)
	s.Assert().Nil(err)

	err = RestartAerospikeContainer(GetAerospikeContainerName(0), asconf.ToConfFile())
	s.Assert().Nil(err)

	asinfo2 := info.NewAsInfo(logr.Discard(), host, asPolicy)

	genConf2, err := asconfig.GenerateConf(logr.Discard(), asinfo2, true)
	s.Assert().Nil(err)
	genConfWithDefaults2, err := asconfig.GenerateConf(logr.Discard(), asinfo2, false)
	s.Assert().Nil(err)

	asconf2, err := asconfig.NewMapAsConfig(logr.Discard(), genConf2.Conf)
	s.Assert().Nil(err)
	asconfWithDefaults2, err := asconfig.NewMapAsConfig(
		logr.Discard(),
		genConfWithDefaults2.Conf,
	)
	s.Assert().Nil(err)

	s.Assert().Equal(asconf, asconf2)
	s.Assert().Equal(asconfWithDefaults, asconfWithDefaults2)
}

func TestGenerateE2ETestSuiteSuite(t *testing.T) {
	suite.Run(t, new(GenerateE2eTestSuite))
}
