package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	opt Procedure = iota
	freq
	none
)

// Procedure defines a type of molpro calculation. This includes
// optimization (opt), harmonic frequency (freq), and single point
// (none)
type Procedure int

func (p Procedure) String() string {
	return []string{"opt", "freq", "ref"}[p]
}

// Molpro holds the data for writing molpro input files
type Molpro struct {
	Dir   string
	Head  string
	Geom  string
	Tail  string
	Opt   string
	Extra string
}

func (m *Molpro) SetDir(dir string) {
	m.Dir = dir
}
func (m *Molpro) GetDir() string {
	return m.Dir
}

func (m *Molpro) SetGeom(geom string) {
	m.Geom = geom
}

func (m *Molpro) GetGeom() string {
	return m.Geom
}

// LoadMolpro loads a template molpro input file
func LoadMolpro(filename string) (*Molpro, error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}
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
	buf.WriteString(m.Geom + "\n")
	buf.WriteString(m.Tail)
	switch p {
	case opt:
		buf.WriteString(m.Opt)
	case freq:
		buf.WriteString("{frequencies}\n")
	}
	buf.WriteString(m.Extra)
	os.WriteFile(filename, buf.Bytes(), 0755)
}

// FormatZmat formats a z-matrix for use in Molpro input and places it
// in the Geometry field of m
func (m *Molpro) FormatZmat(geom string) (err error) {
	var out []string
	err = errors.New("improper z-matrix")
	split := strings.Split(geom, "\n")
	for i, line := range split {
		if strings.Contains(line, "=") {
			out = append(append(append(out, split[:i]...), "}"), split[i:]...)
			err = nil
			break
		}
	}
	m.Geom = strings.Join(out, "\n")
	return
}

// FormatCart formats a Cartesian geometry for use in Molpro input and
// places it in the Geometry field of m
func (m *Molpro) FormatCart(geom string) (err error) {
	m.Geom = geom + "\n}\n"
	return
}

// UpdateZmat updates an old zmat with new parameters
func (m *Molpro) UpdateZmat(new string) {
	old := m.Geom
	lines := strings.Split(old, "\n")
	for i, line := range lines {
		if strings.Contains(line, "}") {
			lines = lines[:i+1]
			break
		}
	}
	updated := strings.Join(lines, "\n")
	m.Geom = updated + "\n" + new
}

// ReadOut reads a molpro output file and returns the resulting
// energy, the real time taken, the gradient vector, and an error
// describing the status of the output
// TODO signal error on problem reading gradient
func (m Molpro) ReadOut(filename string) (result, time float64, grad []float64, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return brokenFloat, 0, grad, ErrFileNotFound
	}
	scanner := bufio.NewScanner(f)
	err = ErrEnergyNotFound
	time = 0
	result = brokenFloat
	var (
		i                   int
		gradx, grady, gradz []string
	)
	processGrad := func(line string) []string {
		coords := strings.Fields(line)
		coords = coords[3 : len(coords)-1] // trim front and back
		coords[len(coords)-1] = strings.TrimRight(coords[len(coords)-1], "]")
		return coords
	}
	var line string
	for i = 0; scanner.Scan(); i++ {
		line = scanner.Text()
		switch {
		// kill switch
		case i == 0 && strings.Contains(strings.ToUpper(line), "PANIC"):
			panic("panic requested in output file")
		case i == 0 && strings.Contains(strings.ToUpper(line), "ERROR"):
			return result, time, grad, ErrFileContainsError
		case strings.Contains(strings.ToLower(line), "error") &&
			ErrorLine.MatchString(line):
			return result, time, grad, ErrFileContainsError
			// since we assume the line contains an '='
			// below, gate the regex match with that
		case strings.Contains(line, "=") &&
			!strings.Contains(line, "gthresh") &&
			!strings.Contains(line, "hf") &&
			Conf.RE(EnergyLine).MatchString(line):
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
		case strings.Contains(line, "REAL TIME"):
			fields := strings.Fields(line)
			timeStr := fields[len(fields)-2]
			time, _ = strconv.ParseFloat(timeStr, 64)
		case strings.Contains(line, "GRADX"):
			gradx = processGrad(line)
		case strings.Contains(line, "GRADY"):
			grady = processGrad(line)
		case strings.Contains(line, "GRADZ"):
			gradz = processGrad(line)
		case strings.Contains(line, molproTerminated) && err != nil:
			err = ErrFinishedButNoEnergy
		}
	}
	if i == 0 {
		return result, time, grad, ErrBlankOutput
	}
	if gradx != nil {
		grad = func(xs, ys, zs []string) []float64 {
			lx := len(xs)
			if !(lx == len(ys) && lx == len(zs)) {
				panic("Gradient dimension mismatch")
			}
			ret := make([]float64, 0, 3*lx)
			for i := range xs {
				x, _ := strconv.ParseFloat(xs[i], 64)
				y, _ := strconv.ParseFloat(ys[i], 64)
				z, _ := strconv.ParseFloat(zs[i], 64)
				ret = append(ret, x, y, z)
			}
			return ret
		}(gradx, grady, gradz)
	}
	return result, time, grad, err
}

