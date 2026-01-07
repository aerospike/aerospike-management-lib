package info

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	aero "github.com/aerospike/aerospike-client-go/v8"
	lib "github.com/aerospike/aerospike-management-lib"
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
			map[string]string{"logs": "0:/var/log/aerospike.log"},
			[]string{"log/0"},
			map[string]string{
				"log/0": "misc:CRITICAL;alloc:CRITICAL;arenax:CRITICAL;hardware:CRITICAL;msg:CRITICAL;rbuffer:CRITICAL;socket:CRITICAL;tls:CRITICAL;vmapx:CRITICAL",
			},
			lib.Stats{"logging": lib.Stats{
				"/var/log/aerospike.log": lib.Stats{"misc": "CRITICAL", "alloc": "CRITICAL", "arenax": "CRITICAL", "hardware": "CRITICAL", "msg": "CRITICAL", "rbuffer": "CRITICAL", "socket": "CRITICAL", "tls": "CRITICAL", "vmapx": "CRITICAL"},
			}},
		},
		{
			"racks",
			map[string]string{cmdMetaEdition: "Aerospike Enterprise Edition"},
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
			s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "get-config:context=xdr", "sindex/", "logs", "build", "edition"}).Return(tc.coreInfoResp, nil)

			if tc.req != nil {
				s.mockConn.EXPECT().RequestInfo(tc.req).Return(tc.resp, nil)
			}

			result, err := s.asinfo.GetAsConfig(tc.context)

			s.Assert().Nil(err)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *AsParserTestSuite) TestAsInfoGetAsConfigXDREnabled() {
	context := "xdr"
	coreInfoResp := map[string]string{"build": "6.4.0.0"}

	// Call GetAsInfo with the input from the test case
	s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "get-config:context=xdr", "sindex/", "logs", "build", "edition"}).Return(coreInfoResp, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr"}).Return(map[string]string{"get-config:context=xdr": "dcs=DC1,DC2;src-id=0;trace-sample=0"}, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr;dc=DC1"}).Return(map[string]string{"get-config:context=xdr;dc=DC1": "auth-mode=none;auth-password-file=null;auth-user=null;connector=false;max-recoveries-interleaved=0;node-address-port=;period-ms=100;tls-name=null;use-alternate-access-address=false;namespaces=test"}, nil)
	s.mockConn.EXPECT().RequestInfo([]string{"get-config:context=xdr;dc=DC2"}).Return(map[string]string{"get-config:context=xdr;dc=DC2": "auth-mode=none;auth-password-file=null;auth-user=null;connector=false;max-recoveries-interleaved=0;node-address-port=;period-ms=100;tls-name=null;use-alternate-access-address=false;namespaces=bar"}, nil)
	s.mockConn.EXPECT().RequestInfo(gomock.Any()).DoAndReturn(
		func(cmds ...string) (map[string]string, aero.Error) {
			expectedCmds := []string{
				"get-config:context=xdr;dc=DC1;namespace=test",
				"get-config:context=xdr;dc=DC2;namespace=bar",
			}
			s.ElementsMatch(expectedCmds, cmds)

			return map[string]string{
				"get-config:context=xdr;dc=DC1;namespace=test": "enabled=true;bin-policy=all;compression-level=1;compression-threshold=128;delay-ms=0;enable-compression=false;forward=false;hot-key-ms=100;ignored-bins=",
				"get-config:context=xdr;dc=DC2;namespace=bar":  "enabled=true;bin-policy=all;compression-level=1;compression-threshold=128;delay-ms=0;enable-compression=false;forward=false;hot-key-ms=100;ignored-bins=",
			}, nil
		},
	)

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

