package main

import (
	"bytes"
	"strings"
	"fmt"
	"os"
	"time"
	"log"
	"errors"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	ssh "github.com/zywillc/drone-chef-client/ssh"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "centos"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute
)

type (
	Config struct {
		// plugin-specific parameters and secrets
		User       string
		Password   string
		Private_Key string
		Host       string
		Host_Key    string
		Port       int
		Agent      bool
		Timeout    string
		Bastion_User       string
		Bastion_Password   string
		Bastion_Private_Key string
		Bastion_Host       string
		Bastion_Host_Key    string
		Bastion_Port       int
		Agent_Identity string

		runList []string
		sudopwd string
	}

	Plugin struct {
		Config Config
	}
)
/***********************************************
* parse out connectionInfo from Plugin Config *
**********************************************/
func parseConnectionInfo(config *Config) (*ssh.ConnectionInfo, error) {
	connInfo := &ssh.ConnectionInfo{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           connInfo,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	m := structs.Map(*config)
	if err := dec.Decode(m); err != nil {
		return nil, err
	}

	// To default Agent to true, we need to check the raw string, since the
	// decoded boolean can't represent "absence of config".
	//
	// And if SSH_AUTH_SOCK is not set, there's no agent to connect to, so we
	// shouldn't try.
	if m["Agent"] == "" && os.Getenv("SSH_AUTH_SOCK") != "" {
		connInfo.Agent = true
	}

	if connInfo.User == "" {
		connInfo.User = DefaultUser
	}

	if connInfo.Port == 0 {
		connInfo.Port = DefaultPort
	}
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
	}

	// Default all bastion config attrs to their non-bastion counterparts
	if connInfo.BastionHost != "" {

		if connInfo.BastionUser == "" {
			connInfo.BastionUser = connInfo.User
		}
		if connInfo.BastionPassword == "" {
			connInfo.BastionPassword = connInfo.Password
		}
		if connInfo.BastionPrivateKey == "" {
			connInfo.BastionPrivateKey = connInfo.PrivateKey
		}
		if connInfo.BastionPort == 0 {
			connInfo.BastionPort = connInfo.Port
		}
	}

	return connInfo, nil
}

// safeDuration returns either the parsed duration or a default value
func safeDuration(dur string, defaultDur time.Duration) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		log.Printf("Invalid duration '%s', using default of %s", dur, defaultDur)
		return defaultDur
	}
	return d
}



// Plugin execution implementation
func (p Plugin) Exec() error {
	// plugin logic goes here
	conf := p.Config
	connInfo, err := parseConnectionInfo(&conf)
	if err != nil {
		return err
	}

	c, err := ssh.New(connInfo)
	if err != nil {
		return errors.New(fmt.Sprintf("error creating ssh communicator: %s", err))
	}

	var cmd ssh.Cmd
	stdout := new(bytes.Buffer)
	var b strings.Builder
	cmd_string := fmt.Sprintf("echo %s | sudo -S chef-client", conf.sudopwd)
	b.WriteString(cmd_string)
	if len(conf.runList) > 0 {
		b.WriteString(" -r ")
		b.WriteString(strings.Join(conf.runList, ","))
	}
	cmd.Command = b.String()
	cmd.Stdout = stdout

	err = c.Start(&cmd)
	if err != nil {
		return errors.New(fmt.Sprintf("error executing remote command: %s", err))
	}

	return err
}