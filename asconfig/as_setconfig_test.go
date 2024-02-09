package asconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-management-lib/commons"
	"github.com/aerospike/aerospike-management-lib/deployment"
)

type AsSetConfigTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockASConn *deployment.MockASConnInterface
}

func (s *AsSetConfigTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	mockASConn := deployment.NewMockASConnInterface(s.ctrl)
	s.mockASConn = mockASConn
}

func (s *AsSetConfigTestSuite) TestDeploymentSetConfig() {
	testCases := []struct {
		name      string
		inputConf commons.DynamicConfigMap
		expected  []string
	}{
		{
			"commands",
			commons.DynamicConfigMap{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {commons.Update: "no-bins"},
				"security.log.report-data-op":               {commons.Add: []string{"ns3:set2"}, "remove": []string{"ns2:set2"}},
				"xdr.dcs.{DC3}.name":                        {commons.Remove: "DC3"},
				"xdr.dcs.{DC1}.node-address-ports": {
					"remove": []string{"1.1.1.1:3000"},
				},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":        {commons.Add: "ns1"},
				"xdr.dcs.{DC1}.name":                         {commons.Add: "DC1"},
				"security.privilege-refresh-period":          {commons.Update: "100"},
				"xdr.src-id":                                 {commons.Update: "10"},
				"logging.{console}.any":                      {commons.Update: "info"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.ignore-sets": {commons.Add: []string{"set1"}},
			},
			[]string{"set-config:context=xdr;dc=DC1;action=create",
				"set-config:context=xdr;dc=DC1;node-address-port=1.1.1.1:3000;action=remove",
				"set-config:context=xdr;dc=DC1;namespace=ns1;action=add",
				"set-config:context=xdr;dc=DC1;namespace=ns1;bin-policy=no-bins",
				"set-config:context=security;log.report-data-op=true;namespace=ns3;set=set2",
				"set-config:context=security;log.report-data-op=false;namespace=ns2;set=set2",
				"set-config:context=xdr;dc=DC3;action=delete",
				"set-config:context=xdr;src-id=10",
				"log-set:id=0;any=info",
				"set-config:context=security;privilege-refresh-period=100",
				"set-config:context=xdr;dc=DC1;namespace=ns1;ignore-set=set1",
			},
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()
			policy := &aero.ClientPolicy{}

			s.mockASConn.EXPECT().RunInfo(gomock.Any(), gomock.Any()).Return(map[string]string{
				"logs": "0:stderr"}, nil).AnyTimes()
			result, err := CreateSetConfigCmdList(logger, tc.inputConf, s.mockASConn, policy)

			s.Assert().Nil(err)
			s.Assert().True(gomock.InAnyOrder(result).Matches(tc.expected))
		})
	}
}

func (s *AsSetConfigTestSuite) TestDeploymentSetConfigOrdered() {
	testCases := []struct {
		name      string
		inputConf commons.DynamicConfigMap
		expected  []string
	}{
		{
			"commands",
			commons.DynamicConfigMap{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {commons.Update: "no-bins"},
				"xdr.dcs.{DC3}.name":                        {commons.Add: "DC3"},
				"xdr.dcs.{DC1}.namespaces.{ns2}.name":       {commons.Remove: "ns2"},
				"xdr.dcs.{DC1}.node-address-ports":          {commons.Add: []string{"1.1.1.1:3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {commons.Add: "ns1"},
			},
			[]string{"set-config:context=xdr;dc=DC1;namespace=ns2;action=remove",
				"set-config:context=xdr;dc=DC3;action=create",
				"set-config:context=xdr;dc=DC1;node-address-port=1.1.1.1:3000;action=add",
				"set-config:context=xdr;dc=DC1;namespace=ns1;action=add",
				"set-config:context=xdr;dc=DC1;namespace=ns1;bin-policy=no-bins",
			},
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()
			policy := &aero.ClientPolicy{}
			result, err := CreateSetConfigCmdList(logger, tc.inputConf, nil, policy)

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func TestAsSetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(AsSetConfigTestSuite))
}
