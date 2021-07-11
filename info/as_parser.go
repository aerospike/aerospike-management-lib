package info

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	aero "github.com/ashishshinde/aerospike-client-go/v5"
	ast "github.com/ashishshinde/aerospike-client-go/v5/types"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/bcrypt"
	log "github.com/inconshreveable/log15"
)

type ClusterAsStat = lib.Stats

type NodeAsStats = lib.Stats

var pkglog = log.New(log.Ctx{"module": "lib.info"})

// InvalidNamespaceErr specifies that the namespace is invalid on the cluster.
var InvalidNamespaceErr = fmt.Errorf("invalid namespace")

// InvalidDCErr specifies that the dc is invalid on the cluster.
var InvalidDCErr = fmt.Errorf("invalid dc")

const (
	_DEFAULT_TIMEOUT = 2 * time.Second

	_STAT        = "statistics"     // Stat
	_STAT_XDR    = "statistics/xdr" // StatXdr
	_STAT_NS     = "namespace/"     // StatNamespace
	_STAT_DC     = "dc/"            // StatDC
	_STAT_SET    = "sets/"          // StatSets
	_STAT_BIN    = "bins/"          // StatBins
	_STAT_SINDEX = "sindex/"        // StatSindex

	_STAT_NS_NAMES = "namespaces" // StatNamespaces
	_STAT_DC_NAMES = "dcs"        // StatDcs need dc names
	_STAT_LOG_IDS  = "logs"       // StatLogs need logging id

	_CONFIG_NETWORK   = "get-config:context=network"       // ConfigNetwork
	_CONFIG_SERVICE   = "get-config:context=service"       // ConfigService
	_CONFIG_NAMESPACE = "get-config:context=namespace;id=" // ConfigNamespace
	_CONFIG_XDR       = "get-config:context=xdr"           // ConfigXdr
	_CONFIG_SECURITY  = "get-config:context=security"      // ConfigSecurity
	_CONFIG_DC        = "get-dc-config:context=dc:dc="     // ConfigDC
	_CONFIG_MCAST     = "mcast"                            // ConfigMulticast
	_CONFIG_MESH      = "mesh"                             // ConfigMesh
	_CONFIG_RACKS     = "racks:"                           // ConfigRacks
	_CONFIG_LOGGING   = "log/"                             // ConfigLog

	_LATENCY    = "latency:"
	_THROUGHPUT = "throughput:"

	_META_BUILD              = "build"              // Build
	_META_VERSION            = "version"            // Version
	_META_BUILD_OS           = "build_os"           // BUILD OS
	_META_NODE_ID            = "node"               // NodeID
	_META_CLUSTER_NAME       = "cluster-name"       // Cluster Name
	_META_SERVICE            = "service"            // Service
	_META_SERVICES           = "services"           // Services
	_META_SERVICES_ALUMNI    = "services-alumni"    // ServicesAlumni
	_META_SERVICES_ALTERNATE = "services-alternate" // ServiceAlternate
	_META_FEATURES           = "features"           // Features
	_META_EDITION            = "edition"            // Edition
)

// other metainfos
// "cluster-generation", "partition-generation", "build_time",
// "udf-list", "cluster-name", "service-clear-std", "service-tls-std",

// Aerospike Config Context
const (
	ConfigServiceContext   = "service"
	ConfigNetworkContext   = "network"
	ConfigNamespaceContext = "namespace"
	ConfigSetContext       = "set"
	ConfigXDRContext       = "xdr"
	ConfigDCContext        = "dc"
	ConfigSecurityContext  = "security"
	ConfigLoggingContext   = "logging"
	ConfigRacksContext     = "racks"
)

const (
	_ConfigDCNames        = "dc_names"
	_ConfigNamespaceNames = "namespace_names"
	_ConfigLogIDs         = "log_ids"
)

//var asCmds = []string{"statistics", "configs", "metadata", "latency", "throughput", "hist-dump", "udf-list", "udf-get"}

var asCmds = []string{"statistics", "configs", "metadata", "latency", "throughput"}

var networkTLSNameRe = regexp.MustCompile("^tls\\[([0-9]+)].name$")

// AsInfo provides info calls on an aerospike cluster.
type AsInfo struct {
	policy *aero.ClientPolicy
	host   *aero.Host
	conn   *aero.Connection
	mutex  sync.Mutex

	log log.Logger
}

func NewAsInfo(h *aero.Host, cp *aero.ClientPolicy) *AsInfo {
	return &AsInfo{
		host:   h,
		policy: cp,
		conn:   nil,
		log:    pkglog.New(log.Ctx{"node": h}),
	}
}

var maxInfoRetries = 3
var asTimeout = time.Second * 100

// RequestInfo get aerospike info
func (info *AsInfo) RequestInfo(cmd ...string) (result map[string]string, err error) {
	if len(cmd) == 0 {
		return map[string]string{}, nil
	}
	for i := 0; i < maxInfoRetries; i++ {
		result, err = info.doInfo(cmd...)
		if err == nil {
			return result, nil
		}
		// TODO: only retry for EOF or Timeout errors
	}
	return result, err
}

