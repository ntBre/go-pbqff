package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	opt Procedure = iota
	freq
	none
)

type Procedure int

type Molpro struct {
	Head     string
	Geometry string
	Tail     string
	Opt      string
}

// Load a template molpro input file
func LoadMolpro(filename string) *Molpro {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		buf  bytes.Buffer
		line string
		mp   Molpro
	)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "optg") && !strings.Contains(line, "gthresh") {
			mp.Tail = buf.String()
			buf.Reset()
		}
		buf.WriteString(line + "\n")
		if strings.Contains(line, "geometry=") {
			mp.Head = buf.String()
			buf.Reset()
		}
	}
	mp.Opt = buf.String()
	return &mp
}

// Write a Molpro input file
func (m *Molpro) WriteInput(filename string, p Procedure) {
	var buf bytes.Buffer
	buf.WriteString(m.Head)
	buf.WriteString(m.Geometry + "\n")
	buf.WriteString(m.Tail)
	switch {
	case p == opt:
		buf.WriteString(m.Opt)
	case p == freq:
		buf.WriteString("{frequencies}\n")
	}
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
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
// Return optimized Cartesian geometry (in Bohr) and zmat
func (m Molpro) HandleOutput(filename string) (string, string, error) {
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
			return "", "", ErrFileContainsError
		}
	}
	// ReadLog(logfile)
	// looking for optimized geometry in bohr
	cart, zmat := ReadLog(logfile)
	return cart, zmat, nil
}

// Read the log file and return the Cartesian geometry
// (in Bohr) and the zmat variables
func ReadLog(filename string) (string, string) {
	lines := ReadFile(filename)
	var cart, zmat bytes.Buffer
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "ATOMIC COORDINATES") {
			cart.Reset() // only want the last of these
			for ; !strings.Contains(lines[i], "Bond lengths in Bohr"); i++ {
				if !strings.Contains(lines[i], "ATOM") {
					fields := strings.Fields(strings.TrimSpace(lines[i]))
					fmt.Fprintf(&cart, "%s %s %s %s\n",
						fields[1], fields[3], fields[4], fields[5])
				}
			}
		} else if strings.Contains(lines[i], "Current variables") {
			zmat.Reset()
			i++
			for ; !strings.Contains(lines[i], "***"); i++ {
				fmt.Fprintf(&zmat, "%s\n", lines[i])
			}
		}
	}
	return cart.String(), zmat.String()
}

// Read a Molpro frequency calculation output file
// and return a slice of the harmonic frequencies
func (m Molpro) ReadFreqs(filename string) (freqs []float64) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "Wavenumbers") {
			fields := strings.Fields(line)[2:]
			for _, val := range fields {
				val, _ := strconv.ParseFloat(val, 64)
				freqs = append(freqs, val)
			}
		}
		if strings.Contains(line, "low/zero") {
			break
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(freqs)))
	return
}
