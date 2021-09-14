package system

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
)

// General errors returned by this package.
var (
	ErrUnknownServiceSystem = errors.New("unknown service system")
	ErrUknownPackageManager = errors.New("unknown package manager")
)

// PkgMgr represents the package manager of the system
type PkgMgr int

const (
	UnknownPkgMgr PkgMgr = iota // not one of the known package managers that the system can work with
	DpkgPkgMgr
	RpmPkgMgr
)

// UpdateMethod represents the package update method on the system
type UpdateMethod int

const (
	UnknownUpdateMethod UpdateMethod = iota //  not one of the known package update methods that the system can work with
	YumUpdateMethod
	AptUpdateMethod
)

// ServiceSystem represents the init system
type ServiceSystem int

const (
	UknownServiceSystem ServiceSystem = iota // not one of the known service systems that the system can work with
	InitServiceSystem
	SystemdServiceSystem
	UpstartServiceSystem
)

// Job represents a job that is running on the system.
type Job struct {
	cmd       BigCommand
	cancel    func()
	cancelled bool
	completed bool
}

// newJob creates a new job for the command.
func newJob(cmd BigCommand, cancel func()) *Job {
	return &Job{
		cmd:    cmd,
		cancel: cancel,
	}
}

// Cancel cancels the job.
func (j *Job) Cancel() {
	if j.cancelled || j.completed {
		return
	}

	defer func() {
		j.cancelled = true
	}()
	j.cancel()
}

// IsComplete returns true if the job completed.
func (j *Job) IsComplete() bool {
	return j.completed
}

// IsCancelled returns true if the job was cancelled.
func (j *Job) IsCancelled() bool {
	return j.cancelled
}

// Cmd returns the command the job was run with.
func (j *Job) Cmd() BigCommand {
	return j.cmd
}

// Command represents a command line command to run on the system.
//
// Command should be used for short running commands with not much output.
// For long running commands or commands with lot of output to process please
// see BigCommand.
type Command interface {
	// Sudo should return true if the command needs sudo privileges to run.
	Sudo() bool

	// Cmd should return the command line command to run.
	Cmd() string

	// Parse should parse the output of the command.
	//
	// This method is called after the command has executed. stdout, stderr are what
	// is printed by the command on the standard output and standard error when run.
	// err holds the error value of running the command.
	Parse(stdout, stderr string, err error)
}

// BigCommand represents a command line command that is long running or produces
// a lot of output.
type BigCommand interface {
	// Sudo should return true if the command needs sudo privileges to run.
	Sudo() bool

	// Cmd should return the command line command to run.
	Cmd() string

	// Parse should parse the output of the command.
	//
	// This method is called while the command is executing.
	// stdout, stderr are the output printed by the command on the standard output
	// and standard error. It is the responsibility of the parser to consume the
	// outputs at the rate at which it is produced.
	// If there is an error on command execute it is passed on the channel err.
	// The cancel channel will be closed when the job is cancelled. It is the
	// responsibility of the implementation to stop parsing, do cleanup and return.
	Parse(stdout, stderr io.Reader, err <-chan error, cancel <-chan struct{})
}

// System is an abstraction over the software capabilities of a machine.
type System struct {
	hostName string

	sshConfig *ssh.ClientConfig
	sshPort   int
	sudo      *Sudo

	homeDir   string
	osName    string
	osVersion string

	pkgMgr        PkgMgr
	updateMethod  UpdateMethod
	serviceSystem ServiceSystem

	// SSH policies might be such that the connection is valid only for certain
	// time period, connection might be idle for certain time period, etc. In
	// these cases ssh pacakge does not provide a way to validate a connection.
	// These data structures help to alleviate these issues. See newSession for
	// futher details.
	clientMut   sync.Mutex
	client      *ssh.Client   // client to be used
	prevClients []*ssh.Client // all previous clients

	log logr.Logger
}

