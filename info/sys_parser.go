package info

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	log "github.com/inconshreveable/log15"

	lib "github.com/aerospike/aerospike-management-lib"
	"github.com/aerospike/aerospike-management-lib/system"
)

type NodeSysStats = lib.Stats

var RunCmd = map[string][]string{
	"hostname":   []string{"hostname -I", "hostname"},
	"top":        []string{"top -n1 -b", "top -l 1"},
	"lsb":        []string{"lsb_release -a", "ls /etc|grep release|xargs -I f cat /etc/f"},
	"meminfo":    []string{"cat /proc/meminfo", "vmstat -s"},
	"dmesg":      []string{"dmesg -T", "dmesg"},
	"interrupts": []string{"cat /proc/interrupts"},
	"iostat":     []string{"iostat -x 1 1"},
	"limits":     []string{"sudo  pgrep asd | xargs -I f sh -c 'sudo cat /proc/f/limits'"},
	"lscpu":      []string{"lscpu"},
	"sysctlall":  []string{"sudo sysctl vm fs"},
	"iptables":   []string{"sudo iptables -S"},
	"hdparm":     []string{"sudo fdisk -l |grep Disk |grep dev | cut -d ' ' -f 2 | cut -d ':' -f 1 | xargs sudo hdparm -I 2>/dev/null", "sudo fdisk -l |grep Disk |grep dev | cut -d ' ' -f 2 | cut -d ':' -f 1 | xargs sudo hdparm -I 2>/dev/null"},
	"df":         []string{"df -h"},
	"free-m":     []string{"free -m"},
	"uname":      []string{"uname -a"},
}

var RunCmdKeys = []string{"hostname", "top", "lsb", "meminfo", "interrupts", "iostat",
	"dmesg", "limits", "lscpu", "sysctlall", "iptables", "hdparm", "df", "free-m", "uname"}

var sysCmdMaxRetry = 2

// SysInfo provides system level information on a machine.
type SysInfo struct {
	// FIXME should it have all the shit about system connection
	system *system.System
	log    log.Logger
}

// NewSysInfo returns a new SysInfo
func NewSysInfo(system *system.System) (*SysInfo, error) {
	if system == nil {
		return nil, fmt.Errorf("system connection object nil")
	}
	sys := &SysInfo{
		system: system,
		log:    pkglog.New(log.Ctx{"node": system}),
	}
	return sys, nil
}

// Close closes all the connections to the system.
// SysInfo is not usable after this call.
func (s *SysInfo) Close() error {
	if s.system != nil {
		return s.system.Close()
	}
	s.system = nil
	return nil
}

//***************************************************************************
// wrapper function for system command parser
//***************************************************************************

// GetSysInfo fetch and parse info for given commands
func (s *SysInfo) GetSysInfo(cmdList ...string) NodeSysStats {

	if len(cmdList) == 0 {
		cmdList = RunCmdKeys
	}
	var wg sync.WaitGroup
	var lock = sync.RWMutex{}
	sysMap := make(NodeSysStats)

	wg.Add(len(cmdList))

	for _, cmd := range cmdList {
		go func(cmd string) {
			defer wg.Done()

			cmdOutput, stderr, err := s.RunSysCmd(RunCmd[cmd]...)
			if err != nil {
				s.log.Debug("failed to run system command", log.Ctx{"command": cmd, "stderr": stderr, "err": err})
				return
			}

			var m lib.Stats

			switch cmd {
			case "uname":
				m = parseUnameInfo(cmdOutput)
			case "meminfo":
				m = parseMemInfo(cmdOutput)
			case "df":
				m = parseDfInfo(cmdOutput)
			case "free-m":
				m = parseFreeMInfo(cmdOutput)
			case "hostname":
				m = parseHostnameInfo(cmdOutput)
			case "dmesg":
				m = parseDmesgInfo(cmdOutput)
			case "lscpu":
				m = parseLscpuInfo(cmdOutput)
			case "iptables":
				m = parseIptablesInfo(cmdOutput)
			case "sysctlall":
				m = parseSysctlallInfo(cmdOutput)
			case "hdparm":
				m = parseHdparmInfo(cmdOutput)
			case "limits":
				m = parseLimitsInfo(cmdOutput)
			case "interrupts":
				m = parseInterruptsInfo(cmdOutput)
			case "top":
				m = parseTopInfo(cmdOutput)
			case "lsb":
				m = parseLsbInfo(cmdOutput)
			case "iostat":
				m = parseIostatInfo(cmdOutput)
			default:
				s.log.Debug("invalid cmd to parse sysinfo", log.Ctx{"command": cmd})
			}

			lock.Lock()
			sysMap[cmd] = m
			lock.Unlock()

		}(cmd)
	}
	wg.Wait()
	return sysMap
}

