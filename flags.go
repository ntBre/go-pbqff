package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

const (
	help = `Requirements (* denotes requirements for SICs only):
- pbqff input file minimally specifying the geometry as a
  space-delimited Zmat or XYZ and the
- paths to intder*, anpass*, and spectro executables
- template intder.in*, anpass.in*, spectro.in, and molpro.in files
  - intder.in should be a pts intder input and have the old geometry
    to serve as a template
  - anpass.in should be a first run anpass file, not a stationary
    point
  - spectro.in should not have any resonance information
  - molpro.in should have the geometry removed and have no closing
    brace to the geometry section
    - on sequoia, the custom energy parameter pbqff=energy is required
      for parsing
    - for gradients, use forces,varsav and show[f20.15],grad{x,y,z} to
      print the gradients
Flags:
`
)

var (
	checkpoint = flag.Bool("c", false, "resume from checkpoint")
	count      = flag.Bool("count", false, "read the input file and print the number of calculations needed then exit")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	debug      = flag.Bool("debug", false, "for debugging, print 2nd derivative energies array")
	debugStack = flag.Bool("debugstack", false, "dump stack traces on garbage heap clears")
	dump       = flag.Bool("dump", false, "dump geom:energy pairs for Cartesian points")
	format     = flag.Bool("fmt", false, "parse existing output files and print them in anpass format")
	freqs      = flag.Bool("freqs", false, "start from running anpass on the pts output")
	irdy       = flag.String("irdy", "", "intder file is ready to be used in pts; specify the atom order")
	maxthreads = flag.Int("maxthreads", 40, "maximum number of OS threads usable by the program. If < 1 there is no limit")
	maxprocs   = flag.Int("maxprocs", 1, "maximum number of simultaneous OS threads. If < 1 default to number of CPUs")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	nodel      = flag.Bool("nodel", false, "don't delete used output files")
	nomatch    = flag.Bool("nomatch", false, "SICs: use the input geometry directly in intder")
	nosym      = flag.Bool("nosym", false, "disable the use of symmetry relationships")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
	pts        = flag.Bool("pts", false, "start by running pts on optimized geometry from opt")
	read       = flag.Bool("r", false, "read reference energy from pts/inp/ref.out")
	test       = flag.Bool("test", false, "shorten wait for signal")
	version    = flag.Bool("version", false, "print the version and exit")
)

var stackDump = func() {
	return
}

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
	if *version {
		fmt.Printf("pbqff version: %s\ncompiled at %s\n", VERSION, COMP_TIME)
		os.Exit(0)
	}
	switch {
	case *freqs:
		FREQS = true
	case *pts:
		PTS = true
		FREQS = true
	default:
		OPT = true
		PTS = true
		FREQS = true
	}
	if *debugStack {
		stackDump = func() {
			fmt.Fprintf(os.Stderr, "\n\n%d goroutines:\n",
				runtime.NumGoroutine())
			buf := make([]byte, 1<<32)
			byts := runtime.Stack(buf, true)
			fmt.Fprintf(os.Stderr, "%s\n\n", buf[:byts])
		}
	}
	return flag.Args()
}
