package ssh

import (
	"net"
	"time"
	"log"
	"fmt"
	"golang.org/x/crypto/ssh"
)

// ConnectFunc is a convenience method for returning a function
// that just uses net.Dial to communicate with the remote end that
// is suitable for use with the SSH communicator configuration.
func ConnectFunc(network, addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		c, err := net.DialTimeout(network, addr, 15*time.Second)
		if err != nil {
			return nil, err
		}

		if tcpConn, ok := c.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
		}

		return c, nil
	}
}

// BastionConnectFunc is a convenience method for returning a function
// that connects to a host over a bastion connection.
func BastionConnectFunc(
	bProto string,
	bAddr string,
	bConf *ssh.ClientConfig,
	proto string,
	addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		log.Printf("[DEBUG] Connecting to bastion: %s", bAddr)
		bastion, err := ssh.Dial(bProto, bAddr, bConf)
		if err != nil {
			return nil, fmt.Errorf("Error connecting to bastion: %s", err)
		}

		log.Printf("[DEBUG] Connecting via bastion (%s) to host: %s", bAddr, addr)
		conn, err := bastion.Dial(proto, addr)
		if err != nil {
			bastion.Close()
			return nil, err
		}

		// Wrap it up so we close both things properly
		return &bastionConn{
			Conn:    conn,
			Bastion: bastion,
		}, nil
	}
}

type bastionConn struct {
	net.Conn
	Bastion *ssh.Client
}

func (c *bastionConn) Close() error {
	c.Conn.Close()
	return c.Bastion.Close()
}