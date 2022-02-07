package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var GaussErrorLine = regexp.MustCompile(`(?i)error termination`)

// copying molpro for now, not sure how many are actually applicable
type Gaussian struct {
	Dir  string
	Head string
	Opt  string
	Body string
	Geom string
	Tail string
}

func (g *Gaussian) String() string {
	var str strings.Builder
	fmt.Fprintf(&str, "Head:\n%s", g.Head)
	fmt.Fprintf(&str, "Opt:\n%s", g.Opt)
	fmt.Fprintf(&str, "Body:\n%s", g.Body)
	fmt.Fprintf(&str, "Geom:\n%s", g.Geom)
	fmt.Fprintf(&str, "Tail:\n%s", g.Tail)
	return str.String()
}

func (g *Gaussian) SetDir(dir string) {
	g.Dir = dir
}

func (g *Gaussian) GetDir() string {
	return g.Dir
}

func (g *Gaussian) SetGeom(geom string) {
	g.Geom = geom
}

func (g *Gaussian) GetGeom() string {
	return g.Geom
}

// LoadGaussian loads a template Gaussian input file
func LoadGaussian(filename string) (*Gaussian, error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	var (
		str  strings.Builder
		line string
		g    Gaussian
		geom bool
	)
	chargeSpin := regexp.MustCompile(`^\s*\d\s+\d\s*$`)
	proc := regexp.MustCompile(`(?i)(opt|freq(=[^ ])*)`)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.Contains(line, "#"):
			g.Head = str.String()
			str.Reset()
			// TODO might actually want to keep this in
			// some cases:
			// remove opt/freq from input line
			g.Opt = proc.ReplaceAllString(line, "") + "\n"
		case chargeSpin.MatchString(line):
			str.WriteString(line + "\n")
			g.Body = str.String()
			str.Reset()
			geom = true
		case geom && line == "":
			geom = false
		case geom:
			// skip the geometry
		default:
			str.WriteString(line + "\n")
		}
	}
	g.Tail = str.String()
	return &g, nil
}

func (g *Gaussian) makeInput(w io.Writer, p Procedure) {
	fmt.Fprintf(w, "%s%s ", g.Head, strings.TrimSpace(g.Opt))
	switch p {
	case opt:
		fmt.Fprint(w, "opt=VeryTight\n")
	case freq:
		fmt.Fprint(w, "freq\n")
	default:
		fmt.Fprint(w, "\n")
	}
	fmt.Fprintf(w, "%s%s\n\n", g.Body, strings.TrimSpace(g.Geom))
	fmt.Fprintf(w, "%s", g.Tail)
}

// WriteInput writes a Gaussian input file
func (g *Gaussian) WriteInput(filename string, p Procedure) {
	basename := TrimExt(filename)
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	// might have to make this a basename, but only on eland T.T
	fmt.Fprintf(f, "%%chk=%s.chk\n", basename)
	g.makeInput(f, p)
}

// FormatZmat formats a z-matrix for use in Gaussian input and places
// it in the Geom field of g
func (g *Gaussian) FormatZmat(geom string) (err error) {
	var out []string
	err = errors.New("improper z-matrix")
	split := strings.Split(geom, "\n")
	unit := regexp.MustCompile(`(?i)\s+(ang|deg)`)
	var (
		i    int
		line string
	)
	for i, line = range split {
		if strings.Contains(line, "=") {
			out = append(append(out, split[:i]...), "")
			err = nil
			break
		}
	}
	// in case there are units in the zmat params, remove them
	for _, line := range split[i:] {
		out = append(out, unit.ReplaceAllString(line, ""))
	}
	out = append(out, "")
	g.Geom = strings.Join(out, "\n")
	return
}

// FormatCart formats a Cartesian geometry for use in Gaussian input
// and places it in the Geometry field of m
func (g *Gaussian) FormatCart(geom string) (err error) {
	g.Geom = geom
	return
}

// UpdateZmat updates an old zmat with new parameters
func (g *Gaussian) UpdateZmat(new string) {
	old := g.Geom
	lines := strings.Split(old, "\n")
	for i, line := range lines {
		if strings.Contains(line, "=") {
			lines = lines[:i]
			break
		}
	}
	updated := strings.Join(lines, "\n")
	g.Geom = updated + "\n" + new
}

// readChk reads a Gaussian fchk file and return the SCF energy
func readChk(filename string) float64 {
	f, err := os.Open(filename)
	defer f.Close()
	for err != nil {
		fmt.Printf("trying to open %s again\n", filename)
		time.Sleep(1 * time.Second)
		f, err = os.Open(filename)
	}
	scanner := bufio.NewScanner(f)
	for {
		for scanner.Scan() {
			if line := scanner.Text(); strings.Contains(line, "SCF Energy") {
				fields := strings.Fields(line)
				v, _ := strconv.ParseFloat(fields[3], 64)
				return v
			}
		}
		fmt.Printf("can't find energy in %s, retrying\n", filename)
		f.Seek(0, io.SeekStart)
		scanner = bufio.NewScanner(f)
		time.Sleep(1 * time.Second)
	}
}

