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
	// filterArgs.Add("name", name)
	filterArgs.Add("label", fmt.Sprint("io.kubernetes.pod.name=", name))
	containers, err := cli.client.ContainerList(ctx, types.ContainerListOptions{Filters: filterArgs})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		//確認用
		fmt.Println(container.ID[:12], container.Image, container.NetworkSettings.Networks[container.HostConfig.NetworkMode].IPAddress)
	}
	return containers
}

func (cli dockerClient) tcCommand(ctx context.Context, tcimage string, c types.Container, cmd []string) {
	if tcimage == "" {
		cli.execOnContainer(ctx, c, cmd)
	} else {
		cli.tcContainerCommand(ctx, c, tcimage, cmd)
	}
}

func (cli dockerClient) execOnContainer(ctx context.Context, c types.Container, args []string) {
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
	// if command found execute it
	log.WithField("command", "tc").Info("command found: continue execution")

	// prepare exec config
	config := types.ExecConfig{
		Privileged: true,
		Cmd:        append([]string{"tc"}, args...),
	}
	// execute the command
	exec, err = cli.client.ContainerExecCreate(ctx, c.ID, config)
	if err != nil {
		fmt.Println("failed to create exec configuration for a command")
		panic(err)
	}
	log.Infof("starting exec tc %s (%s)", args, exec.ID)
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
	if exitInspect.ExitCode != 0 {
		panic(errors.Errorf("command tc failed in %s container; run it in manually to Info", c.ID))
	}
}

func (cli dockerClient) tcContainerCommand(ctx context.Context, c types.Container, tcimage string, args []string) {
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
		Entrypoint: []string{"tc"},
		Cmd:        args,
		Tty:        false,
	}
	hconfig := container.HostConfig{
		// auto remove container on tc command exit
		// AutoRemove: true,
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

	//StdCopy は `src` をデマルチプレックスします。これは、StdWriter のインスタンスを使用してマルチプレックスされた 2 つのストリームを含んでいると仮定しています。src` から読み込むと、StdCopy は `dstout` と `dsterr` に書き込みを行います。
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

func (cli dockerClient) Netemcontainer(name string, tcimage string, cmd []string) {
	ctx := context.Background()
	containers := cli.Listcontainer(ctx, name)
	for _, c := range containers {
		cli.tcCommand(ctx, tcimage, c, cmd)
	}
}
