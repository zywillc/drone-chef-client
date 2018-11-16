package ssh

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

/**************
 SSHCommunicator
**************/
type Communicator interface {
	// Connect is used to setup the connection
	Connect() error

	// Disconnect is used to terminate the connection
	Disconnect() error

	// Timeout returns the configured connection timeout
	Timeout() time.Duration

	// Start executes a remote command in a new session
	Start(*Cmd) error
}

// SSHCommunicator represents the SSH SSHCommunicator

type SSHCommunicator struct {
	connInfo *ConnectionInfo
	client   *ssh.Client
	config   *sshConfig
	conn     net.Conn
	address  string
}

// New creates a new SSHCommunicator implementation over SSH.
func New(connInfo *ConnectionInfo) (*SSHCommunicator, error) {
	config, err := prepareSSHConfig(connInfo)
	if err != nil {
		return nil, err
	}

	comm := &SSHCommunicator{
		connInfo: connInfo,
		config:   config,
	}

	return comm, nil
}

func (c *SSHCommunicator) newSession() (session *ssh.Session, err error) {
	log.Println("[DEBUG] opening new ssh session")
	if c.client == nil {
		err = errors.New("ssh client is not connected")
	} else {
		session, err = c.client.NewSession()
	}

	if err != nil {
		log.Printf("[WARN] ssh session open error: '%s', attempting reconnect", err)
		if err := c.Connect(); err != nil {
			return nil, err
		}

		return c.client.NewSession()
	}

	return session, nil
}
// Connect implementation of Communicator.SSHCommunicator interface
func (c *SSHCommunicator) Connect() (err error) {

	if c.conn != nil {
		c.conn.Close()
	}

	// Set the conn and client to nil since we'll recreate it
	c.conn = nil
	c.client = nil


	log.Printf(fmt.Sprintf(
		"[DEBUG] Connecting to remote host via SSH...\n"+
			"  Host: %s\n"+
			"  User: %s\n"+
			"  Password: %t\n"+
			"  Private key: %t\n"+
			"  SSH Agent: %t\n"+
			"  Checking Host Key: %t",
		c.connInfo.Host, c.connInfo.User,
		c.connInfo.Password != "",
		c.connInfo.PrivateKey != "",
		c.connInfo.Agent,
		c.connInfo.HostKey != "",
	))

	if c.connInfo.BastionHost != "" {
		log.Printf(fmt.Sprintf(
			"[DEBUG] Using configured bastion host...\n"+
				"  Host: %s\n"+
				"  User: %s\n"+
				"  Password: %t\n"+
				"  Private key: %t\n"+
				"  SSH Agent: %t\n"+
				"  Checking Host Key: %t",
			c.connInfo.BastionHost, c.connInfo.BastionUser,
			c.connInfo.BastionPassword != "",
			c.connInfo.BastionPrivateKey != "",
			c.connInfo.Agent,
			c.connInfo.BastionHostKey != "",
		))
	}

	log.Printf("[DEBUG] connecting to TCP connection for SSH")
	c.conn, err = c.config.connection()
	if err != nil {
		c.conn = nil

		log.Printf("[ERROR] connection error: %s", err)
		return err
	}

	log.Printf("[DEBUG] handshaking with SSH")
	host := fmt.Sprintf("%s:%d", c.connInfo.Host, c.connInfo.Port)
	sshConn, sshChan, req, err := ssh.NewClientConn(c.conn, host, c.config.config)
	if err != nil {
		log.Printf("[WARN] %s", err)
		return err
	}

	c.client = ssh.NewClient(sshConn, sshChan, req)

	if c.config.sshAgent != nil {
		log.Printf("[DEBUG] Telling SSH config to forward to agent")
		if err := c.config.sshAgent.ForwardToAgent(c.client); err != nil {
			return fatalError{err}
		}

		log.Printf("[DEBUG] Setting up a session to request agent forwarding")
		session, err := c.newSession()
		if err != nil {
			return err
		}
		defer session.Close()

		err = agent.RequestAgentForwarding(session)

		if err == nil {
			log.Printf("[INFO] agent forwarding enabled")
		} else {
			log.Printf("[WARN] error forwarding agent: %s", err)
		}
	}

	return err
}

// Disconnect implementation of Communicator.SSHCommunicator interface
func (c *SSHCommunicator) Disconnect() error {

	if c.config.sshAgent != nil {
		if err := c.config.sshAgent.Close(); err != nil {
			return err
		}
	}

	if c.conn != nil {
		conn := c.conn
		c.conn = nil
		return conn.Close()
	}

	return nil
}

// Timeout implementation of Communicator.SSHCommunicator interface
func (c *SSHCommunicator) Timeout() time.Duration {
	return c.connInfo.TimeoutVal
}


// Start implementation of Communicator.SSHCommunicator interface

func (c *SSHCommunicator) Start(cmd *Cmd) error {
	cmd.Init()

	session, err := c.newSession()
	if err != nil {
		return err
	}

	// Setup our session
	session.Stdin = cmd.Stdin
	session.Stdout = cmd.Stdout
	session.Stderr = cmd.Stderr

	if !c.config.noPty {
		// Request a PTY
		termModes := ssh.TerminalModes{
			ssh.ECHO:          0,     // do not echo
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 80, 40, termModes); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] starting remote command: %s", cmd.Command)
	err = session.Start(strings.TrimSpace(cmd.Command) + "\n")
	if err != nil {
		return err
	}

	// Start a goroutine to wait for the session to end and set the
	// exit boolean and status.
	go func() {
		defer session.Close()

		err := session.Wait()
		exitStatus := 0
		if err != nil {
			exitErr, ok := err.(*ssh.ExitError)
			if ok {
				exitStatus = exitErr.ExitStatus()
			}
		}

		cmd.SetExitStatus(exitStatus, err)
		log.Printf("[DEBUG] remote command exited with '%d': %s", exitStatus, cmd.Command)
	}()

	return nil
}



