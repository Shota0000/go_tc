package main

import (
	"go_tc/pkg/netem"
	"os"

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
					Value: "supercord530/iproute2",
				},
			},
			Subcommands: []cli.Command{
				// {
				// 	Name:  "reset",
				// 	Usage: "reset delay",
				// 	Action: func(cli *cli.Context) error {
				// 		netem.Reset(cli, cli.String("name"))
				// 		return nil
				// 	},
				// 	Flags: []cli.Flag{
				// 		cli.StringFlag{
				// 			Name:  "n, name",
				// 			Usage: "Specify the name of the container",
				// 		},
				// 	},
				// },
				// {
				// 	Name:  "init",
				// 	Usage: "initialize delay",
				// 	Action: func(cli *cli.Context) error {
				// 		netem.Initialize(cli, cli.String("name"))
				// 		return nil
				// 	},
				// 	Flags: []cli.Flag{
				// 		cli.StringFlag{
				// 			Name:  "n, name",
				// 			Usage: "Specify the name of the container",
				// 		},
				// 	},
				// },
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
							Name:  "n, name",
							Usage: "Specify the name of the container",
						},
					},
					Action: func(cli *cli.Context) error {
						netem.Set(cli)
						return nil
					},
				},
			},
		},
	}
}
