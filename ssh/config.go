package ssh

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/xanzy/ssh-agent"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)


type ConnectionInfo struct {
	User       string
	Password   string
	PrivateKey string `mapstructure:"Private_Key"`
	Host       string
	HostKey    string `mapstructure:"Host_Key"`
	Port       int
	Agent      bool
	Timeout    string
	TimeoutVal time.Duration `mapstructure:"-"`

	BastionUser       string `mapstructure:"Bastion_User"`
	BastionPassword   string `mapstructure:"Bastion_Password"`
	BastionPrivateKey string `mapstructure:"Bastion_Private_Key"`
	BastionHost       string `mapstructure:"Bastion_Host"`
	BastionHostKey    string `mapstructure:"Bastion_Host_Key"`
	BastionPort       int    `mapstructure:"Bastion_Port"`

	AgentIdentity string `mapstructure:"Agent_Identity"`
}

/***********************************************
ssh config
**********************************************/

// prepare ssh agent
func connectToAgent(connInfo *ConnectionInfo) (*sshAgent, error) {
	if connInfo.Agent != true {
		// No agent configured
		return nil, nil
	}

	agent, conn, err := sshagent.New()
	if err != nil {
		return nil, err
	}

	// connection close is handled over in Communicator
	return &sshAgent{
		agent: agent,
		conn:  conn,
		id:    connInfo.AgentIdentity,
	}, nil

}

// SSH Config Options
type sshClientConfigOpts struct {
	privateKey string
	password   string
	sshAgent   *sshAgent
	user       string
	host       string
	hostKey    string
}

func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode([]byte(pk))
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	signer, err := ssh.ParsePrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key file %q: %s", pk, err)
	}

	return ssh.PublicKeys(signer), nil
}

func buildSSHClientConfig(opts sshClientConfigOpts) (*ssh.ClientConfig, error) {
	hkCallback := ssh.InsecureIgnoreHostKey()

	if opts.hostKey != "" {

		tf, err := ioutil.TempFile("", "drone_known_hosts")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp known_hosts file: %s", err)
		}
		defer tf.Close()
		defer os.RemoveAll(tf.Name())

		if _, err := tf.WriteString(fmt.Sprintf("@cert-authority %s %s\n", opts.host, opts.hostKey)); err != nil {
			return nil, fmt.Errorf("failed to write temp known_hosts file: %s", err)
		}
		tf.Sync()

		hkCallback, err = knownhosts.New(tf.Name())
		if err != nil {
			return nil, err
		}
	}

	conf := &ssh.ClientConfig{
		HostKeyCallback: hkCallback,
		User:            opts.user,
	}

	if opts.privateKey != "" {
		pubKeyAuth, err := readPrivateKey(opts.privateKey)
		if err != nil {
			return nil, err
		}
		conf.Auth = append(conf.Auth, pubKeyAuth)
	}

	if opts.password != "" {
		conf.Auth = append(conf.Auth, ssh.Password(opts.password))
	}

	if opts.sshAgent != nil {
		conf.Auth = append(conf.Auth, opts.sshAgent.Auth())
	}

	return conf, nil
}

// prepare ssh config
type sshConfig struct {
	config *ssh.ClientConfig

	// connection returns a new connection. The current connection
	// in use will be closed as part of the Close method, or in the
	// case an error occurs.
	connection func() (net.Conn, error)

	// noPty, if true, will not request a pty from the remote end.
	noPty bool

	sshAgent *sshAgent
}
func prepareSSHConfig(connInfo *ConnectionInfo) (*sshConfig, error) {
	sshAgent, err := connectToAgent(connInfo)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", connInfo.Host, connInfo.Port)

	sshConf, err := buildSSHClientConfig(sshClientConfigOpts{
		user:       connInfo.User,
		host:       host,
		privateKey: connInfo.PrivateKey,
		password:   connInfo.Password,
		hostKey:    connInfo.HostKey,
		sshAgent:   sshAgent,
	})
	if err != nil {
		return nil, err
	}

	connectFunc := ConnectFunc("tcp", host)

	var bastionConf *ssh.ClientConfig
	if connInfo.BastionHost != "" {
		bastionHost := fmt.Sprintf("%s:%d", connInfo.BastionHost, connInfo.BastionPort)

		bastionConf, err = buildSSHClientConfig(sshClientConfigOpts{
			user:       connInfo.BastionUser,
			host:       bastionHost,
			privateKey: connInfo.BastionPrivateKey,
			password:   connInfo.BastionPassword,
			hostKey:    connInfo.HostKey,
			sshAgent:   sshAgent,
		})
		if err != nil {
			return nil, err
		}

		connectFunc = BastionConnectFunc("tcp", bastionHost, bastionConf, "tcp", host)
	}

	config := &sshConfig{
		config:     sshConf,
		connection: connectFunc,
		sshAgent:   sshAgent,
	}
	return config, nil
}
