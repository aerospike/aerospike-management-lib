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
		name        string
		inputMap    map[string]interface{}
		expected    *Conf    // nil for error cases
		errContains []string // non-empty when an error is expected
	}{
		// --- success cases: verify the resulting flat map ---
		{
			name: "namespace context",
			inputMap: map[string]interface{}{
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
			expected: &Conf{
				"namespaces.{test}.<index>":             0,
				"namespaces.{test}.name":                "test",
				"namespaces.{test}.storage-engine.type": "memory",
				"namespaces.{bar}.<index>":              1,
				"namespaces.{bar}.name":                 "bar",
				"namespaces.{bar}.storage-engine.type":  "memory",
			},
		},
		{
			name: "valid config with unique names across all list sections",
			inputMap: map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{"name": "ns1", "replication-factor": 2},
					{"name": "ns2", "replication-factor": 3},
				},
				"network": map[string]interface{}{
					"tls": []map[string]interface{}{
						{"name": "tls-fabric"},
						{"name": "tls-service"},
					},
				},
			},
		},

		// --- error cases: duplicate names in list sections ---
		{
			name: "duplicate namespace names",
			inputMap: map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{"name": "test", "replication-factor": 2},
					{"name": "test", "replication-factor": 3},
				},
			},
			errContains: []string{"test", "namespaces"},
		},
		{
			// namespaces.sets is an array-of-object keyed by "name" in 8.x schema.
			name: "duplicate set names within a namespace",
			inputMap: map[string]interface{}{
				"namespaces": []map[string]interface{}{
					{
						"name": "test",
						"sets": []map[string]interface{}{
							{"name": "setA", "disable-eviction": true},
							{"name": "setA", "disable-eviction": false},
						},
					},
				},
			},
			errContains: []string{"setA", "sets"},
		},
		{
			name: "duplicate TLS names",
			inputMap: map[string]interface{}{
				"network": map[string]interface{}{
					"tls": []map[string]interface{}{
						{"name": "abc", "cert-file": "/etc/certs/cert.pem"},
						{"name": "abc", "cert-file": "/etc/certs/cert2.pem"},
					},
				},
			},
			errContains: []string{"network:", "abc", "tls"},
		},
		{
			name: "duplicate XDR DC names",
			inputMap: map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{"name": "DC1", "node-address-ports": "1.1.1.1 3000"},
						{"name": "DC1", "node-address-ports": "2.2.2.2 3000"},
					},
				},
			},
			errContains: []string{"xdr:", "DC1", "dcs"},
		},
		{
			name: "duplicate logging sink names",
			inputMap: map[string]interface{}{
				"logging": []map[string]interface{}{
					{"name": "/var/log/aerospike/aerospike.log"},
					{"name": "/var/log/aerospike/aerospike.log"},
				},
			},
			errContains: []string{"/var/log/aerospike/aerospike.log", "logging"},
		},
		{
			name: "duplicate namespace names within an XDR DC",
			inputMap: map[string]interface{}{
				"xdr": map[string]interface{}{
					"dcs": []map[string]interface{}{
						{
							"name": "DC1",
							"namespaces": []map[string]interface{}{
								{"name": "ns1", "bin-policy": "all"},
								{"name": "ns1", "bin-policy": "no-bins"},
							},
						},
					},
				},
			},
			errContains: []string{"xdr:", "ns1", "namespaces"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			logger := logr.Discard()

			asConfig, err := NewMapAsConfig(logger, tc.inputMap)

			if len(tc.errContains) > 0 {
				s.Assert().Error(err)

				for _, substr := range tc.errContains {
					s.Assert().ErrorContains(err, substr)
				}

				s.Assert().Nil(asConfig)
			} else {
				s.Assert().NoError(err)

				if tc.expected != nil {
					s.Assert().Equal(tc.expected, asConfig.GetFlatMap())
				}
			}
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
		{
			// expandKey must assign a single-segment key (no dot) directly into
			// the parent Conf. If it mistakenly runs processKey on the single
			// key, processKey creates an intermediate Conf node, which causes
			// the value to be wrapped: input["k"] = Conf{"k": val} instead of
			// input["k"] = val.
			"single-segment top-level scalar key",
			map[string]interface{}{
				"proto-fd-max": 15000,
			},
			Conf{
				"proto-fd-max": 15000,
			},
		},
		{
			// SplitKey returns an empty slice for an empty string key, which
			// previously caused a panic at keys[len(keys)-1]. expandKey must
			// guard against this and silently skip the entry.
			"empty string key does not panic",
			map[string]interface{}{
				"": 42,
			},
			Conf{},
		},
		{
			// Verify that single-segment and multi-segment keys are both
			// expanded correctly when they appear in the same flat map.
			"mixed single-segment and multi-segment keys",
			map[string]interface{}{
				"proto-fd-max": 15000,
				"service": map[string]interface{}{
					"threads": 4,
				},
			},
			Conf{
				"proto-fd-max": 15000,
				"service": Conf{
					"threads": 4,
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
