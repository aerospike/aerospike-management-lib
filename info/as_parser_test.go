package info

import (
	"testing"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v6"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AsParserTestSuite struct {
	suite.Suite
	asinfo   *AsInfo
	ctrl     *gomock.Controller
	mockConn *MockConnection
}

func (s *AsParserTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	mockConnFact := NewMockConnectionFactory(s.ctrl)
	s.mockConn = NewMockConnection(s.ctrl)
	policy := &aero.ClientPolicy{}
	host := &aero.Host{}
	mockConnFact.EXPECT().NewConnection(policy, host).Return(s.mockConn, nil).AnyTimes()
	s.mockConn.EXPECT().IsConnected().Return(true).AnyTimes()
	s.mockConn.EXPECT().Login(policy).Return(nil).AnyTimes()
	s.mockConn.EXPECT().SetTimeout(gomock.Any(), time.Second*100).AnyTimes()
	s.mockConn.EXPECT().Close().Return().AnyTimes()
	s.asinfo = NewAsInfoWithConnFactory(logr.Discard(), host, policy, mockConnFact)
}

func (s *AsParserTestSuite) TestAsInfoGetAsConfig() {
	testCases := []struct {
		context      string
		coreInfoResp map[string]string
		req          []string
		resp         map[string]string
		expected     lib.Stats
	}{
		{
			"service",
			nil,
			[]string{"get-config:context=service"},
			map[string]string{"get-config:context=service": "advertise-ipv6=false;auto-pin=none;batch-index-threads=8;batch-max-buffers-per-queue=255;batch-max-unused-buffers=256"},
			lib.Stats{"service": lib.Stats{"advertise-ipv6": false, "auto-pin": "none", "batch-index-threads": int64(8), "batch-max-buffers-per-queue": int64(255), "batch-max-unused-buffers": int64(256)}},
		},
		{
			"network",
			nil,
			[]string{"get-config:context=network"},
			map[string]string{"get-config:context=network": "service.access-port=0;service.address=any;service.alternate-access-port=0;service.port=3000"},
			lib.Stats{"network": lib.Stats{"service.access-port": int64(0), "service.address": "any", "service.alternate-access-port": int64(0), "service.port": int64(3000)}},
		},
		{
			"namespaces",
			map[string]string{"namespaces": "test;bar"},
			[]string{"get-config:context=namespace;id=test", "get-config:context=namespace;id=bar"},
			map[string]string{"get-config:context=namespace;id=test": "allow-ttl-without-nsup=false;background-query-max-rps=10000;conflict-resolution-policy=generation;conflict-resolve-writes=false", "get-config:context=namespace;id=bar": "allow-ttl-without-nsup=true;background-query-max-rps=10000;conflict-resolution-policy=generation;conflict-resolve-writes=false"},
			lib.Stats{"namespaces": lib.Stats{"test": lib.Stats{"allow-ttl-without-nsup": false, "background-query-max-rps": int64(10000), "conflict-resolution-policy": "generation", "conflict-resolve-writes": false}, "bar": lib.Stats{"allow-ttl-without-nsup": true, "background-query-max-rps": int64(10000), "conflict-resolution-policy": "generation", "conflict-resolve-writes": false}}},
		},
		{
			"sets",
			map[string]string{"namespaces": "test;bar"},
			[]string{"sets/test", "sets/bar"},
			map[string]string{"sets/test": "ns=test:set=testset:objects=1:tombstones=0:memory_data_bytes=311142:device_data_bytes=0:truncate_lut=0:sindexes=0:index_populating=false:truncating=false:disable-eviction=false:enable-index=false:stop-writes-count=1:stop-writes-size=1;", "sets/bar": "ns=test:set=testset:objects=2:tombstones=0:memory_data_bytes=311142:device_data_bytes=0:truncate_lut=0:sindexes=0:index_populating=false:truncating=false:disable-eviction=false:enable-index=false:stop-writes-count=2:stop-writes-size=2;"},
			lib.Stats{"namespaces": lib.Stats{"test": lib.Stats{"sets": lib.Stats{"testset": lib.Stats{"disable-eviction": false, "enable-index": false, "stop-writes-count": int64(1), "stop-writes-size": int64(1)}}}, "bar": lib.Stats{"sets": lib.Stats{"testset": lib.Stats{"disable-eviction": false, "enable-index": false, "stop-writes-count": int64(2), "stop-writes-size": int64(2)}}}}},
		},
		{
			"xdr",
			map[string]string{"build": "4.9.0.35"}, // xdr5 test is below
			[]string{"get-config:context=xdr"},
			map[string]string{"get-config:context=xdr": "enable-xdr=false;enable-change-notification=false;forward-xdr-writes=false;xdr-delete-shipping-enabled=true;xdr-nsup-deletes-enabled=false"},
			lib.Stats{"xdr": lib.Stats{"enable-xdr": false, "enable-change-notification": false, "forward-xdr-writes": false, "xdr-delete-shipping-enabled": true, "xdr-nsup-deletes-enabled": false}},
		},
		{
			"dcs",
			map[string]string{"build": "4.9.0.35", "dcs": "DC1;DC2"},
			[]string{"get-dc-config:context=dc;dc=DC1", "get-dc-config:context=dc;dc=DC2"},
			map[string]string{
				"get-dc-config:context=dc;dc=DC1": "dc-name=DC1:dc-type=aerospike:tls-name=:dc-security-config-file=:dc-ship-bins=true:nodes=1.1.1.1+3000:auth-mode=internal:int-ext-ipmap=:dc-connections=64:dc-connections-idle-ms=55000:dc-use-alternate-services=false:namespaces=",
				"get-dc-config:context=dc;dc=DC2": "dc-name=DC2:dc-type=aerospike:tls-name=:dc-security-config-file=:dc-ship-bins=true:nodes=1.1.1.1+3000:auth-mode=internal:int-ext-ipmap=:dc-connections=64:dc-connections-idle-ms=55000:dc-use-alternate-services=false:namespaces=",
			},
			lib.Stats{"dcs": lib.Stats{
				"DC1": lib.Stats{"dc-name": "DC1", "dc-type": "aerospike", "tls-name": "", "dc-security-config-file": "", "dc-ship-bins": true, "nodes": "1.1.1.1+3000", "auth-mode": "internal", "int-ext-ipmap": "", "dc-connections": int64(64), "dc-connections-idle-ms": int64(55000), "dc-use-alternate-services": false, "namespaces": ""},
				"DC2": lib.Stats{"dc-name": "DC2", "dc-type": "aerospike", "tls-name": "", "dc-security-config-file": "", "dc-ship-bins": true, "nodes": "1.1.1.1+3000", "auth-mode": "internal", "int-ext-ipmap": "", "dc-connections": int64(64), "dc-connections-idle-ms": int64(55000), "dc-use-alternate-services": false, "namespaces": ""},
			}},
		},
		{
			"dcs",
			map[string]string{"build": "5.0.0.0", "dcs": "DC1;DC2"},
			nil,
			nil,
			lib.Stats{},
		},
		{
			"security",
			nil,
			[]string{"get-config:context=security"},
			map[string]string{
				"get-config:context=security": "enable-ldap=false;enable-security=true;ldap-login-threads=8;privilege-refresh-period=300;ldap.disable-tls=false;ldap.polling-period=300",
			},
			lib.Stats{"security": lib.Stats{"enable-ldap": false, "enable-security": true, "ldap-login-threads": int64(8), "privilege-refresh-period": int64(300), "ldap.disable-tls": false, "ldap.polling-period": int64(300)}},
		},
		{
			"logging",
			map[string]string{"logs": "0:/var/log/aerospike.log;1:stderr"},
			[]string{"log/0", "log/1"},
			map[string]string{
				"log/0": "misc:CRITICAL;alloc:CRITICAL;arenax:CRITICAL;hardware:CRITICAL;msg:CRITICAL;rbuffer:CRITICAL;socket:CRITICAL;tls:CRITICAL;vmapx:CRITICAL",
				"log/1": "misc:CRITICAL;alloc:CRITICAL;arenax:CRITICAL;hardware:CRITICAL;msg:CRITICAL;rbuffer:CRITICAL;socket:CRITICAL;tls:CRITICAL;vmapx:CRITICAL",
			},
			lib.Stats{"logging": lib.Stats{
				"/var/log/aerospike.log": lib.Stats{"misc": "CRITICAL", "alloc": "CRITICAL", "arenax": "CRITICAL", "hardware": "CRITICAL", "msg": "CRITICAL", "rbuffer": "CRITICAL", "socket": "CRITICAL", "tls": "CRITICAL", "vmapx": "CRITICAL"},
				"stderr":                 lib.Stats{"misc": "CRITICAL", "alloc": "CRITICAL", "arenax": "CRITICAL", "hardware": "CRITICAL", "msg": "CRITICAL", "rbuffer": "CRITICAL", "socket": "CRITICAL", "tls": "CRITICAL", "vmapx": "CRITICAL"},
			}},
		},
		{
			"racks",
			nil,
			[]string{"racks:"},
			map[string]string{
				"racks:": "ns=test:rack_0=1B;ns=bar:rack_0=2B;",
			},
			lib.Stats{"racks": []lib.Stats{
				{"ns": "test", "rack_0": "1B"},
				{"ns": "bar", "rack_0": "2B"},
			}},
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		s.Run(tc.context, func() {
			// Call GetAsInfo with the input from the test case
			s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "dcs", "sindex/", "logs", "build"}).Return(tc.coreInfoResp, nil)

			if tc.req != nil {
				s.mockConn.EXPECT().RequestInfo(tc.req).Return(tc.resp, nil)
			}

			result, err := s.asinfo.GetAsConfig(tc.context)

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *AsParserTestSuite) TestAsInfoGetAsConfigXDR5Enabled() {
	context := "xdr"
	coreInfoResp := map[string]string{"build": "5.0.0.0"}

	// Call GetAsInfo with the input from the test case
	s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "dcs", "sindex/", "logs", "build"}).Return(coreInfoResp, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr"}).Return(map[string]string{"get-config:context=xdr": "dcs=DC1,DC2;src-id=0;trace-sample=0"}, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr;dc=DC1"}).Return(map[string]string{"get-config:context=xdr;dc=DC1": "auth-mode=none;auth-password-file=null;auth-user=null;connector=false;max-recoveries-interleaved=0;node-address-port=;period-ms=100;tls-name=null;use-alternate-access-address=false;namespaces=test"}, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr;dc=DC2"}).Return(map[string]string{"get-config:context=xdr;dc=DC2": "auth-mode=none;auth-password-file=null;auth-user=null;connector=false;max-recoveries-interleaved=0;node-address-port=;period-ms=100;tls-name=null;use-alternate-access-address=false;namespaces=bar"}, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr;dc=DC2;namespace=bar", "get-config:context=xdr;dc=DC1;namespace=test"}).Return(map[string]string{"get-config:context=xdr;dc=DC1;namespace=test": "enabled=true;bin-policy=all;compression-level=1;compression-threshold=128;delay-ms=0;enable-compression=false;forward=false;hot-key-ms=100;ignored-bins=", "get-config:context=xdr;dc=DC2;namespace=bar": "enabled=true;bin-policy=all;compression-level=1;compression-threshold=128;delay-ms=0;enable-compression=false;forward=false;hot-key-ms=100;ignored-bins="}, nil)

	expected := lib.Stats{"xdr": lib.Stats{
		"src-id":       int64(0),
		"trace-sample": int64(0),
		"dcs": lib.Stats{
			"DC1": lib.Stats{"auth-mode": "none", "auth-password-file": "null", "auth-user": "null", "connector": false, "max-recoveries-interleaved": int64(0), "node-address-port": "", "period-ms": int64(100), "tls-name": "null", "use-alternate-access-address": false, "namespaces": lib.Stats{
				"test": lib.Stats{"enabled": true, "bin-policy": "all", "compression-level": int64(1), "compression-threshold": int64(128), "delay-ms": int64(0), "enable-compression": false, "forward": false, "hot-key-ms": int64(100), "ignored-bins": ""},
			}},
			"DC2": lib.Stats{"auth-mode": "none", "auth-password-file": "null", "auth-user": "null", "connector": false, "max-recoveries-interleaved": int64(0), "node-address-port": "", "period-ms": int64(100), "tls-name": "null", "use-alternate-access-address": false, "namespaces": lib.Stats{
				"bar": lib.Stats{"enabled": true, "bin-policy": "all", "compression-level": int64(1), "compression-threshold": int64(128), "delay-ms": int64(0), "enable-compression": false, "forward": false, "hot-key-ms": int64(100), "ignored-bins": ""},
			}},
		},
	}}

	result, err := s.asinfo.GetAsConfig(context)

	s.Assert().Nil(err)
	s.Assert().Equal(expected, result)
}

func (s *AsParserTestSuite) TestAsInfoGetAsConfigXDR5Disabled() {
	context := "xdr"
	coreInfoResp := map[string]string{"build": "5.0.0.0"}

	// Call GetAsInfo with the input from the test case
	s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "dcs", "sindex/", "logs", "build"}).Return(coreInfoResp, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr"}).Return(map[string]string{"get-config:context=xdr": "dcs=;src-id=0;trace-sample=0"}, nil)

	expected := lib.Stats{"xdr": lib.Stats{
		"src-id":       int64(0),
		"trace-sample": int64(0),
		"dcs":          lib.Stats{},
	}}
	result, err := s.asinfo.GetAsConfig(context)

	s.Assert().Nil(err)
	s.Assert().Equal(expected, result)
}

func TestAsParserTestSuite(t *testing.T) {
	suite.Run(t, new(AsParserTestSuite))
}
