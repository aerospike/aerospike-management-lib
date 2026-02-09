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

// coreInfoCommandsAsAny returns getCoreInfoCommands() as []any for gomock expectations.
func coreInfoCommandsAsAny() []any {
	cmds := getCoreInfoCommands()

	out := make([]any, len(cmds))
	for i := range cmds {
		out[i] = cmds[i]
	}

	return out
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
			s.mockConn.EXPECT().RequestInfo(coreInfoCommandsAsAny()...).Return(tc.coreInfoResp, nil)

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
	s.mockConn.EXPECT().RequestInfo(coreInfoCommandsAsAny()...).Return(coreInfoResp, nil)
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
	s.mockConn.EXPECT().RequestInfo(coreInfoCommandsAsAny()...).Return(coreInfoResp, nil)
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

func TestExtractAddressesFromNodeList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple peers",
			input:    "[[A1,,[10.128.0.71:31207]],[A2,,[10.128.0.98:30352]]]",
			expected: []string{"10.128.0.71:31207", "10.128.0.98:30352"},
		},
		{
			name:     "single peer",
			input:    "[[A0,,[10.128.0.94:32354]]]",
			expected: []string{"10.128.0.94:32354"},
		},
		{
			name:     "empty array",
			input:    "[]",
			expected: []string{},
		},
		{
			name:     "invalid - no brackets",
			input:    "invalid",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractAddressesFromNodeList(tc.input)

			if len(tc.expected) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d addresses, got %d", len(tc.expected), len(result))
				return
			}

			for i, addr := range tc.expected {
				if result[i] != addr {
					t.Errorf("Expected address[%d] = %q, got %q", i, addr, result[i])
				}
			}
		})
	}
}

// TestParseNodeEndpointList_Endpoints exercises ParseNodeEndpointList on the full peers/alumni
// info format (<gen>,<port>,[[...]]) and asserts on .Endpoints() (address list only).
func TestParseNodeEndpointList_Endpoints(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		// Non-TLS cluster scenarios
		{
			name:     "non-TLS cluster - peers-clear-std with multiple nodes",
			input:    "20,3000,[[A1,,[10.128.0.71:31207]],[A2,,[10.128.0.98:30352]],[A3,,[10.128.0.97:30256]],[A4,,[10.128.0.64:32136]]]",
			expected: []string{"10.128.0.71:31207", "10.128.0.98:30352", "10.128.0.97:30256", "10.128.0.64:32136"},
		},
		{
			name:     "non-TLS cluster - peers-tls-std empty (TLS not enabled)",
			input:    "20,3000,[]",
			expected: []string{},
		},
		{
			name:     "non-TLS cluster - peers-tls-alt empty (TLS not enabled)",
			input:    "11,3000,[]",
			expected: []string{},
		},
		{
			name:     "non-TLS cluster - alumni-tls-std empty (TLS not enabled)",
			input:    "5,3000,[]",
			expected: []string{},
		},
		{
			name:     "non-TLS cluster - alumni-clear-std with nodes",
			input:    "10,3000,[[BB9050011AC4202,,[172.17.0.5]]]",
			expected: []string{"172.17.0.5"},
		},
		{
			name:     "non-TLS cluster - peers-clear-alt with external IPs",
			input:    "20,3000,[[A1,,[34.134.36.120:31207]],[A2,,[34.56.228.239:30352]],[A3,,[34.30.117.181:30256]],[A4,,[34.28.75.232:32136]]]",
			expected: []string{"34.134.36.120:31207", "34.56.228.239:30352", "34.30.117.181:30256", "34.28.75.232:32136"},
		},
		// TLS cluster scenarios
		{
			name:     "TLS cluster - peers-tls-std with tls-name",
			input:    "15,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333]],[BB9050011AC4203,clusternode,[172.17.0.6:4333]]]",
			expected: []string{"172.17.0.5:4333", "172.17.0.6:4333"},
		},
		{
			name:     "TLS cluster - peers-tls-alt with tls-name",
			input:    "15,4333,[[BB9050011AC4202,clusternode,[34.134.36.120:4333]],[BB9050011AC4203,clusternode,[34.56.228.239:4333]]]",
			expected: []string{"34.134.36.120:4333", "34.56.228.239:4333"},
		},
		{
			name:     "TLS cluster - alumni-tls-std with tls-name",
			input:    "10,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333]]]",
			expected: []string{"172.17.0.5:4333"},
		},
		{
			name:     "TLS cluster - single node with tls-name",
			input:    "5,4333,[[A0,my-tls-node,[10.128.0.94:4333]]]",
			expected: []string{"10.128.0.94:4333"},
		},
		// Edge cases
		{
			name:     "single node without tls-name",
			input:    "8,3000,[[A0,,[10.128.0.94:32354]]]",
			expected: []string{"10.128.0.94:32354"},
		},
		{
			name:     "node with multiple addresses",
			input:    "10,3000,[[A0,,[10.128.0.94:32354,192.168.1.1:3000]]]",
			expected: []string{"10.128.0.94:32354", "192.168.1.1:3000"},
		},
		{
			name:     "TLS node with multiple addresses",
			input:    "10,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333,172.17.0.5:3000]]]",
			expected: []string{"172.17.0.5:4333", "172.17.0.5:3000"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "invalid - no bracket found",
			input:    "invalid",
			expected: []string{},
		},
		{
			name:     "invalid - malformed response",
			input:    "20,3000",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseNodeEndpointList(tc.input).Endpoints()

			if len(tc.expected) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d addresses, got %d", len(tc.expected), len(result))
				return
			}

			for i, addr := range tc.expected {
				if result[i] != addr {
					t.Errorf("Expected address[%d] = %q, got %q", i, addr, result[i])
				}
			}
		})
	}
}