// AllConfigs returns all the dynamic configurations of the node.
//
// The returned map can be converted to asconfig.Conf.
func (info *AsInfo) AllConfigs() (map[string]interface{}, error) {
	key := "configs"
	values, err := info.GetAsInfo(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config info from node: %v", err)
	}
	configs, ok := values[key].(lib.Stats)
	if !ok {
		typ := reflect.TypeOf(values[key])
		return nil, fmt.Errorf("failed to convert to lib.Stats, is of type %v", typ)
	}
	return configs, nil
}

func hashPassword(password string) ([]byte, error) {
	// Hashing the password with the cost of 10, with a static salt
	const salt = "$2a$10$7EqJtq98hPqEX7fNZaFWoO"
	hashedPassword, err := bcrypt.Hash(password, salt)
	if err != nil {
		return nil, err
	}
	return []byte(hashedPassword), nil
}

func (info *AsInfo) doInfo(commands ...string) (map[string]string, error) {
	// This is thread safe
	info.mutex.Lock()
	defer info.mutex.Unlock()

	// TODO Check for error
	if info.conn == nil || !info.conn.IsConnected() {
		var err error
		info.conn, err = aero.NewConnection(info.policy, info.host)
		if err != nil {
			return nil, fmt.Errorf("failed to create secure connection for aerospike info: %v", err)
		}

		aerr := info.conn.Login(info.policy)
		if aerr != nil {
			return nil, fmt.Errorf("failed to authenticate user `%s` in aerospike server: %v", info.policy.User, aerr.resultCode())
		}
		info.log.Debug("secure connection created for aerospike info")
	}

	var deadline time.Time
	if asTimeout > 0 {
		deadline = time.Now().Add(asTimeout)
	}
	info.conn.SetTimeout(deadline, asTimeout)

	result, err := info.conn.RequestInfo(commands...)
	if err != nil {
		info.log.Debug("failed to run aerospike info command", log.Ctx{"err": err})
		if err == io.EOF {
			// Peer closed connection.
			info.conn.Close()
			return nil, fmt.Errorf("connection reset: %v", err)
		}
		// FIXME: timeout is also closing connection
		info.conn.Close()
		return nil, err
	}
	return result, err
}

// Close closes all the connections to the system.
func (info *AsInfo) Close() error {
	// This is thread safe
	info.mutex.Lock()
	defer info.mutex.Unlock()

	if info.conn != nil {
		info.conn.Close()
	}
	info.conn = nil
	return nil
}

//*******************************************************************************************
// Public API to get parsed data
//*******************************************************************************************

// GetAsInfo function fetch and parse data for given commands from given host
// Input: cmdList - Options [statistics, configs, metadata, latency, throughput]
func (info *AsInfo) GetAsInfo(cmdList ...string) (NodeAsStats, error) {

	// These info will be used for creating other info commands
	//  _STAT_NS_NAMES, _STAT_DC_NAMES, _STAT_SINDEX, _STAT_LOG_IDS
	m, err := info.getCoreInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get basic ns/dc/sindex info: %v", err)
	}

	if len(cmdList) == 0 {
		cmdList = asCmds
	}
	rawCmdList := info.createCmdList(m, cmdList...)
	return info.execute(rawCmdList, m, cmdList...)
}

// GetAsConfig function fetch and parse config data for given context from given host
// Input: cmdList - Options [service, network, namespace, xdr, dc, security, logging]
func (info *AsInfo) GetAsConfig(contextList ...string) (lib.Stats, error) {

	// These info will be used for creating other info commands
	//  _STAT_NS_NAMES, _STAT_DC_NAMES, _STAT_SINDEX, _STAT_LOG_IDS
	m, err := info.getCoreInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get basic ns/dc/sindex info: %v", err)
	}

	if len(contextList) == 0 {
		contextList = []string{
			ConfigServiceContext, ConfigNetworkContext, ConfigNamespaceContext, ConfigSetContext,
			ConfigXDRContext, ConfigDCContext, ConfigSecurityContext, ConfigLoggingContext,
		}
	}

	rawCmdList := info.createConfigCmdList(m, contextList...)
	key := "configs"
	configs, err := info.execute(rawCmdList, m, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config info from aerospike server: %v", err)
	}
	c, ok := configs[key].(lib.Stats)
	if !ok {
		typ := reflect.TypeOf(configs[key])
		return nil, fmt.Errorf("failed to convert to lib.Stats, is of type %v", typ)
	}
	return c, nil
}

// GetNamespaceNamesCmd
func GetNamespaceNamesCmd() string {
	return _STAT_NS_NAMES
}

// GetDCNamesCmd
func GetDCNamesCmd() string {
	return _STAT_DC_NAMES
}

// GetTLSNamesCmd
func GetTLSNamesCmd() string {
	return _CONFIG_NETWORK
}

// GetLogNamesCmd
func GetLogNamesCmd() string {
	return _STAT_LOG_IDS
}

// GetSindexNamesCmd
func GetSindexNamesCmd() string {
	return _STAT_SINDEX
}

