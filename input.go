package main

import (
	"regexp"
	"strings"
)

// Regexp combines a *regexp.Regexp and a Key
type Regexp struct {
	*regexp.Regexp
	Name Key
}

// ProcessInput extracts keywords from a line of input
func ProcessInput(line string) {
	for k, kword := range Conf {
		if kword.Extract != nil &&
			kword.Re != nil &&
			kword.Re.MatchString(line) {
			split := strings.SplitN(line, "=", 2)
			Conf[Key(k)].Value =
				kword.Extract(split[len(split)-1])
			break
		}
	}
}

// ParseInfile parses an input file specified by filename and stores
// the results in the array Input
func ParseInfile(filename string) {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	var (
		block   strings.Builder
		inblock bool
	)
	for i := 0; i < len(lines); i++ {
		switch {
		case lines[i][0] == '#': // comment
		case strings.Contains(lines[i], "}"):
			inblock = false
			ProcessInput(strings.TrimSpace(block.String()))
			block.Reset()
		case strings.Contains(lines[i], "{"):
			keyword := strings.SplitN(lines[i], "{", 2)[0]
			block.WriteString(keyword)
			inblock = true
		case inblock:
			block.WriteString(lines[i] + "\n")
		default:
			ProcessInput(lines[i])
		}
	}
	// Post-parse processing on some of the keywords
	Conf.WhichProgram()
	Conf.WhichCluster() // Cluster EnergyLine overwrites ChemProg
	if Conf.ProcessGeom() {
		Conf.ParseDeltas()
	}
}

// TODO flag for reading pbs template file
// - need to update docs to include that
// - dump subcommand to dump the internal default

// if joblimit is not a multiple of chunksize, we should
// increase joblimit until it is
// if some fields are not present, need to error
// - geometry and the cmds I think are the only ones
//   - and the latter only if needed
//     (not intder and anpass in carts)
