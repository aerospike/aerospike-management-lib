package asconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type GenerateTestSuite struct {
	suite.Suite
	mockGetter *MockConfGetter
	// asinfo *AsInfo
	ctrl *gomock.Controller
}

func (suite *GenerateTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockGetter = NewMockConfGetter(suite.ctrl)
}

type GenerateTC struct {
	name           string
	removeDefaults bool
	allConfigs     Conf
	metadata       Conf
	expected       Conf
}

func convertIntToInt64(conf Conf) Conf {
	for key, value := range conf {
		switch v := value.(type) {
		case int:
			conf[key] = int64(v)
		case Conf:
			conf[key] = convertIntToInt64(v)
		case []Conf:
			for i, c := range v {
				v[i] = convertIntToInt64(c)
			}
		}
	}
	return conf
}

func (suite *GenerateTestSuite) TestGenerate() {
	testCases := []GenerateTC{
		loggingTC,
		namespaceTC,
		networkTC,
		serviceTC,
		securityEnabled57TC,
		securityDisabled57TC,
		securityEnabled56TC,
		securityDisabled56TC,
		xdr5TC,
		loggingDefaultsTC,
		namespacesDefaultsTC,
		networkDefaultsTC,
		serviceDefaultsTC,
		security57DefaultsTC,
		security57AllDefaultsTC,
		xdr5DefaultsTC,
	}

	InitFromMap(logr.Discard(), testSchemas)

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockGetter.EXPECT().AllConfigs().Return(convertIntToInt64(tc.allConfigs), nil)
			suite.mockGetter.EXPECT().GetAsInfo("metadata").Return(tc.metadata, nil)
			logger := logr.Discard()

			actual, err := GenerateConf(logger, suite.mockGetter, tc.removeDefaults)

			suite.Assert().Nil(err)
			suite.Assert().Equal(convertIntToInt64(tc.expected), actual)
		})
	}
}

func TestGenerateTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(GenerateTestSuite))
}