// GetSetNamesCmd
func GetSetNamesCmd() string {
	return _STAT_SET
}

// ParseNamespaceNames parses all namespace names
func ParseNamespaceNames(m map[string]string) []string {
	return getNames(m[_STAT_NS_NAMES])
}

// ParseDCNames parses all DC names
func ParseDCNames(m map[string]string) []string {
	return getNames(m[_STAT_DC_NAMES])
}

// ParseTLSNames parses all TLS names
func ParseTLSNames(m map[string]string) []string {
	names := make([]string, 0)
	nc := parseBasicConfigInfo(m[_CONFIG_NETWORK], "=")
	for k, v := range nc {
		if networkTLSNameRe.MatchString(k) {
			names = append(names, v.(string))
		}
	}
	return names
}

// ParseLogNames parses all log names
func ParseLogNames(m map[string]string) []string {
	logs := parseIntoMap(m[_STAT_LOG_IDS], ";", ":")
	names := make([]string, 0, len(logs))
	for _, l := range logs {
		lStr, _ := l.(string)
		if lStr == "stderr" {
			lStr = "console"
		}
		names = append(names, lStr)
	}
	return names
}

// ParseSindexNames parses all sindex names for namespace
func ParseSindexNames(m map[string]string, ns string) []string {
	return sindexNames(m[_STAT_SINDEX], ns)
}

// ParseSetNames parses all set names for namespace
func ParseSetNames(m map[string]string, ns string) []string {
	return setNames(m[_STAT_SET], ns)
}

//*******************************************************************************************
// create raw cmd list
//*******************************************************************************************

func (info *AsInfo) getCoreInfo() (map[string]string, error) {
	m, err := info.RequestInfo(_STAT_NS_NAMES, _STAT_DC_NAMES, _STAT_SINDEX, _STAT_LOG_IDS)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (info *AsInfo) createCmdList(m map[string]string, cmdList ...string) []string {
	var rawCmdList []string

	for _, cmd := range cmdList {
		switch cmd {
		case "statistics":
			cmds := info.createStatCmdList(m)
			rawCmdList = append(rawCmdList, cmds...)
		case "configs":
			cmds := info.createConfigCmdList(m)
			rawCmdList = append(rawCmdList, cmds...)
		case "metadata":
			cmds := info.createMetaCmdList()
			rawCmdList = append(rawCmdList, cmds...)
		case "latency":
			rawCmdList = append(rawCmdList, _LATENCY)
		case "throughput":
			rawCmdList = append(rawCmdList, _THROUGHPUT)

		default:
			info.log.Debug("invalid cmd to parse asinfo", log.Ctx{"command": cmd})
		}
	}
	return rawCmdList
}

func (info *AsInfo) createStatCmdList(m map[string]string) []string {
	cmdList := []string{_STAT, _STAT_XDR, _STAT_NS_NAMES, _STAT_DC_NAMES}

	nsNames := getNames(m[_STAT_NS_NAMES])
	for _, ns := range nsNames {
		// namespace, sets, bins, sindex
		cmdList = append(cmdList, _STAT_NS+ns, _STAT_SET+ns, _STAT_BIN+ns, _STAT_SINDEX+ns)

		indxNames := sindexNames(m[_STAT_SINDEX], ns)
		for _, indx := range indxNames {
			cmdList = append(cmdList, _STAT_SINDEX+ns+"/"+indx)
		}
	}

	dcNames := getNames(m[_STAT_DC_NAMES])
	for _, dc := range dcNames {
		cmdList = append(cmdList, _STAT_DC+dc)
	}

	return cmdList
}

// createConfigCmdList creates get-config commands for all context from contextList
func (info *AsInfo) createConfigCmdList(m map[string]string, contextList ...string) []string {
	if len(contextList) == 0 {
		contextList = []string{ConfigServiceContext, ConfigNetworkContext, ConfigNamespaceContext,
			ConfigSetContext, ConfigXDRContext, ConfigDCContext, ConfigSecurityContext,
			ConfigLoggingContext, _ConfigDCNames, _ConfigNamespaceNames, _ConfigLogIDs,
			ConfigRacksContext,
		}
	}

	cmdList := make([]string, 0, len(contextList))

	for _, c := range contextList {
		switch c {
		case ConfigServiceContext:
			cmdList = append(cmdList, _CONFIG_SERVICE)

		case ConfigNetworkContext:
			cmdList = append(cmdList, _CONFIG_NETWORK)

		case ConfigNamespaceContext:
			cmdList = append(cmdList, info.createNamespaceConfigCmdList(getNames(m[_STAT_NS_NAMES])...)...)

		case ConfigSetContext:
			cmdList = append(cmdList, info.createSetConfigCmdList(getNames(m[_STAT_NS_NAMES])...)...)

		case ConfigXDRContext:
			cmdList = append(cmdList, _CONFIG_XDR)

		case ConfigDCContext:
			cmdList = append(cmdList, info.createDCConfigCmdList(getNames(m[_STAT_DC_NAMES])...)...)

		case ConfigSecurityContext:
			cmdList = append(cmdList, _CONFIG_SECURITY)

		case ConfigLoggingContext:
			logs := parseIntoMap(m[_STAT_LOG_IDS], ";", ":")
			for id := range logs {
				cmdList = append(cmdList, _CONFIG_LOGGING+id)
			}
		case ConfigRacksContext:
			cmdList = append(cmdList, _CONFIG_RACKS)
		case _ConfigDCNames:
			cmdList = append(cmdList, _STAT_DC_NAMES)

		case _ConfigNamespaceNames:
			cmdList = append(cmdList, _STAT_NS_NAMES)

		case _ConfigLogIDs:
			cmdList = append(cmdList, _STAT_LOG_IDS)

		default:
			info.log.Debug("invalid context to parse AsConfig", log.Ctx{"context": c})
		}
	}

	return cmdList
}

// createNamespaceConfigCmdList creates get-config command for namespace
func (info *AsInfo) createNamespaceConfigCmdList(nsNames ...string) []string {
	cmdList := make([]string, 0, len(nsNames))

	for _, ns := range nsNames {
		cmdList = append(cmdList, _CONFIG_NAMESPACE+ns)
	}
	return cmdList
}

// createSetConfigCmdList creates get-config command for set
func (info *AsInfo) createSetConfigCmdList(nsNames ...string) []string {
	cmdList := make([]string, 0, len(nsNames))

	for _, ns := range nsNames {
		cmdList = append(cmdList, _STAT_SET+ns)
	}
	return cmdList
}

// createDCConfigCmdList creates get-config command for DC
func (info *AsInfo) createDCConfigCmdList(dcNames ...string) []string {
	cmdList := make([]string, 0, len(dcNames))

	for _, dc := range dcNames {
		cmdList = append(cmdList, _CONFIG_DC+dc)
	}
	return cmdList
}

func (info *AsInfo) createMetaCmdList() []string {
	cmdList := []string{_META_NODE_ID, _META_BUILD, _META_SERVICE,
		_META_SERVICES, _META_SERVICES_ALUMNI, _META_SERVICES_ALTERNATE, _META_VERSION,
		_META_BUILD_OS, _META_CLUSTER_NAME, _META_FEATURES, _META_EDITION}
	return cmdList
}

func getNames(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ";")
}

