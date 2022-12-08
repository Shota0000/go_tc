package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func Listcontainer() {
	ctx := context.Background()
	// client.FromEnv：環境変数からdockerサーバーへのURL，apiバージョン，証明書をとってくる関数．
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)
	filterArgs := filters.NewArgs()

	// ここを任意の名前にする。ラベルにも対応したいね
	filterArgs.Add("name", "docker_goapp")
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filterArgs})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Println(container.ID[:12], container.Image)
	}
}
