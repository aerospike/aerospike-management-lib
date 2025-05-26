package info

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"

	aero "github.com/aerospike/aerospike-client-go/v8"
	ast "github.com/aerospike/aerospike-client-go/v8/types"
	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/utils"
)

type ClusterAsStat = lib.Stats

type NodeAsStats = lib.Stats

// ErrInvalidNamespace specifies that the namespace is invalid on the cluster.
var ErrInvalidNamespace = fmt.Errorf("invalid namespace")

// ErrInvalidDC specifies that the dc is invalid on the cluster.
var ErrInvalidDC = fmt.Errorf("invalid dc")

// ErrConnNotAuthenticated specifies that the connection is not authenticated.
var ErrConnNotAuthenticated = fmt.Errorf("connection not authenticated")

// ASInfo top level map keys
const (
	ConstStat     = "statistics" // stat
	ConstConfigs  = "configs"    // configs
	ConstLatency  = "latency"    // latency
	ConstMetadata = "metadata"   // metadata
)

const (
	// Explicit constants are defined with `const` prefix when
	// 1. string values which are not commands
	// 2. string values which are used to generate other commands
	// 3. string values which are both command and constant
	constStatNS      = "namespace/" // StatNamespace
	constStatDC      = "get-stats:context=xdr;dc="
	constStatSet     = "sets/"      // StatSets
	constStatBin     = "bins/"      // StatBins
	constStatSIndex  = "sindex/"    // StatSindex
	constStatNSNames = "namespaces" // StatNamespaces
	constStatDCNames = "dcs"        // StatDcs need dc names
	constStatLogIDs  = "logs"       // StatLogs need logging id

	cmdConfigNetwork   = "get-config:context=network"       // ConfigNetwork
	cmdConfigService   = "get-config:context=service"       // ConfigService
	cmdConfigNamespace = "get-config:context=namespace;id=" // ConfigNamespace
	cmdConfigXDR       = "get-config:context=xdr"           // ConfigXDR
	cmdConfigSecurity  = "get-config:context=security"      // ConfigSecurity
	cmdConfigDC        = "get-config:context=xdr;dc="       // ConfigDC
	cmdConfigMESH      = "mesh"                             // ConfigMesh
	cmdConfigRacks     = "racks:"                           // configRacks
	cmdConfigLogging   = "log/"                             // ConfigLog

	cmdLatency = "latency:"

	cmdMetaBuild             = "build"              // Build
	cmdMetaVersion           = "version"            // Version
	cmdMetaBuildOS           = "build_os"           // BUILD OS
	cmdMetaNodeID            = "node"               // NodeID
	cmdMetaClusterName       = "cluster-name"       // Cluster Name
	cmdMetaService           = "service"            // Service
	cmdMetaServices          = "services"           // Services
	cmdMetaServicesAlumni    = "services-alumni"    // ServicesAlumni
	cmdMetaServicesAlternate = "services-alternate" // ServiceAlternate
	cmdMetaFeatures          = "features"           // Features
	cmdMetaEdition           = "edition"            // Edition
)

// other meta-info
// "cluster-generation", "partition-generation", "build_time",
// "udf-list", "cluster-name", "service-clear-std", "service-tls-std",

// Aerospike Config Context
const (
	ConfigServiceContext   = "service"
	ConfigNetworkContext   = "network"
	ConfigNamespaceContext = "namespaces"
	ConfigSetContext       = "sets"
	ConfigXDRContext       = "xdr"
	ConfigDCContext        = "dcs"
	ConfigSecurityContext  = "security"
	ConfigLoggingContext   = "logging"
	ConfigRacksContext     = "racks"
)

// Aerospike Metadata Context
const (
	MetaBuild             = "asd_build"
	MetaVersion           = cmdMetaVersion
	MetaBuildOS           = cmdMetaBuildOS
	MetaNodeID            = cmdMetaNodeID
	MetaClusterName       = cmdMetaClusterName
	MetaService           = cmdMetaService
	MetaServices          = cmdMetaServices
	MetaServicesAlumni    = cmdMetaServicesAlumni
	MetaServicesAlternate = cmdMetaServicesAlternate
	MetaFeatures          = cmdMetaFeatures
	MetaEdition           = cmdMetaEdition
)

