package main

import (
	"os"

	"go_tc/pkg/netem"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "edge-emulate"
	app.Usage = "emulate edge environment latency"
	app.Version = "0.1.0"
	app.Commands = initializeCLICommands()
	app.Run(os.Args)
}

func initializeCLICommands() []cli.Command {
	return []cli.Command{
		{
			Name:  "delay",
			Usage: "if use set reset,init or add",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "n, name",
					Usage: "Specify the name of the container",
				},
				cli.StringFlag{
					Name:  "tc-image",
					Usage: "Docker image with tc (iproute2 package); try 'supercord530/iproute2'",
					// Value: "supercord530/iproute2",
				},
			},
			Subcommands: []cli.Command{
				{
					Name:  "reset",
					Usage: "reset delay",
					Action: func(cli *cli.Context) error {
						netem.Reset(cli)
						return nil
					},
					// Flags: []cli.Flag{
					// 	cli.StringFlag{
					// 		Name:  "n, name",
					// 		Usage: "Specify the name of the container",
					// 	},
					// },
				},
				{
					Name:  "init",
					Usage: "initialize delay",
					Action: func(cli *cli.Context) error {
						netem.Initialize(cli)
						return nil
					},
				},
				{
					Name:  "set",
					Usage: "if use set -t,ーf,-s",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "t, time",
							Usage: "Decide how much to delay",
						},
						cli.StringFlag{
							Name:  "f, file",
							Usage: "Set delay by referencing json",
						},
						cli.StringFlag{
							Name:  "s, source",
							Usage: "When using json，Specify the source ip. Default is the ip address of eth0",
						},
					},
					Action: func(cli *cli.Context) error {
						netem.Set(cli)
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "if use set -t,-p,ーf,-s",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "t, time",
							Usage: "Decide how much to delay",
						},
						cli.StringFlag{
							Name:  "p,priority",
							Usage: "Specify priority as an integer",
						},
						cli.StringFlag{
							Name:  "f, file",
							Usage: "Set delay by referencing json",
						},
						cli.StringFlag{
							Name:  "s, source",
							Usage: "When using json，Specify the source ip",
						},
					},
					Action: func(cli *cli.Context) error {
						if cli.String("file") == "" {
							netem.Add("", cli.Args(), cli.String("time"))
							return nil
						} else {
							netem.AddFromJson(cli)
							return nil
						}
					},
				},
			},
		},
	}
}