// ns=test:set=testset:indexname=idx_foo:bin=loop:type=NUMERIC:indextype=NONE:path=loop:state=RW;
func sindexNames(str, ns string) []string {
	var indxNames []string
	sindexStrList := strings.Split(str, ";")
	for _, str := range sindexStrList {
		if str == "" {
			continue
		}
		idxMap := parseIntoMap(str, ":", "=")

		nsIdx := idxMap.TryString("ns", "")
		if nsIdx != ns {
			continue
		}
		// Assume indexname is always there
		indxNames = append(indxNames, idxMap.TryString("indexname", ""))
	}
	return indxNames
}

// ns=test:set=demo:objects=2:tombstones=0:memory_data_bytes=28:truncate_lut=0:stop-writes-count=0:set-enable-xdr=use-default:disable-eviction=false;
func setNames(str, ns string) []string {
	var setNames []string
	setStrList := strings.Split(str, ";")
	for _, str := range setStrList {
		if str == "" {
			continue
		}
		setMap := parseIntoMap(str, ":", "=")

		if setMap.TryString("ns", "") != ns {
			continue
		}

		// Assume set is always there
		setNames = append(setNames, setMap.TryString("set", ""))
	}
	return setNames
}

//*******************************************************************************************
// execute raw cmds
//*******************************************************************************************

func (info *AsInfo) execute(rawCmdList []string, m map[string]string, cmdList ...string) (NodeAsStats, error) {
	rawMap, err := info.RequestInfo(rawCmdList...)
	if err != nil {
		return nil, err
	}

	// Add all core info also in rawMap, This info will be further used in parsing
	for k, v := range m {
		rawMap[k] = v
	}

	parsedMap := parseCmdResults(rawMap, cmdList...)
	return parsedMap, nil
}

//*******************************************************************************************
// parse raw cmd results
//*******************************************************************************************

func parseCmdResults(rawMap map[string]string, cmdList ...string) lib.Stats {
	asMap := make(lib.Stats)

	for _, cmd := range cmdList {
		switch cmd {
		case "statistics":
			asMap[cmd] = parseStatInfo(rawMap)
		case "configs":
			asMap[cmd] = parseConfigInfo(rawMap)
		case "metadata":
			asMap[cmd] = parseMetadataInfo(rawMap)
		case "latency":
			asMap[cmd] = parseLatencyInfo(rawMap[_LATENCY])
		case "throughput":
			asMap[cmd] = parseThroughputInfo(rawMap[_THROUGHPUT])

		default:
			pkglog.Debug("invalid cmd to parse asinfo", log.Ctx{"command": cmd})
		}
	}

	if _, ok := asMap["metadata"]; ok {
		updateExtraMetadata(asMap)
	}
	return asMap
}

