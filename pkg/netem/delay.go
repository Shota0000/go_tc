package netem

import (
	"encoding/json"
	"fmt"
	"go_tc/pkg/container"
	"io/ioutil"
	"os"

	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func Reset(cli *cli.Context) [][]string {
	var cmds [][]string
	cmd, _ := shellwords.Parse("qdisc del dev eth0 root")
	cmds = append(cmds, cmd)
	fmt.Println("Reset command is ready")
	return cmds
}

func AddQdisc() [][]string {
	var cmds [][]string
	cmd, _ := shellwords.Parse("qdisc add dev eth0 root handle 1: htb default 1")
	cmds = append(cmds, cmd)
	cmd, _ = shellwords.Parse("class add dev eth0 parent 1: classid 1:1 htb rate 1000Gbit")
	cmds = append(cmds, cmd)
	return cmds
}

func Initialize(cli *cli.Context, name string) [][]string {
	// resetコマンドを配列に追加
	// netem qdiscを追加するための準備を行うコマンド
	var cmds [][]string
	cmds = append(cmds, Reset(cli)...)
	cmds = append(cmds, AddQdisc()...)
	fmt.Println("Initialization command is ready")
	return cmds
}

func Set(cli *cli.Context) {
	// 実行するコマンド
	var cmds [][]string
	if cli.String("file") == "" {
		// 初期化に必要なコマンドを配列に追加
		initialCommands := Initialize(cli, cli.String("name"))
		cmds = append(cmds, initialCommands...)
		// ipアドレス毎に場合分
		addCommands := Add("", cli.Args(), cli.String("time"), 1)
		cmds = append(cmds, addCommands...)
		Netemcontainer(cli.String("name"), cli.GlobalString("tc-image"), cmds)
	} else {
		SetFromJson(cli)
	}
}

func Add(prio string, ip []string, time string, id int) [][]string {
	if prio == "" {
		prio = "100"
	}
	var cmds [][]string
	classCommand := fmt.Sprint("class add dev eth0 parent 1:1 classid 1:", id, "0 htb rate 10Gbit")
	cmd, _ := shellwords.Parse(classCommand)
	cmds = append(cmds, cmd)

	// shell scriptのcreate classを実装
	qdiscCommand := fmt.Sprint("qdisc add dev eth0 parent 1:", id, "0 handle 1", id, ": netem delay ", time)
	cmd, _ = shellwords.Parse(qdiscCommand)
	cmds = append(cmds, cmd)

	// shell scriptのadd filterを実装
	for i := 1; i <= len(ip); i++ {
		filterCommand := fmt.Sprint("filter add dev eth0 protocol ip parent 1: prio ", prio, " u32 match ip dst ", ip[i-1], " flowid 1:", id, "0")
		cmd, _ = shellwords.Parse(filterCommand)
		cmds = append(cmds, cmd)
	}
	fmt.Println("Add command is ready")
	return cmds
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
		Service   string
		Namespace string
		Latency   []Latency `json:"latency"`
	}
	var (
		cg Config
	)

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
	if err != nil {
		panic(err)
	}
	//latency配列分を回す
	for _, latency := range cg.Latency {
		var cmds [][]string
		cmds = append(cmds, Initialize(cli, latency.From)...)
		//delay内に書かれている設定分を回す
		for i, delay := range latency.Delay {
			//toの中身突っ込む.名前解決は2重for文内で
			// for _, to := range delay.To {
			// 	ip, err := net.ResolveIPAddr("ip", fmt.Sprint(to, ".", cg.Service, ".", cg.Namespace, ".svc.cluster.local"))
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// 	ips := append(ips, ip.IP.String())
			// }

			//一時的にip直打ちに変更
			fmt.Println("ip:", delay.To)
			//class名の被りを防ぐためにiを渡す
			cmds = append(cmds, Add(delay.Prio, delay.To, delay.Time, i)...)
		}
		Netemcontainer(latency.From, cli.GlobalString("tc-image"), cmds)
	}
}

func Netemcontainer(name string, tcimage string, cmds [][]string) {
	client, err := container.NewClient()
	if err != nil {
		panic(err)
	}
	client.Netemcontainer(name, tcimage, cmds)
}