// RunSysCmd execute first valid command from given command list
func (s *SysInfo) RunSysCmd(cmds ...string) (stdout, stderr string, err error) {
	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		for i := 0; i < sysCmdMaxRetry; i++ {
			stdout, stderr, err = s.system.RunWithSudo(cmd)
			if stdout != "" {
				return stdout, stderr, err
			}
		}
	}
	return stdout, stderr, err
}

//***************************************************************************
// system commands parser
//***************************************************************************

// parseInterruptsInfo parse interrupts info
// cmdOutput: (output of command - "cat /proc/interrupts")
func parseInterruptsInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	intList := make([]interface{}, 0)
	lines := strings.Split(cmdOutput, "\n")

	var cpuToks []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "CPU") {
			cpuToks = strings.Fields(line)
			continue
		}
		if strings.Contains(line, "Tx_rx") {
			toks := strings.Fields(line)
			cpuList := toks[1 : len(toks)-2]

			devObj := make(lib.Stats)
			devObj["device_name"] = toks[len(toks)-1]
			devObj["interrupt_id"] = strings.Replace(toks[0], ":", "", -1)
			devObj["interrupt_type"] = toks[len(toks)-2]

			m := make(lib.Stats)
			for idx, cpu := range cpuToks {
				m[cpu] = cpuList[idx]
			}
			devObj["interrupts"] = m
			intList = append(intList, devObj)
		}

	}
	m["device_interrupts"] = intList
	return m
}

// parseLimitsInfo parse limits info
// cmdOutput: (output of command - "sudo  pgrep asd | xargs -I f sh -c “sudo cat /proc/f/limits”")
// Max cpu time              unlimited            unlimited            seconds
// Max file size             unlimited            unlimited            bytes
var regexLimitInfo = regexp.MustCompile("  +")

func parseLimitsInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" || !strings.Contains(line, "Max") {
			continue
		}
		toks := regexLimitInfo.Split(line, -1)
		if len(toks) < 4 {
			continue
		}
		k := strings.Trim(toks[0], " ")
		m["Soft "+k] = strings.Trim(toks[1], " ")
		m["Hard "+k] = strings.Trim(toks[2], " ")
	}
	return m
}

// parseSysctlallInfo parse sysctl info
// cmdOutput: (output of command - "sudo sysctl vm fs")
func parseSysctlallInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		kv := strings.Split(line, "=")
		m[strings.Trim(kv[0], " ")] = strings.Trim(kv[1], " ")
	}
	return m
}

// parseHdparmInfo parse hdparm info
// cmdOutput: (output of command - "sudo fdisk -l |grep Disk |grep dev | cut -d ” ” -f 2 | cut -d “:” -f 1 | xargs sudo hdparm -I 2>/dev/null")
func parseHdparmInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")

	var device string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "/dev") {
			device = line
		}
		if strings.Contains(line, "Sector size") ||
			strings.Contains(line, "device size") ||
			strings.Contains(line, "Model Number") ||
			strings.Contains(line, "Serial Number") ||
			strings.Contains(line, "Firmware Revision") ||
			strings.Contains(line, "Transport") ||
			strings.Contains(line, "Queue Depth") {
			kv := strings.Split(line, ":")
			if len(kv) != 2 {
				continue
			}
			k := device + strings.Trim(kv[0], " ")
			v := strings.Trim(kv[1], " ")
			m[k] = v
		}
	}
	return m
}

// parseIptablesInfo parse iptables info
// cmdOutput: (output of command - "sudo iptables -S")
func parseIptablesInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	m["has_firewall"] = false

	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "DROP") {
			m["has_firewall"] = true
			return m
		}
	}
	return m
}

// parseLscpuInfo parse lscpu info
// cmdOutput: (output of command - "lscpu")
func parseLscpuInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		kv := strings.Split(line, ":")
		m[strings.Trim(kv[0], " ")] = strings.Trim(kv[1], " ")
	}
	return m
}

