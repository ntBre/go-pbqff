package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Molpro struct {
	Geometry string
	Basis    string
	Charge   string
	Spin     string
	Method   string
}

// Takes an input filename and template filename
// and writes an input file
func (m *Molpro) WriteInput(infile, tfile string) {
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t := LoadTemplate(tfile)
	t.Execute(f, m)
}

// Format z-matrix for use in Molpro input
func FormatZmat(geom string) string {
	var out []string
	split := strings.Split(geom, "\n")
	for i, line := range split {
		if strings.Contains(line, "=") {
			out = append(append(append(out, split[:i]...), "}"), split[i:]...)
			break
		}
	}
	return strings.Join(out, "\n")
}

func (m Molpro) ReadOut(filename string) (result float64, err error) {
	runtime.LockOSThread()
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
		return brokenFloat, ErrFileNotFound
	}
	error := regexp.MustCompile(`(?i)[^_]error`)
	err = ErrEnergyNotFound
	result = brokenFloat
	lines := ReadFile(filename)
	// ASSUME blank file is only created when PBS runs
	// blank file has a single newline - which is stripped by this ReadLines
	if len(lines) == 1 {
		if strings.Contains(strings.ToUpper(lines[0]), "ERROR") {
			return result, ErrFileContainsError
		}
		return result, ErrBlankOutput
	} else if len(lines) == 0 {
		return result, ErrBlankOutput
	}

	for _, line := range lines {
		if error.MatchString(line) {
			return result, ErrFileContainsError
		}
		if strings.Contains(line, energyLine) {
			split := strings.Fields(line)
			for i, _ := range split {
				if strings.Contains(split[i], energyLine) {
					// take the thing right after search term
					// not the last entry in the line
					if i+1 < len(split) {
						// assume we found energy so no error
						// from default EnergyNotFound
						err = nil
						result, err = strconv.ParseFloat(split[i+1], 64)
						if err != nil {
							result = math.NaN()
						}
					}
				}
			}
		}
		if strings.Contains(line, molproTerminated) && err != nil {
			err = ErrFinishedButNoEnergy
		}
	}
	runtime.UnlockOSThread()
	return result, err
}

// Handle .out and .log files for filename, assumes no extension
func (m Molpro) HandleOutput(filename string) error {
	outfile := filename + ".out"
	logfile := filename + ".log"
	lines := ReadFile(outfile)
	warn := regexp.MustCompile(`(?i)warning`)
	error := regexp.MustCompile(`(?i)[^_]error`)
	warned := false
	// notify about warnings or errors in output file
	// apparently warnings are not printed in the log
	for _, line := range lines {
		if warn.MatchString(line) && !warned {
			fmt.Fprintf(os.Stderr,
				"HandleOutput: warning found in %s, continuing\n",
				outfile)
			warned = true
		}
		if error.MatchString(line) {
			fmt.Fprintf(os.Stderr,
				"HandleOutput: error %q, found in %s, aborting\n",
				line, outfile)
			return ErrFileContainsError
		}
	}
	// looking for optimized geometry in bohr
	lines = ReadFile(logfile)
	for _, line := range lines {
		if strings.Contains(line, "ATOMIC COORDINATES") {
			// inGeom = true
		}
		// TODO this is the important part
	}
	return nil
}
