package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "edge-emulate"
	app.Usage = "emulate edge environment latency"
	app.Version = "0.1.0"
	app.Commands = []cli.Command{
		{
			Name:  "delay",
			Usage: "if use set reset,init or add",
			Subcommands: []cli.Command{
				{
					Name:  "reset",
					Usage: "reset delay",
					Action: func(c *cli.Context) error {
						reset(c)
						return nil
					},
				},
				{
					Name:  "init",
					Usage: "if use set -t，ーi",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "t, time",
							Usage: "Decide how much to delay",
						},
						cli.StringSliceFlag{
							Name:  "i, ip",
							Usage: "Specify ip address for delay",
						},
						cli.StringFlag{
							Name:  "f, file",
							Usage: "Set delay by referencing json",
						},
					},
					Action: func(c *cli.Context) error {
						initialize(c)
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "if use set -t，ーi",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "t, time",
							Usage: "Decide how much to delay",
						},
						cli.StringSliceFlag{
							Name:  "i, ip",
							Usage: "Specifies an ip address. Multiple addresses are possible.",
						},
						cli.StringFlag{
							Name:  "p,priority",
							Usage: "Specify priority as an integer",
						},
						cli.StringFlag{
							Name:  "f, file",
							Usage: "Set delay by referencing json",
						},
					},
					Action: func(c *cli.Context) error {
						if c.String("file") == "" {
							add("", c.StringSlice("ip"), c.String("time"))
							return nil
						} else {
							addFromJson(c)
							return nil
						}
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

func reset(c *cli.Context) {
	cmd, _ := shellwords.Parse("tc qdisc del dev eth0 root")
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
		return
	}
	fmt.Println("reset completed!")
}

func initialize(c *cli.Context) {
	reset(c)
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
	if c.String("file") == "" {
		add("", c.StringSlice("ip"), c.String("time"))
	} else {
		addFromJson(c)
	}
	fmt.Println("init completed!")
}

func add(prio string, ip []string, time string) {
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

func addFromJson(c *cli.Context) {
	type DelayInfo struct {
		Time string   `json:"time"`
		From string   `json:"from"`
		To   []string `json:"to"`
		Prio string   `json:"priority"`
	}

	type Config struct {
		Latency []DelayInfo `json:"latency"`
	}
	var (
		cg Config
	)

	raw, err := ioutil.ReadFile(c.String("file"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = json.Unmarshal(raw, &cg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, di := range cg.Latency {
		// fmt.Println(di.Prio, di.To, di.Time)
		add(di.Prio, di.To, di.Time)
	}
}
