// Package process has code that runs graphviz dot command as a subprocess.
package process

import "os"

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
