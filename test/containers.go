package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/aerospike/aerospike-client-go/v7"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var ClusterName = "mgmt-lib-test"
var PortStart = 10000
var IP = "127.0.0.1"
var WordDirAbs = "test/work"
var Image = "aerospike/aerospike-server-enterprise:7.0.0.2"
var ContainerPrefix = "aerospike_mgmt_lib_test_"
var User = "admin"
var Password = "admin"

var configTemplate = fmt.Sprintf(`
security {
}

service {
	cluster-name %s
	# Uncomment if multi-node EE tests are needed
    # feature-key-file env-b64:FEATURES
	run-as-daemon false
	proto-fd-max 1024
	transaction-retry-ms 10000
	transaction-max-ms 10000
}

logging {
	console {
		context any info
        context security info
	}
}

network {
	service {
		port {{.ServicePort}}
		address any
		access-address {{.AccessAddress}}
	}

	heartbeat {
		mode mesh
		address any
		port {{.HeartbeatPort}}
		interval 100
		timeout 10
		connect-timeout-ms 100
		{{.PeerConnection}}
	}

	fabric {
		port {{.FabricPort}}
		address any
	}

	info {
		port {{.InfoPort}}
		address any
	}
}

namespace test {
	default-ttl 30d # use 0 to never expire/evict.
	nsup-period 120
	replication-factor 1
	storage-engine memory {
		data-size 1G
	}
}
`, ClusterName)

type AerospikeContainer struct {
	ip         string
	configPath string
	portBase   int
}

type Containers struct {
	namesToContainers map[string]*AerospikeContainer
	dockerCLI         *client.Client
	workDir           string
}

var containers = &Containers{make(map[string]*AerospikeContainer), nil, ""}

func Start(size int) error {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	ctx := context.Background()
	containers.dockerCLI = cli
	containers.workDir, _ = filepath.Abs(WordDirAbs)
	reader, err := cli.ImagePull(ctx, Image, types.ImagePullOptions{})

	if err != nil {
		log.Printf("Unable to pull aerospike image: %s", err)
		return err
	}

	defer reader.Close()
	_, err = io.Copy(os.Stdout, reader)

	if err != nil {
		log.Printf("Unable to pull aerospike image: %s", err)
		return err
	}

	for i := 0; i < size; i++ {
		peerConnection := ""

		if i > 0 {
			peerConnection = fmt.Sprintf("mesh-seed-address-port %s %d", IP, PortStart+2)
		}

		name := GetAerospikeContainerName(i)
		RmAerospikeContainer(name) //nolint:errcheck // Removing containers just in case they were left over from a previous run

		asContainer, err := RunAerospikeContainer(i, name, IP, PortStart+(i*4), peerConnection)

		if err != nil {
			log.Printf("Unable to start testing containers")
			return err
		}

		containers.namesToContainers[name] = asContainer
	}

	return nil
}

func Stop() error {
	log.Println("Stopping test containers")

	for name := range containers.namesToContainers {
		err := RmAerospikeContainer(name)

		if err != nil {
			log.Printf("Unable to remove container %s: %s", name, err)
			return err
		}
	}

	abs, _ := filepath.Abs(containers.workDir)
	err := os.RemoveAll(abs)

	if err != nil {
		log.Printf("Unable to remove work directory: %s", err)
		return err
	}

	return nil
}

func GetAerospikeContainerName(index int) string {
	return ContainerPrefix + fmt.Sprintf("%d", index)
}

func createConfigFile(portBase int, accessAddress, peerConnection string) (string, error) {
	templateInput := struct {
		FeaturePath    string
		AccessAddress  string
		PeerConnection string
		Namespace      string
		ServicePort    int
		HeartbeatPort  int
		FabricPort     int
		InfoPort       int
	}{
		// Uncomment if multi-node EE tests are needed
		// FeaturePath:    "/opt/aerospike/features.conf",
		AccessAddress:  accessAddress,
		PeerConnection: peerConnection,
		Namespace:      "test",
		ServicePort:    portBase,
		HeartbeatPort:  portBase + 1,
		FabricPort:     portBase + 2,
		InfoPort:       portBase + 3,
	}

	tmpl, _ := template.New("config").Parse(configTemplate)

	err := os.MkdirAll(containers.workDir, 0o755)

	if err != nil {
		log.Printf("Unable to create work directory: %s", err)
		return "", err
	}

	file, err := os.CreateTemp(containers.workDir, "aerospike_*.conf")

	if err != nil {
		log.Printf("Unable to create config file: %s", err)
		return "", err
	}

	defer file.Close()

	err = tmpl.Execute(file, templateInput)

	if err != nil {
		log.Printf("Unable to create config file using template: %s", err)
		return "", err
	}

	return file.Name(), nil
}

