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
		{
			"xdr 4.9 context",
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
				"xdr.datacenters.{DC1}.<index>":              0,
				"xdr.datacenters.{DC1}.name":                 "DC1",
				"xdr.datacenters.{DC1}.dc-node-address-port": []string{"1.1.1.1:3000"},
				"xdr.datacenters.{DC1}.dc-int-ext-ipmap":     []string{"1.1.1.1 2.2.2.2", "3.3.3.3 4.4.4.4"},
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
		expected   map[string]map[string]interface{}
	}{
		{
			"namespace context",
			map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{
						"name": "test",
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
									"name":       "ns1",
									"bin-policy": "no-bins",
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
						"name": "bar",
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

			map[string]map[string]interface{}{
				"namespaces.{test}.storage-engine.type":     {"update": "device"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {"update": "no-bins"},
				"security.log.report-data-op":               {"add": []string{"ns3 set2"}, "remove": []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                        {"remove": "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":          {"update": []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {"add": "ns1"},
				"xdr.dcs.{DC1}.name":                        {"add": "DC1"},
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
		inputConf map[string]map[string]interface{}
		expected  bool
	}{
		{
			"static fields",
			map[string]map[string]interface{}{
				"namespaces.{test}.storage-engine.type":     {"update": "device"},
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {"update": "no-bins"},
				"security.log.report-data-op":               {"add": []string{"ns3 set2"}, "remove": []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                        {"remove": "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":          {"update": []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {"add": "ns1"},
				"xdr.dcs.{DC1}.name":                        {"add": "DC1"},
			},

			false,
		},
		{
			"dynamic fields",
			map[string]map[string]interface{}{
				"xdr.dcs.{DC1}.namespaces.{ns1}.bin-policy": {"update": "no-bins"},
				"security.log.report-data-op":               {"add": []string{"ns3 set2"}, "remove": []string{"ns2 set2"}},
				"xdr.dcs.{DC3}.name":                        {"remove": "DC3"},
				"xdr.dcs.{DC1}.node-address-ports":          {"update": []string{"1.1.1.1 3000"}},
				"xdr.dcs.{DC1}.namespaces.{ns1}.name":       {"add": "ns1"},
				"xdr.dcs.{DC1}.name":                        {"add": "DC1"},
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
