package asconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v8"
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
				"xdr.dcs.{DC1}.node-address-ports":          {Add: []string{"1.1.1.1:3000:tls-name"}},
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

func (s *AsSetConfigTestSuite) TestCreateSetConfigCmdListForNamespaceParamByBuild() {
	// Namespace set-config: build >= 7.2 uses "namespace=", pre-7.2 uses legacy "id=".
	inputConf := DynamicConfigMap{
		"namespaces.{test}.replication-factor": {Update: "2"},
	}
	logger := logr.Discard()
	policy := &aero.ClientPolicy{}

	tests := []struct {
		name         string
		buildVersion string
		wantContains string
		msg          string
	}{
		{"build 7.2+ uses namespace=",
			"7.2.0.0",
			"set-config:context=namespace;namespace=test",
			"build 7.2+ must use namespace= parameter, got: %s",
		},
		{"pre-7.2 uses id=",
			"7.1.0.0",
			"set-config:context=namespace;id=test",
			"pre-7.2 must use id= parameter, got: %s",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := CreateSetConfigCmdListWithBuildVersion(logger, inputConf, nil, policy, tt.buildVersion)
			s.Require().NoError(err)
			s.Require().Len(result, 1)
			s.Require().Contains(result[0], tt.wantContains, tt.msg, result[0])
		})
	}
}

func (s *AsSetConfigTestSuite) TestNamespaceSetConfigCommandList() {
	testCases := []struct {
		name  string
		build string
		want  string
	}{
		{
			name:  "empty build returns legacy id prefix",
			build: "",
			want:  "set-config:context=namespace;id=",
		},
		{
			name:  "pre-7.2 uses id",
			build: "7.0.0.0",
			want:  "set-config:context=namespace;id=",
		},
		{
			name:  "7.2 uses namespace",
			build: "7.2.0.0",
			want:  "set-config:context=namespace;namespace=",
		},
		{
			name:  "8.0 uses namespace",
			build: "8.0.0.0",
			want:  "set-config:context=namespace;namespace=",
		},
		{
			name:  "invalid build falls back to id",
			build: "not-a-version",
			want:  "set-config:context=namespace;id=",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			got := namespaceSetConfigCmd(tc.build)
			s.Equal(tc.want, got)
		})
	}
}

func TestAsSetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(AsSetConfigTestSuite))
}