// parseDmesgInfo parse dmesg info
// cmdOutput: (output of command - "dmesg -T", "dmesg")
func parseDmesgInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	m["OOM"] = false
	m["Blocked"] = false

	lines := strings.Split(cmdOutput, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "OOM") {
			m["OOM"] = true
		}
		if strings.Contains(line, "blocked for more than") {
			m["Blocked"] = true
		}
		if strings.Contains(line, "Linux version") {
			m["OS"] = line
		}
	}
	return m
}

// parseUnameInfo parse uname info
// cmdOutput: (output of command - "uname -a")
func parseUnameInfo(cmdOutput string) lib.Stats {
	// Linux ubuntu 4.8.0-39-generic #42-Ubuntu SMP Mon Feb 20 11:47:27 UTC 2017
	//fmt.Println(cmdOutput)

	m := make(lib.Stats)
	cmdOutput = strings.Trim(strings.Split(cmdOutput, "#")[0], " ")
	dataList := strings.Split(cmdOutput, " ")

	m["kernel_name"] = dataList[0]
	m["nodename"] = dataList[1]
	m["kernel_release"] = dataList[2]

	return m
}

// parseMemInfo parse mem info
// cmdOutput: (output of command - "cat /proc/meminfo", "vmstat -s")
func parseMemInfo(cmdOutput string) lib.Stats {
	// MemTotal:        8415676 kB
	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var key, val string
		if strings.Contains(line, ":") {
			// "cat /proc/meminfo" -> MemTotal:        8415684 kB
			// TODO: str to lower, and match
			if !strings.Contains(line, " kB") {
				continue
			}
			kv := strings.Split(line, ":")
			key = kv[0]
			val = strings.Split(strings.Trim(kv[1], " "), " ")[0]
		} else {
			// "vmstat -s" -> 8415684 K total memory
			if !strings.Contains(line, " K ") {
				continue
			}
			kv := strings.Split(line, " K ")
			key = strings.Trim(kv[1], " ")
			val = strings.Trim(kv[0], " ")
		}
		// TODO should replace or not. check lscpu
		key = strings.Replace(key, " ", "_", -1)
		vali, err := strconv.Atoi(val)
		if err != nil {
			pkglog.Debug("failed to parse memInfo val %s: %v", val, err)
		}
		vali = vali * 1024
		m[key] = vali
	}
	return m
}

// parseDfInfo parse df info
// cmdOutput: (output of command - "df -h")
// "Filesystem             Size  Used Avail Use% Mounted on\n",
// "/dev/xvda1             7.8_g  1.6_g  5.9_g  21% /\n",
// "none                   4.0_k     0  4.0_k   0% /sys/fs/cgroup\n",
//
// Filesystem           1K-blocks      Used Available Use% Mounted on
// /dev/mapper/VolGroup-lv_root
//                     37613080   1237888  34457856   4% /
// tmpfs                   961076         0    961076   0% /dev/shm
//
// output: [{name: A, size: A, used: A, avail: A, %use: A, mount_point: A},{}]
func parseDfInfo(cmdOutput string) lib.Stats {
	m := make(lib.Stats)
	var fsList []lib.Stats
	tokCount := 6
	secStart := false
	sizeInKb := false
	skip := false

	lines := strings.Split(cmdOutput, "\n")
	for i, line := range lines {
		// Handle multiline output
		// /dev/mapper/VolGroup-lv_root
		//             37613080   1237888  34457856   4% /
		if skip {
			skip = false
			continue
		}

		fs := make(lib.Stats)
		if line == "" {
			continue
		}
		if strings.Contains(line, "Filesystem") &&
			strings.Contains(line, "Used") {
			secStart = true
			continue
		}
		if strings.Contains(line, "1_k-block") || strings.Contains(line, "1K-blocks") {
			sizeInKb = true
			continue
		}

		if secStart {
			tokens := strings.Fields(line)
			if len(tokens) != tokCount {
				if i >= len(lines)-1 {
					continue
				}
				nextLineToks := strings.Fields(lines[i+1])
				if len(tokens) == 1 && len(nextLineToks) == tokCount-1 {
					tokens = append(tokens, nextLineToks...)
					skip = true
				} else {
					fs["Error"] = fmt.Sprintf("Token len mismatch %s", line)
					fsList = append(fsList, fs)
					continue
				}
			}
			// TODO: should we fail here? in case there is parsing err
			var e error
			fs["name"] = tokens[0]
			fs["size"], e = getMemInByteFromStr(tokens[1], 1)
			if e != nil {
				pkglog.Debug("failed to parse `size` from `df` cmd output `%s`: %v", tokens[1], e)
			}
			fs["used"], e = getMemInByteFromStr(tokens[2], 1)
			if e != nil {
				pkglog.Debug("failed to parse `used` from `df` cmd output `%s`: %v", tokens[2], e)
			}
			fs["avail"], e = getMemInByteFromStr(tokens[3], 1)
			if e != nil {
				pkglog.Debug("failed to parse `avail` from `df` cmd output `%s`: %v", tokens[3], e)
			}
			fs["%use"] = strings.Replace(tokens[4], "%", "", -1)
			fs["mount_point"] = tokens[5]
			if sizeInKb {
				fs["size"] = fs["size"].(int64) * 1024
			}
			fsList = append(fsList, fs)
		}
	}
	m["Filesystems"] = fsList
	return m
}

