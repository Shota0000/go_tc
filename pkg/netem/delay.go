package netem

import (
	"encoding/json"
	"fmt"
	"go_tc/pkg/container"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-pipeline"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func Reset(cli *cli.Context) {
	cmd, _ := shellwords.Parse("qdisc del dev eth0 root")
	Netemcontainer(cli.GlobalString("name"), cli.GlobalString("tc-image"), cmd)
	fmt.Println("reset completed!")
}

func Initialize(cli *cli.Context) {
	Reset(cli)
	cmd, _ := shellwords.Parse("tc qdisc add dev eth0 root handle 1: htb default 1")
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
		return
	}
	// fmt.Println(string(out))

	cmd, _ = shellwords.Parse("tc class add dev eth0 parent 1: classid 1:1 htb rate 1000Gbit")
	out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
		return
	}
	fmt.Println("init completed!")
}

func Set(cli *cli.Context) {
	Initialize(cli)
	if cli.String("file") == "" {
		Add("", cli.Args(), cli.String("time"))
	} else {
		AddFromJson(cli)
	}
	fmt.Println("set completed!")
}

func Add(prio string, ip []string, time string) {
	var (
		cmd []string
		out []byte
		err error
	)

	roop := 1

	if prio == "" {
		prio = "100"
	}

	for {
		cmd, _ = shellwords.Parse(fmt.Sprint("tc class add dev eth0 parent 1:1 classid 1:", roop, "0 htb rate 10Gbit"))
		_, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		// out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			// fmt.Println(err, string(out), roop)
			roop++
			continue
		}
		// fmt.Println(string(out))
		// fmt.Println(fmt.Sprint("tc class add dev eth0 parent 1:1 classid 1:", roop, "0 htb rate 10Gbit"))
		break
	}

	// shell scriptのcreate classを実装

	cmd, _ = shellwords.Parse(fmt.Sprint("tc qdisc add dev eth0 parent 1:", roop, "0 handle 1", roop, ": netem delay ", time))
	out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
		return
	}
	// fmt.Println(string(out))
	// fmt.Println(fmt.Sprint("tc qdisc add dev eth0 parent 1:", roop, "0 handle 1", roop, ": netem delay ", c.String("time")))

	// shell scriptのadd filterを実装
	for i := 1; i <= len(ip); i++ {
		cmd, _ = shellwords.Parse(fmt.Sprint("tc filter add dev eth0 protocol ip parent 1: prio ", prio, " u32 match ip dst ", ip[i-1], " flowid 1:", roop, "0"))
		out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			fmt.Println(err)
			fmt.Println(string(out))
			return
		}
		// fmt.Println(string(out))
		// fmt.Println(fmt.Sprint("tc filter add dev eth0 protocol ip parent 1: prio 1 u32 match ip dst ", c.StringSlice("ip")[i-1], " flowid 1:", roop, "0"))
	}
	fmt.Println("add completed!")
}

func AddFromJson(cli *cli.Context) {
	type DelayInfo struct {
		Time string   `json:"time"`
		From string   `json:"from"`
		To   []string `json:"to"`
		Prio string   `json:"priority"`
	}

	type Config struct {
		Latency []DelayInfo `json:"latency"`
	}
	var cg Config
	var ip string

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

	if cli.String("source") == "" {
		out, err := pipeline.Output(
			[]string{"ip", "a"},
			[]string{"grep", "-x", ".*eth0"},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		ip = strings.TrimLeft(string(out), "inet ")
		ip = ip[0:strings.Index(ip, "/")]
	} else {
		ip = cli.String("source")
	}
	fmt.Println(ip)

	for _, di := range cg.Latency {
		if di.From == ip {
			Add(di.Prio, di.To, di.Time)
		} else {
			continue
		}
	}
}

func Netemcontainer(name string, tcimage string, cmd []string) {
	client, err := container.NewClient()
	if err != nil {
		panic(err)
	}
	// client.Netemcontainer("docker_test1", "supercord530/iproute2")
	client.Netemcontainer(name, tcimage, cmd)
}