func updateExtraMetadata(m lib.Stats) {
	serviceMap := m.GetInnerVal("statistics", "service")
	nsStatMap := m.GetInnerVal("statistics", "namespace")
	configMap := m.GetInnerVal("configs", "service")
	metaMap := m.GetInnerVal("metadata")

	if len(serviceMap) != 0 {
		metaMap["cluster_size"] = serviceMap.Get("cluster_size")
		metaMap["uptime"] = serviceMap.Get("uptime")
		metaMap["principal"] = serviceMap.Get("paxos_principal")
	}
	if len(configMap) != 0 {
		metaMap["cluster_name"] = configMap.Get("cluster-name")
	}

	if len(nsStatMap) != 0 {
		flag := false
		var nsList []string
		for ns := range nsStatMap {
			nsList = append(nsList, ns)

			if flag {
				continue
			}
			// Get info time for each node, for all ns of a node, its same.
			nsServiceMap := nsStatMap.GetInnerVal(ns, "service")
			if v := nsServiceMap.TryInt("current_time", 0); v != 0 {
				metaMap["current_time"] = v + ast.CITRUSLEAF_EPOCH
				flag = true
			}
		}
		metaMap["ns_list"] = nsList
	}
}

//***************************************************************************
// parse statistics

func parseStatInfo(rawMap map[string]string) lib.Stats {

	statMap := make(lib.Stats)

	statMap["service"] = parseBasicInfo(rawMap[_STAT])
	statMap["xdr"] = parseBasicInfo(rawMap[_STAT_XDR])
	statMap["dc"] = parseAllDcStats(rawMap)
	statMap["namespace"] = parseAllNsStats(rawMap)

	return statMap
}

// AllDCStats returns statistics of all dc's on the host.
func parseAllDcStats(rawMap map[string]string) lib.Stats {
	dcStats := make(lib.Stats)
	dcNames := getNames(rawMap[_STAT_DC_NAMES])

	for _, dc := range dcNames {
		newCmd := _STAT_DC + "/" + dc
		s := parseBasicInfo(rawMap[newCmd])
		dcStats[dc] = s
	}
	return dcStats
}

func parseAllNsStats(rawMap map[string]string) lib.Stats {
	nsStatMap := make(lib.Stats)
	nsNames := getNames(rawMap[_STAT_NS_NAMES])
	for _, ns := range nsNames {
		m := make(lib.Stats)
		m["service"] = parseStatNsInfo(rawMap[_STAT_NS+ns])
		m["set"] = parseStatSetsInfo(rawMap[_STAT_SET+ns])
		m["bin"] = parseStatBinsInfo(rawMap[_STAT_BIN+ns])
		m["sindex"] = parseStatSindexsInfo(rawMap, ns)

		nsStatMap[ns] = m
	}
	return nsStatMap
}

func parseBasicInfo(res string) lib.Stats {
	return parseIntoMap(res, ";", "=")
}

func parseStatNsInfo(res string) lib.Stats {
	m := parseBasicInfo(res)
	// some stats are of form {nsname}-statname
	newMap := parseNsKeys(m)
	return newMap
}

func parseStatSindexsInfo(rawMap map[string]string, ns string) lib.Stats {
	indxMap := make(lib.Stats)
	indxNames := sindexNames(rawMap[_STAT_SINDEX], ns)
	for _, indx := range indxNames {
		indxMap[indx] = parseBasicInfo(rawMap[_STAT_SINDEX+ns+"/"+indx])
	}
	return indxMap
}

func parseStatSetsInfo(res string) lib.Stats {
	// Parse
	ml := parseIntoListOfMap(res, ";", ":", "=")

	// Change this list in map
	stats := make(lib.Stats)
	for _, setStat := range ml {
		stats[setStat.TryString("set", "")] = setStat
	}
	return stats
}

func parseStatBinsInfo(res string) lib.Stats {
	// This can be optimize, bin has only 2 stats, so just parse those 2.
	var binStatStr string
	binStr := strings.Split(res, ",")
	for _, s := range binStr {
		if strings.Contains(s, "=") {
			binStatStr = binStatStr + "," + s
		}
	}
	stats := parseIntoMap(binStatStr, ",", "=")
	return stats
}

//***************************************************************************
// parse configs
//

func parseConfigInfo(rawMap map[string]string) lib.Stats {

	configMap := make(lib.Stats)

	sc := parseBasicConfigInfo(rawMap[_CONFIG_SERVICE], "=")
	if len(sc) > 0 {
		configMap[ConfigServiceContext] = sc
	}

	nc := parseBasicConfigInfo(rawMap[_CONFIG_NETWORK], "=")
	if len(nc) > 0 {
		configMap[ConfigNetworkContext] = nc
	}

	nsc := parseAllNsConfig(rawMap, _CONFIG_NAMESPACE)
	if len(nsc) > 0 {
		configMap[ConfigNamespaceContext] = nsc
	}

	xc := parseBasicConfigInfo(rawMap[_CONFIG_XDR], "=")
	if len(xc) > 0 {
		configMap[ConfigXDRContext] = xc
	}

	dcc := parseAllDcConfig(rawMap, _CONFIG_DC)
	if len(dcc) > 0 {
		configMap[ConfigDCContext] = dcc
	}

	sec := parseBasicConfigInfo(rawMap[_CONFIG_SECURITY], "=")
	if len(sec) > 0 {
		configMap[ConfigSecurityContext] = sec
	}

	lc := parseAllLoggingConfig(rawMap, _CONFIG_LOGGING)
	if len(lc) > 0 {
		configMap[ConfigLoggingContext] = lc
	}

	rc := parseConfigRacksInfo(rawMap[_CONFIG_RACKS])
	if len(rc) > 0 {
		configMap[ConfigRacksContext] = rc
	}

	return configMap
}

