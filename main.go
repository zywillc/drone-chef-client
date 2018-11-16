package main

import (
	"fmt"
	"os"
	"log"
	"errors"

	"github.com/urfave/cli"
)

var (
	version = "0.0.0"
	build = "0"
)


func main() {
	app := cli.NewApp()
	app.Name = "chef client plugin"
	app.Usage = "chef client plugin "
	app.Action = run
	app.Version = fmt.Sprintf("%s+%s", version, build)
	app.Flags = []cli.Flag{

		cli.StringFlag{
			Name:   "user",
			Usage:  "ssh user name",
			EnvVar: "PLUGIN_USER",
		},
		cli.StringFlag{
			Name:   "password",
			Usage:  "ssh password",
			EnvVar: "PLUGIN_PASSWORD",
		},
		cli.StringFlag{
			Name:   "private-key",
			Usage:  "ssh private key",
			EnvVar: "PLUGIN_PRIVATE_KEY, SSH_PRIVATE_KEY",
		},
		cli.StringFlag{
			Name:   "host",
			Usage:  "ssh host ip",
			EnvVar: "PLUGIN_HOST",
		},
		cli.StringFlag{
			Name:   "host-key",
			Usage:  "ssh host key",
			EnvVar: "PLUGIN_HOST_KEY, SSH_HOST_KEY",
		},
		cli.IntFlag{
			Name:   "port",
			Usage:  "ssh port",
			EnvVar: "PLUGIN_PORT",
		},
		cli.BoolFlag{
			Name:   "agent",
			Usage:  "ssh agent",
			EnvVar: "PLUGIN_AGENT",
		},
		cli.StringFlag{
			Name:   "timeout",
			Usage:  "ssh timeout",
			EnvVar: "PLUGIN_TIMEOUT",
		},
		cli.StringFlag{
			Name:   "bastion-user",
			Usage:  "ssh bastion user",
			EnvVar: "PLUGIN_BASTION_USER",
		},
		cli.StringFlag{
			Name:   "bastion-password",
			Usage:  "ssh bastion password",
			EnvVar: "PLUGIN_BASTION_PASSWORD",
		},
		cli.StringFlag{
			Name:   "bastion private key",
			Usage:  "ssh bastion private key",
			EnvVar: "PLUGIN_BASTION_PRIVATE_KEY, SSH_BASTION_PRIVATE_KEY",
		},
		cli.StringFlag{
			Name:   "bastion-host",
			Usage:  "ssh bastion host ip",
			EnvVar: "PLUGIN_BASTION_HOST",
		},
		cli.StringFlag{
			Name:   "bastion-host-key",
			Usage:  "ssh bastion host key",
			EnvVar: "PLUGIN_BASTION_HOST_KEY, SSH_BASTION_HOST_KEY",
		},
		cli.IntFlag{
			Name:   "bastion-port",
			Usage:  "ssh bastion port",
			EnvVar: "PLUGIN_BASTION_PORT",
		},
		cli.StringFlag{
			Name:   "agent-indentity",
			Usage:  "ssh agent identity",
			EnvVar: "PLUGIN_AGENT_IDENTITY, SSH_AGENT_IDENTITY",
		},
		cli.StringSliceFlag{
			Name: "run-list",
			Usage: "chef client run list",
			EnvVar: "PLUGIN_RUN_LIST",
		},
		cli.StringFlag{
			Name: "sudo-password",
			Usage: "chef client sudo password",
			EnvVar: "PLUGIN_SUDO_PASSWORD, CHEF_CLIENT_SUDO_PASSWORD, SUDO_PASSWORD",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error{
	plugin := Plugin{
		Config: Config{
			User: 						c.String("user"),
			Password: 					c.String("password"),
			Private_Key: 				c.String("private-key"),
			Host: 						c.String("host"),
			Host_Key:					c.String("host-key"),
			Port:						c.Int("port"),
			Agent:						c.Bool("agent"),
			Timeout:					c.String("timeout"),
			Bastion_User:				c.String("bastion-user"),
			Bastion_Password:			c.String("bastion-password"),
			Bastion_Private_Key:		c.String("bastion-private-key"),
			Bastion_Host_Key:			c.String("bastion-host-key"),
			Bastion_Port:				c.Int("bastion-port"),
			Agent_Identity:				c.String("agent-identity"),
			runList:					c.StringSlice("run-list"),
			sudopwd:					c.String("sudo-password"),
		},
	}

	if err := plugin.Exec(); err != nil {
		return errors.New(fmt.Sprintf("Excecuting plugin fails: %s", err))
	}

	return nil
}
