// Package process has code that runs graphviz dot command as a subprocess.
package process

import "os/exec"

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

func (c *Cmd) StdinPipe() io.WriteCloser {
	if c.err != nil {
		return nil
	}
	var w io.WriteCloser
	w, c.err = c.command.StdinPipe()
	return w
}

func (c *Cmd) Start() {
	if c.err != nil {
		return
	}
	c.err = c.command.Start()
}

func (c *Cmd) Wait() {
	if c.err != nil {
		return
	}
	c.err = c.command.Wait()
}

func (c *Cmd) Error() error {
	return c.err
}
