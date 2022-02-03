package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// ProcessInput extracts keywords from a line of input
func ProcessInput(line string) {
	line = strings.ToLower(line)
	split := strings.SplitN(line, "=", 2)
	key, val := split[0], split[1]
	for _, kword := range reflect.VisibleFields(reflect.TypeOf(Conf)) {
		keyname := kword.Name
		if strings.ToLower(keyname) == key {
			loc := reflect.ValueOf(&Conf).Elem().FieldByName(keyname)
			fmt.Printf("%v %v %s\n", kword.Type, loc, val)
			// switch kword.Type {

			// // .Set(val)
			// }
			// Conf[Key(k)].Value =
			// 	kword.Extract(split[len(split)-1])
			break
		}
	}
}

// ParseInfile parses an input file specified by filename and stores
// the results in the array Input
func ParseInfile(filename string) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	var (
		block   strings.Builder
		inblock bool
		line    string
	)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case len(line) > 0 && line[0] == '#': // comment
		case strings.Contains(line, "}"):
			inblock = false
			ProcessInput(strings.TrimSpace(block.String()))
			block.Reset()
		case strings.Contains(line, "{"):
			keyword := strings.SplitN(line, "{", 2)[0]
			block.WriteString(keyword)
			inblock = true
		case inblock:
			block.WriteString(line + "\n")
		default:
			ProcessInput(line)
		}
	}
	// Post-parse processing on some of the keywords
	Conf.WhichProgram()
	Conf.WhichCluster() // Cluster EnergyLine overwrites ChemProg
	if Conf.ProcessGeom() {
		Conf.ParseDeltas("TODO")
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