// New returns a system connected by ssh on the hostName, sshPort with the given sudo privileges.
//
// The sudo privileges will be used across all commands that require it.
func New(log logr.Logger, hostName string, sshPort int, sudo *Sudo, config *ssh.ClientConfig) (*System, error) {
	logger := log.WithValues("ip", hostName, "port", sshPort)
	logger.V(1).Info("creating new ssh system")

	client, err := newClient(hostName, sshPort, config)
	if err != nil {
		return nil, err
	}

	logger = log.WithValues("host", hostName)
	s := &System{
		hostName:  hostName,
		sshPort:   sshPort,
		sshConfig: config,
		sudo:      sudo,
		client:    client,
		log:       logger,
	}

	s.init()

	return s, nil
}

// NewSystem creates ssh.config and returns a system connected by ssh on the hostName, sshPort with the given sudo privileges.
//
// The sudo privileges will be used across all commands that require it.
func NewSystem(log logr.Logger, hostName string, sshPort int, sudo *Sudo, userName, password string, sshKey []byte) (*System, error) {
	config, err := newSSHConfig(userName, password, sshKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh config: %v", err)
	}
	return New(log, hostName, sshPort, sudo, config)
}

// newSession creates a new session.
func (sys *System) newSession() (*ssh.Session, error) {
	sys.clientMut.Lock()
	defer sys.clientMut.Unlock()

	if sys.client != nil {
		session, err := sys.client.NewSession()
		if err == nil {
			return session, nil
		}

		// NewSession might be rejected for a lot of reasons one of them being
		// stale, invalid client connection. To handle the case of invalid
		// create a new client connection and send requests. But in the other
		// cases the client might still be valid and in use by long running
		// commands. So not closing the client. Collect all these clients and
		// close them when Close is called on the System.
		sys.log.V(-1).Info("failed to create new session", "err", err, "clientPoolSize", len(sys.prevClients))
		sys.prevClients = append(sys.prevClients, sys.client)
		sys.client = nil
	}

	sys.log.V(1).Info("creating new ssh client")
	client, err := newClient(sys.hostName, sys.sshPort, sys.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ssh client: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		// failed to create new session
		// close client and return error
		client.Close()
		return nil, fmt.Errorf("failed to create new ssh session: %v", err)
	}

	sys.client = client
	return session, nil
}

// sshConfig returns the ssh configuration of the cluster.
func newSSHConfig(userName, password string, sshKey []byte) (config *ssh.ClientConfig, err error) {
	if len(sshKey) > 0 {
		signer, err := ssh.ParsePrivateKey(sshKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		config = &ssh.ClientConfig{
			User: userName,
			Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		}
	} else {
		config = &ssh.ClientConfig{
			User: userName,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // FIXME
		}
	}
	return config, err
}

func newClient(hostName string, sshPort int, config *ssh.ClientConfig) (*ssh.Client, error) {
	addr := net.JoinHostPort(hostName, strconv.Itoa(sshPort))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s, %d: %v", hostName, sshPort, err)
	}
	return client, nil
}

// init initializes the set of parameters the system is running with
func (sys *System) init() {
	// os
	os, ver, err := sys.fetchOSType()
	if err == nil {
		sys.osName = os
		sys.osVersion = ver
	} else {
		sys.log.V(1).Info("failed to fetch os info", "err", err)
	}

	// package manager
	switch os {
	case "ubuntu", "debian":
		sys.pkgMgr = DpkgPkgMgr
		sys.updateMethod = AptUpdateMethod
	case "redhat", "centos":
		sys.pkgMgr = RpmPkgMgr
		sys.updateMethod = YumUpdateMethod
	default:
		sys.pkgMgr = UnknownPkgMgr
		sys.updateMethod = UnknownUpdateMethod
		sys.log.V(1).Info("package manager and update method unknown", "os", os)
	}

	// service system
	svc, err := sys.fetchServiceSystem()
	if err == nil {
		sys.serviceSystem = svc
	} else {
		sys.log.V(1).Info("failed to fetch service system info", "err", err)
		sys.serviceSystem = UknownServiceSystem
	}

	// home directory
	out, _, _ := sys.Run("printenv HOME")
	sys.homeDir = strings.TrimSpace(out)
}