func TestExtractAddressesFromNodeEntry(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard entry with empty tls-name",
			input:    "[A1,,[10.128.0.71:31207]]",
			expected: []string{"10.128.0.71:31207"},
		},
		{
			name:     "entry with tls-name",
			input:    "[A1,my-tls,[10.128.0.71:31207]]",
			expected: []string{"10.128.0.71:31207"},
		},
		{
			name:     "entry with multiple addresses",
			input:    "[A1,,[10.128.0.71:31207,192.168.1.1:3000]]",
			expected: []string{"10.128.0.71:31207", "192.168.1.1:3000"},
		},
		{
			name:     "empty addresses",
			input:    "[A1,,]",
			expected: []string{},
		},
		{
			name:     "invalid - no brackets",
			input:    "A1,,[10.128.0.71:31207]",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractAddressesFromNodeEntry(tc.input)

			if len(tc.expected) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d addresses, got %d", len(tc.expected), len(result))
				return
			}

			for i, addr := range tc.expected {
				if result[i] != addr {
					t.Errorf("Expected address[%d] = %q, got %q", i, addr, result[i])
				}
			}
		})
	}
}

func TestParseNodeEndpointList(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		expectedGeneration int
		expectedPort       int
		expectedNodes      []NodeEndpoint
	}{
		// Non-TLS cluster scenarios
		{
			name:               "non-TLS cluster - peers-clear-std with multiple nodes",
			input:              "20,3000,[[A1,,[10.128.0.71:31207]],[A2,,[10.128.0.98:30352]]]",
			expectedGeneration: 20,
			expectedPort:       3000,
			expectedNodes: []NodeEndpoint{
				{NodeID: "A1", TLSName: "", Endpoints: []string{"10.128.0.71:31207"}},
				{NodeID: "A2", TLSName: "", Endpoints: []string{"10.128.0.98:30352"}},
			},
		},
		{
			name:               "non-TLS cluster - peers-tls-std empty (TLS not enabled)",
			input:              "20,3000,[]",
			expectedGeneration: 20,
			expectedPort:       3000,
			expectedNodes:      []NodeEndpoint{},
		},
		{
			name:               "non-TLS cluster - peers-tls-alt empty (TLS not enabled)",
			input:              "11,3000,[]",
			expectedGeneration: 11,
			expectedPort:       3000,
			expectedNodes:      []NodeEndpoint{},
		},
		{
			name:               "non-TLS cluster - alumni-clear-std",
			input:              "10,3000,[[BB9050011AC4202,,[172.17.0.5:3000]]]",
			expectedGeneration: 10,
			expectedPort:       3000,
			expectedNodes: []NodeEndpoint{
				{NodeID: "BB9050011AC4202", TLSName: "", Endpoints: []string{"172.17.0.5:3000"}},
			},
		},

		// TLS cluster scenarios
		{
			name:               "TLS cluster - peers-tls-std with tls-name",
			input:              "15,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333]]]",
			expectedGeneration: 15,
			expectedPort:       4333,
			expectedNodes: []NodeEndpoint{
				{NodeID: "BB9050011AC4202", TLSName: "clusternode", Endpoints: []string{"172.17.0.5:4333"}},
			},
		},
		{
			name:               "TLS cluster - peers-tls-std with multiple nodes",
			input:              "15,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333]],[BB9050011AC4203,clusternode,[172.17.0.6:4333]]]",
			expectedGeneration: 15,
			expectedPort:       4333,
			expectedNodes: []NodeEndpoint{
				{NodeID: "BB9050011AC4202", TLSName: "clusternode", Endpoints: []string{"172.17.0.5:4333"}},
				{NodeID: "BB9050011AC4203", TLSName: "clusternode", Endpoints: []string{"172.17.0.6:4333"}},
			},
		},
		{
			name:               "TLS cluster - alumni-tls-std with tls-name",
			input:              "10,4333,[[BB9050011AC4202,my-tls,[172.17.0.5:4333]]]",
			expectedGeneration: 10,
			expectedPort:       4333,
			expectedNodes: []NodeEndpoint{
				{NodeID: "BB9050011AC4202", TLSName: "my-tls", Endpoints: []string{"172.17.0.5:4333"}},
			},
		},

		// Mixed/Edge cases
		{
			name:               "node with multiple addresses",
			input:              "5,3000,[[A0,,[10.128.0.94:32354,192.168.1.1:3000]]]",
			expectedGeneration: 5,
			expectedPort:       3000,
			expectedNodes: []NodeEndpoint{
				{NodeID: "A0", TLSName: "", Endpoints: []string{"10.128.0.94:32354", "192.168.1.1:3000"}},
			},
		},
		{
			name:               "TLS node with multiple addresses",
			input:              "10,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333,172.17.0.5:3000]]]",
			expectedGeneration: 10,
			expectedPort:       4333,
			expectedNodes: []NodeEndpoint{
				{NodeID: "BB9050011AC4202", TLSName: "clusternode", Endpoints: []string{"172.17.0.5:4333", "172.17.0.5:3000"}},
			},
		},
		{
			name:               "empty string",
			input:              "",
			expectedGeneration: 0,
			expectedPort:       0,
			expectedNodes:      []NodeEndpoint{},
		},
		{
			name:               "invalid - malformed",
			input:              "invalid",
			expectedGeneration: 0,
			expectedPort:       0,
			expectedNodes:      []NodeEndpoint{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseNodeEndpointList(tc.input)

			if result.Generation != tc.expectedGeneration {
				t.Errorf("Expected generation %d, got %d", tc.expectedGeneration, result.Generation)
			}

			if result.DefaultPort != tc.expectedPort {
				t.Errorf("Expected port %d, got %d", tc.expectedPort, result.DefaultPort)
			}

			if len(result.Nodes) != len(tc.expectedNodes) {
				t.Errorf("Expected %d nodes, got %d", len(tc.expectedNodes), len(result.Nodes))
				return
			}

			for i, expectedNode := range tc.expectedNodes {
				actualNode := result.Nodes[i]
				if actualNode.NodeID != expectedNode.NodeID {
					t.Errorf("Node[%d]: expected NodeID %q, got %q", i, expectedNode.NodeID, actualNode.NodeID)
				}

				if actualNode.TLSName != expectedNode.TLSName {
					t.Errorf("Node[%d]: expected TLSName %q, got %q", i, expectedNode.TLSName, actualNode.TLSName)
				}

				if len(actualNode.Endpoints) != len(expectedNode.Endpoints) {
					t.Errorf("Node[%d]: expected %d endpoints, got %d", i, len(expectedNode.Endpoints), len(actualNode.Endpoints))
					continue
				}

				for j, expectedAddr := range expectedNode.Endpoints {
					if actualNode.Endpoints[j] != expectedAddr {
						t.Errorf("Node[%d].Endpoints[%d]: expected %q, got %q", i, j, expectedAddr, actualNode.Endpoints[j])
					}
				}
			}
		})
	}
}

