package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
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

// Procedure defines a type of molpro calculation. This includes
// optimization (opt) and frequencies (freq).
type Procedure int

// Molpro holds the data for writing molpro input files
type Molpro struct {
	Head     string
	Geometry string
	Tail     string
	Opt      string
	Extra    string
}

// LoadMolpro loads a template molpro input file
func LoadMolpro(filename string) (*Molpro, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
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
			mp.Opt = line + "\n"
			continue
		}
		buf.WriteString(line + "\n")
		if strings.Contains(line, "geometry=") {
			mp.Head = buf.String()
			buf.Reset()
		}
	}
	mp.Extra = buf.String()
	return &mp, nil
}

// WriteInput writes a Molpro input file
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
	buf.WriteString(m.Extra)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// FormatZmat formats a z-matrix for use in Molpro input
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

// ReadOut reads a molpro output file and returns the resulting energy
// and an error describing the status of the output
func (m Molpro) ReadOut(filename string) (result float64, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if _, err = os.Stat(filename); os.IsNotExist(err) {
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
		if energyLine.MatchString(line) &&
			!strings.Contains(line, "gthresh") &&
			!strings.Contains(line, "hf") {
			split := strings.Fields(line)
			for i := range split {
				if strings.Contains(split[i], "=") {
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
	return result, err
}

// HandleOutput reads .out and .log files for filename, assumes no extension
// and returns the optimized Cartesian geometry (in Bohr) and the zmat variables
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

// ReadLog reads a molpro log file and returns the optimized Cartesian geometry
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

// ReadFreqs reads a Molpro frequency calculation output file
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

// AugmentHead augments the header of a molpro input file
// with a specification of the geometry type and units
func (mp *Molpro) AugmentHead() {
	lines := strings.Split(mp.Head, "\n")
	add := "geomtyp=xyz\nbohr"
	newlines := make([]string, 0)
	for i, line := range lines {
		if strings.Contains(line, "geometry") &&
			!strings.Contains(lines[i-1], "bohr") {
			newlines = append(newlines, lines[:i]...)
			newlines = append(newlines, add)
			newlines = append(newlines, lines[i:]...)
			mp.Head = strings.Join(newlines, "\n")
			return
		}
	}
}

// BuildPoints uses ./pts/file07 to construct the single-point
// energy calculations and return an array of jobs to run. If write
// is set to true, write the necessary files. Otherwise just return the list
// of jobs.
func (mp *Molpro) BuildPoints(filename string, atomNames []string, target *[]float64, ch chan Calc, write bool) {
	lines := ReadFile(filename)
	l := len(atomNames)
	i := 0
	var (
		buf     bytes.Buffer
		cmdfile string
	)
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
	geom := 0
	count := 0
	pf := 0
	pbs = ptsMaple
	mp.AugmentHead()
	for li, line := range lines {
		if !strings.Contains(line, "#") {
			ind := i % l
			if (ind == 0 && i > 0) || li == len(lines)-1 {
				// last line needs to write first
				if li == len(lines)-1 {
					fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
				}
				mp.Geometry = fmt.Sprint(buf.String(), "}\n")
				basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
				fname := basename + ".inp"
				if write {
					// write the molpro input file and add it to the list of commands
					mp.WriteInput(fname, none)
					cmdfile = fmt.Sprintf("%s/inp/commands%d.txt", dir, pf)
					AddCommand(cmdfile, fname)
					ch <- Calc{Name: basename, Target: target, Index: geom}
					submitted++
					if count == chunkSize || li == len(lines)-1 {
						subfile := fmt.Sprintf("%s/inp/main%d.pbs", dir, pf)
						WritePBS(subfile, &Job{"pts", cmdfile, 35})
						Submit(subfile)
						count = 0
						pf++
					}
					count++
				} else {
					ch <- Calc{Name: basename, Target: target, Index: geom}
				}
				geom++
				buf.Reset()
			}
			fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
			i++
		}
	}
	close(ch)
	return
}

// BuildCartPoints constructs the calculations needed to run a
// Cartesian quartic force field
func (mp *Molpro) BuildCartPoints(names []string, coords []float64, E0 float64, fc2, fc3, fc4 *[]float64, ch chan Calc) {
	// l := len(names)
	// i := 0
	// var (
	// 	buf     bytes.Buffer
	// 	cmdfile string
	// )
	// dir := "pts/inp/"
	// name := strings.Join(names, "")
	// geom := 0
	// count := 0
	// pf := 0
	pbs = ptsMaple
	mp.AugmentHead()
	// TODO insert go-cart code here
	// for li, line := range lines {
	// 	if !strings.Contains(line, "#") {
	// 		ind := i % l
	// 		if (ind == 0 && i > 0) || li == len(lines)-1 {
	// 			// last line needs to write first
	// 			if li == len(lines)-1 {
	// 				fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
	// 			}
	// 			mp.Geometry = fmt.Sprint(buf.String(), "}\n")
	// 			basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
	// 			fname := basename + ".inp"
	// 			if write {
	// 				// write the molpro input file and add it to the list of commands
	// 				mp.WriteInput(fname, none)
	// 				cmdfile = fmt.Sprintf("%s/inp/commands%d.txt", dir, pf)
	// 				AddCommand(cmdfile, fname)
	// 				ch <- Calc{Name: basename, Target: target, Index: geom}
	// 				submitted++
	// 				if count == chunkSize || li == len(lines)-1 {
	// 					subfile := fmt.Sprintf("%s/inp/main%d.pbs", dir, pf)
	// 					WritePBS(subfile, &Job{"pts", cmdfile, 35})
	// 					Submit(subfile)
	// 					count = 0
	// 					pf++
	// 				}
	// 				count++
	// 			} else {
	// 				ch <- Calc{Name: basename, Target: target, Index: geom}
	// 			}
	// 			geom++
	// 			buf.Reset()
	// 		}
	// 		fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
	// 		i++
	// 	}
	// }
	close(ch)
	return
}
