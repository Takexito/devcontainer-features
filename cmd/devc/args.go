package main

import (
	"flag"
	"fmt"
	"os"
)

func newFlagSet(name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet("devc "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "использование: %s\n", usage)
		fs.PrintDefaults()
	}
	return fs
}

// parse разбирает флаги и позиционные аргументы в любом порядке:
// «devc up my-app --stack rust» привычнее, чем флаги строго впереди.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}