func (sys *System) fetchServiceSystem() (ServiceSystem, error) {
	svc := newServiceFacts()
	sys.RunCmd(svc)

	if err := svc.Err(); err != nil {
		return UknownServiceSystem, fmt.Errorf("failed to collect system service fact: %v", err)
	}

	init := svc.Facts()
	switch init {
	case "init":
		return InitServiceSystem, nil
	case "systemd":
		return SystemdServiceSystem, nil
	case "upstart":
		return UpstartServiceSystem, nil
	}

	return UknownServiceSystem, ErrUnknownServiceSystem
}

func (sys *System) fetchOSType() (name, version string, err error) {
	os := newOSFacts()
	sys.RunCmd(os)

	name, version = os.Facts()
	if err := os.Err(); err != nil {
		return "", "", fmt.Errorf("failed to collect system os fact: %v", err)
	}

	return name, version, nil
}

// Close closes the connections held by system.
func (sys *System) Close() error {
	sys.clientMut.Lock()
	defer sys.clientMut.Unlock()

	clients := sys.prevClients
	if sys.client != nil {
		clients = append(clients, sys.client)
	}

	var err error
	for _, client := range clients {
		if e := client.Close(); e != nil {
			err = e // the last error
		}
	}
	return fmt.Errorf("failed to close ssh client: %v", err) // is nil if no errors
}

// OSName returns the name operating system running on the system.
func (sys *System) OSName() string {
	return sys.osName
}

// OSVersion returns the major and minor version of the operating system
// running on the system.
func (sys *System) OSVersion() (major, minor string) {
	s := strings.Split(sys.osVersion, ".")

	major = s[0]
	if len(s) > 1 {
		minor = s[1]
	}

	return
}

// InitManager returns the init manager on the system.
func (sys *System) InitManager() ServiceSystem {
	return sys.serviceSystem
}

// PackageManager returns the package manager on the system.
func (sys *System) PackageManager() PkgMgr {
	return sys.pkgMgr
}

// NetInterfaces returns the list of network interfaces on the system.
func (sys *System) NetInterfaces() []string {
	ni := newNetInterfaceFacts()
	sys.RunCmd(ni)
	return ni.Facts()
}

// Devices returns the list of devices on the system.
func (sys *System) Devices() []string {
	// TODO: Complete code to fetch and parse devices
	// Discussed with Sunil.
	// cat /proc/partition gives us device names but not full path. Also it does not give proper name for mappers.
	// lsblk can give proper tree structure and full path. But it is not working on centos7 docker
	// Script:
	// lsblk --raw --noheadings -o NAME,MOUNTPOINT |  while read name mp; do ([ -e "/dev/$name" ] && echo "/dev/$name" $(basename "$(readlink -f "/sys/class/block/$name/..")")  "$mp") || ([ -e "/dev/mapper/$name" ] && echo "/dev/mapper/$name" $(basename "$(readlink -f "/sys/class/block/$name/..")")  "$mp"); done
	// above script looks good which can give us full path, parent device name, mountpoint.
	// But it assumes everything either in /dev/ or /dev/mapper/, it again fails on centos7 docker.
	// centos7 docker has sda and its partition but its not available in /dev/ or /dev/mapper.

	return nil
}

// wrapSudo returns the cmd wrapped with sudo privileges, error if user does not have sudo.
func (sys *System) wrapSudo(cmd string) (string, error) {
	sudo := sys.sudo
	if !sudo.canSudo() {
		return "", fmt.Errorf("no sudo privileges")
	}

	return sudo.wrap(cmd), nil
}

// wrapCmd wraps the command. returns error if user does not have sudo privileges.
func (sys *System) wrapCmd(cmd string, sudo bool) (string, error) {
	if sudo {
		cmd, err := sys.wrapSudo(cmd)
		if err != nil {
			return "", fmt.Errorf("wrap sudo: %v", err)
		}
		return cmd, nil
	}

	return cmd, nil
}

// RunCmd runs the cmd on the system.
func (sys *System) RunCmd(cmd Command) {
	stdout, stderr, err := sys.run(cmd.Cmd(), cmd.Sudo())
	cmd.Parse(stdout, stderr, err)
}