const (
	ConfigDCNames        = "dc_names"
	ConfigNamespaceNames = "namespace_names"
	ConfigLogIDs         = "log_ids"
)

var asCmds = []string{
	ConstStat, ConstConfigs, ConstMetadata, ConstLatency,
}

var networkTLSNameRe = regexp.MustCompile(`^tls\[(\d+)].name$`)

type Connection interface {
	IsConnected() bool
	Login(*aero.ClientPolicy) aero.Error
	SetTimeout(time.Time, time.Duration) aero.Error
	RequestInfo(...string) (map[string]string, aero.Error)
	Close()
}

type ConnectionFactory interface {
	NewConnection(*aero.ClientPolicy, *aero.Host) (Connection, aero.Error)
}

type aerospikeConnFactory struct{}

func (f *aerospikeConnFactory) NewConnection(
	policy *aero.ClientPolicy, host *aero.Host,
) (Connection, aero.Error) {
	return aero.NewConnection(policy, host)
}

var aeroConnFactory = &aerospikeConnFactory{}

// AsInfo provides info calls on an aerospike cluster.
type AsInfo struct {
	policy   *aero.ClientPolicy
	host     *aero.Host
	conn     Connection
	connFact ConnectionFactory
	log      logr.Logger
	mutex    sync.Mutex
}

func NewAsInfo(log logr.Logger, h *aero.Host, cp *aero.ClientPolicy) *AsInfo {
	return NewAsInfoWithConnFactory(log, h, cp, aeroConnFactory)
}

func NewAsInfoWithConnFactory(
	log logr.Logger, h *aero.Host, cp *aero.ClientPolicy, connFact ConnectionFactory,
) *AsInfo {
	logger := log.WithValues("node", h)

	return &AsInfo{
		host:     h,
		policy:   cp,
		conn:     nil,
		connFact: connFact,
		log:      logger,
	}
}

var maxInfoRetries = 3
var asTimeout = time.Second * 100

// RequestInfo get aerospike info
func (info *AsInfo) RequestInfo(cmd ...string) (
	result map[string]string, err error,
) {
	if len(cmd) == 0 {
		return map[string]string{}, nil
	}

	// TODO: only retry for EOF or Timeout errors
	for i := 0; i < maxInfoRetries; i++ {
		result, err = info.doInfo(cmd...)
		if err == nil {
			return result, nil
		}
	}

	return result, err
}

// AllConfigs returns all the dynamic configurations of the node.
//
// The returned map can be converted to asconfig.Conf.
func (info *AsInfo) AllConfigs() (lib.Stats, error) {
	key := ConstConfigs

	values, err := info.GetAsInfo(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config info from node: %w", err)
	}

	configs, ok := values[key].(lib.Stats)
	if !ok {
		typ := reflect.TypeOf(values[key])

		return nil, fmt.Errorf(
			"failed to convert to lib.Stats, is of type %v", typ,
		)
	}

	return configs, nil
}

func (info *AsInfo) doInfo(commands ...string) (map[string]string, error) {
	// This is thread safe
	info.mutex.Lock()
	defer info.mutex.Unlock()

	// TODO Check for error
	if info.conn == nil || !info.conn.IsConnected() {
		conn, err := info.connFact.NewConnection(info.policy, info.host)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create secure connection for aerospike info: %w",
				err,
			)
		}

		info.conn = conn // NewConnection returns an interface which will fail the nil check

		aerr := info.conn.Login(info.policy)
		if aerr != nil {
			ae := &aero.AerospikeError{}
			if errors.As(err, &ae) {
				return nil, fmt.Errorf(
					"failed to authenticate user `%s` in aerospike server: %v",
					info.policy.User, ae.ResultCode,
				)
			}

			return nil, fmt.Errorf(
				"failed to authenticate user `%s` in aerospike server: %w",
				info.policy.User, aerr,
			)
		}

		info.log.V(1).Info("Secure connection created for aerospike info")
	}

	deadline := time.Now().Add(asTimeout)
	if err := info.conn.SetTimeout(deadline, asTimeout); err != nil {
		return nil, err
	}

	result, err := info.conn.RequestInfo(commands...)
	if err != nil {
		info.log.V(1).Info("Failed to run aerospike info command", "err", err)

		if err == io.EOF {
			// Peer closed connection.
			info.conn.Close()
			return nil, fmt.Errorf("connection reset: %w", err)
		}
		// FIXME: timeout is also closing connection
		info.conn.Close()

		return nil, err
	}

	for k := range result {
		if strings.Contains(k, "not authenticated") {
			info.conn.Close()
			return nil, ErrConnNotAuthenticated
		}

		break
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

// *******************************************************************************************
// Public API to get parsed data
// *******************************************************************************************

// GetAsInfo function fetch and parse data for given commands from given host
// Input: cmdList - Options [statistics, configs, metadata, latency]
func (info *AsInfo) GetAsInfo(cmdList ...string) (NodeAsStats, error) {
	// These info will be used for creating other info commands
	//  statNSNames, statDCNames, statSIndex, statLogIDS
	m, err := info.getCoreInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get basic ns/dc/sindex info: %w", err)
	}

	if len(cmdList) == 0 {
		cmdList = asCmds
	}

	rawCmdList, err := info.createCmdList(m, cmdList...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cmd list: %w", err)
	}

	return info.execute(info.log, rawCmdList, m, cmdList...)
}