func getMemInByteFromStr(memstr string, memUnitLen int) (int64, error) {
	if memstr == "0" {
		return 0, nil
	}
	if strings.Contains(memstr, ",") {
		memstr = strings.Replace(memstr, ",", ".", 1)
	}

	if strings.ContainsAny(memstr, "Kk") {
		return getBytesFromFloat(memstr, 10, memUnitLen)
	} else if strings.ContainsAny(memstr, "Mm") {
		return getBytesFromFloat(memstr, 20, memUnitLen)
	} else if strings.ContainsAny(memstr, "Gg") {
		return getBytesFromFloat(memstr, 30, memUnitLen)
	} else if strings.ContainsAny(memstr, "Tt") {
		return getBytesFromFloat(memstr, 40, memUnitLen)
	} else if strings.ContainsAny(memstr, "Pp") {
		return getBytesFromFloat(memstr, 50, memUnitLen)
	} else if strings.ContainsAny(memstr, "Ee") {
		return getBytesFromFloat(memstr, 60, memUnitLen)
	} else if strings.ContainsAny(memstr, "Zz") {
		return getBytesFromFloat(memstr, 70, memUnitLen)
	} else if strings.ContainsAny(memstr, "Yy") {
		return getBytesFromFloat(memstr, 80, memUnitLen)
	} else {
		return strconv.ParseInt(memstr, 10, 64)
	}
}

func getBytesFromFloat(memstr string, shift uint32, memUnitLen int) (int64, error) {
	var memnum float64
	var err error
	if memUnitLen == 0 {
		memnum, err = strconv.ParseFloat(memstr, 64)
		if err != nil {
			return 0, err
		}
	} else {
		s := memstr[:len(memstr)-memUnitLen]
		memnum, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
	}
	if memstr == "0" {
		memnum, err = strconv.ParseFloat(memstr, 64)
		return int64(memnum), nil
	}

	// Assumption: all conversion float <-> int will work
	i, f := math.Modf(memnum)
	num := 1 << shift
	totalmem := (i * float64(num)) + (f * float64(num))
	return int64(totalmem), nil
}

// compile these at start
var topUptime1 = regexp.MustCompile(`.*up (?P<uptime>.*) days.*`)
var topUptime2 = regexp.MustCompile(`.*up.* (?P<hr>.*):(?P<min>.*),.* load .*`)
var topUptime3 = regexp.MustCompile(`.* (?P<min>.*) min`)

func parseTopLine(line string) lib.Stats {
	obj1 := topUptime1.FindStringSubmatch(line)
	obj2 := topUptime2.FindStringSubmatch(line)
	obj3 := topUptime3.FindStringSubmatch(line)

	var days, hr, mn int
	if len(obj1) != 0 {
		days = toInt(obj1[1])
	}
	if len(obj2) != 0 {
		hr = toInt(obj2[1])
		mn = toInt(obj2[2])
	}
	if len(obj3) != 0 {
		mn = toInt(obj3[1])
	}
	out := lib.Stats{}
	out["seconds"] = (days * 24 * 60 * 60) + (hr * 60 * 60) + (mn * 60)
	return out
}