// RunBigCmd runs the cmd and returns the job representing the command.
func (sys *System) RunBigCmd(cmd BigCommand) (*Job, error) {
	lg := sys.log.WithValues("cmd", cmd.Cmd(), "sudo", cmd.Sudo())
	lg.V(1).Info("running big system command")

	session, err := sys.newSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new ssh session: %v", err)
	}

	outio, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe stdout: %v", err)
	}

	errio, err := session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe stderr: %v", err)
	}

	command := cmd.Cmd()
	if command, err = sys.wrapCmd(command, cmd.Sudo()); err != nil {
		return nil, fmt.Errorf("failed to wrap big system command: %v", err)
	}

	canch := make(chan struct{})
	cancel := func() {
		lg.V(1).Info("cancelled big system command")
		// FIXME session.Close() does not stop command running on the remote machine.
		// see http://grokbase.com/t/gg/golang-nuts/1514k8qk67/go-nuts-how-to-stop-an-infinite-ssh-command-session
		session.Close()
		close(canch)
	}

	job := newJob(cmd, cancel)

	errch := make(chan error, 1)
	go func() {
		err := session.Run(command)
		lg.V(1).Info("finished big system command", "err", err)

		session.Close()
		errch <- err
		job.completed = true
	}()

	go func() {
		errio = sys.sudo.removePromptInReader(errio)
		cmd.Parse(outio, errio, errch, canch)
	}()

	return job, nil
}

// Run runs command line cmd on the system.
//
// Run command should be used to run commands that execute quickly and do not produce
// a lot of output. If not then use RunBigCmd to run those commands.
func (sys *System) Run(cmd string) (stdout, stderr string, err error) {
	return sys.run(cmd, false)
}

// RunWithSudo runs command line cmd on the system with sudo privileges.
//
// RunWithSudo command should be used to run commands that execute quickly and do not produce
// a lot of output. If not then use RunBigCmd to run those commands.
func (sys *System) RunWithSudo(cmd string) (stdout, stderr string, err error) {
	return sys.run(cmd, true)
}

// run runs cmd on the system with/without sudo.
//
// run command should be used to run commands that execute quickly and do not produce
// a lot of output. If not then use RunBigCmd to run those commands.
func (sys *System) run(cmd string, sudo bool) (stdout, stderr string, err error) {
	lg := sys.log.WithValues("cmd", cmd, "sudo", sudo)
	lg.V(1).Info("running system command")

	session, e := sys.newSession()
	if e != nil {
		err = fmt.Errorf("failed to create ssh session: %v", e)
		return
	}
	defer session.Close()

	outio, e := session.StdoutPipe()
	if e != nil {
		err = fmt.Errorf("pipe stdout: %v", e)
		return
	}

	errio, e := session.StderrPipe()
	if e != nil {
		err = fmt.Errorf("pipe stderr: %v", e)
		return
	}

	if cmd, e = sys.wrapCmd(cmd, sudo); e != nil {
		err = fmt.Errorf("failed to wrap system command: %v", e)
		return
	}

	e = session.Run(cmd)
	lg.V(1).Info("finished system command", "err", e)

	tostr := func(in io.Reader) string {
		var b bytes.Buffer
		b.ReadFrom(in)
		return b.String()
	}

	stdout = tostr(outio)
	stderr = sys.sudo.removePrompt(tostr(errio))

	return
}

func parseErr(stderr string, err error) error {
	if err != nil {
		s := err.Error()
		if len(stderr) > 0 {
			s += ": " + stderr
		}

		return errors.New(s)
	}

	return nil
}

// IsRunning returns true if the service is running.
func (sys *System) IsRunning(service string) (bool, error) {
	lg := sys.log.WithValues("service", service)
	lg.V(1).Info("checking system service status")

	cmd, stopped, absent := "", "", ""
	ignoreErr := false

	switch sys.InitManager() {
	case SystemdServiceSystem:
		cmd = fmt.Sprintf("systemctl is-active %s", service)
		stopped = "inactive"
		absent = "unknown"
		ignoreErr = true // systemctl is-active command return non-zero exit code on inactive
	case InitServiceSystem:
		cmd = fmt.Sprintf("service %s status", service)
		stopped = "stop"
		absent = "stopped"
		ignoreErr = false
	default:
		return false, ErrUnknownServiceSystem
	}

	stdout, stderr, err := sys.RunWithSudo(cmd)
	if strings.Contains(stdout, absent) || strings.Contains(stdout, stopped) {
		return false, nil
	}

	if err != nil {
		if ignoreErr {
			return false, nil
		}
		return false, parseErr(stderr, err)
	}

	return true, nil
}