func TestNodeEndpointListEndpoints(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple nodes",
			input:    "20,3000,[[A1,,[10.128.0.71:31207]],[A2,,[10.128.0.98:30352]]]",
			expected: []string{"10.128.0.71:31207", "10.128.0.98:30352"},
		},
		{
			name:     "node with multiple addresses",
			input:    "5,3000,[[A0,,[10.128.0.94:32354,192.168.1.1:3000]]]",
			expected: []string{"10.128.0.94:32354", "192.168.1.1:3000"},
		},
		{
			name:     "empty nodes",
			input:    "20,3000,[]",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseNodeEndpointList(tc.input)
			endpoints := result.Endpoints()

			if len(tc.expected) == 0 && len(endpoints) == 0 {
				return
			}

			if len(endpoints) != len(tc.expected) {
				t.Errorf("Expected %d endpoints, got %d", len(tc.expected), len(endpoints))
				return
			}

			for i, addr := range tc.expected {
				if endpoints[i] != addr {
					t.Errorf("Expected endpoint[%d] = %q, got %q", i, addr, endpoints[i])
				}
			}
		})
	}
}

func TestParseNodeEndpointListAsStats(t *testing.T) {
	testCases := []struct {
		name                   string
		cmd                    string
		input                  string
		expectedGeneration     int
		expectedPort           int
		expectedEndpointsCount int
		expectedNodesCount     int
	}{
		{
			name:                   "non-TLS cluster - peers-clear-std with data",
			cmd:                    cmdMetaPeerClearStd,
			input:                  "20,3000,[[A1,,[10.128.0.71:31207]],[A2,,[10.128.0.98:30352]],[A3,,[10.128.0.97:30256]],[A4,,[10.128.0.64:32136]]]",
			expectedGeneration:     20,
			expectedPort:           3000,
			expectedEndpointsCount: 4,
			expectedNodesCount:     4,
		},
		{
			name:                   "non-TLS cluster - peers-tls-std empty (TLS not enabled)",
			cmd:                    cmdMetaPeerTLSStd,
			input:                  "20,3000,[]",
			expectedGeneration:     20,
			expectedPort:           3000,
			expectedEndpointsCount: 0,
			expectedNodesCount:     0,
		},
		{
			name:                   "non-TLS cluster - peers-tls-alt empty (TLS not enabled)",
			cmd:                    cmdMetaPeerTLSAlt,
			input:                  "11,3000,[]",
			expectedGeneration:     11,
			expectedPort:           3000,
			expectedEndpointsCount: 0,
			expectedNodesCount:     0,
		},
		{
			name:                   "TLS cluster - peers-tls-std with data",
			cmd:                    cmdMetaPeerTLSStd,
			input:                  "15,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333]],[BB9050011AC4203,clusternode,[172.17.0.6:4333]]]",
			expectedGeneration:     15,
			expectedPort:           4333,
			expectedEndpointsCount: 2,
			expectedNodesCount:     2,
		},
		{
			name:                   "peers-clear-alt with external IPs",
			cmd:                    cmdMetaPeerClearAlt,
			input:                  "20,3000,[[A1,,[34.134.36.120:31207]],[A2,,[34.56.228.239:30352]]]",
			expectedGeneration:     20,
			expectedPort:           3000,
			expectedEndpointsCount: 2,
			expectedNodesCount:     2,
		},
		{
			name:                   "alumni-clear-std with data",
			cmd:                    cmdMetaAlumniClearStd,
			input:                  "10,3000,[[BB9050011AC4202,,[172.17.0.5:3000]]]",
			expectedGeneration:     10,
			expectedPort:           3000,
			expectedEndpointsCount: 1,
			expectedNodesCount:     1,
		},
		{
			name:                   "empty response",
			cmd:                    cmdMetaPeerClearStd,
			input:                  "",
			expectedGeneration:     0,
			expectedPort:           0,
			expectedEndpointsCount: 0,
			expectedNodesCount:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rawMap := map[string]string{tc.cmd: tc.input}
			result := parseNodeEndpointListAsStats(rawMap, tc.cmd)

			// Check if empty response
			if tc.input == "" {
				if len(result) != 0 {
					t.Errorf("Expected empty stats for empty input, got %v", result)
				}

				return
			}

			// Check generation
			if gen, ok := result["generation"].(int); !ok || gen != tc.expectedGeneration {
				t.Errorf("Expected generation %d, got %v", tc.expectedGeneration, result["generation"])
			}

			// Check default_port
			if port, ok := result["default_port"].(int); !ok || port != tc.expectedPort {
				t.Errorf("Expected default_port %d, got %v", tc.expectedPort, result["default_port"])
			}

			// Check endpoints count
			endpoints, _ := result["endpoints"].([]string)
			if len(endpoints) != tc.expectedEndpointsCount {
				t.Errorf("Expected %d endpoints, got %d", tc.expectedEndpointsCount, len(endpoints))
			}

			// Check nodes count
			nodes, _ := result["nodes"].([]lib.Stats)
			if len(nodes) != tc.expectedNodesCount {
				t.Errorf("Expected %d nodes, got %d", tc.expectedNodesCount, len(nodes))
			}
		})
	}
}

