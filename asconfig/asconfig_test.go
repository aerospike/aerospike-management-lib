package asconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type AsConfigTestSuite struct {
	suite.Suite
	mockGetter *MockConfGetter
	ctrl       *gomock.Controller
}

func (suite *AsConfigTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockGetter = NewMockConfGetter(suite.ctrl)
}

func (suite *AsConfigTestSuite) TestAsConfigGetFlatMap() {
	testCases := []struct {
		name     string
		version  string
		inputMap map[string]interface{}
		expected *Conf
	}{
		{
			"namespace context",
			"7.0.0",
			map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{
						"name": "test",
						"storage-engine": map[string]interface{}{
							"type": "memory",
						},
					},
					{
						"name": "bar",
						"storage-engine": map[string]interface{}{
							"type": "memory",
						},
					},
				},
			},
			&Conf{
				"namespaces.{test}.index":               0,
				"namespaces.{test}.name":                "test",
				"namespaces.{test}.storage-engine.type": "memory",
				"namespaces.{bar}.index":                1,
				"namespaces.{bar}.name":                 "bar",
				"namespaces.{bar}.storage-engine.type":  "memory",
			},
		},
		{
			"xdr 4.9 context",
			"4.9.0",
			map[string]interface{}{
				"xdr": map[string]interface{}{
					"datacenters": []map[string]interface{}{
						{
							"name":                 "DC1",
							"dc-node-address-port": "1.1.1.1:3000",
							"dc-int-ext-ipmap":     []string{"1.1.1.1 2.2.2.2", "3.3.3.3 4.4.4.4"},
						},
					},
				},
			},
			&Conf{
				"xdr.datacenters.{DC1}.index":                0,
				"xdr.datacenters.{DC1}.name":                 "DC1",
				"xdr.datacenters.{DC1}.dc-node-address-port": []string{"1.1.1.1:3000"},
				"xdr.datacenters.{DC1}.dc-int-ext-ipmap":     []string{"1.1.1.1 2.2.2.2", "3.3.3.3 4.4.4.4"},
			},
		},
	}

	Init(logr.Discard(), "/Users/jesseschmidt/Developer/aerospike-admin/lib/live_cluster/client/config-schemas") // TODO: replace with better location. Maybe a single test schema

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			logger := logr.Discard()

			asConfig, err := NewMapAsConfig(logger, tc.version, tc.inputMap)
			actual := asConfig.GetFlatMap()

			suite.Assert().Nil(err)
			suite.Assert().Equal(tc.expected, actual)
		})
	}
}

func TestAsConfigTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(AsConfigTestSuite))
}