// StartService starts service on system
func (sys *System) StartService(service string) error {
	lg := sys.log.WithValues("service", service)
	lg.V(1).Info("starting system service")

	var cmd string
	switch sys.InitManager() {
	case InitServiceSystem:
		cmd = fmt.Sprintf("service %s start", service)
	case SystemdServiceSystem:
		cmd = fmt.Sprintf("systemctl start %s", service)
	default:
		return ErrUnknownServiceSystem
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished starting system service", "err", err)
	return parseErr(stderr, err)
}

// StopService stops service on the system
func (sys *System) StopService(service string) error {
	lg := sys.log.WithValues("service", service)
	lg.V(1).Info("stopping system service")

	var cmd string
	switch sys.serviceSystem {
	case InitServiceSystem:
		cmd = fmt.Sprintf("service %s stop", service)
	case SystemdServiceSystem:
		cmd = fmt.Sprintf("systemctl stop %s", service)
	default:
		return ErrUnknownServiceSystem
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished stopping system service", "err", err)
	return parseErr(stderr, err)
}

// InstallPkg installs pkg on the system.
func (sys *System) InstallPkg(pkg string) error {
	lg := sys.log.WithValues("package", pkg)
	lg.V(1).Info("installing package")

	var cmd string
	switch sys.updateMethod {
	case AptUpdateMethod:
		cmd = fmt.Sprintf("apt-get install -y %s", pkg)
	case YumUpdateMethod:
		cmd = fmt.Sprintf("yum install -y %s", pkg)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished installing package", "err", err)
	return parseErr(stderr, err)
}

// InstallBinary installs the binary at path on the system.
func (sys *System) InstallBinary(path string) error {
	lg := sys.log.WithValues("binary", path)
	lg.V(1).Info("installing binary")

	var cmd string
	switch sys.pkgMgr {
	case DpkgPkgMgr:
		cmd = fmt.Sprintf("dpkg -i %s", path)
	case RpmPkgMgr:
		cmd = fmt.Sprintf("yum install -y %s", path)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished installing binary", "err", err)
	return parseErr(stderr, err)
}

// uninstallBinary uninstalls the binary on the system.
// In case of yum the config files are removed as well.
//
// Only use this method if the package was installed from a binary.
func (sys *System) uninstallBinary(binaryName string) error {
	lg := sys.log.WithValues("binary", binaryName)
	lg.V(1).Info("uninstalling binary")

	var cmd string
	switch sys.pkgMgr {
	case DpkgPkgMgr:
		cmd = fmt.Sprintf("dpkg --remove %s", binaryName)
	case RpmPkgMgr:
		cmd = fmt.Sprintf("yum remove -y %s", binaryName)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished uninstalling binary", "err", err)
	return parseErr(stderr, err)
}

// PurgeBinary purges the binary on the system.
//
// Only use this method if the package was installed from a binary.
func (sys *System) PurgeBinary(binaryName string) error {
	lg := sys.log.WithValues("binary", binaryName)
	lg.V(1).Info("purging binary")

	var cmd string
	switch sys.pkgMgr {
	case DpkgPkgMgr:
		cmd = fmt.Sprintf("dpkg --purge %s", binaryName)
	case RpmPkgMgr:
		cmd = fmt.Sprintf("yum remove -y %s", binaryName)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished purging binary", "err", err)
	return parseErr(stderr, err)
}

// UpgradeBinary upgrades pkgName with the binary located at binaryPath.
//
// Only use this method if the package was installed from a binary.
func (sys *System) UpgradeBinary(pkgName, binaryPath string) error {
	lg := sys.log.WithValues("binary", binaryPath)
	lg.V(1).Info("upgrading binary")

	var cmd string
	switch sys.pkgMgr {
	case DpkgPkgMgr:
		err := sys.uninstallBinary(pkgName)
		if err == nil {
			err = sys.InstallBinary(binaryPath)
		}
		lg.V(1).Info("finished upgrading binary", "err", err)
		return err

	case RpmPkgMgr:
		cmd = fmt.Sprintf("rpm -U %s", binaryPath)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished purging binary", "err", err)
	return parseErr(stderr, err)
}

// IsPackageInstalled returns true if the pkg is installed
func (sys *System) IsPackageInstalled(pkg string) (bool, error) {
	lg := sys.log.WithValues("package", pkg)
	lg.V(1).Info("is package installed")

	var cmd string
	switch sys.updateMethod {
	case AptUpdateMethod:
		cmd = fmt.Sprintf("dpkg --list %s", pkg)
	case YumUpdateMethod:
		cmd = fmt.Sprintf("yum list %s", pkg)
	default:
		return false, ErrUknownPackageManager
	}

	_, _, err := sys.RunWithSudo(cmd)

	if err != nil {
		if ExitStatus(err) == 1 { // package not present
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// UninstallPkg uninstalls the package on the system.
func (sys *System) UninstallPkg(pkg string) error {
	lg := sys.log.WithValues("package", pkg)
	lg.V(1).Info("uninstalling package")

	var cmd string
	switch sys.updateMethod {
	case AptUpdateMethod:
		cmd = fmt.Sprintf("apt-get remove %s", pkg)
	case YumUpdateMethod:
		cmd = fmt.Sprintf("yum remove %s", pkg)
	default:
		return ErrUknownPackageManager
	}

	_, stderr, err := sys.RunWithSudo(cmd)
	lg.V(1).Info("finished uninstalling package", "err", err)
	return parseErr(stderr, err)
}

// CreateFile creates a file at dir with the contents of in.
func (sys *System) CreateFile(size int64, name string, in io.Reader, dir string) error {
	session, err := sys.newSession()
	if err != nil {
		return err
	}

	lg := sys.log.WithValues("name", name, "size", size, "destination", dir)
	lg.V(1).Info("creating file")

	err = scp.Copy(size, os.ModePerm, name, in, dir, session)

	lg.V(1).Info("finished creating file", "err", err)
	return err
}

// CreateFileFromStr creates a file with name in dir with contents.
func (sys *System) CreateFileFromStr(contents, name, dir string) error {
	size := int64(utf8.RuneCountInString(contents))
	in := strings.NewReader(contents)
	return sys.CreateFile(size, name, in, dir)
}

// Mkdir creates a directory dir with the permissions perms.
// perms syntax is the same as mkdir command, pass empty to create directory
// with default permissions. pass sudo as true to run the command with sudo
// privileges.
func (sys *System) Mkdir(dir string, perms string, sudo bool) error {
	lg := sys.log.WithValues("directory", dir, "permissions", perms)
	lg.V(1).Info("creating directory")

	cmd := ""
	if len(perms) > 0 {
		cmd += fmt.Sprintf("mkdir -m %s %s", perms, dir)
	} else {
		cmd = fmt.Sprintf("mkdir %s", dir)
	}

	_, stderr, err := sys.run(cmd, sudo)

	lg.V(1).Info("finished creating directory", "err", err)
	return parseErr(stderr, err)
}

// GetFile gets the file specified at the the path.
// Do not use this command for huge files.
func (sys *System) GetFile(path string) (string, error) {
	cmd := fmt.Sprintf("cat %s", path)
	conf, _, err := sys.Run(cmd)
	return conf, err
}

// HomeDir returns the path of the logged in user's home directory.
func (sys *System) HomeDir() string {
	return sys.homeDir
}

// ExitStatus returns the exit status from the error value returned by system.RunCmd
// returns -1 if exit status cannot be parsed from err.
func ExitStatus(err error) int {
	// HACK: the ssh api provides no clean way to access the exit status of the
	// command.

	const unknown = -1

	if err == nil {
		return 0
	}

	s := fmt.Sprintf("%s", err)
	s = strings.ToLower(s)

	exit := "process exited with status "
	i := strings.Index(s, exit)

	if i == -1 {
		return unknown
	}

	i += len(exit)
	s = s[i:]
	words := strings.Fields(s)

	if len(words) == 0 {
		return unknown
	}

	status, err := strconv.ParseInt(words[0], 10, 64)
	if err != nil {
		return unknown
	}

	return int(status)
}
