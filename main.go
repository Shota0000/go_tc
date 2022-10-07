package main

import (
	"fmt"
	"os"
	"os/exec"

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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "t, time",
					Value: "",
					Usage: "delay -t time -i ip",
				},
				cli.StringSliceFlag{
					Name:  "i, ip",
					Usage: "delay -t time -i ip",
				},
			},
			Action: func(c *cli.Context) error {
				// fmt.Printf("Hello %s %s \n", c.StringSlice("ip"), c.String("time"))
				out, err := exec.Command("tc", "qdisc del dev eth0 root").Output()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(string(out))

				out, err = exec.Command("tc", "qdisc add dev eth0 root handle 1: htb default 10").Output()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(string(out))

				out, err = exec.Command("tc", "class add dev eth0 parent 1: classid 1:1 htb rate 100Gbit").Output()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(string(out))

				// add qdisc to class
				for i := 0; i < len(c.StringSlice("ip")); i++ {
					out, err = exec.Command("tc", fmt.Sprint("class add dev eth0 parent 1:1 classid 1:", i, "0 htb rate 10Gbit")).Output()
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println(string(out))
				}

				// shell scriptのcreate classを実装
				for i := 0; i < len(c.StringSlice("ip")); i++ {
					out, err = exec.Command("tc", fmt.Sprint("qdisc add dev eth0 parent 1:", i, "0 handle 1", i, ": netem delay 100ms")).Output()
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println(string(out))
				}

				// shell scriptのadd filterを実装
				for i := 0; i < len(c.StringSlice("ip")); i++ {
					out, err = exec.Command("tc", fmt.Sprint("filter add dev eth0 protocol ip parent 1: prio 1 u32 match ip dst ", c.StringSlice("ip")[i], " flowid 1:", i, "0")).Output()
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println(string(out))
				}

				return nil

				// TO DO macでtcコマンド動かんからdocker立ち上げて動かす

			},
		},
	}

	app.Run(os.Args)
}