var loggingTC = GenerateTC{
	"logging",
	false,
	Conf{
		"logging": Conf{
			"/var/log/aerospike/aerospike.log": Conf{
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
			"stderr": Conf{
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
			"/dev/log": Conf{
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
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
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
			{
				"name":         "syslog",
				"path":         "/dev/log",
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
}

var namespaceTC = GenerateTC{
	"namespaces",
	false,
	Conf{
		"racks": []Conf{
			{
				"ns":     "test",
				"rack_1": "BB9030011AC4202",
				"rack_0": "BB9030011AC4203",
			},
			{
				"ns":     "bar",
				"rack_2": "BB9030011AC4202",
				"rack_0": "BB9030011AC4203",
			},
		},
		"namespaces": Conf{
			"bar": Conf{
				"allow-ttl-without-nsup":                 false,
				"background-query-max-rps":               10000,
				"conflict-resolution-policy":             "generation",
				"conflict-resolve-writes":                false,
				"default-ttl":                            0,
				"disable-cold-start-eviction":            false,
				"disable-write-dup-res":                  false,
				"disallow-expunge":                       false,
				"disallow-null-setname":                  false,
				"enable-benchmarks-batch-sub":            false,
				"enable-benchmarks-ops-sub":              false,
				"enable-benchmarks-read":                 false,
				"enable-benchmarks-udf":                  false,
				"enable-benchmarks-udf-sub":              false,
				"enable-benchmarks-write":                false,
				"enable-hist-proxy":                      false,
				"evict-hist-buckets":                     10000,
				"evict-tenths-pct":                       5,
				"force-long-queries":                     false,
				"geo2dsphere-within.earth-radius-meters": 6371000,
				"geo2dsphere-within.level-mod":           1,
				"geo2dsphere-within.max-cells":           12,
				"geo2dsphere-within.max-level":           20,
				"geo2dsphere-within.min-level":           1,
				"geo2dsphere-within.strict":              true,
				"high-water-disk-pct":                    0,
				"high-water-memory-pct":                  0,
				"ignore-migrate-fill-delay":              false,
				"index-stage-size":                       1073741824,
				"index-type":                             "shmem",
				"inline-short-queries":                   false,
				"max-record-size":                        0,
				"memory-size":                            1073741824,
				"migrate-order":                          5,
				"migrate-retransmit-ms":                  5000,
				"migrate-sleep":                          1,
				"nsup-hist-period":                       3600,
				"nsup-period":                            0,
				"nsup-threads":                           1,
				"partition-tree-sprigs":                  256,
				"prefer-uniform-balance":                 true,
				"rack-id":                                0,
				"read-consistency-level-override":        "off",
				"reject-non-xdr-writes":                  false,
				"reject-xdr-writes":                      false,
				"replication-factor":                     1,
				"sets": Conf{
					"testset": Conf{
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":                         1073741824,
				"sindex-type":                               "shmem",
				"single-query-threads":                      4,
				"stop-writes-pct":                           90,
				"stop-writes-sys-memory-pct":                90,
				"storage-engine":                            "device",
				"storage-engine.cache-replica-writes":       false,
				"storage-engine.cold-start-empty":           false,
				"storage-engine.commit-min-size":            0,
				"storage-engine.commit-to-device":           false,
				"storage-engine.compression":                "none",
				"storage-engine.compression-acceleration":   0,
				"storage-engine.compression-level":          0,
				"storage-engine.data-in-memory":             false,
				"storage-engine.defrag-lwm-pct":             50,
				"storage-engine.defrag-queue-min":           0,
				"storage-engine.defrag-sleep":               1000,
				"storage-engine.defrag-startup-minimum":     0,
				"storage-engine.direct-files":               false,
				"storage-engine.disable-odsync":             false,
				"storage-engine.enable-benchmarks-storage":  false,
				"storage-engine.encryption-key-file":        "null",
				"storage-engine.encryption-old-key-file":    "null",
				"storage-engine.file[0]":                    "/opt/aerospike/data/bar.dat",
				"storage-engine.file[0].shadow":             "/opt/aerospike/data/bar-shadow.dat",
				"storage-engine.file[1]":                    "/opt/aerospike/data/foo.dat",
				"storage-engine.file[1].shadow":             "/opt/aerospike/data/foo-shadow.dat",
				"storage-engine.filesize":                   1073741824,
				"storage-engine.flush-max-ms":               1000,
				"storage-engine.max-used-pct":               70,
				"storage-engine.max-write-cache":            67108864,
				"storage-engine.min-avail-pct":              5,
				"storage-engine.post-write-queue":           256,
				"storage-engine.read-page-cache":            false,
				"storage-engine.serialize-tomb-raider":      false,
				"storage-engine.sindex-startup-device-scan": false,
				"storage-engine.tomb-raider-sleep":          1000,
				"storage-engine.write-block-size":           1048576,
				"strong-consistency":                        true,
				"strong-consistency-allow-expunge":          false,
				"tomb-raider-eligible-age":                  86400,
				"tomb-raider-period":                        86400,
				"transaction-pending-limit":                 20,
				"truncate-threads":                          4,
				"write-commit-level-override":               "off",
				"xdr-bin-tombstone-ttl":                     86400,
				"xdr-tomb-raider-period":                    120,
				"xdr-tomb-raider-threads":                   1,
			},
			"test": Conf{
				"allow-ttl-without-nsup":                 false,
				"background-query-max-rps":               10000,
				"conflict-resolution-policy":             "generation",
				"conflict-resolve-writes":                false,
				"default-ttl":                            0,
				"disable-cold-start-eviction":            false,
				"disable-write-dup-res":                  false,
				"disallow-expunge":                       false,
				"disallow-null-setname":                  false,
				"enable-benchmarks-batch-sub":            false,
				"enable-benchmarks-ops-sub":              false,
				"enable-benchmarks-read":                 false,
				"enable-benchmarks-udf":                  false,
				"enable-benchmarks-udf-sub":              false,
				"enable-benchmarks-write":                false,
				"enable-hist-proxy":                      false,
				"evict-hist-buckets":                     10000,
				"evict-tenths-pct":                       5,
				"force-long-queries":                     false,
				"geo2dsphere-within.earth-radius-meters": 6371000,
				"geo2dsphere-within.level-mod":           1,
				"geo2dsphere-within.max-cells":           15,
				"geo2dsphere-within.max-level":           20,
				"geo2dsphere-within.min-level":           1,
				"geo2dsphere-within.strict":              true,
				"high-water-disk-pct":                    0,
				"high-water-memory-pct":                  0,
				"ignore-migrate-fill-delay":              false,
				"index-stage-size":                       1073741824,
				"index-type":                             "shmem",
				"inline-short-queries":                   false,
				"max-record-size":                        0,
				"memory-size":                            536870912,
				"migrate-order":                          5,
				"migrate-retransmit-ms":                  5000,
				"migrate-sleep":                          1,
				"nsup-hist-period":                       3600,
				"nsup-period":                            0,
				"nsup-threads":                           1,
				"partition-tree-sprigs":                  256,
				"prefer-uniform-balance":                 true,
				"rack-id":                                0,
				"read-consistency-level-override":        "off",
				"reject-non-xdr-writes":                  false,
				"reject-xdr-writes":                      false,
				"replication-factor":                     1,
				"sets": Conf{
					"testset": Conf{
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":                1073741824,
				"sindex-type":                      "shmem",
				"single-query-threads":             4,
				"stop-writes-pct":                  90,
				"stop-writes-sys-memory-pct":       90,
				"storage-engine":                   "memory",
				"strong-consistency":               false,
				"strong-consistency-allow-expunge": false,
				"tomb-raider-eligible-age":         86400,
				"tomb-raider-period":               86400,
				"transaction-pending-limit":        20,
				"truncate-threads":                 4,
				"write-commit-level-override":      "off",
				"xdr-bin-tombstone-ttl":            86400,
				"xdr-tomb-raider-period":           120,
				"xdr-tomb-raider-threads":          1,
			},
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"namespaces": []Conf{
			{
				"name":                        "bar",
				"allow-ttl-without-nsup":      false,
				"background-query-max-rps":    10000,
				"conflict-resolution-policy":  "generation",
				"conflict-resolve-writes":     false,
				"default-ttl":                 0,
				"disable-cold-start-eviction": false,
				"disable-write-dup-res":       false,
				"disallow-expunge":            false,
				"disallow-null-setname":       false,
				"enable-benchmarks-batch-sub": false,
				"enable-benchmarks-ops-sub":   false,
				"enable-benchmarks-read":      false,
				"enable-benchmarks-udf":       false,
				"enable-benchmarks-udf-sub":   false,
				"enable-benchmarks-write":     false,
				"enable-hist-proxy":           false,
				"evict-hist-buckets":          10000,
				"evict-tenths-pct":            5,
				"geo2dsphere-within": Conf{"earth-radius-meters": 6371000,
					"level-mod": 1,
					"max-cells": 12,
					"max-level": 20,
					"min-level": 1,
					"strict":    true,
				},
				"high-water-disk-pct":       0,
				"high-water-memory-pct":     0,
				"ignore-migrate-fill-delay": false,
				"index-stage-size":          1073741824,
				"index-type":                Conf{"type": "shmem"},
				"inline-short-queries":      false,
				"max-record-size":           0,
				"memory-size":               1073741824,
				"migrate-order":             5,
				"migrate-retransmit-ms":     5000,
				"migrate-sleep":             1,
				"nsup-hist-period":          3600,
				"nsup-period":               0,
				"nsup-threads":              1,
				"partition-tree-sprigs":     256,
				"prefer-uniform-balance":    true,
				"rack-id":                   2,
				"reject-non-xdr-writes":     false,
				"reject-xdr-writes":         false,
				"replication-factor":        1,
				"sets": []Conf{
					{
						"name":              "testset",
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":          1073741824,
				"sindex-type":                Conf{"type": "shmem"},
				"single-query-threads":       4,
				"stop-writes-pct":            90,
				"stop-writes-sys-memory-pct": 90,
				"storage-engine": Conf{
					"type":                      "device",
					"cache-replica-writes":      false,
					"cold-start-empty":          false,
					"commit-min-size":           0,
					"commit-to-device":          false,
					"compression":               "none",
					"compression-acceleration":  0,
					"compression-level":         0,
					"data-in-memory":            false,
					"defrag-lwm-pct":            50,
					"defrag-queue-min":          0,
					"defrag-sleep":              1000,
					"defrag-startup-minimum":    0,
					"direct-files":              false,
					"disable-odsync":            false,
					"enable-benchmarks-storage": false,
					"encryption-key-file":       "",
					"encryption-old-key-file":   "",
					"files": []string{
						"/opt/aerospike/data/bar.dat /opt/aerospike/data/bar-shadow.dat",
						"/opt/aerospike/data/foo.dat /opt/aerospike/data/foo-shadow.dat",
					},
					"filesize":                   1073741824,
					"flush-max-ms":               1000,
					"max-used-pct":               70,
					"max-write-cache":            67108864,
					"min-avail-pct":              5,
					"post-write-queue":           256,
					"read-page-cache":            false,
					"serialize-tomb-raider":      false,
					"sindex-startup-device-scan": false,
					"tomb-raider-sleep":          1000,
					"write-block-size":           1048576,
				},
				"strong-consistency":               true,
				"strong-consistency-allow-expunge": false,
				"tomb-raider-eligible-age":         86400,
				"tomb-raider-period":               86400,
				"transaction-pending-limit":        20,
				"truncate-threads":                 4,
				"xdr-bin-tombstone-ttl":            86400,
				"xdr-tomb-raider-period":           120,
				"xdr-tomb-raider-threads":          1,
			},
			{
				"name":                        "test",
				"allow-ttl-without-nsup":      false,
				"background-query-max-rps":    10000,
				"conflict-resolution-policy":  "generation",
				"conflict-resolve-writes":     false,
				"default-ttl":                 0,
				"disable-cold-start-eviction": false,
				"disable-write-dup-res":       false,
				"disallow-expunge":            false,
				"disallow-null-setname":       false,
				"enable-benchmarks-batch-sub": false,
				"enable-benchmarks-ops-sub":   false,
				"enable-benchmarks-read":      false,
				"enable-benchmarks-udf":       false,
				"enable-benchmarks-udf-sub":   false,
				"enable-benchmarks-write":     false,
				"enable-hist-proxy":           false,
				"evict-hist-buckets":          10000,
				"evict-tenths-pct":            5,
				"geo2dsphere-within": Conf{"earth-radius-meters": 6371000,
					"level-mod": 1,
					"max-cells": 15,
					"max-level": 20,
					"min-level": 1,
					"strict":    true,
				},
				"high-water-disk-pct":             0,
				"high-water-memory-pct":           0,
				"ignore-migrate-fill-delay":       false,
				"index-stage-size":                1073741824,
				"index-type":                      Conf{"type": "shmem"},
				"inline-short-queries":            false,
				"max-record-size":                 0,
				"memory-size":                     536870912,
				"migrate-order":                   5,
				"migrate-retransmit-ms":           5000,
				"migrate-sleep":                   1,
				"nsup-hist-period":                3600,
				"nsup-period":                     0,
				"nsup-threads":                    1,
				"partition-tree-sprigs":           256,
				"prefer-uniform-balance":          true,
				"rack-id":                         1,
				"read-consistency-level-override": "off",
				"reject-non-xdr-writes":           false,
				"reject-xdr-writes":               false,
				"replication-factor":              1,
				"sets": []Conf{
					{
						"name":              "testset",
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":          1073741824,
				"sindex-type":                Conf{"type": "shmem"},
				"single-query-threads":       4,
				"stop-writes-pct":            90,
				"stop-writes-sys-memory-pct": 90,
				"storage-engine": Conf{
					"type": "memory",
				},
				"strong-consistency":               false,
				"strong-consistency-allow-expunge": false,
				"tomb-raider-eligible-age":         86400,
				"tomb-raider-period":               86400,
				"transaction-pending-limit":        20,
				"truncate-threads":                 4,
				"write-commit-level-override":      "off",
				"xdr-bin-tombstone-ttl":            86400,
				"xdr-tomb-raider-period":           120,
				"xdr-tomb-raider-threads":          1,
			},
		},
	},
}

var networkTC = GenerateTC{
	"network",
	false,
	Conf{
		"network": Conf{
			"fabric.channel-bulk-fds":           2,
			"fabric.channel-bulk-recv-threads":  4,
			"fabric.channel-ctrl-fds":           1,
			"fabric.channel-ctrl-recv-threads":  4,
			"fabric.channel-meta-fds":           1,
			"fabric.channel-meta-recv-threads":  4,
			"fabric.channel-rw-fds":             8,
			"fabric.channel-rw-recv-pools":      1,
			"fabric.channel-rw-recv-threads":    16,
			"fabric.keepalive-enabled":          true,
			"fabric.keepalive-intvl":            1,
			"fabric.keepalive-probes":           10,
			"fabric.keepalive-time":             1,
			"fabric.latency-max-ms":             5,
			"fabric.port":                       3001,
			"fabric.recv-rearm-threshold":       1024,
			"fabric.send-threads":               8,
			"fabric.tls-name":                   "null",
			"fabric.tls-port":                   0,
			"heartbeat.connect-timeout-ms":      500,
			"heartbeat.interval":                150,
			"heartbeat.mode":                    "multicast",
			"heartbeat.mtu":                     65535,
			"heartbeat.multicast-group":         "239.1.99.222,239.1.99.223",
			"heartbeat.multicast-ttl":           0,
			"heartbeat.port":                    9918,
			"heartbeat.protocol":                "v3",
			"heartbeat.timeout":                 10,
			"info.port":                         3003,
			"service.access-address":            "1.1.1.1,2.2.2.2",
			"service.access-port":               0,
			"service.address":                   "any",
			"service.alternate-access-port":     0,
			"service.disable-localhost":         false,
			"service.port":                      3000,
			"service.tls-access-port":           0,
			"service.tls-alternate-access-port": 0,
			"service.tls-name":                  "null",
			"service.tls-port":                  0,
			"service.tls-authenticate-client":   0,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"network": Conf{
			"fabric": Conf{
				"channel-bulk-fds":          2,
				"channel-bulk-recv-threads": 4,
				"channel-ctrl-fds":          1,
				"channel-ctrl-recv-threads": 4,
				"channel-meta-fds":          1,
				"channel-meta-recv-threads": 4,
				"channel-rw-fds":            8,
				"channel-rw-recv-pools":     1,
				"channel-rw-recv-threads":   16,
				"keepalive-enabled":         true,
				"keepalive-intvl":           1,
				"keepalive-probes":          10,
				"keepalive-time":            1,
				"latency-max-ms":            5,
				"port":                      3001,
				"recv-rearm-threshold":      1024,
				"send-threads":              8,
				"tls-name":                  "",
				"tls-port":                  0,
			},
			"heartbeat": Conf{
				"connect-timeout-ms": 500,
				"interval":           150,
				"mode":               "multicast",
				"mtu":                65535,
				"multicast-groups":   []string{"239.1.99.222", "239.1.99.223"},
				"multicast-ttl":      0,
				"port":               9918,
				"protocol":           "v3",
				"timeout":            10,
			},
			"info": Conf{
				"port": 3003,
			},
			"service": Conf{
				"access-addresses":          []string{"1.1.1.1", "2.2.2.2"},
				"access-port":               0,
				"addresses":                 []string{"any"},
				"alternate-access-port":     0,
				"disable-localhost":         false,
				"port":                      3000,
				"tls-access-port":           0,
				"tls-alternate-access-port": 0,
				"tls-name":                  "",
				"tls-port":                  0,
				"tls-authenticate-client":   0,
			},
		},
	},
}

var securityEnabled57TC = GenerateTC{
	"security post 5.7",
	false,
	Conf{
		"security": Conf{
			"enable-quotas":             true,
			"enable-security":           true,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"privilege-refresh-period":  300,
			"session-ttl":               86400,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"security": Conf{
			"enable-quotas": true,
			"log": Conf{
				"report-authentication": false,
				"report-sys-admin":      false,
				"report-user-admin":     false,
				"report-violation":      false,
			},
			"privilege-refresh-period": 300,
			"session-ttl":              86400,
			"tps-weight":               2,
		},
	},
}

var securityDisabled57TC = GenerateTC{
	"security post 5.7",
	false,
	Conf{
		"security": Conf{
			"enable-quotas":             true,
			"enable-security":           false,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"privilege-refresh-period":  300,
			"session-ttl":               86400,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{},
}

var securityEnabled56TC = GenerateTC{
	"security pre 5.7",
	false,
	Conf{
		"security": Conf{
			"enable-quotas":             true,
			"enable-security":           true,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"privilege-refresh-period":  300,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "5.6.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"security": Conf{
			"enable-quotas":   true,
			"enable-security": true,
			"log": Conf{
				"report-authentication": false,
				"report-sys-admin":      false,
				"report-user-admin":     false,
				"report-violation":      false,
			},
			"privilege-refresh-period": 300,
			"tps-weight":               2,
		},
	},
}

var securityDisabled56TC = GenerateTC{
	"security pre 5.7",
	false,
	Conf{
		"security": Conf{
			"enable-quotas":             true,
			"enable-security":           false,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"ldap.session-ttl":          86400,
			"privilege-refresh-period":  300,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "5.6.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"security": Conf{
			"enable-quotas":   true,
			"enable-security": false,
			"log": Conf{
				"report-authentication": false,
				"report-sys-admin":      false,
				"report-user-admin":     false,
				"report-violation":      false,
			},
			"ldap": Conf{
				"session-ttl": 86400,
			},
			"privilege-refresh-period": 300,
			"tps-weight":               2,
		},
	},
}

var serviceTC = GenerateTC{
	"service",
	false,
	Conf{
		"service": Conf{
			"advertise-ipv6":              false,
			"auto-pin":                    "none",
			"batch-index-threads":         8,
			"batch-max-buffers-per-queue": 255,
			"batch-max-unused-buffers":    256,
			"cluster-name":                "6.x-cluster-security",
			"debug-allocations":           "none",
			"disable-udf-execution":       false,
			"downgrading":                 false,
			"enable-benchmarks-fabric":    false,
			"enable-health-check":         false,
			"enable-hist-info":            false,
			"enforce-best-practices":      false,
			"feature-key-file[0]":         "/etc/aerospike/features.conf",
			"indent-allocations":          false,
			"info-max-ms":                 10000,
			"info-threads":                16,
			"keep-caps-ssd-health":        false,
			"log-local-time":              false,
			"log-millis":                  false,
			"microsecond-histograms":      false,
			"migrate-fill-delay":          0,
			"migrate-max-num-incoming":    4,
			"migrate-threads":             1,
			"min-cluster-size":            1,
			"node-id":                     "BB9050011AC4202",
			"node-id-interface":           "null",
			"os-group-perms":              false,
			"pidfile":                     "null",
			"proto-fd-idle-ms":            0,
			"proto-fd-max":                15000,
			"query-max-done":              100,
			"query-threads-limit":         128,
			"run-as-daemon":               true,
			"salt-allocations":            false,
			"secrets-address-port":        "null",
			"secrets-tls-context":         "null",
			"service-threads":             8,
			"sindex-builder-threads":      4,
			"sindex-gc-period":            10,
			"stay-quiesced":               false,
			"ticker-interval":             10,
			"transaction-max-ms":          1000,
			"transaction-retry-ms":        1002,
			"vault-ca":                    "null",
			"vault-namespace":             "null",
			"vault-path":                  "null",
			"vault-token-file":            "null",
			"vault-url":                   "null",
			"work-directory":              "/opt/aerospike",
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"service": Conf{
			"advertise-ipv6":              false,
			"auto-pin":                    "none",
			"batch-index-threads":         8,
			"batch-max-buffers-per-queue": 255,
			"batch-max-unused-buffers":    256,
			"cluster-name":                "6.x-cluster-security",
			"debug-allocations":           "none",
			"disable-udf-execution":       false,
			"enable-benchmarks-fabric":    false,
			"enable-health-check":         false,
			"enable-hist-info":            false,
			"enforce-best-practices":      false,
			"feature-key-files":           []string{"/etc/aerospike/features.conf"},
			"indent-allocations":          false,
			"info-max-ms":                 10000,
			"info-threads":                16,
			"keep-caps-ssd-health":        false,
			"log-local-time":              false,
			"log-millis":                  false,
			"microsecond-histograms":      false,
			"migrate-fill-delay":          0,
			"migrate-max-num-incoming":    4,
			"migrate-threads":             1,
			"min-cluster-size":            1,
			"node-id":                     "BB9050011AC4202",
			"node-id-interface":           "",
			"os-group-perms":              false,
			"pidfile":                     "",
			"proto-fd-idle-ms":            0,
			"proto-fd-max":                15000,
			"query-max-done":              100,
			"query-threads-limit":         128,
			"run-as-daemon":               true,
			"salt-allocations":            false,
			"secrets-address-port":        "",
			"secrets-tls-context":         "",
			"service-threads":             8,
			"sindex-builder-threads":      4,
			"sindex-gc-period":            10,
			"stay-quiesced":               false,
			"ticker-interval":             10,
			"transaction-max-ms":          1000,
			"transaction-retry-ms":        1002,
			"vault-ca":                    "",
			"vault-namespace":             "",
			"vault-path":                  "",
			"vault-token-file":            "",
			"vault-url":                   "",
			"work-directory":              "/opt/aerospike",
		},
	},
}

var xdr5TC = GenerateTC{
	"xdr5",
	false,
	Conf{
		"xdr": Conf{
			"dcs": Conf{
				"DC1": Conf{
					"auth-mode":                  "none",
					"auth-password-file":         "null",
					"auth-user":                  "null",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": Conf{
						"test": Conf{
							"bin-policy":               "changed-or-specified",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignored-bins":             "",
							"ignored-sets":             "",
							"max-throughput":           100000,
							"remote-namespace":         "null",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"shipped-bins":             "foo,bar",
							"shipped-sets":             "blah,blee",
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-port":            "",
					"period-ms":                    100,
					"tls-name":                     "null",
					"use-alternate-access-address": false,
				},
				"DC2": Conf{
					"auth-mode":                  "none",
					"auth-password-file":         "null",
					"auth-user":                  "null",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": Conf{
						"bar": Conf{
							"bin-policy":               "all",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"enabled":                  true,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignored-bins":             "tip,wip",
							"ignored-sets":             "zip,zap",
							"max-throughput":           100000,
							"remote-namespace":         "null",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"shipped-bins":             "",
							"shipped-sets":             "",
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-port":            "",
					"period-ms":                    100,
					"tls-name":                     "null",
					"use-alternate-access-address": false,
				},
			},
			"src-id":       0,
			"trace-sample": 0,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"xdr": Conf{
			"dcs": []Conf{
				{
					"name":                       "DC1",
					"auth-mode":                  "none",
					"auth-password-file":         "",
					"auth-user":                  "",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": []Conf{
						{
							"name":                     "test",
							"bin-policy":               "changed-or-specified",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignore-bins":              []string{},
							"ignore-sets":              []string{},
							"max-throughput":           100000,
							"remote-namespace":         "",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"ship-bins":                []string{"foo", "bar"},
							"ship-sets":                []string{"blah", "blee"},
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-ports":           []string{},
					"period-ms":                    100,
					"tls-name":                     "",
					"use-alternate-access-address": false,
				},
				{
					"name":                       "DC2",
					"auth-mode":                  "none",
					"auth-password-file":         "",
					"auth-user":                  "",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": []Conf{
						{
							"name":                     "bar",
							"bin-policy":               "all",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignore-bins":              []string{"tip", "wip"},
							"ignore-sets":              []string{"zip", "zap"},
							"max-throughput":           100000,
							"remote-namespace":         "",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"ship-bins":                []string{},
							"ship-sets":                []string{},
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-ports":           []string{},
					"period-ms":                    100,
					"tls-name":                     "",
					"use-alternate-access-address": false,
				},
			},
			"src-id": 0,
		},
	},
}

// Same as above but remove defaults
var loggingDefaultsTC = GenerateTC{
	"logging with remove default",
	true,
	Conf{
		"logging": Conf{
			"/var/log/aerospike/aerospike.log": Conf{
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
			"stderr": Conf{
				"aggr":         "CRITICAL",
				"alloc":        "WARNING",
				"appeal":       "DEBUG",
				"arenax":       "DETAIL",
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
			"/dev/log": Conf{
				"some-other":   "keep this",
				"aggr":         "CRITICAL",
				"alloc":        "CRITICAL",
				"appeal":       "CRITICAL",
				"arenax":       "CRITICAL",
				"as":           "CRITICAL",
				"audit":        "CRITICAL",
				"batch":        "CRITICAL",
				"batch-sub":    "CRITICAL",
				"bin":          "CRITICAL",
				"clustering":   "CRITICAL",
				"config":       "CRITICAL",
				"drv_pmem":     "CRITICAL",
				"drv_ssd":      "CRITICAL",
				"exchange":     "CRITICAL",
				"exp":          "CRITICAL",
				"fabric":       "CRITICAL",
				"flat":         "CRITICAL",
				"geo":          "CRITICAL",
				"hardware":     "CRITICAL",
				"hb":           "CRITICAL",
				"health":       "CRITICAL",
				"hlc":          "CRITICAL",
				"index":        "CRITICAL",
				"info":         "CRITICAL",
				"info-port":    "CRITICAL",
				"key-busy":     "CRITICAL",
				"migrate":      "CRITICAL",
				"misc":         "CRITICAL",
				"msg":          "CRITICAL",
				"namespace":    "CRITICAL",
				"nsup":         "CRITICAL",
				"os":           "CRITICAL",
				"particle":     "CRITICAL",
				"partition":    "CRITICAL",
				"proto":        "CRITICAL",
				"proxy":        "CRITICAL",
				"proxy-divert": "CRITICAL",
				"query":        "CRITICAL",
				"record":       "CRITICAL",
				"roster":       "CRITICAL",
				"rw":           "CRITICAL",
				"rw-client":    "CRITICAL",
				"secrets":      "CRITICAL",
				"security":     "CRITICAL",
				"service":      "CRITICAL",
				"service-list": "CRITICAL",
				"sindex":       "CRITICAL",
				"skew":         "CRITICAL",
				"smd":          "CRITICAL",
				"socket":       "CRITICAL",
				"storage":      "CRITICAL",
				"tls":          "CRITICAL",
				"truncate":     "CRITICAL",
				"tsvc":         "CRITICAL",
				"udf":          "CRITICAL",
				"vault":        "CRITICAL",
				"vmapx":        "CRITICAL",
				"xdr":          "CRITICAL",
				"xdr-client":   "CRITICAL",
				"xmem":         "CRITICAL",
			},
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"logging": []Conf{
			{
				"name": "/var/log/aerospike/aerospike.log",
				"any":  "INFO",
			},
			{
				"name":   "console",
				"aggr":   "CRITICAL",
				"alloc":  "WARNING",
				"appeal": "DEBUG",
				"arenax": "DETAIL",
				"any":    "INFO",
			},
			{
				"name": "syslog",
				"path": "/dev/log",
				"any":  "CRITICAL",
			},
		},
	},
}

var namespacesDefaultsTC = GenerateTC{
	"namespaces with remove defaults",
	true,
	Conf{
		"racks": []Conf{
			{
				"ns":     "test",
				"rack_1": "BB9030011AC4202",
				"rack_0": "BB9030011AC4203",
			},
			{
				"ns":     "bar",
				"rack_2": "BB9030011AC4202",
				"rack_0": "BB9030011AC4203",
			},
		},
		"namespaces": Conf{
			"bar": Conf{
				"allow-ttl-without-nsup":                 false,
				"background-query-max-rps":               10000,
				"conflict-resolution-policy":             "generation",
				"conflict-resolve-writes":                false,
				"default-ttl":                            0,
				"disable-cold-start-eviction":            false,
				"disable-write-dup-res":                  false,
				"disallow-expunge":                       false,
				"disallow-null-setname":                  false,
				"enable-benchmarks-batch-sub":            false,
				"enable-benchmarks-ops-sub":              false,
				"enable-benchmarks-read":                 false,
				"enable-benchmarks-udf":                  false,
				"enable-benchmarks-udf-sub":              false,
				"enable-benchmarks-write":                false,
				"enable-hist-proxy":                      false,
				"evict-hist-buckets":                     10000,
				"evict-tenths-pct":                       5,
				"force-long-queries":                     false,
				"geo2dsphere-within.earth-radius-meters": 6371000,
				"geo2dsphere-within.level-mod":           1,
				"geo2dsphere-within.max-cells":           12,
				"geo2dsphere-within.max-level":           20,
				"geo2dsphere-within.min-level":           1,
				"geo2dsphere-within.strict":              true,
				"high-water-disk-pct":                    0,
				"high-water-memory-pct":                  0,
				"ignore-migrate-fill-delay":              false,
				"index-stage-size":                       1073741824,
				"index-type":                             "shmem",
				"inline-short-queries":                   false,
				"max-record-size":                        0,
				"memory-size":                            1073741824,
				"migrate-order":                          5,
				"migrate-retransmit-ms":                  5000,
				"migrate-sleep":                          1,
				"nsup-hist-period":                       3600,
				"nsup-period":                            0,
				"nsup-threads":                           1,
				"partition-tree-sprigs":                  256,
				"prefer-uniform-balance":                 true,
				"rack-id":                                0,
				"read-consistency-level-override":        "off",
				"reject-non-xdr-writes":                  false,
				"reject-xdr-writes":                      false,
				"replication-factor":                     1,
				"sets": Conf{
					"testset": Conf{
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":                         1073741824,
				"sindex-type":                               "shmem",
				"single-query-threads":                      4,
				"stop-writes-pct":                           90,
				"stop-writes-sys-memory-pct":                90,
				"storage-engine":                            "device",
				"storage-engine.cache-replica-writes":       false,
				"storage-engine.cold-start-empty":           false,
				"storage-engine.commit-min-size":            0,
				"storage-engine.commit-to-device":           false,
				"storage-engine.compression":                "none",
				"storage-engine.compression-acceleration":   0,
				"storage-engine.compression-level":          0,
				"storage-engine.data-in-memory":             false,
				"storage-engine.defrag-lwm-pct":             50,
				"storage-engine.defrag-queue-min":           0,
				"storage-engine.defrag-sleep":               1000,
				"storage-engine.defrag-startup-minimum":     0,
				"storage-engine.direct-files":               false,
				"storage-engine.disable-odsync":             false,
				"storage-engine.enable-benchmarks-storage":  false,
				"storage-engine.encryption-key-file":        "null",
				"storage-engine.encryption-old-key-file":    "null",
				"storage-engine.file[0]":                    "/opt/aerospike/data/bar.dat",
				"storage-engine.file[0].shadow":             "/opt/aerospike/data/bar-shadow.dat",
				"storage-engine.file[1]":                    "/opt/aerospike/data/foo.dat",
				"storage-engine.file[1].shadow":             "/opt/aerospike/data/foo-shadow.dat",
				"storage-engine.filesize":                   1073741824,
				"storage-engine.flush-max-ms":               1000,
				"storage-engine.max-used-pct":               70,
				"storage-engine.max-write-cache":            67108864,
				"storage-engine.min-avail-pct":              5,
				"storage-engine.post-write-queue":           256,
				"storage-engine.read-page-cache":            false,
				"storage-engine.serialize-tomb-raider":      false,
				"storage-engine.sindex-startup-device-scan": false,
				"storage-engine.tomb-raider-sleep":          1000,
				"storage-engine.write-block-size":           1048576,
				"strong-consistency":                        true,
				"strong-consistency-allow-expunge":          false,
				"tomb-raider-eligible-age":                  86400,
				"tomb-raider-period":                        86400,
				"transaction-pending-limit":                 20,
				"truncate-threads":                          4,
				"write-commit-level-override":               "off",
				"xdr-bin-tombstone-ttl":                     86400,
				"xdr-tomb-raider-period":                    120,
				"xdr-tomb-raider-threads":                   1,
			},
			"test": Conf{
				"allow-ttl-without-nsup":                 false,
				"background-query-max-rps":               10000,
				"conflict-resolution-policy":             "generation",
				"conflict-resolve-writes":                false,
				"default-ttl":                            0,
				"disable-cold-start-eviction":            false,
				"disable-write-dup-res":                  false,
				"disallow-expunge":                       false,
				"disallow-null-setname":                  false,
				"enable-benchmarks-batch-sub":            false,
				"enable-benchmarks-ops-sub":              false,
				"enable-benchmarks-read":                 false,
				"enable-benchmarks-udf":                  false,
				"enable-benchmarks-udf-sub":              false,
				"enable-benchmarks-write":                false,
				"enable-hist-proxy":                      false,
				"evict-hist-buckets":                     10000,
				"evict-tenths-pct":                       5,
				"force-long-queries":                     false,
				"geo2dsphere-within.earth-radius-meters": 6371000,
				"geo2dsphere-within.level-mod":           1,
				"geo2dsphere-within.max-cells":           15,
				"geo2dsphere-within.max-level":           20,
				"geo2dsphere-within.min-level":           1,
				"geo2dsphere-within.strict":              true,
				"high-water-disk-pct":                    0,
				"high-water-memory-pct":                  0,
				"ignore-migrate-fill-delay":              false,
				"index-stage-size":                       1073741824,
				"index-type":                             "shmem",
				"inline-short-queries":                   false,
				"max-record-size":                        0,
				"memory-size":                            536870912,
				"migrate-order":                          5,
				"migrate-retransmit-ms":                  5000,
				"migrate-sleep":                          1,
				"nsup-hist-period":                       3600,
				"nsup-period":                            0,
				"nsup-threads":                           1,
				"partition-tree-sprigs":                  256,
				"prefer-uniform-balance":                 true,
				"rack-id":                                0,
				"read-consistency-level-override":        "off",
				"reject-non-xdr-writes":                  false,
				"reject-xdr-writes":                      false,
				"replication-factor":                     1,
				"sets": Conf{
					"testset": Conf{
						"disable-eviction":  false,
						"enable-index":      false,
						"stop-writes-count": 0,
						"stop-writes-size":  0,
					},
				},
				"sindex-stage-size":                1073741824,
				"sindex-type":                      "shmem",
				"single-query-threads":             4,
				"stop-writes-pct":                  90,
				"stop-writes-sys-memory-pct":       90,
				"storage-engine":                   "memory",
				"strong-consistency":               false,
				"strong-consistency-allow-expunge": false,
				"tomb-raider-eligible-age":         86400,
				"tomb-raider-period":               86400,
				"transaction-pending-limit":        20,
				"truncate-threads":                 4,
				"write-commit-level-override":      "off",
				"xdr-bin-tombstone-ttl":            86400,
				"xdr-tomb-raider-period":           120,
				"xdr-tomb-raider-threads":          1,
			},
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"namespaces": []Conf{
			{
				"name":               "bar",
				"memory-size":        1073741824,
				"rack-id":            2,
				"replication-factor": 1,
				"sets": []Conf{
					{
						"name": "testset",
					},
				},
				"index-type": Conf{
					"type": "shmem",
				},
				"sindex-type": Conf{
					"type": "shmem",
				},
				"storage-engine": Conf{
					"type": "device",
					"files": []string{
						"/opt/aerospike/data/bar.dat /opt/aerospike/data/bar-shadow.dat",
						"/opt/aerospike/data/foo.dat /opt/aerospike/data/foo-shadow.dat",
					},
					"filesize":         1073741824,
					"data-in-memory":   false,
					"post-write-queue": 256,
				},
				"strong-consistency": true,
			},
			{
				"name":        "test",
				"memory-size": 536870912,
				"geo2dsphere-within": Conf{
					"max-cells": 15,
				},
				"rack-id":            1,
				"replication-factor": 1,
				"sets": []Conf{
					{
						"name": "testset",
					},
				},
				"index-type": Conf{
					"type": "shmem",
				},
				"sindex-type": Conf{
					"type": "shmem",
				},
				"storage-engine": Conf{
					"type": "memory",
				},
			},
		},
	},
}

var networkDefaultsTC = GenerateTC{
	"network with remove defaults",
	true,
	Conf{
		"network": Conf{
			"fabric.channel-bulk-fds":           2,
			"fabric.channel-bulk-recv-threads":  4,
			"fabric.channel-ctrl-fds":           1,
			"fabric.channel-ctrl-recv-threads":  4,
			"fabric.channel-meta-fds":           1,
			"fabric.channel-meta-recv-threads":  4,
			"fabric.channel-rw-fds":             8,
			"fabric.channel-rw-recv-pools":      1,
			"fabric.channel-rw-recv-threads":    16,
			"fabric.keepalive-enabled":          true,
			"fabric.keepalive-intvl":            1,
			"fabric.keepalive-probes":           10,
			"fabric.keepalive-time":             1,
			"fabric.latency-max-ms":             5,
			"fabric.port":                       3001,
			"fabric.recv-rearm-threshold":       1024,
			"fabric.send-threads":               8,
			"fabric.tls-name":                   "null",
			"fabric.tls-port":                   0,
			"heartbeat.connect-timeout-ms":      500,
			"heartbeat.interval":                151,
			"heartbeat.mode":                    "multicast",
			"heartbeat.mtu":                     65535,
			"heartbeat.multicast-group":         "239.1.99.222,239.1.99.223",
			"heartbeat.multicast-ttl":           0,
			"heartbeat.port":                    9918,
			"heartbeat.protocol":                "v3",
			"heartbeat.timeout":                 10,
			"info.port":                         3003,
			"service.access-address":            "1.1.1.1,2.2.2.2",
			"service.access-port":               0,
			"service.address":                   "any",
			"service.alternate-access-port":     0,
			"service.disable-localhost":         false,
			"service.port":                      3000,
			"service.tls-access-port":           0,
			"service.tls-alternate-access-port": 0,
			"service.tls-name":                  "null",
			"service.tls-port":                  0,
			"service.tls-authenticate-client":   "any",
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"network": Conf{
			"fabric": Conf{
				"port": 3001,
			},
			"heartbeat": Conf{
				"interval":         151,
				"mode":             "multicast",
				"multicast-groups": []string{"239.1.99.222", "239.1.99.223"},
				"port":             9918,
				"mtu":              65535,
			},
			"info": Conf{
				"port": 3003,
			},
			"service": Conf{
				"access-addresses": []string{"1.1.1.1", "2.2.2.2"},
				"addresses":        []string{"any"},
				"port":             3000,
			},
		},
	},
}

var security57DefaultsTC = GenerateTC{
	"security post 5.7 with remove defaults",
	true,
	Conf{
		"security": Conf{
			"enable-quotas":             true,
			"enable-security":           true,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"privilege-refresh-period":  300,
			"session-ttl":               86400,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"security": Conf{
			"enable-quotas": true,
		},
	},
}

var security57AllDefaultsTC = GenerateTC{
	"security post 5.7 with remove defaults",
	true,
	Conf{
		"security": Conf{
			"enable-quotas":             false,
			"enable-security":           true,
			"log.report-authentication": false,
			"log.report-sys-admin":      false,
			"log.report-user-admin":     false,
			"log.report-violation":      false,
			"privilege-refresh-period":  300,
			"session-ttl":               86400,
			"tps-weight":                2,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"security": Conf{},
	},
}

var serviceDefaultsTC = GenerateTC{
	"service with remove defaults",
	true,
	Conf{
		"service": Conf{
			"advertise-ipv6":              false,
			"auto-pin":                    "none",
			"batch-index-threads":         8,
			"batch-max-buffers-per-queue": 255,
			"batch-max-unused-buffers":    256,
			"cluster-name":                "6.x-cluster-security",
			"debug-allocations":           "none",
			"disable-udf-execution":       false,
			"downgrading":                 false,
			"enable-benchmarks-fabric":    false,
			"enable-health-check":         false,
			"enable-hist-info":            false,
			"enforce-best-practices":      false,
			"feature-key-file[0]":         "/etc/aerospike/features-non-default.conf",
			"indent-allocations":          false,
			"info-max-ms":                 10000,
			"info-threads":                16,
			"keep-caps-ssd-health":        false,
			"log-local-time":              false,
			"log-millis":                  false,
			"microsecond-histograms":      false,
			"migrate-fill-delay":          0,
			"migrate-max-num-incoming":    4,
			"migrate-threads":             1,
			"min-cluster-size":            1,
			"node-id":                     "BB9050011AC4202",
			"node-id-interface":           "null",
			"os-group-perms":              false,
			"pidfile":                     "null",
			"proto-fd-idle-ms":            0,
			"proto-fd-max":                15001,
			"query-max-done":              100,
			"query-threads-limit":         128,
			"run-as-daemon":               true,
			"salt-allocations":            false,
			"secrets-address-port":        "null",
			"secrets-tls-context":         "null",
			"service-threads":             8,
			"sindex-builder-threads":      4,
			"sindex-gc-period":            10,
			"stay-quiesced":               false,
			"ticker-interval":             10,
			"transaction-max-ms":          1000,
			"transaction-retry-ms":        1002,
			"vault-ca":                    "null",
			"vault-namespace":             "null",
			"vault-path":                  "null",
			"vault-token-file":            "null",
			"vault-url":                   "null",
			"work-directory":              "/opt/aerospike",
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"service": Conf{
			"cluster-name": "6.x-cluster-security",
			"proto-fd-max": 15001,
			"feature-key-files": []string{
				"/etc/aerospike/features-non-default.conf",
			},
			"node-id":             "BB9050011AC4202",
			"service-threads":     8,
			"batch-index-threads": 8,
		},
	},
}

var xdr5DefaultsTC = GenerateTC{
	"xdr5 with remove defaults",
	true,
	Conf{
		"xdr": Conf{
			"dcs": Conf{
				"DC1": Conf{
					"auth-mode":                  "none",
					"auth-password-file":         "null",
					"auth-user":                  "null",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": Conf{
						"test": Conf{
							"bin-policy":               "changed-or-specified",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"enabled":                  true,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignored-bins":             "",
							"ignored-sets":             "",
							"max-throughput":           100000,
							"remote-namespace":         "null",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"shipped-bins":             "foo,bar",
							"shipped-sets":             "blah,blee",
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-port":            "",
					"period-ms":                    100,
					"tls-name":                     "null",
					"use-alternate-access-address": false,
				},
				"DC2": Conf{
					"auth-mode":                  "none",
					"auth-password-file":         "null",
					"auth-user":                  "null",
					"connector":                  false,
					"max-recoveries-interleaved": 0,
					"namespaces": Conf{
						"bar": Conf{
							"bin-policy":               "all",
							"compression-level":        1,
							"compression-threshold":    128,
							"delay-ms":                 0,
							"enable-compression":       false,
							"enabled":                  true,
							"forward":                  false,
							"hot-key-ms":               100,
							"ignore-expunges":          false,
							"ignored-bins":             "tip,wip",
							"ignored-sets":             "zip,zap",
							"max-throughput":           100000,
							"remote-namespace":         "null",
							"sc-replication-wait-ms":   100,
							"ship-bin-luts":            false,
							"ship-nsup-deletes":        false,
							"ship-only-specified-sets": false,
							"shipped-bins":             "",
							"shipped-sets":             "",
							"transaction-queue-limit":  16384,
							"write-policy":             "auto",
						},
					},
					"node-address-port":            "",
					"period-ms":                    100,
					"tls-name":                     "null",
					"use-alternate-access-address": false,
				},
			},
			"src-id":       1,
			"trace-sample": 0,
		},
	},
	Conf{"metadata": Conf{"asd_build": "6.4.0.0", "node_id": "BB9030011AC4202"}},
	Conf{
		"xdr": Conf{
			"dcs": []Conf{
				{
					"name": "DC1",
					"namespaces": []Conf{
						{
							"name":       "test",
							"bin-policy": "changed-or-specified",
							"ship-bins":  []string{"foo", "bar"},
							"ship-sets":  []string{"blah", "blee"},
						},
					},
				},
				{
					"name": "DC2",
					"namespaces": []Conf{
						{
							"name":        "bar",
							"ignore-bins": []string{"tip", "wip"},
							"ignore-sets": []string{"zip", "zap"},
						},
					},
				},
			},
			"src-id": 1,
		},
	},
}

var testSchemas = map[string]string{
	"5.6.0": `{
		"$schema": "http://json-schema.org/draft-06/schema",
		"additionalProperties": false,
		"type": "object",
		"required": [
		  "network",
		  "namespaces"
		],
		"properties": {
		  "service": {
			"type": "object",
			"additionalProperties": false,
			"oneOf": [
			  {
				"required": [
				  "feature-key-file"
				]
			  },
			  {
				"required": [
				  "feature-key-files"
				]
			  }
			],
			"properties": {
			  "query-buf-size": {
				"type": "integer",
				"default": 2097152,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "user": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "group": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "paxos-single-replica-limit": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 256,
				"description": "",
				"dynamic": false
			  },
			  "pidfile": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "proto-fd-max": {
				"type": "integer",
				"default": 15000,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "proto-slow-netio-sleep-ms": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "advertise-ipv6": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "auto-pin": {
				"type": "string",
				"description": "",
				"dynamic": false,
				"default": "none",
				"enum": [
				  "none",
				  "cpu",
				  "numa",
				  "adq"
				]
			  },
			  "batch-index-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 1,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "batch-max-buffers-per-queue": {
				"type": "integer",
				"default": 255,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "batch-max-requests": {
				"type": "integer",
				"default": 5000,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "batch-max-unused-buffers": {
				"type": "integer",
				"default": 256,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "batch-without-digests": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "cluster-name": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": true
			  },
			  "disable-udf-execution": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "enable-benchmarks-fabric": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "enable-health-check": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "enable-hist-info": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "feature-key-file": {
				"type": "string",
				"default": "/opt/aerospike/data/features.conf",
				"description": "",
				"dynamic": false
			  },
			  "feature-key-files": {
				"type": "array",
				"items": {
				  "type": "string"
				},
				"description": "",
				"dynamic": false,
				"default": [
				  "/opt/aerospike/data/features.conf"
				]
			  },
			  "info-threads": {
				"type": "integer",
				"default": 16,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "keep-caps-ssd-health": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "log-local-time": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "log-millis": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "microsecond-histograms": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "migrate-fill-delay": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "migrate-max-num-incoming": {
				"type": "integer",
				"default": 4,
				"minimum": 0,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "migrate-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 100,
				"description": "",
				"dynamic": true
			  },
			  "min-cluster-size": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "node-id": {
				"type": "string",
				"default": "BB9138953270008",
				"description": "",
				"dynamic": false
			  },
			  "node-id-interface": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "os-group-perms": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "proto-fd-idle-ms": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "query-batch-size": {
				"type": "integer",
				"default": 100,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "query-bufpool-size": {
				"type": "integer",
				"default": 256,
				"minimum": 1,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "query-in-transaction-thread": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "query-long-q-max-size": {
				"type": "integer",
				"default": 500,
				"minimum": 1,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "query-microbenchmark": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "query-pre-reserve-partitions": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "query-priority": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "query-priority-sleep-us": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 18446744073709551615,
				"description": "",
				"dynamic": true
			  },
			  "query-rec-count-bound": {
				"type": "integer",
				"default": 18446744073709551615,
				"minimum": 1,
				"maximum": 18446744073709551615,
				"description": "",
				"dynamic": true
			  },
			  "query-req-in-query-thread": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "query-req-max-inflight": {
				"type": "integer",
				"default": 100,
				"minimum": 1,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "query-short-q-max-size": {
				"type": "integer",
				"default": 500,
				"minimum": 1,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "query-threads": {
				"type": "integer",
				"default": 6,
				"minimum": 1,
				"maximum": 32,
				"description": "",
				"dynamic": true
			  },
			  "query-threshold": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "query-untracked-time-ms": {
				"type": "integer",
				"default": 1000,
				"minimum": 1,
				"maximum": 18446744073709551615,
				"description": "",
				"dynamic": true
			  },
			  "query-worker-threads": {
				"type": "integer",
				"default": 15,
				"minimum": 1,
				"maximum": 480,
				"description": "",
				"dynamic": true
			  },
			  "run-as-daemon": {
				"type": "boolean",
				"default": true,
				"description": "",
				"dynamic": false
			  },
			  "scan-max-done": {
				"type": "integer",
				"default": 100,
				"minimum": 0,
				"maximum": 1000,
				"description": "",
				"dynamic": true
			  },
			  "scan-threads-limit": {
				"type": "integer",
				"default": 128,
				"minimum": 1,
				"maximum": 1024,
				"description": "",
				"dynamic": true
			  },
			  "service-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 1,
				"maximum": 4096,
				"description": "",
				"dynamic": true
			  },
			  "sindex-builder-threads": {
				"type": "integer",
				"default": 4,
				"minimum": 1,
				"maximum": 32,
				"description": "",
				"dynamic": true
			  },
			  "sindex-gc-max-rate": {
				"type": "integer",
				"default": 50000,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "sindex-gc-period": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "stay-quiesced": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "ticker-interval": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "transaction-max-ms": {
				"type": "integer",
				"default": 1000,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "transaction-retry-ms": {
				"type": "integer",
				"default": 1002,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "vault-ca": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-path": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-token-file": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-url": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "work-directory": {
				"type": "string",
				"default": "/opt/aerospike",
				"description": "",
				"dynamic": false
			  },
			  "debug-allocations": {
				"type": "string",
				"description": "",
				"dynamic": false,
				"default": "none",
				"enum": [
				  "none",
				  "transient",
				  "persistent",
				  "all"
				]
			  },
			  "indent-allocations": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  }
			}
		  },
		  "logging": {
			"type": "array",
			"items": {
			  "type": "object",
			  "additionalProperties": false,
			  "properties": {
				"name": {
				  "type": "string",
				  "default": " ",
				  "description": "",
				  "dynamic": false
				},
				"misc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"alloc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"arenax": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hardware": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"msg": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"rbuffer": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"socket": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"tls": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"vault": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"vmapx": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xmem": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"aggr": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"appeal": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"as": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"batch": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"bin": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"config": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"clustering": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"drv_pmem": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"drv_ssd": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"exchange": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"exp": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"fabric": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"flat": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"geo": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hb": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"health": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hlc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"index": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"info": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"info-port": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"job": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"migrate": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"mon": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"namespace": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"nsup": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"particle": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"partition": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"paxos": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proto": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proxy": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proxy-divert": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"query": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"record": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"roster": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"rw": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"rw-client": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"scan": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"security": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"service": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"service-list": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"sindex": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"skew": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"smd": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"storage": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"truncate": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"tsvc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"udf": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xdr": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xdr-client": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"any": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				}
			  }
			}
		  },
		  "network": {
			"type": "object",
			"additionalProperties": false,
			"required": [
			  "service",
			  "heartbeat",
			  "fabric"
			],
			"properties": {
			  "service": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "port"
					]
				  },
				  {
					"required": [
					  "tls-port",
					  "tls-name"
					]
				  }
				],
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "alternate-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "alternate-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "disable-localhost": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "tls-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-alternate-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-alternate-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "tls-authenticate-client": {
					"oneOf": [
					  {
						"type": "string",
						"description": "",
						"dynamic": false,
						"default": "any",
						"enum": [
						  "any",
						  "false"
						]
					  },
					  {
						"type": "array",
						"items": {
						  "type": "string",
						  "format": "hostname",
						  "not": {
							"enum": [
							  "any",
							  "false"
							]
						  }
						}
					  }
					]
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "heartbeat": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "mode",
					  "port"
					]
				  },
				  {
					"required": [
					  "mode",
					  "tls-port",
					  "tls-name"
					]
				  }
				],
				"properties": {
				  "mode": {
					"type": "string",
					"description": "",
					"dynamic": false,
					"default": "",
					"enum": [
					  "mesh",
					  "multicast"
					]
				  },
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "multicast-groups": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "mesh-seed-address-ports": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "interval": {
					"type": "integer",
					"default": 150,
					"minimum": 50,
					"maximum": 600000,
					"description": "",
					"dynamic": true
				  },
				  "timeout": {
					"type": "integer",
					"default": 10,
					"minimum": 3,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "connect-timeout-ms": {
					"type": "integer",
					"default": 500,
					"minimum": 50,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "mtu": {
					"type": "integer",
					"default": 0,
					"minimum": 0,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "multicast-ttl": {
					"type": "integer",
					"default": 0,
					"minimum": 0,
					"maximum": 255,
					"description": "",
					"dynamic": false
				  },
				  "protocol": {
					"type": "string",
					"description": "",
					"dynamic": true,
					"default": "v3",
					"enum": [
					  "none",
					  "v3"
					]
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-mesh-seed-address-ports": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "fabric": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "port"
					]
				  },
				  {
					"required": [
					  "tls-port",
					  "tls-name"
					]
				  }
				],
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "channel-bulk-fds": {
					"type": "integer",
					"default": 2,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-bulk-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-ctrl-fds": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-ctrl-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-meta-fds": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-meta-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-rw-fds": {
					"type": "integer",
					"default": 8,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-rw-recv-pools": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 16,
					"description": "",
					"dynamic": false
				  },
				  "channel-rw-recv-threads": {
					"type": "integer",
					"default": 16,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "keepalive-enabled": {
					"type": "boolean",
					"default": true,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-intvl": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-probes": {
					"type": "integer",
					"default": 10,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-time": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "latency-max-ms": {
					"type": "integer",
					"default": 5,
					"minimum": 0,
					"maximum": 1000,
					"description": "",
					"dynamic": false
				  },
				  "recv-rearm-threshold": {
					"type": "integer",
					"default": 1024,
					"minimum": 0,
					"maximum": 1048576,
					"description": "",
					"dynamic": true
				  },
				  "send-threads": {
					"type": "integer",
					"default": 8,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "info": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "tls": {
				"type": "array",
				"items": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"name": {
					  "type": "string",
					  "default": " ",
					  "description": "",
					  "dynamic": false
					},
					"ca-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"ca-path": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cert-blacklist": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cert-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cipher-suite": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"key-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"key-file-password": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"protocols": {
					  "type": "string",
					  "default": "TLSv1.2",
					  "description": "",
					  "dynamic": false
					}
				  }
				}
			  }
			}
		  },
		  "namespaces": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "type": "object",
			  "additionalProperties": false,
			  "required": [
				"memory-size"
			  ],
			  "properties": {
				"name": {
				  "type": "string",
				  "default": " ",
				  "description": "",
				  "dynamic": false
				},
				"replication-factor": {
				  "type": "integer",
				  "default": 2,
				  "minimum": 1,
				  "maximum": 256,
				  "description": "",
				  "dynamic": false
				},
				"memory-size": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 18446744073709551615,
				  "description": "",
				  "dynamic": true
				},
				"default-ttl": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 315360000,
				  "description": "",
				  "dynamic": true
				},
				"storage-engine": {
				  "oneOf": [
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "memory",
						  "enum": [
							"memory"
						  ]
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "oneOf": [
						{
						  "required": [
							"type",
							"devices"
						  ]
						},
						{
						  "required": [
							"type",
							"files"
						  ]
						}
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "device",
						  "enum": [
							"device"
						  ]
						},
						"devices": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"files": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"filesize": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1048576,
						  "maximum": 2199023255552,
						  "description": "",
						  "dynamic": false
						},
						"scheduler-mode": {
						  "type": "string",
						  "enum": [
							"anticipatory",
							"cfq",
							"deadline",
							"noop",
							"null"
						  ],
						  "description": "",
						  "dynamic": false
						},
						"write-block-size": {
						  "type": "integer",
						  "default": 1048576,
						  "minimum": 1024,
						  "maximum": 8388608,
						  "description": "",
						  "dynamic": false
						},
						"data-in-memory": {
						  "type": "boolean",
						  "default": true,
						  "description": "",
						  "dynamic": false
						},
						"cache-replica-writes": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"cold-start-empty": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"commit-to-device": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"commit-min-size": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 8388608,
						  "description": "",
						  "dynamic": false
						},
						"compression": {
						  "type": "string",
						  "description": "",
						  "dynamic": true,
						  "default": "none",
						  "enum": [
							"none",
							"lz4",
							"snappy",
							"zstd"
						  ]
						},
						"compression-level": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 9,
						  "description": "",
						  "dynamic": true
						},
						"defrag-lwm-pct": {
						  "type": "integer",
						  "default": 50,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": true
						},
						"defrag-queue-min": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-startup-minimum": {
						  "type": "integer",
						  "default": 10,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": false
						},
						"direct-files": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"disable-odsync": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"enable-benchmarks-storage": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"encryption": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "aes-128",
						  "enum": [
							"aes-128",
							"aes-256"
						  ]
						},
						"encryption-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"flush-max-ms": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 1000,
						  "description": "",
						  "dynamic": true
						},
						"max-write-cache": {
						  "type": "integer",
						  "default": 67108864,
						  "minimum": 0,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						},
						"min-avail-pct": {
						  "type": "integer",
						  "default": 5,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"post-write-queue": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4096,
						  "description": "",
						  "dynamic": true
						},
						"read-page-cache": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"serialize-tomb-raider": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"sindex-startup-device-scan": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"tomb-raider-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"files"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "pmem",
						  "enum": [
							"pmem"
						  ]
						},
						"files": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"filesize": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1048576,
						  "maximum": 2199023255552,
						  "description": "",
						  "dynamic": false
						},
						"commit-to-device": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"compression": {
						  "type": "string",
						  "description": "",
						  "dynamic": true,
						  "default": "none",
						  "enum": [
							"none",
							"lz4",
							"snappy",
							"zstd"
						  ]
						},
						"compression-level": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 9,
						  "description": "",
						  "dynamic": true
						},
						"defrag-lwm-pct": {
						  "type": "integer",
						  "default": 50,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": true
						},
						"defrag-queue-min": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-startup-minimum": {
						  "type": "integer",
						  "default": 10,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": false
						},
						"direct-files": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"disable-odsync": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"enable-benchmarks-storage": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"encryption": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "aes-128",
						  "enum": [
							"aes-128",
							"aes-256"
						  ]
						},
						"encryption-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"flush-max-ms": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 1000,
						  "description": "",
						  "dynamic": true
						},
						"max-write-cache": {
						  "type": "integer",
						  "default": 67108864,
						  "minimum": 0,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						},
						"min-avail-pct": {
						  "type": "integer",
						  "default": 5,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"serialize-tomb-raider": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"tomb-raider-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						}
					  }
					}
				  ]
				},
				"allow-ttl-without-nsup": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"background-scan-max-rps": {
				  "type": "integer",
				  "default": 10000,
				  "minimum": 1,
				  "maximum": 1000000,
				  "description": "",
				  "dynamic": true
				},
				"conflict-resolution-policy": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "generation",
				  "enum": [
					"generation",
					"last-update-time"
				  ]
				},
				"conflict-resolve-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"data-in-index": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"disable-cold-start-eviction": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"disable-write-dup-res": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"disallow-null-setname": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-batch-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-ops-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-read": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-udf": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-udf-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-write": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-hist-proxy": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"evict-hist-buckets": {
				  "type": "integer",
				  "default": 10000,
				  "minimum": 100,
				  "maximum": 10000000,
				  "description": "",
				  "dynamic": true
				},
				"evict-tenths-pct": {
				  "type": "integer",
				  "default": 5,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"high-water-disk-pct": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"high-water-memory-pct": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"ignore-migrate-fill-delay": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"index-stage-size": {
				  "type": "integer",
				  "default": 1073741824,
				  "minimum": 134217728,
				  "maximum": 17179869184,
				  "description": "",
				  "dynamic": false
				},
				"index-type": {
				  "oneOf": [
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "shmem",
						  "enum": [
							"shmem"
						  ]
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "pmem",
						  "enum": [
							"pmem"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1073741824,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "flash",
						  "enum": [
							"flash"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 4294967296,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					}
				  ]
				},
				"migrate-order": {
				  "type": "integer",
				  "default": 5,
				  "minimum": 1,
				  "maximum": 10,
				  "description": "",
				  "dynamic": true
				},
				"migrate-retransmit-ms": {
				  "type": "integer",
				  "default": 5000,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"migrate-sleep": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-hist-period": {
				  "type": "integer",
				  "default": 3600,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-period": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-threads": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"partition-tree-sprigs": {
				  "type": "integer",
				  "default": 256,
				  "minimum": 16,
				  "maximum": 4096,
				  "description": "",
				  "dynamic": false
				},
				"prefer-uniform-balance": {
				  "type": "boolean",
				  "default": true,
				  "description": "",
				  "dynamic": true
				},
				"rack-id": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 1000000,
				  "description": "",
				  "dynamic": true
				},
				"read-consistency-level-override": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "off",
				  "enum": [
					"all",
					"off",
					"one"
				  ]
				},
				"reject-non-xdr-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"reject-xdr-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"sets": {
				  "type": "array",
				  "items": {
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "name": {
						"type": "string",
						"default": " ",
						"description": "",
						"dynamic": false
					  },
					  "disable-eviction": {
						"type": "boolean",
						"default": false,
						"description": "",
						"dynamic": true
					  },
					  "enable-index": {
						"type": "boolean",
						"default": false,
						"description": "",
						"dynamic": true
					  },
					  "stop-writes-count": {
						"type": "integer",
						"default": 0,
						"minimum": 0,
						"maximum": 18446744073709551615,
						"description": "",
						"dynamic": true
					  }
					}
				  }
				},
				"sindex": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"num-partitions": {
					  "type": "integer",
					  "default": 32,
					  "minimum": 1,
					  "maximum": 256,
					  "description": "",
					  "dynamic": false
					}
				  }
				},
				"geo2dsphere-within": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"strict": {
					  "type": "boolean",
					  "default": true,
					  "description": "",
					  "dynamic": false
					},
					"min-level": {
					  "type": "integer",
					  "default": 1,
					  "minimum": 0,
					  "maximum": 30,
					  "description": "",
					  "dynamic": true
					},
					"max-level": {
					  "type": "integer",
					  "default": 30,
					  "minimum": 0,
					  "maximum": 30,
					  "description": "",
					  "dynamic": true
					},
					"max-cells": {
					  "type": "integer",
					  "default": 12,
					  "minimum": 1,
					  "maximum": 256,
					  "description": "",
					  "dynamic": true
					},
					"level-mod": {
					  "type": "integer",
					  "default": 1,
					  "minimum": 1,
					  "maximum": 3,
					  "description": "",
					  "dynamic": false
					},
					"earth-radius-meters": {
					  "type": "integer",
					  "default": 6371000,
					  "minimum": 0,
					  "maximum": 4294967295,
					  "description": "",
					  "dynamic": false
					}
				  }
				},
				"single-bin": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"single-scan-threads": {
				  "type": "integer",
				  "default": 4,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"stop-writes-pct": {
				  "type": "integer",
				  "default": 90,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"strong-consistency": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"strong-consistency-allow-expunge": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"tomb-raider-eligible-age": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"tomb-raider-period": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"transaction-pending-limit": {
				  "type": "integer",
				  "default": 20,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"truncate-threads": {
				  "type": "integer",
				  "default": 4,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"write-commit-level-override": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "off",
				  "enum": [
					"all",
					"master",
					"off"
				  ]
				},
				"xdr-bin-tombstone-ttl": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 315360000,
				  "description": "",
				  "dynamic": true
				},
				"xdr-tomb-raider-period": {
				  "type": "integer",
				  "default": 120,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"xdr-tomb-raider-threads": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				}
			  }
			}
		  },
		  "mod-lua": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "cache-enabled": {
				"type": "boolean",
				"default": true,
				"description": "",
				"dynamic": false
			  },
			  "user-path": {
				"type": "string",
				"default": "/opt/aerospike/usr/udf/lua",
				"description": "",
				"dynamic": false
			  }
			}
		  },
		  "security": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "enable-ldap": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "enable-quotas": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "enable-security": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "ldap-login-threads": {
				"type": "integer",
				"default": 8,
				"minimum": 1,
				"maximum": 64,
				"description": "",
				"dynamic": false
			  },
			  "privilege-refresh-period": {
				"type": "integer",
				"default": 300,
				"minimum": 10,
				"maximum": 86400,
				"description": "",
				"dynamic": true
			  },
			  "tps-weight": {
				"type": "integer",
				"default": 2,
				"minimum": 2,
				"maximum": 20,
				"description": "",
				"dynamic": true
			  },
			  "ldap": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "disable-tls": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "polling-period": {
					"type": "integer",
					"default": 300,
					"minimum": 0,
					"maximum": 86400,
					"description": "",
					"dynamic": true
				  },
				  "query-base-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "query-user-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "query-user-password-file": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "role-query-base-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "role-query-patterns": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "role-query-search-ou": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "server": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "session-ttl": {
					"type": "integer",
					"default": 86400,
					"minimum": 120,
					"maximum": 864000,
					"description": "",
					"dynamic": true
				  },
				  "tls-ca-file": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "token-hash-method": {
					"type": "string",
					"default": "sha-256",
					"description": "",
					"dynamic": false
				  },
				  "user-dn-pattern": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "user-query-pattern": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "log": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "report-authentication": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-data-op": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-role": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-user": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-sys-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-user-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-violation": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  }
				}
			  },
			  "syslog": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "local": {
					"type": "integer",
					"default": -1,
					"minimum": 0,
					"maximum": 7,
					"description": "",
					"dynamic": false
				  },
				  "report-authentication": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-data-op": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-role": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-user": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-sys-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-user-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-violation": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  }
				}
			  }
			}
		  },
		  "xdr": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "dcs": {
				"type": "array",
				"minItems": 1,
				"items": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"name": {
					  "type": "string",
					  "default": " ",
					  "description": "",
					  "dynamic": false
					},
					"node-address-ports": {
					  "type": "array",
					  "items": {
						"type": "string"
					  },
					  "description": "",
					  "dynamic": false,
					  "default": []
					},
					"namespaces": {
					  "type": "array",
					  "items": {
						"type": "object",
						"additionalProperties": false,
						"properties": {
						  "name": {
							"type": "string",
							"default": " ",
							"description": "",
							"dynamic": false
						  },
						  "bin-policy": {
							"type": "string",
							"description": "",
							"dynamic": true,
							"default": "all",
							"enum": [
							  "all",
							  "only-changed",
							  "changed-and-specified",
							  "changed-or-specified"
							]
						  },
						  "compression-level": {
							"type": "integer",
							"default": 1,
							"minimum": 1,
							"maximum": 9,
							"description": "",
							"dynamic": true
						  },
						  "delay-ms": {
							"type": "integer",
							"default": 0,
							"minimum": 0,
							"maximum": 5000,
							"description": "",
							"dynamic": true
						  },
						  "enable-compression": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "forward": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "hot-key-ms": {
							"type": "integer",
							"default": 100,
							"minimum": 0,
							"maximum": 5000,
							"description": "",
							"dynamic": true
						  },
						  "ignore-bins": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "ignore-expunges": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ignore-sets": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "max-throughput": {
							"type": "integer",
							"default": 100000,
							"minimum": 0,
							"maximum": 4294967295,
							"description": "",
							"dynamic": true
						  },
						  "remote-namespace": {
							"type": "string",
							"default": "",
							"description": "",
							"dynamic": true
						  },
						  "sc-replication-wait-ms": {
							"type": "integer",
							"default": 100,
							"minimum": 5,
							"maximum": 1000,
							"description": "",
							"dynamic": true
						  },
						  "ship-bins": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "ship-bin-luts": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-nsup-deletes": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-only-specified-sets": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-sets": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "transaction-queue-limit": {
							"type": "integer",
							"default": 16384,
							"minimum": 1024,
							"maximum": 1048576,
							"description": "",
							"dynamic": true
						  },
						  "write-policy": {
							"type": "string",
							"description": "",
							"dynamic": true,
							"default": "auto",
							"enum": [
							  "auto",
							  "update",
							  "replace"
							]
						  }
						}
					  }
					},
					"auth-mode": {
					  "type": "string",
					  "description": "",
					  "dynamic": true,
					  "default": "internal",
					  "enum": [
						"internal",
						"external",
						"external-insecure"
					  ]
					},
					"auth-password-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"auth-user": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"connector": {
					  "type": "boolean",
					  "default": false,
					  "description": "",
					  "dynamic": true
					},
					"max-recoveries-interleaved": {
					  "type": "integer",
					  "default": 0,
					  "minimum": 0,
					  "maximum": 4294967295,
					  "description": "",
					  "dynamic": true
					},
					"max-used-service-threads": {
					  "type": "integer",
					  "default": 0,
					  "minimum": 0,
					  "maximum": 4096,
					  "description": "",
					  "dynamic": true
					},
					"period-ms": {
					  "type": "integer",
					  "default": 100,
					  "minimum": 5,
					  "maximum": 1000,
					  "description": "",
					  "dynamic": true
					},
					"tls-name": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"use-alternate-access-address": {
					  "type": "boolean",
					  "default": false,
					  "description": "",
					  "dynamic": true
					}
				  }
				}
			  },
			  "src-id": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 255,
				"description": "",
				"dynamic": true
			  }
			}
		  }
		}
	  }`,
	"6.4.0": `
	{
		"$schema": "http://json-schema.org/draft-06/schema",
		"additionalProperties": false,
		"type": "object",
		"required": [
		  "network",
		  "namespaces"
		],
		"properties": {
		  "service": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "advertise-ipv6": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "auto-pin": {
				"type": "string",
				"description": "",
				"dynamic": false,
				"default": "none",
				"enum": [
				  "none",
				  "cpu",
				  "numa",
				  "adq"
				]
			  },
			  "batch-index-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 1,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "batch-max-buffers-per-queue": {
				"type": "integer",
				"default": 255,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "batch-max-unused-buffers": {
				"type": "integer",
				"default": 256,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "cluster-name": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": true
			  },
			  "debug-allocations": {
				"type": "string",
				"description": "",
				"dynamic": false,
				"default": "none",
				"enum": [
				  "none",
				  "transient",
				  "persistent",
				  "all"
				]
			  },
			  "disable-udf-execution": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "enable-benchmarks-fabric": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "enable-health-check": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "enable-hist-info": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "enforce-best-practices": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "feature-key-file": {
				"type": "string",
				"default": "/opt/aerospike/data/features.conf",
				"description": "",
				"dynamic": false
			  },
			  "feature-key-files": {
				"type": "array",
				"items": {
				  "type": "string"
				},
				"description": "",
				"dynamic": false,
				"default": [
				  "/opt/aerospike/data/features.conf"
				]
			  },
			  "group": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "indent-allocations": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "info-max-ms": {
				"type": "integer",
				"default": 10000,
				"minimum": 500,
				"maximum": 10000,
				"description": "",
				"dynamic": true
			  },
			  "info-threads": {
				"type": "integer",
				"default": 16,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "keep-caps-ssd-health": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "log-local-time": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "log-millis": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "microsecond-histograms": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": true
			  },
			  "migrate-fill-delay": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "migrate-max-num-incoming": {
				"type": "integer",
				"default": 4,
				"minimum": 0,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "migrate-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 100,
				"description": "",
				"dynamic": true
			  },
			  "min-cluster-size": {
				"type": "integer",
				"default": 1,
				"minimum": 0,
				"maximum": 256,
				"description": "",
				"dynamic": true
			  },
			  "node-id": {
				"type": "string",
				"default": "BB9C0E8CD290C00",
				"description": "",
				"dynamic": false
			  },
			  "node-id-interface": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "os-group-perms": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "pidfile": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "proto-fd-idle-ms": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "proto-fd-max": {
				"type": "integer",
				"default": 15000,
				"minimum": 0,
				"maximum": 2147483647,
				"description": "",
				"dynamic": true
			  },
			  "query-max-done": {
				"type": "integer",
				"default": 100,
				"minimum": 0,
				"maximum": 10000,
				"description": "",
				"dynamic": true
			  },
			  "query-threads-limit": {
				"type": "integer",
				"default": 128,
				"minimum": 1,
				"maximum": 1024,
				"description": "",
				"dynamic": true
			  },
			  "run-as-daemon": {
				"type": "boolean",
				"default": true,
				"description": "",
				"dynamic": false
			  },
			  "salt-allocations": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "secrets-address-port": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "secrets-tls-context": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "secrets-uds-path": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "service-threads": {
				"type": "integer",
				"default": 1,
				"minimum": 1,
				"maximum": 4096,
				"description": "",
				"dynamic": true
			  },
			  "sindex-builder-threads": {
				"type": "integer",
				"default": 4,
				"minimum": 1,
				"maximum": 32,
				"description": "",
				"dynamic": true
			  },
			  "sindex-gc-period": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "stay-quiesced": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "ticker-interval": {
				"type": "integer",
				"default": 10,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "transaction-max-ms": {
				"type": "integer",
				"default": 1000,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "transaction-retry-ms": {
				"type": "integer",
				"default": 1002,
				"minimum": 0,
				"maximum": 4294967295,
				"description": "",
				"dynamic": true
			  },
			  "user": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-ca": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-namespace": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-path": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "vault-token-file": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": true
			  },
			  "vault-url": {
				"type": "string",
				"default": "",
				"description": "",
				"dynamic": false
			  },
			  "work-directory": {
				"type": "string",
				"default": "/opt/aerospike",
				"description": "",
				"dynamic": false
			  }
			}
		  },
		  "logging": {
			"type": "array",
			"items": {
			  "type": "object",
			  "additionalProperties": false,
			  "properties": {
				"name": {
				  "type": "string",
				  "default": " ",
				  "description": "",
				  "dynamic": false
				},
				"misc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"alloc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"arenax": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hardware": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"msg": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"os": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"secrets": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"socket": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"tls": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"vault": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"vmapx": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xmem": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"aggr": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"appeal": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"as": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"audit": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"batch": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"batch-sub": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"bin": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"config": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"clustering": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"drv_pmem": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"drv_ssd": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"exchange": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"exp": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"fabric": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"flat": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"geo": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hb": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"health": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"hlc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"index": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"info": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"info-port": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"key-busy": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"migrate": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"namespace": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"nsup": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"particle": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"partition": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proto": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proxy": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"proxy-divert": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"query": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"record": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"roster": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"rw": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"rw-client": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"security": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"service": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"service-list": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"sindex": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"skew": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"smd": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"storage": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"truncate": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"tsvc": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"udf": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xdr": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"xdr-client": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"any": {
				  "enum": [
					"CRITICAL",
					"critical",
					"WARNING",
					"warning",
					"INFO",
					"info",
					"DEBUG",
					"debug",
					"DETAIL",
					"detail"
				  ],
				  "description": "",
				  "dynamic": true,
				  "default": "CRITICAL"
				},
				"facility": {
				  "enum": [
					"auth",
					"authpriv",
					"cron",
					"daemon",
					"ftp",
					"kern",
					"lpr",
					"mail",
					"news",
					"syslog",
					"user",
					"uucp",
					"local0",
					"local1",
					"local2",
					"local3",
					"local4",
					"local5",
					"local6",
					"local7"
				  ],
				  "description": "",
				  "dynamic": false,
				  "default": "local0"
				},
				"path": {
				  "type": "string",
				  "default": "/dev/log",
				  "description": "",
				  "dynamic": false
				},
				"tag": {
				  "type": "string",
				  "default": "asd",
				  "description": "",
				  "dynamic": false
				}
			  }
			}
		  },
		  "network": {
			"type": "object",
			"additionalProperties": false,
			"required": [
			  "service",
			  "heartbeat",
			  "fabric"
			],
			"properties": {
			  "service": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "port"
					]
				  },
				  {
					"required": [
					  "tls-name",
					  "tls-port"
					]
				  }
				],
				"properties": {
				  "access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "alternate-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "alternate-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "disable-localhost": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "tls-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-alternate-access-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-alternate-access-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "tls-authenticate-client": {
					"oneOf": [
					  {
						"type": "string",
						"description": "",
						"dynamic": false,
						"default": "any",
						"enum": [
						  "any",
						  "false"
						]
					  },
					  {
						"type": "array",
						"items": {
						  "type": "string",
						  "format": "hostname",
						  "not": {
							"enum": [
							  "any",
							  "false"
							]
						  }
						}
					  }
					]
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "heartbeat": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "mode",
					  "port"
					]
				  },
				  {
					"required": [
					  "mode",
					  "tls-name",
					  "tls-port"
					]
				  }
				],
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "connect-timeout-ms": {
					"type": "integer",
					"default": 500,
					"minimum": 50,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "interval": {
					"type": "integer",
					"default": 150,
					"minimum": 50,
					"maximum": 600000,
					"description": "",
					"dynamic": true
				  },
				  "mesh-seed-address-ports": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "mode": {
					"type": "string",
					"description": "",
					"dynamic": false,
					"default": "",
					"enum": [
					  "mesh",
					  "multicast"
					]
				  },
				  "mtu": {
					"type": "integer",
					"default": 0,
					"minimum": 0,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "multicast-groups": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "multicast-ttl": {
					"type": "integer",
					"default": 0,
					"minimum": 0,
					"maximum": 255,
					"description": "",
					"dynamic": false
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "protocol": {
					"type": "string",
					"description": "",
					"dynamic": true,
					"default": "v3",
					"enum": [
					  "none",
					  "v3"
					]
				  },
				  "timeout": {
					"type": "integer",
					"default": 10,
					"minimum": 3,
					"maximum": 4294967295,
					"description": "",
					"dynamic": true
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-mesh-seed-address-ports": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "fabric": {
				"type": "object",
				"additionalProperties": false,
				"anyOf": [
				  {
					"required": [
					  "port"
					]
				  },
				  {
					"required": [
					  "tls-name",
					  "tls-port"
					]
				  }
				],
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "channel-bulk-fds": {
					"type": "integer",
					"default": 2,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-bulk-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-ctrl-fds": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-ctrl-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-meta-fds": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-meta-recv-threads": {
					"type": "integer",
					"default": 4,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "channel-rw-fds": {
					"type": "integer",
					"default": 8,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "channel-rw-recv-pools": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 16,
					"description": "",
					"dynamic": false
				  },
				  "channel-rw-recv-threads": {
					"type": "integer",
					"default": 16,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": true
				  },
				  "keepalive-enabled": {
					"type": "boolean",
					"default": true,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-intvl": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-probes": {
					"type": "integer",
					"default": 10,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "keepalive-time": {
					"type": "integer",
					"default": 1,
					"minimum": 1,
					"maximum": 2147483647,
					"description": "",
					"dynamic": false
				  },
				  "latency-max-ms": {
					"type": "integer",
					"default": 5,
					"minimum": 0,
					"maximum": 1000,
					"description": "",
					"dynamic": false
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  },
				  "recv-rearm-threshold": {
					"type": "integer",
					"default": 1024,
					"minimum": 0,
					"maximum": 1048576,
					"description": "",
					"dynamic": true
				  },
				  "send-threads": {
					"type": "integer",
					"default": 8,
					"minimum": 1,
					"maximum": 128,
					"description": "",
					"dynamic": false
				  },
				  "tls-addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "tls-name": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "info": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "addresses": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "port": {
					"type": "integer",
					"default": 0,
					"minimum": 1024,
					"maximum": 65535,
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "tls": {
				"type": "array",
				"items": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"name": {
					  "type": "string",
					  "default": " ",
					  "description": "",
					  "dynamic": false
					},
					"ca-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"ca-path": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cert-blacklist": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cert-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"cipher-suite": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"key-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"key-file-password": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": false
					},
					"protocols": {
					  "type": "string",
					  "default": "TLSv1.2",
					  "description": "",
					  "dynamic": false
					}
				  }
				}
			  }
			}
		  },
		  "namespaces": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "type": "object",
			  "additionalProperties": false,
			  "required": [
				"memory-size"
			  ],
			  "properties": {
				"name": {
				  "type": "string",
				  "default": " ",
				  "description": "",
				  "dynamic": false
				},
				"allow-ttl-without-nsup": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"background-query-max-rps": {
				  "type": "integer",
				  "default": 10000,
				  "minimum": 1,
				  "maximum": 1000000,
				  "description": "",
				  "dynamic": true
				},
				"conflict-resolution-policy": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "generation",
				  "enum": [
					"generation",
					"last-update-time"
				  ]
				},
				"conflict-resolve-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"default-ttl": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 315360000,
				  "description": "",
				  "dynamic": true
				},
				"disable-cold-start-eviction": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"disable-write-dup-res": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"disallow-expunge": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"disallow-null-setname": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-batch-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-ops-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-read": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-udf": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-udf-sub": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-benchmarks-write": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"enable-hist-proxy": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"evict-hist-buckets": {
				  "type": "integer",
				  "default": 10000,
				  "minimum": 100,
				  "maximum": 10000000,
				  "description": "",
				  "dynamic": true
				},
				"evict-tenths-pct": {
				  "type": "integer",
				  "default": 5,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"high-water-disk-pct": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"high-water-memory-pct": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"ignore-migrate-fill-delay": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"index-stage-size": {
				  "type": "integer",
				  "default": 1073741824,
				  "minimum": 134217728,
				  "maximum": 17179869184,
				  "description": "",
				  "dynamic": false
				},
				"inline-short-queries": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"max-record-size": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"memory-size": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 18446744073709551615,
				  "description": "",
				  "dynamic": true
				},
				"migrate-order": {
				  "type": "integer",
				  "default": 5,
				  "minimum": 1,
				  "maximum": 10,
				  "description": "",
				  "dynamic": true
				},
				"migrate-retransmit-ms": {
				  "type": "integer",
				  "default": 5000,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"migrate-sleep": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-hist-period": {
				  "type": "integer",
				  "default": 3600,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-period": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"nsup-threads": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"partition-tree-sprigs": {
				  "type": "integer",
				  "default": 256,
				  "minimum": 16,
				  "maximum": 268453456,
				  "description": "",
				  "dynamic": false
				},
				"prefer-uniform-balance": {
				  "type": "boolean",
				  "default": true,
				  "description": "",
				  "dynamic": true
				},
				"rack-id": {
				  "type": "integer",
				  "default": 0,
				  "minimum": 0,
				  "maximum": 1000000,
				  "description": "",
				  "dynamic": true
				},
				"read-consistency-level-override": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "off",
				  "enum": [
					"all",
					"off",
					"one"
				  ]
				},
				"reject-non-xdr-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"reject-xdr-writes": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"replication-factor": {
				  "type": "integer",
				  "default": 2,
				  "minimum": 1,
				  "maximum": 256,
				  "description": "",
				  "dynamic": false
				},
				"sindex-stage-size": {
				  "type": "integer",
				  "default": 1073741824,
				  "minimum": 134217728,
				  "maximum": 4294967296,
				  "description": "",
				  "dynamic": false
				},
				"single-query-threads": {
				  "type": "integer",
				  "default": 4,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"stop-writes-pct": {
				  "type": "integer",
				  "default": 90,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"stop-writes-sys-memory-pct": {
				  "type": "integer",
				  "default": 90,
				  "minimum": 0,
				  "maximum": 100,
				  "description": "",
				  "dynamic": true
				},
				"strong-consistency": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": false
				},
				"strong-consistency-allow-expunge": {
				  "type": "boolean",
				  "default": false,
				  "description": "",
				  "dynamic": true
				},
				"tomb-raider-eligible-age": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"tomb-raider-period": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"transaction-pending-limit": {
				  "type": "integer",
				  "default": 20,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"truncate-threads": {
				  "type": "integer",
				  "default": 4,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"write-commit-level-override": {
				  "type": "string",
				  "description": "",
				  "dynamic": true,
				  "default": "off",
				  "enum": [
					"all",
					"master",
					"off"
				  ]
				},
				"xdr-bin-tombstone-ttl": {
				  "type": "integer",
				  "default": 86400,
				  "minimum": 0,
				  "maximum": 315360000,
				  "description": "",
				  "dynamic": true
				},
				"xdr-tomb-raider-period": {
				  "type": "integer",
				  "default": 120,
				  "minimum": 0,
				  "maximum": 4294967295,
				  "description": "",
				  "dynamic": true
				},
				"xdr-tomb-raider-threads": {
				  "type": "integer",
				  "default": 1,
				  "minimum": 1,
				  "maximum": 128,
				  "description": "",
				  "dynamic": true
				},
				"geo2dsphere-within": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"strict": {
					  "type": "boolean",
					  "default": true,
					  "description": "",
					  "dynamic": false
					},
					"min-level": {
					  "type": "integer",
					  "default": 1,
					  "minimum": 0,
					  "maximum": 30,
					  "description": "",
					  "dynamic": true
					},
					"max-level": {
					  "type": "integer",
					  "default": 20,
					  "minimum": 0,
					  "maximum": 30,
					  "description": "",
					  "dynamic": true
					},
					"max-cells": {
					  "type": "integer",
					  "default": 12,
					  "minimum": 1,
					  "maximum": 256,
					  "description": "",
					  "dynamic": true
					},
					"level-mod": {
					  "type": "integer",
					  "default": 1,
					  "minimum": 1,
					  "maximum": 3,
					  "description": "",
					  "dynamic": false
					},
					"earth-radius-meters": {
					  "type": "integer",
					  "default": 6371000,
					  "minimum": 0,
					  "maximum": 4294967295,
					  "description": "",
					  "dynamic": false
					}
				  }
				},
				"index-type": {
				  "oneOf": [
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "shmem",
						  "enum": [
							"shmem"
						  ]
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "pmem",
						  "enum": [
							"pmem"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1073741824,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "flash",
						  "enum": [
							"flash"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 4294967296,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					}
				  ]
				},
				"sets": {
				  "type": "array",
				  "items": {
					"type": "object",
					"additionalProperties": false,
					"properties": {
					  "name": {
						"type": "string",
						"default": " ",
						"description": "",
						"dynamic": false
					  },
					  "disable-eviction": {
						"type": "boolean",
						"default": false,
						"description": "",
						"dynamic": true
					  },
					  "enable-index": {
						"type": "boolean",
						"default": false,
						"description": "",
						"dynamic": true
					  },
					  "stop-writes-count": {
						"type": "integer",
						"default": 0,
						"minimum": 0,
						"maximum": 18446744073709551615,
						"description": "",
						"dynamic": true
					  },
					  "stop-writes-size": {
						"type": "integer",
						"default": 0,
						"minimum": 0,
						"maximum": 18446744073709551615,
						"description": "",
						"dynamic": true
					  }
					}
				  }
				},
				"sindex-type": {
				  "oneOf": [
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "shmem",
						  "enum": [
							"shmem"
						  ]
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "pmem",
						  "enum": [
							"pmem"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1073741824,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"mounts",
						"mounts-size-limit"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "flash",
						  "enum": [
							"flash"
						  ]
						},
						"mounts": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"mounts-high-water-pct": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"mounts-size-limit": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1073741824,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						}
					  }
					}
				  ]
				},
				"storage-engine": {
				  "oneOf": [
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "memory",
						  "enum": [
							"memory"
						  ]
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "oneOf": [
						{
						  "required": [
							"type",
							"devices"
						  ]
						},
						{
						  "required": [
							"type",
							"files"
						  ]
						}
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "device",
						  "enum": [
							"device"
						  ]
						},
						"cache-replica-writes": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"cold-start-empty": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"commit-to-device": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"commit-min-size": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 8388608,
						  "description": "",
						  "dynamic": false
						},
						"compression": {
						  "type": "string",
						  "description": "",
						  "dynamic": true,
						  "default": "none",
						  "enum": [
							"none",
							"lz4",
							"snappy",
							"zstd"
						  ]
						},
						"compression-acceleration": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1,
						  "maximum": 65537,
						  "description": "",
						  "dynamic": true
						},
						"compression-level": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1,
						  "maximum": 9,
						  "description": "",
						  "dynamic": true
						},
						"data-in-memory": {
						  "type": "boolean",
						  "default": true,
						  "description": "",
						  "dynamic": false
						},
						"defrag-lwm-pct": {
						  "type": "integer",
						  "default": 50,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": true
						},
						"defrag-queue-min": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-startup-minimum": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 99,
						  "description": "",
						  "dynamic": false
						},
						"devices": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"direct-files": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"disable-odsync": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"enable-benchmarks-storage": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"encryption": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "aes-128",
						  "enum": [
							"aes-128",
							"aes-256"
						  ]
						},
						"encryption-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"encryption-old-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"files": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"filesize": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1048576,
						  "maximum": 2199023255552,
						  "description": "",
						  "dynamic": false
						},
						"flush-max-ms": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 1000,
						  "description": "",
						  "dynamic": true
						},
						"max-used-pct": {
						  "type": "integer",
						  "default": 70,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"max-write-cache": {
						  "type": "integer",
						  "default": 67108864,
						  "minimum": 0,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						},
						"min-avail-pct": {
						  "type": "integer",
						  "default": 5,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"post-write-queue": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4096,
						  "description": "",
						  "dynamic": true
						},
						"read-page-cache": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"serialize-tomb-raider": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"sindex-startup-device-scan": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"tomb-raider-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"write-block-size": {
						  "type": "integer",
						  "default": 1048576,
						  "minimum": 1024,
						  "maximum": 8388608,
						  "description": "",
						  "dynamic": false
						}
					  }
					},
					{
					  "type": "object",
					  "additionalProperties": false,
					  "required": [
						"type",
						"files"
					  ],
					  "properties": {
						"type": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "pmem",
						  "enum": [
							"pmem"
						  ]
						},
						"commit-to-device": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"compression": {
						  "type": "string",
						  "description": "",
						  "dynamic": true,
						  "default": "none",
						  "enum": [
							"none",
							"lz4",
							"snappy",
							"zstd"
						  ]
						},
						"compression-acceleration": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1,
						  "maximum": 65537,
						  "description": "",
						  "dynamic": true
						},
						"compression-level": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1,
						  "maximum": 9,
						  "description": "",
						  "dynamic": true
						},
						"defrag-lwm-pct": {
						  "type": "integer",
						  "default": 50,
						  "minimum": 1,
						  "maximum": 99,
						  "description": "",
						  "dynamic": true
						},
						"defrag-queue-min": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						},
						"defrag-startup-minimum": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 0,
						  "maximum": 99,
						  "description": "",
						  "dynamic": false
						},
						"direct-files": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"disable-odsync": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"enable-benchmarks-storage": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": true
						},
						"encryption": {
						  "type": "string",
						  "description": "",
						  "dynamic": false,
						  "default": "aes-128",
						  "enum": [
							"aes-128",
							"aes-256"
						  ]
						},
						"encryption-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"encryption-old-key-file": {
						  "type": "string",
						  "default": "",
						  "description": "",
						  "dynamic": false
						},
						"files": {
						  "type": "array",
						  "items": {
							"type": "string"
						  },
						  "description": "",
						  "dynamic": false,
						  "default": []
						},
						"filesize": {
						  "type": "integer",
						  "default": 0,
						  "minimum": 1048576,
						  "maximum": 2199023255552,
						  "description": "",
						  "dynamic": false
						},
						"flush-max-ms": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 1000,
						  "description": "",
						  "dynamic": true
						},
						"max-used-pct": {
						  "type": "integer",
						  "default": 70,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"max-write-cache": {
						  "type": "integer",
						  "default": 67108864,
						  "minimum": 0,
						  "maximum": 18446744073709551615,
						  "description": "",
						  "dynamic": true
						},
						"min-avail-pct": {
						  "type": "integer",
						  "default": 5,
						  "minimum": 0,
						  "maximum": 100,
						  "description": "",
						  "dynamic": true
						},
						"serialize-tomb-raider": {
						  "type": "boolean",
						  "default": false,
						  "description": "",
						  "dynamic": false
						},
						"tomb-raider-sleep": {
						  "type": "integer",
						  "default": 1000,
						  "minimum": 0,
						  "maximum": 4294967295,
						  "description": "",
						  "dynamic": true
						}
					  }
					}
				  ]
				}
			  }
			}
		  },
		  "mod-lua": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "cache-enabled": {
				"type": "boolean",
				"default": true,
				"description": "",
				"dynamic": false
			  },
			  "user-path": {
				"type": "string",
				"default": "/opt/aerospike/usr/udf/lua",
				"description": "",
				"dynamic": false
			  }
			}
		  },
		  "security": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "enable-quotas": {
				"type": "boolean",
				"default": false,
				"description": "",
				"dynamic": false
			  },
			  "privilege-refresh-period": {
				"type": "integer",
				"default": 300,
				"minimum": 10,
				"maximum": 86400,
				"description": "",
				"dynamic": true
			  },
			  "session-ttl": {
				"type": "integer",
				"default": 86400,
				"minimum": 120,
				"maximum": 864000,
				"description": "",
				"dynamic": true
			  },
			  "tps-weight": {
				"type": "integer",
				"default": 2,
				"minimum": 2,
				"maximum": 20,
				"description": "",
				"dynamic": true
			  },
			  "ldap": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "disable-tls": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "login-threads": {
					"type": "integer",
					"default": 8,
					"minimum": 1,
					"maximum": 64,
					"description": "",
					"dynamic": false
				  },
				  "polling-period": {
					"type": "integer",
					"default": 300,
					"minimum": 0,
					"maximum": 86400,
					"description": "",
					"dynamic": true
				  },
				  "query-base-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "query-user-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "query-user-password-file": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "role-query-base-dn": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "role-query-patterns": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": false,
					"default": []
				  },
				  "role-query-search-ou": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": false
				  },
				  "server": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "tls-ca-file": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "token-hash-method": {
					"type": "string",
					"default": "sha-256",
					"description": "",
					"dynamic": false
				  },
				  "user-dn-pattern": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  },
				  "user-query-pattern": {
					"type": "string",
					"default": "",
					"description": "",
					"dynamic": false
				  }
				}
			  },
			  "log": {
				"type": "object",
				"additionalProperties": false,
				"properties": {
				  "report-authentication": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-data-op": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-role": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-data-op-user": {
					"type": "array",
					"items": {
					  "type": "string"
					},
					"description": "",
					"dynamic": true,
					"default": []
				  },
				  "report-sys-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-user-admin": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  },
				  "report-violation": {
					"type": "boolean",
					"default": false,
					"description": "",
					"dynamic": true
				  }
				}
			  }
			}
		  },
		  "xdr": {
			"type": "object",
			"additionalProperties": false,
			"properties": {
			  "src-id": {
				"type": "integer",
				"default": 0,
				"minimum": 0,
				"maximum": 255,
				"description": "",
				"dynamic": true
			  },
			  "dcs": {
				"type": "array",
				"items": {
				  "type": "object",
				  "additionalProperties": false,
				  "properties": {
					"name": {
					  "type": "string",
					  "default": " ",
					  "description": "",
					  "dynamic": false
					},
					"auth-mode": {
					  "type": "string",
					  "description": "",
					  "dynamic": true,
					  "default": "none",
					  "enum": [
						"none",
						"internal",
						"external",
						"external-insecure",
						"pki"
					  ]
					},
					"auth-password-file": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"auth-user": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"connector": {
					  "type": "boolean",
					  "default": false,
					  "description": "",
					  "dynamic": true
					},
					"max-recoveries-interleaved": {
					  "type": "integer",
					  "default": 0,
					  "minimum": 0,
					  "maximum": 4294967295,
					  "description": "",
					  "dynamic": true
					},
					"node-address-ports": {
					  "type": "array",
					  "items": {
						"type": "string"
					  },
					  "description": "",
					  "dynamic": false,
					  "default": []
					},
					"period-ms": {
					  "type": "integer",
					  "default": 100,
					  "minimum": 5,
					  "maximum": 1000,
					  "description": "",
					  "dynamic": true
					},
					"tls-name": {
					  "type": "string",
					  "default": "",
					  "description": "",
					  "dynamic": true
					},
					"use-alternate-access-address": {
					  "type": "boolean",
					  "default": false,
					  "description": "",
					  "dynamic": true
					},
					"namespaces": {
					  "type": "array",
					  "items": {
						"type": "object",
						"additionalProperties": false,
						"properties": {
						  "name": {
							"type": "string",
							"default": " ",
							"description": "",
							"dynamic": false
						  },
						  "bin-policy": {
							"type": "string",
							"description": "",
							"dynamic": true,
							"default": "all",
							"enum": [
							  "all",
							  "no-bins",
							  "only-changed",
							  "changed-and-specified",
							  "changed-or-specified"
							]
						  },
						  "compression-level": {
							"type": "integer",
							"default": 1,
							"minimum": 1,
							"maximum": 9,
							"description": "",
							"dynamic": true
						  },
						  "compression-threshold": {
							"type": "integer",
							"default": 128,
							"minimum": 128,
							"maximum": 4294967295,
							"description": "",
							"dynamic": true
						  },
						  "delay-ms": {
							"type": "integer",
							"default": 0,
							"minimum": 0,
							"maximum": 5000,
							"description": "",
							"dynamic": true
						  },
						  "enable-compression": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "forward": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "hot-key-ms": {
							"type": "integer",
							"default": 100,
							"minimum": 0,
							"maximum": 5000,
							"description": "",
							"dynamic": true
						  },
						  "ignore-bins": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "ignore-expunges": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ignore-sets": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "max-throughput": {
							"type": "integer",
							"default": 100000,
							"minimum": 0,
							"maximum": 4294967295,
							"description": "",
							"dynamic": true
						  },
						  "remote-namespace": {
							"type": "string",
							"default": "",
							"description": "",
							"dynamic": true
						  },
						  "sc-replication-wait-ms": {
							"type": "integer",
							"default": 100,
							"minimum": 5,
							"maximum": 1000,
							"description": "",
							"dynamic": true
						  },
						  "ship-bins": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "ship-bin-luts": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-nsup-deletes": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-only-specified-sets": {
							"type": "boolean",
							"default": false,
							"description": "",
							"dynamic": true
						  },
						  "ship-sets": {
							"type": "array",
							"items": {
							  "type": "string"
							},
							"description": "",
							"dynamic": true,
							"default": []
						  },
						  "transaction-queue-limit": {
							"type": "integer",
							"default": 16384,
							"minimum": 1024,
							"maximum": 1048576,
							"description": "",
							"dynamic": true
						  },
						  "write-policy": {
							"type": "string",
							"description": "",
							"dynamic": true,
							"default": "auto",
							"enum": [
							  "auto",
							  "update",
							  "replace"
							]
						  }
						}
					  }
					}
				  }
				}
			  }
			}
		  }
		}
	  }
	`,
}
