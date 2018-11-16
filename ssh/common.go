package ssh

import "fmt"

/****************
 Error definition
****************/
type fatalError struct {
	error
}

func (e fatalError) FatalError() error {
	return e.error
}

// ExitError is returned by Wait to indicate and error executing the remote
// command, or a non-zero exit status.
type ExitError struct {
	Command    string
	ExitStatus int
	Err        error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("error executing %q: %v", e.Command, e.Err)
	}
	return fmt.Sprintf("%q exit status: %d", e.Command, e.ExitStatus)
}