func (s *AsParserTestSuite) TestAsInfoGetAsConfigXDRDisabled() {
	context := "xdr"
	coreInfoResp := map[string]string{"build": "6.4.0.0"}

	// Call GetAsInfo with the input from the test case
	s.mockConn.EXPECT().RequestInfo([]string{"namespaces", "get-config:context=xdr", "sindex/", "logs", "build", "edition"}).Return(coreInfoResp, nil)
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

func (s *AsParserTestSuite) TestBuildValidation() {
	testCases := []struct {
		name    string
		resp    map[string]string
		want    string
		wantErr bool
	}{
		{
			name:    "valid build",
			resp:    map[string]string{"build": "7.1.0.0"},
			want:    "7.1.0.0",
			wantErr: false,
		},
		{
			name:    "empty build",
			resp:    map[string]string{"build": ""},
			wantErr: true,
		},
		{
			name:    "error prefix lower",
			resp:    map[string]string{"build": "error: something bad"},
			wantErr: true,
		},
		{
			name:    "error prefix upper",
			resp:    map[string]string{"build": "ERROR: something bad"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.mockConn.EXPECT().RequestInfo("build").Return(tc.resp, nil)

			got, err := s.asinfo.Build()
			if tc.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tc.want, got)
		})
	}
}

func (s *AsParserTestSuite) TestNamespaceConfigCmd() {
	testCases := []struct {
		name  string
		build string
		ns    string
		want  string
	}{
		{
			name:  "pre-7.2 uses id",
			build: "7.1.0.0",
			ns:    "test",
			want:  "get-config:context=namespace;id=test",
		},
		{
			name:  "7.2 and above uses namespace",
			build: "7.2.0.0",
			ns:    "test",
			want:  "get-config:context=namespace;namespace=test",
		},
		{
			name:  "invalid build falls back to id",
			build: "not-a-version",
			ns:    "test",
			want:  "get-config:context=namespace;id=test",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			got := NamespaceConfigCmd(tc.ns, tc.build)
			s.Equal(tc.want, got)
		})
	}
}

func TestAsParserTestSuite(t *testing.T) {
	suite.Run(t, new(AsParserTestSuite))
}

func TestNewAsInfo_ConnectionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnFact := NewMockConnectionFactory(ctrl)
	expectedErr := &aero.AerospikeError{ResultCode: 1}
	policy := &aero.ClientPolicy{}
	host := &aero.Host{}
	mockConnFact.EXPECT().NewConnection(policy, host).Return(nil, expectedErr).AnyTimes()

	asinfo := NewAsInfoWithConnFactory(logr.Discard(), host, policy, mockConnFact)

	r, actualErr := asinfo.RequestInfo("connection will fail")

	if r != nil {
		t.Errorf("Expected nil response, got %v", r)
	}

	if errors.Is(actualErr, expectedErr) == false {
		t.Errorf("Expected error %v, got %v", expectedErr, actualErr)
	}
}

func TestNewAsInfo_NotAuthenticatedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnFact := NewMockConnectionFactory(ctrl)
	mockConn := NewMockConnection(ctrl)
	policy := &aero.ClientPolicy{}
	host := &aero.Host{}
	mockConnFact.EXPECT().NewConnection(policy, host).Return(mockConn, nil).AnyTimes()
	mockConn.EXPECT().IsConnected().Return(true).AnyTimes()
	mockConn.EXPECT().Login(policy).Return(nil).AnyTimes()
	mockConn.EXPECT().SetTimeout(gomock.Any(), time.Second*100).AnyTimes()
	mockConn.EXPECT().Close().Return().AnyTimes()
	mockConn.EXPECT().RequestInfo([]string{"auth will fail"}).Return(map[string]string{"ERROR:80:not authenticated": ""}, nil)

	maxInfoRetries = 1 // Should be configurable
	asinfo := NewAsInfoWithConnFactory(logr.Discard(), host, policy, mockConnFact)

	r, acutalErr := asinfo.RequestInfo("auth will fail")

	if r != nil {
		t.Errorf("Expected nil response, got %v", r)
	}

	if errors.Is(acutalErr, ErrConnNotAuthenticated) == false {
		t.Errorf("Expected error %v, got %v", ErrConnNotAuthenticated, acutalErr)
	}
}