func TestGetEndpointsFromStats(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name: "valid stats with endpoints",
			input: lib.Stats{
				"generation":   20,
				"default_port": 3000,
				"endpoints":    []string{"10.128.0.71:31207", "10.128.0.98:30352"},
				"nodes":        []lib.Stats{},
			},
			expected: []string{"10.128.0.71:31207", "10.128.0.98:30352"},
		},
		{
			name: "stats with empty endpoints (non-TLS cluster querying TLS endpoint)",
			input: lib.Stats{
				"generation":   20,
				"default_port": 3000,
				"endpoints":    []string{},
				"nodes":        []lib.Stats{},
			},
			expected: []string{},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "empty stats",
			input:    lib.Stats{},
			expected: []string{},
		},
		{
			name:     "wrong type - not lib.Stats",
			input:    "not a stats map",
			expected: []string{},
		},
		{
			name: "endpoints is wrong type",
			input: lib.Stats{
				"endpoints": "not a slice",
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getEndpointsFromStats(tc.input)

			if len(tc.expected) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d endpoints, got %d", len(tc.expected), len(result))
				return
			}

			for i, addr := range tc.expected {
				if result[i] != addr {
					t.Errorf("Expected endpoint[%d] = %q, got %q", i, addr, result[i])
				}
			}
		})
	}
}