// ReadOut reads a molpro output file and returns the resulting
// energy, the wall time taken in seconds, the gradient vector, and an
// error describing the status of the output
// TODO signal error on problem reading gradient
func (g *Gaussian) ReadOut(filename string) (result, time float64,
	grad []float64, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		err = ErrGaussNotFound
		return
	}
	scanner := bufio.NewScanner(f)
	err = ErrEnergyNotFound
	var (
		i                   int
		gradx, grady, gradz []string
	)
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
			GaussErrorLine.MatchString(line):
			return result, time, grad, ErrFileContainsError
			// since we assume the line contains an '='
			// below, gate the regex match with that
		case strings.Contains(line, "Normal termination of Gaussian"):
			basename := TrimExt(filename)
			result = readChk(basename + ".fchk")
			err = nil
		case strings.Contains(line, "Elapsed time:"):
			// TODO this only pulls the seconds portion of
			// the time
			fields := strings.Fields(line)
			timeStr := fields[8]
			time, _ = strconv.ParseFloat(timeStr, 64)
		}
	}
	if i == 0 {
		return result, time, grad, ErrBlankOutput
	}
	// TODO extract gradients one day - fixes nilness
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

// HandleOutput extracts the optimized geometry in Cartesian (bohr)
// and Z-matrix (angstrom) form from a Gaussian output file. It also
// checks the output file for warnings and errors.
func (g *Gaussian) HandleOutput(filename string) (string, string, error) {
	f, err := os.Open(filename + OutExt)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	warn := regexp.MustCompile(`(?i)warning`)
	var (
		vars       bool
		cart, zmat strings.Builder
		skip       int
		geom       bool
	)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if skip > 0 {
			skip--
			continue
		}
		line := scanner.Text()
		if warn.MatchString(line) &&
			!strings.Contains(line, "This program") {
			Warn("HandleOutput: warning %q, found in %s",
				line, filename)
		}
		if GaussErrorLine.MatchString(line) {
			fmt.Fprintf(os.Stderr,
				"HandleOutput: error %q, found in %s, aborting\n",
				line, filename)
			return "", "", ErrFileContainsError
		}
		switch {
		case strings.Contains(line, "Variables:"):
			vars = true
		case vars && !strings.Contains(line, "=") ||
			strings.Contains(line, "GINC"):
			vars = false
		case vars:
			fmt.Fprintln(&zmat, strings.TrimSpace(line))
		case strings.Contains(line, "Standard orientation"):
			skip += 4
			geom = true
			cart.Reset()
		case geom && strings.Contains(line, "------"):
			geom = false
		case geom:
			fields := strings.Fields(line)[1:]
			coords := make([]float64, 3)
			for i, v := range fields[2:5] {
				coords[i], _ = strconv.ParseFloat(v, 64)
				coords[i] /= angbohr
			}
			// TODO can you get more precision? 6 is
			// pretty low
			fmt.Fprintf(&cart, "%s%10.6f%10.6f%10.6f\n",
				ATOMIC_NUMBERS[fields[0]],
				coords[0], coords[1], coords[2])
		}
	}
	return cart.String(), zmat.String(), nil
}

// ReadFreqs reads a Gaussian harmonic frequency calculation output
// file and return a slice of the harmonic frequencies
func (g Gaussian) ReadFreqs(filename string) (freqs []float64) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(f)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "Frequencies --") {
			fields := strings.Fields(line)[2:]
			for _, val := range fields {
				val, _ := strconv.ParseFloat(val, 64)
				freqs = append(freqs, val)
			}
		}
		if strings.Contains(line, "Thermochemistry") {
			break
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(freqs)))
	return
}

func strBohrToAng(bohrs []string) []float64 {
	ret := make([]float64, len(bohrs))
	for i := range bohrs {
		ret[i], _ = strconv.ParseFloat(bohrs[i], 64)
		ret[i] *= angbohr
	}
	return ret
}

// toAngstrom takes a geometry in bohr like "C 1.0 1.0 1.0\nH 2.0 2.0
// 2.0\n" and returns it converted to angstroms
func toAngstrom(geom string) string {
	lines := strings.Split(geom, "\n")
	new := make([]string, len(lines))
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := strBohrToAng(fields[1:])
			new[i] = fmt.Sprintf("%s %f %f %f",
				fields[0], v[0], v[1], v[2])
		} else {
			new[i] = ""
		}
	}
	return strings.Join(new, "\n")
}

// no-op to meet interface
func (g *Gaussian) AugmentHead() {}

func (g *Gaussian) FormatGeom(coords string) string {
	return toAngstrom(coords)
}

// Run runs a single Gaussian calculation. The type of calculation is
// determined by proc. opt calls for a geometry optimization, freq
// calls for a harmonic frequency calculation, and none calls for a
// single point
func (g *Gaussian) Run(proc Procedure, q Queue) (E0 float64) {
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
	dir = filepath.Join(g.Dir, dir)
	infile := filepath.Join(dir, name+".inp")
	pbsfile := filepath.Join(dir, name+".pbs")
	outfile := filepath.Join(dir, name+OutExt)
	E0, _, _, err := g.ReadOut(outfile)
	if *read && err == nil {
		return
	}
	g.WriteInput(infile, proc)
	q.WritePBS(pbsfile,
		&Job{
			Name: fmt.Sprintf("%s-%s",
				MakeName(Conf.Geometry), proc),
			Filename: infile,
			NumCPUs:  Conf.NumCPUs,
			PBSMem:   Conf.PBSMem,
		}, true)
	jobid := q.Submit(pbsfile)
	jobMap := make(map[string]bool)
	jobMap[jobid] = false
	// only wait for opt and ref to run
	for proc != freq && err != nil {
		E0, _, _, err = g.ReadOut(outfile)
		q.Stat(&jobMap)
		if err == ErrGaussNotFound && !jobMap[jobid] {
			fmt.Fprintf(os.Stderr, "resubmitting %s for %v\n",
				pbsfile, err)
			jobid = q.Submit(pbsfile)
			jobMap[jobid] = false
		}
		time.Sleep(time.Duration(Conf.SleepInt) * time.Second)
	}
	return
}
