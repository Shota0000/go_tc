// This file refers to Pumba project
// https://github.com/alexei-led/pumba

package container

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type dockerClient struct {
	client *client.Client
}

func NewClient() (dockerClient, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return dockerClient{client: nil}, errors.Wrap(err, "failed to create docker client")
	}
	return dockerClient{client: apiClient}, nil
}

func (cli dockerClient) Listcontainer(ctx context.Context, name string) []types.Container {
	cli.client.NegotiateAPIVersion(ctx)
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprint("io.kubernetes.pod.name=", name))
	containers, err := cli.client.ContainerList(ctx, types.ContainerListOptions{Filters: filterArgs})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		//確認用
		fmt.Println(container.ID[:12], container.Image)
	}
	return containers
}

func (cli dockerClient) tcCommand(ctx context.Context, tcimage string, c types.Container, cmds [][]string) {
	if tcimage == "" {
		cli.execOnContainer(ctx, c, cmds)
	} else {
		cli.tcContainerCommand(ctx, c, tcimage, cmds)
	}
}

func (cli dockerClient) execOnContainer(ctx context.Context, c types.Container, args [][]string) {
	checkExists := types.ExecConfig{
		Cmd: []string{"which", "tc"},
	}
	exec, err := cli.client.ContainerExecCreate(ctx, c.ID, checkExists)
	if err != nil {
		fmt.Println("failed to create exec configuration to check if command exists")
		panic(err)
	}
	log.WithField("command", "tc").Infof("checking if command exists")
	//コンテナexecスタート
	err = cli.client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		fmt.Println("failed to check if command exists in a container")
		panic(err)
	}
	//コンテナにtcあるか確認
	checkInspect, err := cli.client.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		fmt.Println("failed to inspect check execution")
		panic(err)
	}
	if checkInspect.ExitCode != 0 {
		panic(errors.Errorf("command tc not found inside the %s container", "tc", c.ID))
	}
	// あったらコンテナ内でtcコマンド実行
	log.WithField("command", "tc").Info("command found: continue execution")
	// コマンド数だけ回す
	for _, cmd := range args {
		// prepare exec config
		config := types.ExecConfig{
			Privileged: true,
			Cmd:        append([]string{"tc"}, cmd...),
		}
		// execute the command
		exec, err = cli.client.ContainerExecCreate(ctx, c.ID, config)
		if err != nil {
			fmt.Println("failed to create exec configuration for a command")
			panic(err)
		}
		log.Infof("starting exec tc %s (%s)", cmd, exec.ID)
		err = cli.client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
		if err != nil {
			fmt.Println("failed to start command execution")
			panic(err)
		}
		exitInspect, err := cli.client.ContainerExecInspect(ctx, exec.ID)
		if err != nil {
			fmt.Println("failed to inspect command execution")
			panic(err)
		}
		//ログ出し
		out, err := cli.client.ContainerLogs(ctx, c.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}
		stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		if exitInspect.ExitCode != 0 {
			log.Errorf("command tc failed in %s container; run it in manually to Info", c.ID)
		}
	}
}

func (cli dockerClient) tcContainerCommand(ctx context.Context, c types.Container, tcimage string, args [][]string) {
	log.WithFields(log.Fields{
		"container": c.ID,
		"tc-image":  tcimage,
		"args":      args,
	}).Info("executing tc command in a separate container joining target container network namespace")

	reader, err := cli.client.ImagePull(ctx, tcimage, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)
	// container config
	config := container.Config{
		Image:      tcimage,
		Entrypoint: []string{"/bin/sh", "-c", "while :; do sleep 10; done"},
		Tty:        false,
	}
	hconfig := container.HostConfig{
		// auto remove container on tc command exit
		AutoRemove: false,
		// NET_ADMIN is required for "tc netem"
		CapAdd: []string{"NET_ADMIN"},
		// use target container network stack
		NetworkMode: container.NetworkMode("container:" + c.ID),
		// others
		PortBindings: nat.PortMap{},
		DNS:          []string{},
		DNSOptions:   []string{},
		DNSSearch:    []string{},
	}
	log.WithField("network", hconfig.NetworkMode).Info("network mode")
	log.WithField("image", config.Image).Info("creating tc-container")
	resp, err := cli.client.ContainerCreate(ctx, &config, &hconfig, nil, nil, "")
	if err != nil {
		panic(err)
	}
	log.WithField("id", resp.ID).Info("tc container created, starting it")
	if err := cli.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	out, err := cli.client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	// コマンド数だけ回す
	for _, cmd := range args {
		// prepare exec config
		config := types.ExecConfig{
			Privileged: true,
			Cmd:        append([]string{"tc"}, cmd...),
		}
		// execute the command
		exec, err := cli.client.ContainerExecCreate(ctx, resp.ID, config)
		if err != nil {
			fmt.Println("failed to create exec configuration for a command")
			panic(err)
		}
		log.Infof("starting exec tc %s (%s)", cmd, exec.ID)
		err = cli.client.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
		if err != nil {
			fmt.Println("failed to start command execution")
			panic(err)
		}
		exitInspect, err := cli.client.ContainerExecInspect(ctx, exec.ID)
		if err != nil {
			fmt.Println("failed to inspect command execution")
			panic(err)
		}
		//ログ出し
		out, err := cli.client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}
		stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		if exitInspect.ExitCode != 0 {
			log.Errorf("command tc failed in %s container; run it in manually to Info", resp.ID)
		}
	}
	err = cli.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		log.Errorf("remove failed %s container", resp.ID)
		fmt.Println(err)
	}
}

func (cli dockerClient) CreateIpContaier(ctx context.Context, c types.Container, tcimage string) {
	log.WithFields(log.Fields{
		"container": c.ID,
		"tc-image":  tcimage,
	}).Info("executing ip a command in a separate container joining target container network namespace")
	reader, err := cli.client.ImagePull(ctx, tcimage, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)
	// container config
	config := container.Config{
		Image:      tcimage,
		Entrypoint: []string{"ip"},
		Cmd:        []string{"a"},
		Tty:        false,
	}
	hconfig := container.HostConfig{
		NetworkMode: container.NetworkMode("container:" + c.ID),
		// others
		PortBindings: nat.PortMap{},
		DNS:          []string{},
		DNSOptions:   []string{},
		DNSSearch:    []string{},
	}
	log.WithField("network", hconfig.NetworkMode).Info("network mode")
	log.WithField("image", config.Image).Info("creating tc-container")
	resp, err := cli.client.ContainerCreate(ctx, &config, &hconfig, nil, nil, "")
	if err != nil {
		panic(err)
	}
	log.WithField("id", resp.ID).Info("tc container created, starting it")
	if err := cli.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	out, err := cli.client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

func (cli dockerClient) Netemcontainer(name string, tcimage string, cmds [][]string) {
	ctx := context.Background()
	containers := cli.Listcontainer(ctx, name)
	for _, c := range containers {
		cli.tcCommand(ctx, tcimage, c, cmds)
	}
}
