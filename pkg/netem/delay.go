package netem

import (
	"context"
	"encoding/json"
	"fmt"
	"go_tc/pkg/container"
	"io/ioutil"
	"os"

	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func Reset(cli *cli.Context, name string) {
	cmd, _ := shellwords.Parse("qdisc del dev eth0 root")
	// cmd, _ := shellwords.Parse("qdisc add dev eth0 root handle 1: htb default 1")
	Netemcontainer(name, cli.GlobalString("tc-image"), cmd)
	fmt.Println("reset completed!")
}

func Initialize(cli *cli.Context, name string) {
	Reset(cli, name)
	cmd, _ := shellwords.Parse("qdisc add dev eth0 root handle 1: htb default 1")
	Netemcontainer(name, cli.GlobalString("tc-image"), cmd)
	// fmt.Println(string(out))
	cmd, _ = shellwords.Parse("class add dev eth0 parent 1: classid 1:1 htb rate 1000Gbit")
	Netemcontainer(name, cli.GlobalString("tc-image"), cmd)
	fmt.Println("init completed!")
}

func Set(cli *cli.Context) {
	// Initialize(cli)
	// if cli.String("file") == "" {
	// 	Add("", cli.Args(), cli.String("time"), cli.GlobalString("name"), cli.GlobalString("tc-image"), 1)
	// } else {
	// 	AddFromJson(cli)
	// }
	// fmt.Println("set completed!")
	if cli.String("file") == "" {
		Initialize(cli, cli.String("name"))
		Add("", cli.Args(), cli.String("time"), cli.String("name"), cli.GlobalString("tc-image"), 1)
	} else {
		SetFromJson(cli)
	}
}

func Add(prio string, ip []string, time string, name string, tcimage string, id int) {
	var (
		cmd []string
		out []byte
		err error
	)
	if prio == "" {
		prio = "100"
	}
	cmd, _ = shellwords.Parse(fmt.Sprint("class add dev eth0 parent 1:1 classid 1:", id, "0 htb rate 10Gbit"))
	Netemcontainer(name, tcimage, cmd)

	// shell scriptのcreate classを実装
	cmd, _ = shellwords.Parse(fmt.Sprint("qdisc add dev eth0 parent 1:", id, "0 handle 1", id, ": netem delay ", time))
	Netemcontainer(name, tcimage, cmd)

	// shell scriptのadd filterを実装
	for i := 1; i <= len(ip); i++ {
		cmd, _ = shellwords.Parse(fmt.Sprint("filter add dev eth0 protocol ip parent 1: prio ", prio, " u32 match ip dst ", ip[i-1], " flowid 1:", id, "0"))
		Netemcontainer(name, tcimage, cmd)
		if err != nil {
			fmt.Println(err)
			fmt.Println(string(out))
			return
		}
	}
	fmt.Println("add completed!")
}

func SetFromJson(cli *cli.Context) {
	type DelayInfo struct {
		Time string   `json:"time"`
		To   []string `json:"to"`
		Prio string   `json:"priority"`
	}

	type Latency struct {
		From  string      `json:"from"`
		Delay []DelayInfo `json:"delay"`
	}

	type Config struct {
		Latency []Latency `json:"latency"`
	}
	var cg Config
	var ip []string

	raw, err := ioutil.ReadFile(cli.String("file"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = json.Unmarshal(raw, &cg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	containercli, err := container.NewClient()
	if err != nil {
		panic(err)
	}
	//latency配列分を回す
	for _, latency := range cg.Latency {
		ctx := context.Background()
		Initialize(cli, latency.From)
		//delay内に書かれている設定分を回す
		for i, delay := range latency.Delay {
			//toの数だけ回す
			for _, to := range delay.To {
				containers := containercli.Listcontainer(ctx, to)
				// toをlabelにしてコンテナ一覧を取得。その後、それらのipアドレスを格納
				for _, container := range containers {
					ip = append(ip, container.NetworkSettings.Networks[container.HostConfig.NetworkMode].IPAddress)
				}
			}
			//class名の被りを防ぐためにiを渡す
			Add(delay.Prio, ip, delay.Time, latency.From, cli.GlobalString("tc-image"), i)
		}
	}

	// //自分のipアドレスを確認。その後、json内のipと照合
	// if cli.String("source") == "" {
	// 	out, err := pipeline.Output(
	// 		[]string{"ip", "a"},
	// 		[]string{"grep", "-x", ".*eth0"},
	// 	)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	// 	ip = strings.TrimLeft(string(out), "inet ")
	// 	ip = ip[0:strings.Index(ip, "/")]
	// } else {
	// 	ip = cli.String("source")
	// }
	// fmt.Println(ip)

	// for _, di := range cg.Latency {
	// 	if di.From == ip {
	// 		Add(di.Prio, di.To, di.Time, cli.GlobalString("name"), cli.GlobalString("tc-image"))
	// 	} else {
	// 		continue
	// 	}
	// }
}

func Netemcontainer(name string, tcimage string, cmd []string) {
	client, err := container.NewClient()
	if err != nil {
		panic(err)
	}
	client.Netemcontainer(name, tcimage, cmd)
}