// GetAsConfig function fetch and parse config data for given context from given host
// Input: cmdList - Options [service, network, namespace, xdr, dc, security, logging]
func (info *AsInfo) GetAsConfig(contextList ...string) (lib.Stats, error) {
	// These info will be used for creating other info commands
	//  statNSNames, statDCNames, statSIndex, statLogIDS
	m, err := info.getCoreInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get basic ns/dc/sindex info: %w", err)
	}

	if len(contextList) == 0 {
		contextList = []string{
			ConfigServiceContext, ConfigNetworkContext, ConfigNamespaceContext,
			ConfigSetContext,
			ConfigXDRContext, ConfigDCContext, ConfigSecurityContext,
			ConfigLoggingContext,
		}
	}

	rawCmdList, err := info.createConfigCmdList(m, contextList...)
	if err != nil {
		return nil, fmt.Errorf("failed to create config cmd list: %w", err)
	}

	key := ConstConfigs
	configs, err := info.execute(info.log, rawCmdList, m, key)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get config info from aerospike server: %w", err,
		)
	}

	c, ok := configs[key].(lib.Stats)
	if !ok {
		typ := reflect.TypeOf(configs[key])

		return nil, fmt.Errorf(
			"failed to convert to lib.Stats, is of type %v", typ,
		)
	}

	return c, nil
}

// GetNamespaceNamesCmd returns the command to get namespace names
func GetNamespaceNamesCmd() string {
	return constStatNSNames
}

// GetDCNamesCmd returns the command to get DC namespace
func GetDCNamesCmd() string {
	return constStatDCNames
}

// GetTLSNamesCmd returns the command to get TLS names
func GetTLSNamesCmd() string {
	return cmdConfigNetwork
}

// GetLogNamesCmd returns the command to get log names
func GetLogNamesCmd() string {
	return constStatLogIDs
}

// GetSindexNamesCmd returns the command to get sindex names
func GetSindexNamesCmd() string {
	return constStatSIndex
}

// GetSetNamesCmd returns the command to get set names
func GetSetNamesCmd() string {
	return constStatSet
}

// ParseNamespaceNames parses all namespace names
func ParseNamespaceNames(m map[string]string) []string {
	return getNames(m[constStatNSNames])
}

// ParseDCNames parses all DC names
func ParseDCNames(m map[string]string) []string {
	rawXDRConfig, exists := m[cmdConfigXDR]
	if !exists || rawXDRConfig == "" {
		return []string{}
	}

	xdrConfig := ParseIntoMap(rawXDRConfig, ";", "=")

	rawNames, ok := xdrConfig[constStatDCNames].(string)
	if !ok || rawNames == "" {
		return []string{}
	}

	return strings.Split(rawNames, ",")
}

// ParseTLSNames parses all TLS names
func ParseTLSNames(m map[string]string) []string {
	names := make([]string, 0)
	nc := parseBasicConfigInfo(m[cmdConfigNetwork], "=")

	for k, v := range nc {
		if networkTLSNameRe.MatchString(k) {
			names = append(names, v.(string))
		}
	}

	return names
}

// ParseLogNames parses all log names
func ParseLogNames(m map[string]string) []string {
	logs := ParseIntoMap(m[constStatLogIDs], ";", ":")
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
	return sindexNames(m[constStatSIndex], ns)
}

