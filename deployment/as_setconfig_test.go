package deployment

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v6"
)

type AsSetConfigTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func (s *AsSetConfigTestSuite) TestDeploymentSetConfig() {
	testCases := []struct {
		name      string
		inputConf map[string]map[string]interface{}
		expected  []string
	}{
		{
			"commands",
			map[string]map[string]interface{}{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {"update": "no-bins"},
				"security.log.report-data-op":               {"add": []string{"ns3 set2"}, "remove": []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                        {"remove": "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":          {"remove": []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {"add": "ns1"},
				"xdr.dcs.{DC1}.name":                        {"add": "DC1"},
			},
			[]string{"set-config:context=xdr;dc=DC1;action=create",
				"set-config:context=xdr;dc=DC1;node-address-port=1.1.1.1:3000;action=remove",
				"set-config:context=xdr;dc=DC1;namespace=ns1;action=add",
				"set-config:context=xdr;dc=DC1;namespace=ns1;bin-policy=no-bins",
				"set-config:context=security;log.report-data-op=true;namespace=ns3;set=set2",
				"set-config:context=security;log.report-data-op=false;namespace=ns2;set=set2",
				"set-config:context=xdr;dc=DC3;action=delete"},
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()
			policy := &aero.ClientPolicy{}
			result, err := CreateConfigSetCmdList(logger, tc.inputConf, nil, policy)

			s.Assert().Nil(err)
			s.Assert().True(gomock.InAnyOrder(result).Matches(tc.expected))
		})
	}
}

func (s *AsSetConfigTestSuite) TestDeploymentSetConfigOrdered() {
	testCases := []struct {
		name      string
		inputConf map[string]map[string]interface{}
		expected  []string
	}{
		{
			"commands",
			map[string]map[string]interface{}{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {"update": "no-bins"},
				"xdr.dcs.{DC1}.name":                        {"add": "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":          {"add": []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {"add": "ns1"},
			},
			[]string{"set-config:context=xdr;dc=DC1;action=create",
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
			result, err := CreateConfigSetCmdList(logger, tc.inputConf, nil, policy)

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func TestAsSetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(AsSetConfigTestSuite))
}
