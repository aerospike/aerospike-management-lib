package asconfig

import (
	"log"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AsConfigTestSuite struct {
	suite.Suite
	mockGetter *MockConfGetter
	ctrl       *gomock.Controller
}

func (s *AsConfigTestSuite) SetupSuite() {
	schemaDir := os.Getenv("TEST_SCHEMA_DIR")
	if schemaDir == "" {
		log.Printf("Env var TEST_SCHEMA_DIR must be set.")
		s.T().Fail()
	}

	err := Init(logr.Discard(), schemaDir)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *AsConfigTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockGetter = NewMockConfGetter(s.ctrl)
}

func (s *AsConfigTestSuite) TestAsConfigGetFlatMap() {
	testCases := []struct {
		name     string
		inputMap map[string]interface{}
		expected *Conf
	}{
		{
			"namespace context",
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
				"namespaces.{test}.<index>":             0,
				"namespaces.{test}.name":                "test",
				"namespaces.{test}.storage-engine.type": "memory",
				"namespaces.{bar}.<index>":              1,
				"namespaces.{bar}.name":                 "bar",
				"namespaces.{bar}.storage-engine.type":  "memory",
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()

			asConfig, err := NewMapAsConfig(logger, tc.inputMap)
			actual := asConfig.GetFlatMap()

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, actual)
		})
	}
}

func (s *AsConfigTestSuite) TestAsConfigGetDiff() {
	testCases := []struct {
		name       string
		inputConf1 map[string]interface{}
		inputConf2 map[string]interface{}
		expected   DynamicConfigMap
	}{
		{
			"General differences",
			map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{
						"name":               "test",
						"replication-factor": 3,
						"storage-engine": map[string]interface{}{
							"type": "device",
						},
					},
					{
						"name": "bar",
						"storage-engine": map[string]interface{}{
							"type": "memory",
						},
					},
				},
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name":               "DC1",
							"node-address-ports": "1.1.1.1 3000",
							"namespaces": []map[string]interface{}{
								{
									"name":        "ns1",
									"bin-policy":  "no-bins",
									"ignore-sets": []string{"set1"},
								},
							},
						},
					},
				},
				"security": map[string]interface{}{
					"log": map[string]interface{}{
						"report-data-op": []string{"ns1 set1", "ns3 set2"},
					},
				},
			},
			map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{
						"name": "test",
						"storage-engine": map[string]interface{}{
							"type": "memory",
						},
					},
					{
						"name":               "bar",
						"replication-factor": 3,
						"storage-engine": map[string]interface{}{
							"type": "memory",
						},
					},
				},
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name":               "DC3",
							"node-address-ports": "1.1.1.1 3000",
							"namespaces": []map[string]interface{}{
								{
									"name":       "ns1",
									"bin-policy": "all",
								},
							},
						},
					},
				},
				"security": map[string]interface{}{
					"log": map[string]interface{}{
						"report-data-op": []string{"ns1 set1", "ns2 set2"},
					},
				},
			},

			DynamicConfigMap{
				"namespaces.{test}.storage-engine.type":     {Update: "device"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {Update: "no-bins"},
				"security.log.report-data-op": {Add: []string{"ns3 set2"},
					Remove: []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                         {Remove: "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":           {Add: []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":        {Add: "ns1"},
				"xdr.dcs.{DC1}.name":                         {Add: "DC1"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.ignore-sets": {Add: []string{"set1"}},
				"namespaces.{test}.replication-factor":       {Update: 3},
				"namespaces.{bar}.replication-factor":        {Update: uint64(2)},
			},
		},
		{
			"Partial Context differences: adding",
			map[string]interface{}{
				"security": map[string]interface{}{
					"log": map[string]interface{}{
						"report-data-op": []string{"ns1 set1", "ns3 set2"},
					},
				},
			},
			map[string]interface{}{
				"security": map[string]interface{}{},
			},

			DynamicConfigMap{
				"security.log.report-data-op": {Add: []string{"ns1 set1", "ns3 set2"}},
			},
		},

		{
			"Partial Context differences: removing",
			map[string]interface{}{
				"security": map[string]interface{}{},
			},
			map[string]interface{}{
				"security": map[string]interface{}{
					"log": map[string]interface{}{
						"report-authentication": true,
						"report-data-op":        []string{"ns1 set1", "ns3 set2"},
					},
				},
			},
			DynamicConfigMap{
				"security.log.report-authentication": {Update: false},
				"security.log.report-data-op":        {Remove: []string{"ns1 set1", "ns3 set2"}},
			},
		},

		{
			"DCS field differences",
			map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name":               "DC3",
							"node-address-ports": []string{"1.1.1.1 3000"},
							"namespaces": []map[string]interface{}{
								{
									"name":       "ns1",
									"bin-policy": "all",
								},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name":               "DC3",
							"node-address-ports": []string{"1.1.2.1 3000"},
							"namespaces": []map[string]interface{}{
								{
									"name":       "ns1",
									"bin-policy": "no-bins",
								},
							},
						},
					},
				},
			},
			DynamicConfigMap{
				"xdr.dcs.{DC3}.node-address-ports": {Add: []string{"1.1.1.1 3000"},
					Remove: []string{"1.1.2.1 3000"}},
				"xdr.dcs.{DC3}.namespaces.{ns1}.bin-policy": {Update: "all"},
			},
		},
		{
			"DCS Namespace add/remove field differences",
			map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name": "DC3",
							"namespaces": []map[string]interface{}{
								{
									"name":       "ns1",
									"bin-policy": "all",
								},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name": "DC3",
							"namespaces": []map[string]interface{}{
								{
									"name":       "ns2",
									"bin-policy": "all",
								},
							},
						},
					},
				},
			},
			DynamicConfigMap{
				"xdr.dcs.{DC3}.namespaces.{ns1}.name":       {Add: "ns1"},
				"xdr.dcs.{DC3}.namespaces.{ns2}.name":       {Remove: "ns2"},
				"xdr.dcs.{DC3}.namespaces.{ns1}.bin-policy": {Update: "all"},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()

			asConfig1, err := NewMapAsConfig(logger, tc.inputConf1)
			s.Assert().Nil(err)
			asConfig2, err := NewMapAsConfig(logger, tc.inputConf2)
			s.Assert().Nil(err)
			diff, err := ConfDiff(logger, *asConfig1.baseConf, *asConfig2.baseConf, false, "7.0.0")

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, diff)
		})
	}
}