func parseAllLoggingConfig(rawMap map[string]string, cmd string) lib.Stats {
	logConfigMap := make(lib.Stats)
	logs := parseIntoMap(rawMap[_STAT_LOG_IDS], ";", ":")

	for id := range logs {
		m := parseBasicConfigInfo(rawMap[cmd+id], ":")
		if len(m) > 0 {
			logConfigMap[logs.TryString(id, "")] = m
		}
	}
	return logConfigMap
}

// {test}-configname -> configname
func parseAllNsConfig(rawMap map[string]string, cmd string) lib.Stats {
	nsConfigMap := make(lib.Stats)
	nsNames := getNames(rawMap[_STAT_NS_NAMES])

	for _, ns := range nsNames {
		m := parseBasicConfigInfo(rawMap[cmd+ns], "=")
		setM := parseConfigSetsInfo(rawMap[_STAT_SET+ns])
		if len(setM) > 0 {
			if len(m) == 0 {
				m = make(lib.Stats)
			}
			m["set"] = setM
		}

		newM := parseNsKeys(m)
		// Some config are like {test}-configname
		if len(newM) > 0 {
			nsConfigMap[ns] = newM
		}
	}
	return nsConfigMap
}

func parseConfigSetsInfo(res string) lib.Stats {
	// Parse
	ml := parseIntoListOfMap(res, ";", ":", "=")

	// Change this list in map
	stats := make(lib.Stats)
	for _, setStat := range ml {
		set := setStat.TryString("set", "")
		if len(set) > 0 {
			for k := range setStat {
				if !strings.Contains(k, "-") {
					// TODO: Is it good enough to consider keys with '-' as config?
					delete(setStat, k)
				}
			}
			stats[set] = setStat
		}
	}
	return stats
}

func parseAllDcConfig(rawMap map[string]string, cmd string) lib.Stats {
	dcConfigMap := make(lib.Stats)
	dcNames := getNames(rawMap[_STAT_DC_NAMES])

	for _, dc := range dcNames {
		m := parseIntoDcMap(rawMap[cmd+dc], ":", "=")
		if len(m) > 0 {
			dcConfigMap[dc] = m
		}
	}
	return dcConfigMap
}

func parseBasicConfigInfo(res string, sep string) lib.Stats {
	// Parse
	conf := parseIntoMap(res, ";", sep)
	return conf
}

func parseConfigRacksInfo(res string) []lib.Stats {
	ml := parseIntoListOfMap(res, ";", ":", "=")
	return ml
}

//***************************************************************************
// parse metadata
//

func parseMetadataInfo(rawMap map[string]string) lib.Stats {
	metaMap := make(lib.Stats)

	metaMap["node_id"] = rawMap[_META_NODE_ID]
	metaMap["asd_build"] = rawMap[_META_BUILD]
	metaMap["service"] = parseListTypeMetaInfo(rawMap, _META_SERVICE)
	metaMap["services"] = parseListTypeMetaInfo(rawMap, _META_SERVICES)
	metaMap["services-alumni"] = parseListTypeMetaInfo(rawMap, _META_SERVICES_ALUMNI)
	metaMap["services-alternate"] = parseListTypeMetaInfo(rawMap, _META_SERVICES_ALTERNATE)
	metaMap["features"] = parseListTypeMetaInfo(rawMap, _META_FEATURES)
	metaMap["edition"] = rawMap[_META_EDITION]
	metaMap["version"] = rawMap[_META_VERSION]
	metaMap["build_os"] = rawMap[_META_BUILD_OS]

	return metaMap
}

func parseListTypeMetaInfo(rawMap map[string]string, cmd string) []string {
	// Parse
	str := strings.TrimSpace(rawMap[cmd])
	if str == "" {
		return []string{}
	}
	l := strings.Split(str, ";")
	return l
}

