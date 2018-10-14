// Package main is the entrypoint for the program "typediag", which prints a
// dot digraph of a go package.
package main

import (
	"flag"

	"github.com/filmil/typediag/pkg/dot"
	log "github.com/sirupsen/logrus"
)

var (
	// path is the go package path to analyze.
	path string

	exported bool
)

func main() {
	flag.StringVar(&path, "path", "", "Package path to analyze")
	flag.BoolVar(&exported, "exported-only", true, "If set, only exported types are analyzed")
	flag.Parse()

	if path == "" {
		log.Fatalf("flag --path=... is required")
	}

	d := dot.NewDiagram(exported, path)

	if err := d.Render(); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}