// HandleOutput is a wrapper around ReadLog that reads the .out and
// .log files for filename, first checking the .out file for warnings
// and errors before calling ReadLog on the .log file
func (m *Molpro) HandleOutput(filename string) (string, string, error) {
	outfile := filename + ".out"
	logfile := filename + ".log"
	f, err := os.Open(outfile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	warn := regexp.MustCompile(`(?i)warning`)
	error := regexp.MustCompile(`(?i)[^_]error`)
	// notify about warnings or errors in output file
	// apparently warnings are not printed in the log
	scanner := bufio.NewScanner(f)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if warn.MatchString(line) {
			Warn("HandleOutput: warning %q, found in %s",
				line, outfile)
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

// ReadLog reads a molpro log file and returns the optimized Cartesian
// geometry (in Bohr) and the zmat variables
func ReadLog(filename string) (string, string) {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
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
	defer f.Close()
	if err != nil {
		return
	}
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

// AugmentHead augments the header of a Molpro input file with a
// specification of the geometry type and units
func (m *Molpro) AugmentHead() {
	lines := strings.Split(m.Head, "\n")
	add := "geomtyp=xyz\nbohr"
	newlines := make([]string, 0)
	for i, line := range lines {
		if strings.Contains(line, "geometry") &&
			!strings.Contains(lines[i-1], "bohr") {
			newlines = append(newlines, lines[:i]...)
			newlines = append(newlines, add)
			newlines = append(newlines, lines[i:]...)
			m.Head = strings.Join(newlines, "\n")
			return
		}
	}
}

func (m *Molpro) FormatGeom(coords string) string {
	return fmt.Sprint(coords, "}\n")
}

// Run runs a single Molpro calculation. The type of calculation is
// determined by proc. opt calls for a geometry optimization, freq
// calls for a harmonic frequency calculation, and none calls for a
// single point
func (m *Molpro) Run(proc Procedure, q Queue) (E0 float64) {
	var (
		dir  string
		name string
	)
	switch proc {
	case opt:
		dir = "opt"
		name = "opt"
	case freq:
		dir = "freq"
		name = "freq"
	case none:
		dir = "pts/inp"
		name = "ref"
	}
	dir = filepath.Join(m.Dir, dir)
	infile := filepath.Join(dir, name+".inp")
	pbsfile := filepath.Join(dir, name+".pbs")
	outfile := filepath.Join(dir, name+".out")
	E0, _, _, err := m.ReadOut(outfile)
	if *read && err == nil {
		return
	}
	m.WriteInput(infile, proc)
	q.WritePBS(pbsfile,
		&Job{
			Name: fmt.Sprintf("%s-%s",
				MakeName(Conf.Str(Geometry)), proc),
			Filename: infile,
			NumCPUs:  Conf.Int(NumCPUs),
			PBSMem:   Conf.Int(PBSMem),
		}, q.SinglePBS())
	jobid := q.Submit(pbsfile)
	jobMap := make(map[string]bool)
	jobMap[jobid] = false
	// only wait for opt and ref to run
	for proc != freq && err != nil {
		E0, _, _, err = m.ReadOut(outfile)
		q.Stat(&jobMap)
		if err == ErrFileNotFound && !jobMap[jobid] {
			fmt.Fprintf(os.Stderr, "resubmitting %s for %v\n",
				pbsfile, err)
			jobid = q.Submit(pbsfile)
			jobMap[jobid] = false
		}
		time.Sleep(time.Duration(Conf.Int(SleepInt)) * time.Second)
	}
	return
}