//***************************************************************************
// parse latency and throughput
//
func parseThroughputInfo(rawStr string) lib.Stats {

	ip := lib.NewInfoParser(rawStr)

	//typical format is {test}-read:15:43:18-GMT,ops/sec;15:43:28,0.0;
	//nodeStats := map[string]float64{}
	//res := map[string]map[string]float64{}
	nodeStats := lib.Stats{}
	res := map[string]lib.Stats{}
	for {
		if err := ip.Expect("{"); err != nil {
			// it's an error string, read to next section
			if _, err := ip.ReadUntil(';'); err != nil {
				break
			}
			continue
		}
		ns, err := ip.ReadUntil('}')
		if err != nil {
			break
		}
		if err := ip.Expect("-"); err != nil {
			break
		}
		op, err := ip.ReadUntil(':')
		if err != nil {
			break
		}
		// first timestamp
		if _, err := ip.ReadUntil(','); err != nil {
			break
		}
		// ops/sec
		if _, err := ip.ReadUntil(';'); err != nil {
			break
		}
		// second timestamp
		if _, err = ip.ReadUntil(','); err != nil {
			break
		}
		opsCount, err := ip.ReadFloat(';')
		if err != nil {
			break
		}
		if res[ns] == nil {
			res[ns] = lib.Stats{
				op: opsCount,
			}
		} else {
			res[ns][op] = opsCount
		}
	}
	// TODO Cross check with khosrow, was it double accounting. for latency also
	// calc totals
	for _, mp := range res {
		for op, tps := range mp {
			// nodeStats[op] is interface{} type, so will be nil at start.
			if nodeStats[op] == nil {
				nodeStats[op] = float64(0)
			}
			nodeStats[op] = nodeStats[op].(float64) + tps.(float64)
		}
	}

	throughputMap := make(lib.Stats)
	newNodeStats := make(lib.Stats)
	newRes := make(lib.Stats)

	for k, v := range nodeStats {
		newNodeStats[k] = v
	}
	for k, v := range res {
		newRes[k] = v
	}

	throughputMap["namespace"] = newRes
	throughputMap["total"] = newNodeStats
	return throughputMap
}

// TODO: check diff lat bucket in agg
//typical format is {test}-read:10:17:37-GMT,ops/sec,>1ms,>8ms,>64ms;10:17:47,29648.2,3.44,0.08,0.00;
func parseLatencyInfo(rawStr string) lib.Stats {

	ip := lib.NewInfoParser(rawStr)
	nodeStats := make(map[string]lib.Stats)
	res := make(map[string]lib.Stats)

	for {
		if err := ip.Expect("{"); err != nil {
			// it's an error string, read to next section
			if _, err := ip.ReadUntil(';'); err != nil {
				break
			}
			continue
		}
		ns, err := ip.ReadUntil('}')
		if err != nil {
			break
		}
		if err := ip.Expect("-"); err != nil {
			break
		}
		op, err := ip.ReadUntil(':')
		if err != nil {
			break
		}
		timestamp, err := ip.ReadUntil(',')
		if err != nil {
			break
		}
		if _, err := ip.ReadUntil(','); err != nil {
			break
		}
		bucketsStr, err := ip.ReadUntil(';')
		if err != nil {
			break
		}
		buckets := strings.Split(bucketsStr, ",")

		_, err = ip.ReadUntil(',')
		if err != nil {
			break
		}
		opsCount, err := ip.ReadFloat(',')
		if err != nil {
			break
		}
		valBucketsStr, err := ip.ReadUntil(';')
		if err != nil && err != io.EOF {
			break
		}
		valBuckets := strings.Split(valBucketsStr, ",")
		valBucketsFloat := make([]float64, len(valBuckets))
		for i := range valBuckets {
			valBucketsFloat[i], _ = strconv.ParseFloat(valBuckets[i], 64)
		}
		// calc precise in-between percents
		lineAggPct := float64(0)
		for i := len(valBucketsFloat) - 1; i > 0; i-- {
			lineAggPct += valBucketsFloat[i]
			valBucketsFloat[i-1] = math.Max(0, valBucketsFloat[i-1]-lineAggPct)
		}
		if len(buckets) != len(valBuckets) {
			//log.Errorf("Error parsing latency values for node: `%s`. Bucket mismatch: buckets: `%s`, values: `%s`.", n.Address(), bucketsStr, valBucketsStr)
			pkglog.Error("parsing latency values")
			break
		}
		for i := range valBucketsFloat {
			valBucketsFloat[i] *= opsCount
		}
		stats := lib.Stats{
			"tps":        opsCount,
			"buckets":    buckets,
			"valBuckets": valBucketsFloat,
			"timestamp":  timestamp,
			//"timestamp_unix": time.Now().Unix(),
		}
		topct(stats)

		if res[ns] == nil {
			res[ns] = lib.Stats{
				op: stats,
			}
		} else {
			res[ns][op] = stats
		}

		// calc totals
		if nstats := nodeStats[op]; nstats == nil {
			nodeStats[op] = stats
		} else {
			// make a copy, since it is referred in nodeStats
			nstats = _cloneLatency(nstats)

			if timestamp > nstats.TryString("timestamp", "") {
				nstats["timestamp"] = timestamp
			}

			nstats["tps"] = nstats.TryFloat("tps", 0) + opsCount
			nBuckets := nstats["buckets"].([]string)
			if len(buckets) > len(nBuckets) {
				nstats["buckets"] = append(nBuckets, buckets[len(nBuckets):]...)
				nstats["valBuckets"] = append(nstats["valBuckets"].([]float64), make([]float64, len(buckets[len(nBuckets):]))...)
			}

			nValBuckets := nstats["valBuckets"].([]float64)
			for i := range buckets {
				nValBuckets[i] += valBucketsFloat[i]
			}

			nstats["valBuckets"] = nValBuckets
			nodeStats[op] = nstats
		}
	}

	latencyMap := make(lib.Stats)
	newNodeStats := make(lib.Stats)
	newRes := make(lib.Stats)

	for k, v := range nodeStats {
		newNodeStats[k] = v
	}
	for k, v := range res {
		newRes[k] = v
	}

	// Transform before returning
	tNsLat := transformNsLatency(newRes)
	tNodeLat := transformNodeLatency(newNodeStats)
	latencyMap["namespace"] = tNsLat
	latencyMap["total"] = tNodeLat

	return latencyMap
}