func TestParseNodeEndpointListAsStats_NodeDetails(t *testing.T) {
	// Test that node details are properly parsed
	input := "10,4333,[[BB9050011AC4202,clusternode,[172.17.0.5:4333,172.17.0.5:3000]],[A1,,[10.128.0.71:31207]]]"
	rawMap := map[string]string{cmdMetaPeerTLSStd: input}
	result := parseNodeEndpointListAsStats(rawMap, cmdMetaPeerTLSStd)

	nodes, ok := result["nodes"].([]lib.Stats)
	if !ok {
		t.Fatal("Expected nodes to be []lib.Stats")
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}

	// Check first node (TLS enabled)
	node1 := nodes[0]
	if node1["node_id"] != "BB9050011AC4202" {
		t.Errorf("Node 1: expected node_id 'BB9050011AC4202', got %q", node1["node_id"])
	}

	if node1["tls_name"] != "clusternode" {
		t.Errorf("Node 1: expected tls_name 'clusternode', got %q", node1["tls_name"])
	}

	endpoints1, _ := node1["endpoints"].([]string)
	if len(endpoints1) != 2 {
		t.Errorf("Node 1: expected 2 endpoints, got %d", len(endpoints1))
	}

	// Check second node (no TLS)
	node2 := nodes[1]
	if node2["node_id"] != "A1" {
		t.Errorf("Node 2: expected node_id 'A1', got %q", node2["node_id"])
	}

	if node2["tls_name"] != "" {
		t.Errorf("Node 2: expected empty tls_name, got %q", node2["tls_name"])
	}

	endpoints2, _ := node2["endpoints"].([]string)
	if len(endpoints2) != 1 {
		t.Errorf("Node 2: expected 1 endpoint, got %d", len(endpoints2))
	}
}
