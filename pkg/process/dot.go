// Package process has code that runs graphviz dot command as a subprocess.
package process

import (
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

var (
	// dot is the "dot" executable.
	dot = Required("dot")
)

// Required looks up the program in path, and panics if not found.
func Required(program string) string {
	path, err := exec.LookPath(program)
	if err != nil {
		panic(err)
	}
	return path
}

// Cmd is a dot command and execution error.
type Cmd struct {
	command *exec.Cmd
	err     error
}

// NewCommand makes a new dot command.
func NewCommand(output string) *Cmd {
	var c Cmd
	c.command = exec.Command(dot, "-Tpng", "-o", output)
	return &c
}

// StdinPipe gets the input end of the stdin pipe for the executed program.
func (c *Cmd) StdinPipe() io.WriteCloser {
	if c.err != nil {
		return nil
	}
	var w io.WriteCloser
	w, c.err = c.command.StdinPipe()
	return w
}

// Start runs the dot program asynchronously.
func (c *Cmd) Start() {
	if c.err != nil {
		return
	}
	c.err = c.command.Start()
}

// Wait waits for dot to complete the graph rendering.
func (c *Cmd) Wait() {
	if c.err != nil {
		return
	}
	log.Debugf("Wait()")
	c.err = c.command.Wait()
	log.Debugf("Wait() end")
}

// Error returns the error (if any)
func (c *Cmd) Error() error {
	return c.err
}
