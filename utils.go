package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// CleanSplit splits a line using strings.Split and then removes
// empty entries
func CleanSplit(str, sep string) []string {
	lines := strings.Split(str, sep)
	clean := make([]string, 0, len(lines))
	for s := range lines {
		if lines[s] != "" {
			clean = append(clean, lines[s])
		}
	}
	return clean
}

// RunProgram runs a program, redirecting STDIN from filename.in
// and STDOUT to filename.out
func RunProgram(progName, filename string) error {
	current, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fpath := path.Dir(filename)
	if err = os.Chdir(fpath); err != nil {
		panic(err)
	}
	file := path.Base(filename)
	infile := file + ".in"
	outfile := file + ".out"
	out, err := exec.Command("bash", "-c", progName+" < "+infile+" > "+outfile).Output()
	os.Chdir(current)
	if err != nil {
		return fmt.Errorf("error RunProgram: failed with %v running %q on %q"+
			"\nstdout: %q",
			err, progName, infile, out)
	}
	return nil
}
