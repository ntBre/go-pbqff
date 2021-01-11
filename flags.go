package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	help = `Requirements (* denotes requirements for SICs only):
- pbqff input file minimally specifying the geometry and the
- paths to intder*, anpass*, and spectro executables
- template intder.in*, anpass.in*, spectro.in, and molpro.in files
  - intder.in should be a pts intder input and have the old geometry to serve as a template
  - anpass.in should be a first run anpass file, not a stationary point
  - spectro.in should not have any resonance information
  - molpro.in should have the geometry removed and have no closing brace to the geometry section
    - on sequoia, the custom energy parameter pbqff=energy is required for parsing
    - for gradients, use forces,varsav and show[f20.15],grad{x,y,z} to print the gradients
Flags:
`
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
	pts        = flag.Bool("pts", false,
		"start by running pts on optimized geometry from opt")
	freqs = flag.Bool("freqs", false, "start from running anpass on the pts output")
	debug = flag.Bool("debug", false,
		"for debugging, print 2nd derivative energies array")
	checkpoint = flag.Bool("c", false, "resume from checkpoint")
	read       = flag.Bool("r", false, "read reference energy from pts/inp/ref.out")
	irdy       = flag.String("irdy", "",
		"intder file is ready to be used in pts; specify the atom order")
	count = flag.Bool("count", false,
		"read the input file and print the number of calculations needed then exit")
	nodel  = flag.Bool("nodel", false, "don't delete used output files")
	format = flag.Bool("fmt", false,
		"parse existing output files and print them in anpass format")
)

// ParseFlags parses command line flags and returns a slice of
// the remaining arguments
func ParseFlags() []string {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), help)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *format {
		FormatOutput("pts/inp/")
		os.Exit(0)
	}
	switch {
	case *freqs:
		flags = FREQS
	case *pts:
		flags = PTS | FREQS
	default:
		flags = OPT | PTS | FREQS
	}
	if *checkpoint {
		LoadCheckpoint()
	}
	return flag.Args()
}