func transformNsLatency(lat lib.Stats) lib.Stats {
	newNs := lib.Stats{}
	for ns := range lat {
		m := lat.GetInnerVal(ns)
		newM := transformLatencyHistAll(m)
		newNs[ns] = newM
	}
	return newNs
}

func transformNodeLatency(lat lib.Stats) lib.Stats {
	return transformLatencyHistAll(lat)
}

func transformLatencyHistAll(nLatencyMap lib.Stats) lib.Stats {
	newTotal := lib.Stats{}
	for hist := range nLatencyMap {
		m := nLatencyMap.GetInnerVal(hist)
		newM := transformLatencyHist(m)
		newTotal[hist] = newM
	}
	return newTotal
}

func transformLatencyHist(hist lib.Stats) lib.Stats {
	newM := lib.Stats{}
	for i, buk := range hist["buckets"].([]string) {
		newM[buk] = hist["valBuckets"].([]float64)[i]
	}
	newM["tps"] = hist["tps"]
	return newM
}

func topct(stat lib.Stats) {
	tps := stat.TryFloat("tps", 0)
	if tps == 0 {
		tps = 1
	}
	nValBuckets := stat["valBuckets"].([]float64)
	for i := range nValBuckets {
		nValBuckets[i] /= tps
	}
	stat["valBuckets"] = nValBuckets
}

func _cloneLatency(m lib.Stats) lib.Stats {
	vb, _ := m["valBuckets"].([]float64)
	valBuckets := make([]float64, len(vb))
	for i, v := range vb {
		valBuckets[i] = v
	}

	c := lib.Stats{
		"tps":            m["tps"],
		"timestamp":      m["timestamp"],
		"timestamp_unix": m["timestamp_unix"],
		"buckets":        m["buckets"],
		"valBuckets":     valBuckets,
	}

	topct(c)
	return c
}

//***************************************************************************
// utils
//
func parseNsKeys(rawMap lib.Stats) lib.Stats {
	newMap := make(lib.Stats)
	for k, v := range rawMap {
		if strings.Contains(k, "}-") {
			k = strings.Split(k, "}-")[1]
		}
		newMap[k] = v
	}
	return newMap
}

func parseIntoMap(str string, del string, sep string) lib.Stats {
	if str == "" {
		return nil
	}
	m := make(lib.Stats)
	items := strings.Split(str, del)
	for _, item := range items {
		if item == "" {
			continue
		}
		kv := strings.Split(item, sep)
		//m[kv[0]] = kv[1]
		m[kv[0]] = getParsedValue(kv[1])
	}
	//return m.ToParsedValues()
	return m
}

// Type should be map[string]interface{} otherwise same map is returned.
// info_parser needs parsed values
func getParsedValue(val interface{}) interface{} {

	valStr, ok := val.(string)
	if !ok {
		return val
	}
	// TSDB type mismatch problem. 1 solution to change int also to float.
	// Great its working now.
	// FIXME: check with others if health is using int conversion
	if value, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return value
	} else if value, err := strconv.ParseFloat(valStr, 64); err == nil {
		return value
	} else if value, err := strconv.ParseBool(valStr); err == nil {
		return value
	} else {
		return valStr
	}
}

func parseIntoListOfMap(str string, del1 string, del2 string, sep string) []lib.Stats {
	var strListMap []lib.Stats
	strList := strings.Split(str, del1)
	for _, s := range strList {
		if s == "" {
			continue
		}
		m := parseIntoMap(s, del2, sep)
		// Assume indexname is always there
		strListMap = append(strListMap, m)
	}
	return strListMap
}

func parseIntoDcMap(str string, del string, sep string) lib.Stats {
	if str == "" {
		return nil
	}
	m := make(lib.Stats)
	items := strings.Split(str, del)
	newItems := make([]string, len(items))
	nIdx := 0
	//fmt.Println(str)
	for _, item := range items {
		if item == "" {
			newItems[nIdx-1] = newItems[nIdx-1] + del
			continue
		}
		if !strings.Contains(item, "=") {
			newItems[nIdx-1] = newItems[nIdx-1] + del + item
			continue
		}
		newItems[nIdx] = item
		nIdx++
	}
	for _, item := range newItems {
		if item == "" {
			continue
		}
		kv := strings.Split(item, sep)
		m[kv[0]] = kv[1]
	}

	return m.ToParsedValues()
}

func contains(list []string, str string) bool {
	if len(list) == 0 {
		return false
	}

	for _, s := range list {
		if s == str {
			return true
		}
	}

	return false
}
