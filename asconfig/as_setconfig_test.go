package asconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v7"
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

func (s *AsSetConfigTestSuite) TestCreateSetConfigCmdList() {
	testCases := []struct {
		name      string
		inputConf DynamicConfigMap
		expected  []string
	}{
		{
			"commands",
			DynamicConfigMap{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {Update: "no-bins"},
				"security.log.report-data-op":               {Add: []string{"ns3:set2"}, "remove": []string{"ns2:set2"}},
				"xdr.dcs.{DC3}.name":                        {Remove: "DC3"},
				"xdr.dcs.{DC1}.node-address-ports": {
					"remove": []string{"1.1.1.1:3000"},
				},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":        {Add: "ns1"},
				"xdr.dcs.{DC1}.name":                         {Add: "DC1"},
				"security.privilege-refresh-period":          {Update: "100"},
				"xdr.src-id":                                 {Update: "10"},
				"logging.{console}.any":                      {Update: "info"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.ignore-sets": {Add: []string{"set1"}},
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
			policy := &aero.ClientPolicy{}

			s.mockASConn.EXPECT().RunInfo(gomock.Any(), gomock.Any()).Return(map[string]string{
				"logs": "0:stderr"}, nil).AnyTimes()
			result, err := CreateSetConfigCmdList(tc.inputConf, s.mockASConn, policy)

			s.Assert().Nil(err)
			s.Assert().True(gomock.InAnyOrder(result).Matches(tc.expected))
		})
	}
}

/*
func (s *AsSetConfigTestSuite) TestCreateSetConfigCmdListOrdered() {
	testCases := []struct {
		name      string
		inputConf DynamicConfigMap
		expected  []string
	}{
		{
			"commands",
			DynamicConfigMap{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {Update: "no-bins"},
				"xdr.dcs.{DC3}.name":                        {Add: "DC3"},
				"xdr.dcs.{DC1}.namespaces.{ns2}.name":       {Remove: "ns2"},
				"xdr.dcs.{DC1}.node-address-ports":          {Add: []string{"1.1.1.1:3000:tls-name"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {Add: "ns1"},
				"xdr.dcs.{DC1}.tls-name":                    {Update: "tls-name"},
			},
			[]string{"set-config:context=xdr;dc=DC1;namespace=ns2;action=remove",
				"set-config:context=xdr;dc=DC3;action=create",
				"set-config:context=xdr;dc=DC1;tls-name=tls-name",
				"set-config:context=xdr;dc=DC1;node-address-port=1.1.1.1:3000:tls-name;action=add",
				//	"set-config:context=xdr;dc=DC1;node-address-port=1.1.1.1:3000;action=remove",
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
*/

func TestAsSetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(AsSetConfigTestSuite))
}
