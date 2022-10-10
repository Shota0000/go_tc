package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "Delay_tc"
	app.Usage = "Make `delay xxx`"
	app.Version = "0.1.0"
	app.Commands = []cli.Command{
		{
			Name:  "delay",
			Usage: "if use set -i or --ip ",
			Subcommands: []cli.Command{
				{
					Name:  "reset",
					Usage: "delay reset",
					Action: func(c *cli.Context) error {
						cmd, _ := shellwords.Parse("tc qdisc del dev eth0 root")
						out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(string(out))
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "delay add -t time -p ip ...",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "t, time",
							Value: "",
							Usage: "delay -t time -i ip",
						},
						cli.StringSliceFlag{
							Name:  "i, ip",
							Usage: "delay -t time -i ip -i ip ...",
						},
					},
					Action: func(c *cli.Context) error {
						cmd, _ := shellwords.Parse("tc qdisc add dev eth0 root handle 1: htb default 1")
						out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(string(out))

						cmd, _ = shellwords.Parse("tc class add dev eth0 parent 1: classid 1:1 htb rate 100Gbit")
						out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
						if err != nil {
							fmt.Println(err)
						}
						fmt.Println(string(out))

						// add qdisc to class
						for i := 1; i <= len(c.StringSlice("ip")); i++ {
							cmd, _ = shellwords.Parse(fmt.Sprint("tc class add dev eth0 parent 1:1 classid 1:", i, "0 htb rate 10Gbit"))
							out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
							if err != nil {
								fmt.Println(err)
							}
							fmt.Println(string(out))
							fmt.Println(fmt.Sprint("tc class add dev eth0 parent 1:1 classid 1:", i, "0 htb rate 10Gbit"))
						}

						// shell scriptのcreate classを実装
						for i := 1; i <= len(c.StringSlice("ip")); i++ {
							cmd, _ = shellwords.Parse(fmt.Sprint("tc qdisc add dev eth0 parent 1:", i, "0 handle 1", i, ": netem delay ", c.String("time")))
							out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
							if err != nil {
								fmt.Println(err)
							}
							fmt.Println(string(out))
							fmt.Println(fmt.Sprint("tc qdisc add dev eth0 parent 1:", i, "0 handle 1", i, ": netem delay ", c.String("time")))

						}

						// shell scriptのadd filterを実装
						for i := 1; i <= len(c.StringSlice("ip")); i++ {
							cmd, _ = shellwords.Parse(fmt.Sprint("tc filter add dev eth0 protocol ip parent 1: prio 1 u32 match ip dst ", c.StringSlice("ip")[i-1], " flowid 1:", i, "0"))
							out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
							if err != nil {
								fmt.Println(err)
							}
							fmt.Println(string(out))
							fmt.Println(fmt.Sprint("tc filter add dev eth0 protocol ip parent 1: prio 1 u32 match ip dst ", c.StringSlice("ip")[i-1], " flowid 1:", i, "0"))
						}

						return nil
					},
				},
			},
		},
	}
	app.Run(os.Args)
}