func parseTopKeyValLine(str string, del1 string, del2List []string) lib.Stats {
	str = strings.Split(str, ":")[1]
	kvPairs := strings.Split(str, del1)
	out := lib.Stats{}
	for _, kv := range kvPairs {
		kv = strings.Trim(kv, " ")
		for _, del2 := range del2List {
			if strings.Contains(kv, del2) {
				kvList := strings.Split(kv, del2)
				out[kvList[1]] = kvList[0]
				break
			}
		}
	}
	return out
}

// compile these at start
var topSwap1 = regexp.MustCompile(`.*Swap:.* (?P<total>.*).total.* (?P<used>.*).used.* (?P<free>.*).free.* (?P<ca>.*).ca.*`)
var topSwap2 = regexp.MustCompile(`.*Swap:.* (?P<total>.*).total.* (?P<free>.*).free.* (?P<used>.*).used.* (?P<av>.*).av.*`)

func parseSwapLine(line string) lib.Stats {
	obj1 := topSwap1.FindStringSubmatch(line)
	obj2 := topSwap2.FindStringSubmatch(line)

	out := lib.Stats{}
	if len(obj1) != 0 {
		out["total"] = obj1[1]
		out["used"] = obj1[2]
		out["free"] = obj1[3]
		out["cached"] = obj1[4]
	} else if len(obj2) != 0 {
		out["total"] = obj2[1]
		out["free"] = obj2[2]
		out["used"] = obj2[3]
		out["avail"] = obj2[4]
	} else {
		//out["Error"] = "can not parse swap section"
	}
	return out
}

func parseASDLine(line string) lib.Stats {
	fields := strings.Fields(line)
	out := lib.Stats{}
	out["%cpu"] = fields[8]
	out["%mem"] = fields[9]
	return out
}

func toInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return -1
	}
	return i
}

func parseTopInfo(cmdOutput string) lib.Stats {
	lines := strings.Split(cmdOutput, "\n")
	top := lib.Stats{}
	asdFlag := false
	for _, line := range lines {
		if strings.Contains(line, "up") && strings.Contains(line, "load average") {
			top["uptime"] = parseTopLine(line)
		} else if strings.Contains(line, "Tasks") && strings.Contains(line, "total") {
			//top["tasks"] = parseTopKeyValLine(line, ",", []string{" "})
		} else if strings.Contains(line, "Cpu(s):") && strings.Contains(line, "us") {
			top["cpu_utilization"] = parseTopKeyValLine(line, ",", []string{" ", "%"})
		} else if strings.Contains(line, "Mem") && strings.Contains(line, "total") {
			//top["ram"] = parseTopKeyValLine(line, ",", []string{" ", "+"})
		} else if strings.Contains(line, "Swap") && strings.Contains(line, "total") {
			//top["swap"] = parseSwapLine(line)
		} else if !asdFlag && strings.Contains(line, "asd") {
			top["asd_process"] = parseASDLine(line)
		}
	}
	return top
}

// parseFreeMInfo parse free-mem info
// cmdOutput: (output of command - "free -m")
//              total        used        free      shared  buff/cache   available
//Mem:           8218        4379         475          63        3363        3450
//Swap:          2047           6        2041

// "             total       used       free     shared    buffers     cached\n",
// "Mem:         32068      31709        358          0         17      13427\n",
// "-/+ buffers/cache:      18264      13803\n",
// "Swap:         1023        120        903\n",
//
// output: {mem: {}, buffers/cache: {}, swap: {}}
func parseFreeMInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	secStart := false
	var headerTokens []string
	lines := strings.Split(cmdOutput, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "total") {
			// May check other tokens if any new format came
			// Multiple format in header, so can't hardcode
			headerTokens = strings.Fields(line)
			secStart = true
			continue
		}

		tokens := strings.Fields(line)
		// TODO this secStart check can be removed
		if secStart && strings.Contains(line, "Mem:") {
			memObj := make(lib.Stats)
			for idx, tok := range headerTokens {
				memObj[tok] = tokens[idx+1]
			}
			m["mem"] = memObj
			continue
		}
		if secStart && strings.Contains(line, "-/+ buffers/cache:") {
			buffObj := make(lib.Stats)
			buffObj[headerTokens[1]] = tokens[2]
			buffObj[headerTokens[2]] = tokens[3]
			m["buffers/cache"] = buffObj
			continue
		}
		if secStart && strings.Contains(line, "Swap:") {
			swapObj := make(lib.Stats)
			swapObj[headerTokens[0]] = tokens[1]
			swapObj[headerTokens[1]] = tokens[2]
			swapObj[headerTokens[2]] = tokens[3]
			m["swap"] = swapObj
			continue
		}
	}
	return m
}