func waitForASDToStart(name string) error {
	startTime := time.Now()
	timeout := 10 * time.Second
	policy := aerospike.NewClientPolicy()
	policy.User = User
	policy.Password = Password

	for {
		asClient, err := aerospike.NewClientWithPolicy(
			policy, IP, PortStart)

		if err == nil {
			if asClient.IsConnected() {
				break
			}

			asClient.Close()
		}

		if time.Since(startTime) > timeout {
			log.Printf("Timed out waiting for asd %s to start %s", name, err)
			return err
		}

		log.Printf("Waiting for asd %s to start %s", name, err)
		time.Sleep(1 * time.Second)
	}

	log.Printf("asd %s started", name)

	return nil
}

func RunAerospikeContainer(
	index int,
	name,
	ip string,
	portBase int,
	peerConnection string) (*AerospikeContainer, error) {
	ctx := context.Background()
	cli := containers.dockerCLI

	log.Printf("Starting container %s", name)

	confFile, err := createConfigFile(portBase, ip, peerConnection)

	if err != nil {
		log.Printf("Unable to create config file for container %s: %s", name, err)
		return nil, err
	}

	containerWorkDir := "/opt/" + containers.workDir
	containerConfFile := containerWorkDir + "/" + filepath.Base(confFile)

	cmd := []string{
		"/usr/bin/asd",
		"--foreground",
		"--config-file",
		containerConfFile,
		"--instance",
		fmt.Sprintf("%d", index),
	}

	// Uncomment if multi-node EE tests are needed
	// featKey := os.Getenv("FEATKEY")
	// if featKey == "" {
	// 	log.Printf("Env var FEATKEY must be set with a base64 encoded feature key file.")
	// 	return "", fmt.Errorf("env var FEATKEY must be set with a base64 encoded feature key file")
	// }

	ports := make([]string, 4)

	for i := 0; i < 4; i++ {
		ports[i] = fmt.Sprintf("%d/tcp", portBase+i)
	}

	config := container.Config{
		Image:    Image,
		Hostname: ip,
		Cmd:      cmd,
		Tty:      true,
		// Uncomment if multi-node EE tests are needed
		// Env:      []string{fmt.Sprintf("FEATURES=%s", featKey)},
		ExposedPorts: nat.PortSet{
			nat.Port(ports[0]): {},
			nat.Port(ports[1]): {},
			nat.Port(ports[2]): {},
			nat.Port(ports[3]): {},
		},
	}

	if err != nil {
		log.Printf("Unable to get absolute path for work directory: %s", err)
		return nil, err
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(ports[0]): []nat.PortBinding{{
				HostPort: ports[0],
			}}, nat.Port(ports[1]): []nat.PortBinding{{
				HostPort: ports[1],
			}}, nat.Port(ports[2]): []nat.PortBinding{{

				HostPort: ports[2],
			}}, nat.Port(ports[3]): []nat.PortBinding{{

				HostPort: ports[3],
			}},
		},
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: containers.workDir, Target: containerWorkDir},
		},
	}

	if _, err = cli.ContainerCreate(ctx, &config, hostConfig, nil, nil, name); err != nil {
		log.Printf("Unable to create container %s: %s", name, err)
		return nil, err
	}

	err = cli.ContainerStart(ctx, name, types.ContainerStartOptions{})

	if err != nil {
		log.Printf("Unable to start container %s: %s", name, err)
		return nil, err
	}

	inspect, _ := cli.ContainerInspect(ctx, name)

	log.Printf("Started container %s with IP %s", name, inspect.NetworkSettings.IPAddress)
	log.Printf("Waiting for asd %s to start", name)

	if waitForASDToStart(name) != nil {
		return nil, err
	}

	return &AerospikeContainer{inspect.NetworkSettings.IPAddress, confFile, portBase}, nil
}

func RestartAerospikeContainer(name, confFileContents string) error {
	cli := containers.dockerCLI
	ctx := context.Background()
	ip := containers.namesToContainers[name].ip
	log.Printf("Restarting container %s with IP %s", name, ip)

	if confFileContents != "" {
		confPath := containers.namesToContainers[name].configPath
		file, err := os.OpenFile(confPath, os.O_TRUNC|os.O_WRONLY, 0o644)

		if err != nil {
			log.Printf("Unable to open config file %s: %s", confPath, err)
			return err
		}

		_, err = file.WriteString(confFileContents)

		if err != nil {
			log.Printf("Unable to write to config file %s: %s", confPath, err)
			return err
		}
	}

	err := cli.ContainerRestart(ctx, name, container.StopOptions{})

	if err != nil {
		log.Printf("Unable to restart container %s: %s", name, err)
		return err
	}

	log.Printf("Restarted container %s with IP %s", name, ip)
	log.Printf("Waiting for asd %s to start", name)

	if waitForASDToStart(name) != nil {
		return err
	}

	return nil
}

func RmAerospikeContainer(name string) error {
	cli := containers.dockerCLI
	ctx := context.Background()

	return cli.ContainerRemove(ctx, name, types.ContainerRemoveOptions{Force: true})
}
