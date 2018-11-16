package ssh

import (
	"io"
)

// Cmd represents a remote command being prepared or run.
type Cmd struct {
	// Command is the command to run remotely.
	Command string

	// if nil, the process reads from an empty bytes.Buffer.
	Stdin io.Reader

	// If nil, it will be set to ioutil.Discard.
	Stdout io.Writer
	Stderr io.Writer

	exitStatus int

	// Internal fields
	exitCh chan struct{}

	err error
}

// Init must be called by the Communicator before executing the command.
func (c *Cmd) Init() {
	c.exitCh = make(chan struct{})
}


func (c *Cmd) SetExitStatus(status int, err error) {
	c.exitStatus = status
	c.err = err

	close(c.exitCh)
}

func (c *Cmd) Wait() error {
	<-c.exitCh

	if c.err != nil || c.exitStatus != 0 {
		return &ExitError{
			Command:    c.Command,
			ExitStatus: c.exitStatus,
			Err:        c.err,
		}
	}

	return nil
}