func parseLsbInfo(cmdOutput string) lib.Stats {
	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")
	amazon := false
	for _, line := range lines {
		// "LSB Version:\t:base-4.0-amd64:base-4.0-noarch:core-4.0-amd64:""
		// "Description:\t_cent_oS release 6.4 (Final)\n"

		// "Red Hat Enterprise Linux Server release 6.7 (Santiago)\n"
		// "Cent_oS release 6.7 (Final)\n"

		// Few formats have only PRETTY_NAME, so need to add this condition.
		// "PRETTY_NAME=\"Ubuntu 14.04.2 LTS\"\n"

		if strings.Contains(line, "Description") ||
			strings.Contains(line, " release ") ||
			strings.Contains(line, "PRETTY_NAME") {

			m["description"] = line

			if strings.Contains(strings.ToLower(line), "amazon") &&
				strings.Contains(strings.ToLower(line), "ami") {
				amazon = true
			}
			break
		}
	}
	if amazon {
		// TODO: TBD
		// parse os_age_months
	}
	return m
}

// "iostat -x 1 10\n",
// "Linux 2.6.32-279.el6.x86_64 (bfs-dl360g8-02) \t02/02/15 \t_x86_64_\t(24 CPU)\n",
// "avg-cpu:  %user   %nice %system %iowait  %steal   %idle\n",
// "           0.78    0.00    1.44    0.26    0.00   97.51\n",
// "\n",
// "Device:         rrqm/s   wrqm/s     r/s     w/s   rsec/s   wsec/s avgrq-sz avgqu-sz   await  svctm  %util\n",
// "sdb               0.00     4.00    0.00    4.00     0.00    64.00    16.00     0.02    5.75   4.00   1.60\n",
//
// output: [{avg-cpu: {}, device_stat: {}}, .........]
func parseIostatInfo(cmdOutput string) lib.Stats {
	m := make(lib.Stats)
	lines := strings.Split(cmdOutput, "\n")
	var iostatL []lib.Stats
	var sectionList [][]string
	var section []string
	start := false
	for _, line := range lines {
		if strings.Contains(line, "avg-cpu") && strings.Contains(line, "user") {
			if start {
				sectionList = append(sectionList, section)
				section = []string{}
			}
			start = true
		}
		section = append(section, line)
	}
	sectionList = append(sectionList, section)

	for _, sec := range sectionList {
		iostatL = append(iostatL, parseIostatSection(sec))
	}
	m["Iostat_Sections"] = iostatL
	return m
}

func parseIostatSection(section []string) lib.Stats {
	var avgCPUToks, deviceToks []string
	var deviceList []lib.Stats
	avgCPU := false
	device := false
	devMap := lib.Stats{}

	for _, line := range section {
		if line == "" {
			continue
		}
		if strings.Contains(line, "avg-cpu") {
			tokStr := strings.Split(line, ":")[1]
			avgCPUToks = strings.Fields(tokStr)
			avgCPU = true
			device = false
			continue
		} else if strings.Contains(line, "Device") {
			tokStr := strings.Replace(line, ":", "", -1)
			deviceToks = strings.Fields(tokStr)
			avgCPU = false
			device = true
			continue
		}

		if avgCPU {
			m := lib.Stats{}
			toks := strings.Fields(line)
			for i, tok := range avgCPUToks {
				m[tok] = toks[i]
			}
			devMap["avg-cpu"] = m

		} else if device {
			m := lib.Stats{}
			toks := strings.Fields(line)
			for i, tok := range deviceToks {
				m[tok] = toks[i]
			}
			deviceList = append(deviceList, m)
		}
		devMap["device_stat"] = deviceList
	}
	return devMap
}

// parseHostnameInfo parse hostname info
// cmdOutput: (output of command - "hostname -I", "hostname")
// "hostname\n",
// "rs-as01\n",
// output: {hostname: {'hosts': [...................]}}
func parseHostnameInfo(cmdOutput string) lib.Stats {

	m := make(lib.Stats)
	m["hosts"] = strings.Fields(cmdOutput)
	return m
}