func (s *AsConfigTestSuite) TestAsConfigIsDynamic() {
	testCases := []struct {
		name      string
		inputConf DynamicConfigMap
		expected  bool
	}{
		{
			"static fields",
			DynamicConfigMap{
				"namespaces.{test}.storage-engine.type":     {Update: "device"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {Update: "no-bins"},
				"security.log.report-data-op": {Add: []string{"ns3 set2"},
					Remove: []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                  {Remove: "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":    {Update: []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name": {Add: "ns1"},
				"xdr.dcs.{DC1}.name":                  {Add: "DC1"},
			},

			false,
		},
		{
			"dynamic fields",
			DynamicConfigMap{
				"security.log.report-data-op": {Add: []string{"ns3 set2"},
					Remove: []string{"ns2 set2"}},
				"service.proto-fd-max": {Update: "1000"},
			},

			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()

			isDynamic, err := IsAllDynamicConfig(logger, tc.inputConf, "7.0.0")

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, isDynamic)
		})
	}
}

func (s *AsConfigTestSuite) TestAsConfigGetExpandMap() {
	testCases := []struct {
		name     string
		inputMap map[string]interface{}
		expected Conf
	}{
		{
			"namespace context",
			map[string]interface{}{
				"logging": []Conf{
					{
						"name":         "/var/log/aerospike/aerospike.log",
						"aggr":         "INFO",
						"alloc":        "INFO",
						"appeal":       "INFO",
						"arenax":       "INFO",
						"as":           "INFO",
						"audit":        "INFO",
						"batch":        "INFO",
						"batch-sub":    "INFO",
						"bin":          "INFO",
						"clustering":   "INFO",
						"config":       "INFO",
						"drv_pmem":     "INFO",
						"drv_ssd":      "INFO",
						"exchange":     "INFO",
						"exp":          "INFO",
						"fabric":       "INFO",
						"flat":         "INFO",
						"geo":          "INFO",
						"hardware":     "INFO",
						"hb":           "INFO",
						"health":       "INFO",
						"hlc":          "INFO",
						"index":        "INFO",
						"info":         "INFO",
						"info-port":    "INFO",
						"key-busy":     "INFO",
						"migrate":      "INFO",
						"misc":         "INFO",
						"msg":          "INFO",
						"namespace":    "INFO",
						"nsup":         "INFO",
						"os":           "INFO",
						"particle":     "INFO",
						"partition":    "INFO",
						"proto":        "INFO",
						"proxy":        "INFO",
						"proxy-divert": "INFO",
						"query":        "INFO",
						"record":       "INFO",
						"roster":       "INFO",
						"rw":           "INFO",
						"rw-client":    "INFO",
						"secrets":      "INFO",
						"security":     "INFO",
						"service":      "INFO",
						"service-list": "INFO",
						"sindex":       "INFO",
						"skew":         "INFO",
						"smd":          "INFO",
						"socket":       "INFO",
						"storage":      "INFO",
						"tls":          "INFO",
						"truncate":     "INFO",
						"tsvc":         "INFO",
						"udf":          "INFO",
						"vault":        "INFO",
						"vmapx":        "INFO",
						"xdr":          "INFO",
						"xdr-client":   "INFO",
						"xmem":         "INFO",
					},
					{
						"name":         "console",
						"aggr":         "INFO",
						"alloc":        "INFO",
						"appeal":       "INFO",
						"arenax":       "INFO",
						"as":           "INFO",
						"audit":        "INFO",
						"batch":        "INFO",
						"batch-sub":    "INFO",
						"bin":          "INFO",
						"clustering":   "INFO",
						"config":       "INFO",
						"drv_pmem":     "INFO",
						"drv_ssd":      "INFO",
						"exchange":     "INFO",
						"exp":          "INFO",
						"fabric":       "INFO",
						"flat":         "INFO",
						"geo":          "INFO",
						"hardware":     "INFO",
						"hb":           "INFO",
						"health":       "INFO",
						"hlc":          "INFO",
						"index":        "INFO",
						"info":         "INFO",
						"info-port":    "INFO",
						"key-busy":     "INFO",
						"migrate":      "INFO",
						"misc":         "INFO",
						"msg":          "INFO",
						"namespace":    "INFO",
						"nsup":         "INFO",
						"os":           "INFO",
						"particle":     "INFO",
						"partition":    "INFO",
						"proto":        "INFO",
						"proxy":        "INFO",
						"proxy-divert": "INFO",
						"query":        "INFO",
						"record":       "INFO",
						"roster":       "INFO",
						"rw":           "INFO",
						"rw-client":    "INFO",
						"secrets":      "INFO",
						"security":     "INFO",
						"service":      "INFO",
						"service-list": "INFO",
						"sindex":       "INFO",
						"skew":         "INFO",
						"smd":          "INFO",
						"socket":       "INFO",
						"storage":      "INFO",
						"tls":          "INFO",
						"truncate":     "INFO",
						"tsvc":         "INFO",
						"udf":          "INFO",
						"vault":        "INFO",
						"vmapx":        "INFO",
						"xdr":          "INFO",
						"xdr-client":   "INFO",
						"xmem":         "INFO",
					},
				},
			},
			Conf{
				"logging": []Conf{
					{
						"name":         "/var/log/aerospike/aerospike.log",
						"aggr":         "INFO",
						"alloc":        "INFO",
						"appeal":       "INFO",
						"arenax":       "INFO",
						"as":           "INFO",
						"audit":        "INFO",
						"batch":        "INFO",
						"batch-sub":    "INFO",
						"bin":          "INFO",
						"clustering":   "INFO",
						"config":       "INFO",
						"drv_pmem":     "INFO",
						"drv_ssd":      "INFO",
						"exchange":     "INFO",
						"exp":          "INFO",
						"fabric":       "INFO",
						"flat":         "INFO",
						"geo":          "INFO",
						"hardware":     "INFO",
						"hb":           "INFO",
						"health":       "INFO",
						"hlc":          "INFO",
						"index":        "INFO",
						"info":         "INFO",
						"info-port":    "INFO",
						"key-busy":     "INFO",
						"migrate":      "INFO",
						"misc":         "INFO",
						"msg":          "INFO",
						"namespace":    "INFO",
						"nsup":         "INFO",
						"os":           "INFO",
						"particle":     "INFO",
						"partition":    "INFO",
						"proto":        "INFO",
						"proxy":        "INFO",
						"proxy-divert": "INFO",
						"query":        "INFO",
						"record":       "INFO",
						"roster":       "INFO",
						"rw":           "INFO",
						"rw-client":    "INFO",
						"secrets":      "INFO",
						"security":     "INFO",
						"service":      "INFO",
						"service-list": "INFO",
						"sindex":       "INFO",
						"skew":         "INFO",
						"smd":          "INFO",
						"socket":       "INFO",
						"storage":      "INFO",
						"tls":          "INFO",
						"truncate":     "INFO",
						"tsvc":         "INFO",
						"udf":          "INFO",
						"vault":        "INFO",
						"vmapx":        "INFO",
						"xdr":          "INFO",
						"xdr-client":   "INFO",
						"xmem":         "INFO",
					},
					{
						"name":         "console",
						"aggr":         "INFO",
						"alloc":        "INFO",
						"appeal":       "INFO",
						"arenax":       "INFO",
						"as":           "INFO",
						"audit":        "INFO",
						"batch":        "INFO",
						"batch-sub":    "INFO",
						"bin":          "INFO",
						"clustering":   "INFO",
						"config":       "INFO",
						"drv_pmem":     "INFO",
						"drv_ssd":      "INFO",
						"exchange":     "INFO",
						"exp":          "INFO",
						"fabric":       "INFO",
						"flat":         "INFO",
						"geo":          "INFO",
						"hardware":     "INFO",
						"hb":           "INFO",
						"health":       "INFO",
						"hlc":          "INFO",
						"index":        "INFO",
						"info":         "INFO",
						"info-port":    "INFO",
						"key-busy":     "INFO",
						"migrate":      "INFO",
						"misc":         "INFO",
						"msg":          "INFO",
						"namespace":    "INFO",
						"nsup":         "INFO",
						"os":           "INFO",
						"particle":     "INFO",
						"partition":    "INFO",
						"proto":        "INFO",
						"proxy":        "INFO",
						"proxy-divert": "INFO",
						"query":        "INFO",
						"record":       "INFO",
						"roster":       "INFO",
						"rw":           "INFO",
						"rw-client":    "INFO",
						"secrets":      "INFO",
						"security":     "INFO",
						"service":      "INFO",
						"service-list": "INFO",
						"sindex":       "INFO",
						"skew":         "INFO",
						"smd":          "INFO",
						"socket":       "INFO",
						"storage":      "INFO",
						"tls":          "INFO",
						"truncate":     "INFO",
						"tsvc":         "INFO",
						"udf":          "INFO",
						"vault":        "INFO",
						"vmapx":        "INFO",
						"xdr":          "INFO",
						"xdr-client":   "INFO",
						"xmem":         "INFO",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()

			asConfig, err := NewMapAsConfig(logger, tc.inputMap)
			actual := asConfig.ToMap()

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, *actual)
		})
	}
}

func TestAsConfigTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(AsConfigTestSuite))
}