// ParseSetNames parses all set names for namespace
func ParseSetNames(m map[string]string, ns string) []string {
	return setNames(m[constStatSet], ns)
}

// *******************************************************************************************
// create raw cmd list
// *******************************************************************************************

func (info *AsInfo) getCoreInfo() (map[string]string, error) {
	m, err := info.RequestInfo(
		constStatNSNames, cmdConfigXDR, constStatSIndex, constStatLogIDs, cmdMetaBuild,
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (info *AsInfo) createCmdList(
	m map[string]string, cmdList ...string,
) ([]string, error) {
	var rawCmdList []string

	for _, cmd := range cmdList {
		switch cmd {
		case ConstStat:
			cmds := info.createStatCmdList(m)
			rawCmdList = append(rawCmdList, cmds...)
		case ConstConfigs:
			cmds, err := info.createConfigCmdList(m)
			if err != nil {
				return nil, err
			}

			rawCmdList = append(rawCmdList, cmds...)
		case ConstMetadata:
			cmds := info.createMetaCmdList()
			rawCmdList = append(rawCmdList, cmds...)
		case ConstLatency:
			rawCmdList = append(rawCmdList, cmdLatency)

		default:
			info.log.V(1).Info("Invalid cmd to parse asinfo", "command", cmd)
		}
	}

	return rawCmdList, nil
}

func (info *AsInfo) createStatCmdList(m map[string]string) []string {
	cmdList := []string{ConstStat, cmdConfigXDR, constStatNSNames}

	nsNames := getNames(m[constStatNSNames])
	for _, ns := range nsNames {
		// namespace, sets, bins, sindex
		cmdList = append(
			cmdList, constStatNS+ns, constStatSet+ns, constStatSIndex+ns,
		)

		if r, _ := lib.CompareVersions(m[cmdMetaBuild], "7.0"); r == -1 {
			cmdList = append(cmdList, constStatBin+ns)
		}

		indexNames := sindexNames(m[constStatSIndex], ns)
		for _, index := range indexNames {
			cmdList = append(cmdList, constStatSIndex+ns+"/"+index)
		}
	}

	dcNames := ParseDCNames(m)
	for _, dc := range dcNames {
		cmdList = append(cmdList, constStatDC+dc)
	}

	return cmdList
}

// createConfigCmdList creates get-config commands for all context from contextList
func (info *AsInfo) createConfigCmdList(
	m map[string]string, contextList ...string,
) ([]string, error) {
	if len(contextList) == 0 {
		contextList = []string{
			ConfigServiceContext, ConfigNetworkContext, ConfigNamespaceContext,
			ConfigSetContext, ConfigXDRContext, ConfigDCContext,
			ConfigSecurityContext, ConfigLoggingContext, ConfigDCNames,
			ConfigNamespaceNames, ConfigLogIDs, ConfigRacksContext,
		}
	}

	cmdList := make([]string, 0, len(contextList))

	for _, c := range contextList {
		switch c {
		case ConfigServiceContext:
			cmdList = append(cmdList, cmdConfigService)

		case ConfigNetworkContext:
			cmdList = append(cmdList, cmdConfigNetwork)

		case ConfigNamespaceContext:
			cmdList = append(
				cmdList,
				info.createNamespaceConfigCmdList(ParseNamespaceNames(m)...)...,
			)

		case ConfigSetContext:
			cmdList = append(
				cmdList,
				info.createSetConfigCmdList(ParseNamespaceNames(m)...)...,
			)

		case ConfigXDRContext:
			xdrCmdList, err := info.createXDRConfigCmdList(m)
			if err != nil {
				// TODO: log?
				return nil, err
			}

			cmdList = append(cmdList, xdrCmdList...)

		case ConfigSecurityContext:
			cmdList = append(cmdList, cmdConfigSecurity)

		case ConfigLoggingContext:
			logs := ParseIntoMap(m[constStatLogIDs], ";", ":")
			for id := range logs {
				cmdList = append(cmdList, cmdConfigLogging+id)
			}
		case ConfigRacksContext:
			cmdList = append(cmdList, cmdConfigRacks)

		default:
			info.log.V(1).Info(
				"Invalid context to parse AsConfig",
				"context", c,
			)
		}
	}

	return cmdList, nil
}

// createNamespaceConfigCmdList creates get-config command for namespace
func (info *AsInfo) createNamespaceConfigCmdList(nsNames ...string) []string {
	cmdList := make([]string, 0, len(nsNames))

	for _, ns := range nsNames {
		cmdList = append(cmdList, cmdConfigNamespace+ns)
	}

	return cmdList
}

// createSetConfigCmdList creates get-config command for set
func (info *AsInfo) createSetConfigCmdList(nsNames ...string) []string {
	cmdList := make([]string, 0, len(nsNames))

	for _, ns := range nsNames {
		cmdList = append(cmdList, constStatSet+ns)
	}

	return cmdList
}

func (info *AsInfo) createXDRConfigCmdList(m map[string]string) ([]string, error) {
	cmdList := make([]string, 0, 1)

	resp, err := info.doInfo(cmdConfigXDR)
	if err != nil {
		return nil, err
	}

	m = mergeDicts(m, resp)
	dcNames := ParseDCNames(m)
	results := make(chan error, len(dcNames))

	var (
		wg   sync.WaitGroup
		lock sync.Mutex
	)

	for _, dc := range dcNames {
		wg.Add(1)

		go func(dc string) {
			defer wg.Done()

			resp, err := info.doInfo(cmdConfigDC + dc)

			if err != nil {
				results <- err
				return
			}

			lock.Lock()
			m = mergeDicts(m, resp)
			lock.Unlock()

			var nsNames []string

			rawDCConfig := resp[cmdConfigDC+dc]
			dcConfig := ParseIntoMap(rawDCConfig, ";", "=")
			rawNames, ok := dcConfig[constStatNSNames].(string)

			if ok {
				nsNames = strings.Split(rawNames, ",")
			} else {
				nsNames = []string{}
			}

			cmdList = append(cmdList, info.createDCNamespaceConfigCmdList(dc, nsNames...)...)
			results <- nil
		}(dc)
	}

	wg.Wait()
	close(results)

	// Return the first error if one occurred
	for err := range results {
		if err != nil {
			return cmdList, err
		}
	}

	return cmdList, nil
}

// createDCConfigCmdList creates get-config command for DC
func (info *AsInfo) createDCNamespaceConfigCmdList(dc string, namespaces ...string) []string {
	if dc == "" {
		return nil
	}

	cmdList := make([]string, 0, len(namespaces))

	for _, ns := range namespaces {
		cmdList = append(cmdList, cmdConfigDC+dc+";namespace="+ns)
	}

	return cmdList
}

func (info *AsInfo) createMetaCmdList() []string {
	cmdList := []string{
		cmdMetaNodeID, cmdMetaBuild, cmdMetaService,
		cmdMetaServices, cmdMetaServicesAlumni, cmdMetaServicesAlternate,
		cmdMetaVersion,
		cmdMetaBuildOS, cmdMetaClusterName, cmdMetaFeatures, cmdMetaEdition,
	}

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
	sindexStrList := strings.Split(str, ";")
	indexNames := make([]string, 0, len(sindexStrList))

	for _, str := range sindexStrList {
		if str == "" {
			continue
		}

		idxMap := ParseIntoMap(str, ":", "=")

		nsIdx := idxMap.TryString("ns", "")
		if nsIdx != ns {
			continue
		}
		// Assume indexname is always there
		indexNames = append(indexNames, idxMap.TryString("indexname", ""))
	}

	return indexNames
}

// ns=test:set=demo:objects=2:tombstones=0:memory_data_bytes=28:truncate_lut=0:stop-writes-count=0:set-enable-xdr=use-default:disable-eviction=false;
func setNames(str, ns string) []string {
	setStrList := strings.Split(str, ";")
	setNames := make([]string, 0, len(setStrList))

	for _, str := range setStrList {
		if str == "" {
			continue
		}

		setMap := ParseIntoMap(str, ":", "=")

		if setMap.TryString("ns", "") != ns {
			continue
		}

		// Assume set is always there
		setNames = append(setNames, setMap.TryString("set", ""))
	}

	return setNames
}

func mergeDicts(m1, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}

	return m1
}

// *******************************************************************************************
// execute raw cmds
// *******************************************************************************************

func (info *AsInfo) execute(
	log logr.Logger, rawCmdList []string, m map[string]string,
	cmdList ...string,
) (NodeAsStats, error) {
	rawMap, err := info.RequestInfo(rawCmdList...)
	if err != nil {
		return nil, err
	}

	// Add all core info also in rawMap, This info will be further used in parsing
	for k, v := range m {
		rawMap[k] = v
	}

	parsedMap := parseCmdResults(log, rawMap, cmdList...)

	return parsedMap, nil
}

// *******************************************************************************************
// parse raw cmd results
// *******************************************************************************************

func parseCmdResults(
	log logr.Logger, rawMap map[string]string, cmdList ...string,
) lib.Stats {
	asMap := make(lib.Stats)

	for _, cmd := range cmdList {
		switch cmd {
		case ConstStat:
			asMap[cmd] = parseStatInfo(rawMap)
		case ConstConfigs:
			asMap[cmd] = parseConfigInfo(rawMap)
		case ConstMetadata:
			asMap[cmd] = parseMetadataInfo(rawMap)
		case ConstLatency:
			asMap[cmd] = parseLatencyInfo(log, rawMap[cmdLatency])

		default:
			log.V(1).Info("Invalid cmd to parse asinfo", "command", cmd)
		}
	}

	if _, ok := asMap[ConstMetadata]; ok {
		updateExtraMetadata(asMap)
	}

	return asMap
}

func updateExtraMetadata(m lib.Stats) {
	serviceMap := m.GetInnerVal(ConstStat, "service")
	nsStatMap := m.GetInnerVal(ConstStat, "namespace")
	configMap := m.GetInnerVal(ConstConfigs, "service")
	metaMap := m.GetInnerVal(ConstMetadata)

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

// ***************************************************************************
// parse statistics
// ***************************************************************************

func parseStatInfo(rawMap map[string]string) lib.Stats {
	statMap := make(lib.Stats)

	statMap["service"] = parseBasicInfo(rawMap[ConstStat])
	statMap["dc"] = parseAllDcStats(rawMap)
	statMap["namespace"] = parseAllNsStats(rawMap)

	return statMap
}

// AllDCStats returns statistics of all dc's on the host.
func parseAllDcStats(rawMap map[string]string) lib.Stats {
	dcStats := make(lib.Stats)
	dcNames := ParseDCNames(rawMap)

	for _, dc := range dcNames {
		newCmd := constStatDC + dc
		s := parseBasicInfo(rawMap[newCmd])
		dcStats[dc] = s
	}

	return dcStats
}

func parseAllNsStats(rawMap map[string]string) lib.Stats {
	nsStatMap := make(lib.Stats)
	nsNames := getNames(rawMap[constStatNSNames])

	for _, ns := range nsNames {
		m := make(lib.Stats)
		m["service"] = parseStatNsInfo(rawMap[constStatNS+ns])
		m["set"] = parseStatSetsInfo(rawMap[constStatSet+ns])
		m["sindex"] = parseStatSindexsInfo(rawMap, ns)

		if r, _ := lib.CompareVersions(rawMap[cmdMetaBuild], "7.0"); r == -1 {
			m["bin"] = parseStatBinsInfo(rawMap[constStatBin+ns])
		}

		nsStatMap[ns] = m
	}

	return nsStatMap
}

func parseBasicInfo(res string) lib.Stats {
	return ParseIntoMap(res, ";", "=")
}

func parseStatNsInfo(res string) lib.Stats {
	m := parseBasicInfo(res)
	// some stats are of form {nsname}-statname
	newMap := parseNsKeys(m)

	return newMap
}

func parseStatSindexsInfo(rawMap map[string]string, ns string) lib.Stats {
	indexMap := make(lib.Stats)
	indexNames := sindexNames(rawMap[constStatSIndex], ns)

	for _, index := range indexNames {
		indexMap[index] = parseBasicInfo(rawMap[constStatSIndex+ns+"/"+index])
	}

	return indexMap
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
	// This can be optimized, bin has only 2 stats, so just parse those 2.
	var binStatStr string

	binStr := strings.Split(res, ",")

	for _, s := range binStr {
		if strings.Contains(s, "=") {
			binStatStr = binStatStr + "," + s
		}
	}

	stats := ParseIntoMap(binStatStr, ",", "=")

	return stats
}

// ***************************************************************************
// parse configs

func parseConfigInfo(rawMap map[string]string) lib.Stats {
	configMap := make(lib.Stats)

	sc := parseBasicConfigInfo(rawMap[cmdConfigService], "=")
	if len(sc) > 0 {
		configMap[ConfigServiceContext] = sc
	}

	nc := parseBasicConfigInfo(rawMap[cmdConfigNetwork], "=")
	if len(nc) > 0 {
		configMap[ConfigNetworkContext] = nc
	}

	nsc := parseAllNsConfig(rawMap, cmdConfigNamespace)
	if len(nsc) > 0 {
		configMap[ConfigNamespaceContext] = nsc
	}

	xc := parseAllXDRConfig(rawMap, cmdConfigXDR)

	if len(xc) > 0 {
		configMap[ConfigXDRContext] = xc
	}

	sec := parseBasicConfigInfo(rawMap[cmdConfigSecurity], "=")
	if len(sec) > 0 {
		configMap[ConfigSecurityContext] = sec
	}

	lc := parseAllLoggingConfig(rawMap, cmdConfigLogging)
	if len(lc) > 0 {
		configMap[ConfigLoggingContext] = lc
	}

	rc := parseConfigRacksInfo(rawMap[cmdConfigRacks])
	if len(rc) > 0 {
		configMap[ConfigRacksContext] = rc
	}

	return configMap
}

func parseAllLoggingConfig(rawMap map[string]string, cmd string) lib.Stats {
	logConfigMap := make(lib.Stats)
	logs := ParseIntoMap(rawMap[constStatLogIDs], ";", ":")

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
	nsNames := getNames(rawMap[constStatNSNames])

	for _, ns := range nsNames {
		m := parseBasicConfigInfo(rawMap[cmd+ns], "=")
		setM := parseConfigSetsInfo(rawMap[constStatSet+ns])

		if len(setM) > 0 {
			if len(m) == 0 {
				m = make(lib.Stats)
			}

			m[ConfigSetContext] = setM
		}

		newM := parseNsKeys(m)
		// Some configs are like {test}-configname
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
		if set != "" {
			for k := range setStat {
				if !strings.Contains(k, "-") {
					// TODO: Is it good enough to consider keys with '-' as
					// config? Only if a single word config is not added.
					delete(setStat, k)
				}
			}

			stats[set] = setStat
		}
	}

	return stats
}

func parseAllXDRConfig(rawMap map[string]string, cmd string) lib.Stats {
	xdrConfigMap := ParseIntoMap(rawMap[cmd], ";", "=")

	if xdrConfigMap == nil {
		return nil
	}

	var dcNames []string

	dcNamesRaw := xdrConfigMap.TryString(constStatDCNames, "")
	delete(xdrConfigMap, constStatDCNames)

	if dcNamesRaw == "" {
		dcNames = []string{}
		xdrConfigMap[ConfigDCContext] = make(lib.Stats)
	} else {
		dcNames = strings.Split(dcNamesRaw, ",")
		xdrConfigMap[ConfigDCContext] = make(lib.Stats, len(dcNames))
	}

	for _, dc := range dcNames {
		dcMap := ParseIntoMap(rawMap[cmd+";dc="+dc], ";", "=")

		if len(dcMap) == 0 {
			continue
		}

		xdrConfigMap[ConfigDCContext].(lib.Stats)[dc] = dcMap

		var nsNames []string

		nsNamesRaw := dcMap.TryString(constStatNSNames, "")
		delete(dcMap, constStatNSNames)

		if nsNamesRaw == "" {
			nsNames = []string{}
			dcMap[ConfigNamespaceContext] = make(lib.Stats)
		} else {
			nsNames = strings.Split(nsNamesRaw, ",")
			dcMap[ConfigNamespaceContext] = make(lib.Stats, len(nsNames))
		}

		for _, ns := range nsNames {
			nsMap := ParseIntoMap(rawMap[cmd+";dc="+dc+";namespace="+ns], ";", "=")
			dcMap[ConfigNamespaceContext].(lib.Stats)[ns] = nsMap
		}
	}

	return xdrConfigMap
}

func parseAllDcConfig(rawMap map[string]string, cmd string) lib.Stats {
	dcConfigMap := make(lib.Stats)
	dcNames := getNames(rawMap[constStatDCNames])

	for _, dc := range dcNames {
		m := parseIntoDcMap(rawMap[cmd+dc], ":", "=")
		if len(m) > 0 {
			dcConfigMap[dc] = m
		}
	}

	return dcConfigMap
}

func parseBasicConfigInfo(res, sep string) lib.Stats {
	// Parse
	conf := ParseIntoMap(res, ";", sep)
	return conf
}

func parseConfigRacksInfo(res string) []lib.Stats {
	ml := parseIntoListOfMap(res, ";", ":", "=")

	// "racks" command return a list of racks and nodeID per namespace eg:ns=test:rack_1=1A0,1A1:rack_2=2A0,2A1
	// nodeID take hexadecimal values, so value 12345 is also a valid nodeID. So convert all int values to string values.
	for idx := range ml {
		for key, value := range ml[idx] {
			if v, ok := value.(int64); ok {
				ml[idx][key] = strconv.FormatInt(v, 10)
			}
		}
	}

	return ml
}

// ***************************************************************************
// parse metadata
// ***************************************************************************

func parseMetadataInfo(rawMap map[string]string) lib.Stats {
	metaMap := make(lib.Stats)

	metaMap["node_id"] = rawMap[cmdMetaNodeID]
	metaMap["asd_build"] = rawMap[cmdMetaBuild]
	metaMap["service"] = parseListTypeMetaInfo(rawMap, cmdMetaService)
	metaMap["services"] = parseListTypeMetaInfo(rawMap, cmdMetaServices)
	metaMap["services-alumni"] = parseListTypeMetaInfo(
		rawMap, cmdMetaServicesAlumni,
	)
	metaMap["services-alternate"] = parseListTypeMetaInfo(
		rawMap, cmdMetaServicesAlternate,
	)
	metaMap["features"] = parseListTypeMetaInfo(rawMap, cmdMetaFeatures)
	metaMap["edition"] = rawMap[cmdMetaEdition]
	metaMap["version"] = rawMap[cmdMetaVersion]
	metaMap["build_os"] = rawMap[cmdMetaBuildOS]

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

// ***************************************************************************
// parse latency

// TODO: check diff lat bucket in agg
// typical format is {test}-read:10:17:37-GMT,ops/sec,>1ms,>8ms,>64ms;10:17:47,29648.2,3.44,0.08,0.00;
func parseLatencyInfo(log logr.Logger, rawStr string) lib.Stats {
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

		if err = ip.Expect("-"); err != nil {
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

		if _, err = ip.ReadUntil(','); err != nil {
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
			log.Error(fmt.Errorf("parsing latency values"), "buckets not equal")
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
				nstats["valBuckets"] = append(
					nstats["valBuckets"].([]float64),
					make([]float64, len(buckets[len(nBuckets):]))...,
				)
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

	copy(valBuckets, vb)

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

// ***************************************************************************
// utils
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

func ParseIntoMap(str, del, sep string) lib.Stats {
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
		if m[kv[0]] != nil {
			// If key already exists assume it is a list of strings.
			// This was chosen rather than turning the value into []string to
			// remove the possibility of two types (string or []string) or of
			// maintaining a list of fields which could also be strings
			if strKv0, ok := m[kv[0]].(string); ok {
				m[kv[0]] = strKv0 + "," + kv[1]
			}
		} else {
			if utils.IsStringField(kv[0]) {
				m[kv[0]] = kv[1]
			} else {
				m[kv[0]] = getParsedValue(kv[1])
			}
		}
	}

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
	}

	return valStr
}

func parseIntoListOfMap(str, del1, del2, sep string) []lib.Stats {
	strList := strings.Split(str, del1)
	strListMap := make([]lib.Stats, 0, len(strList))

	for _, s := range strList {
		if s == "" {
			continue
		}

		m := ParseIntoMap(s, del2, sep)
		// Assume indexname is always there
		strListMap = append(strListMap, m)
	}

	return strListMap
}

func parseIntoDcMap(str, del, sep string) lib.Stats {
	if str == "" {
		return nil
	}

	m := make(lib.Stats)
	items := strings.Split(str, del)
	newItems := make([]string, len(items))
	nIdx := 0

	for _, item := range items {
		if item == "" {
			newItems[nIdx-1] += del
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
