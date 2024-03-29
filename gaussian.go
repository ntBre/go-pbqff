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

// Load a template Gaussian input file
func (g *Gaussian) Load(filename string) error {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	var (
		str  strings.Builder
		line string
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
	return nil
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

// readChk reads a Gaussian fchk file and returns the total energy
func readChk(filename string) (energy float64, gradient []float64) {
	f, err := os.Open(filename)
	defer f.Close()
	for err != nil {
		fmt.Printf("trying to open %s again\n", filename)
		time.Sleep(1 * time.Second)
		f, err = os.Open(filename)
	}
	scanner := bufio.NewScanner(f)
	var (
		line   string
		fields []string
		ingrad bool
	)
	for {
		for scanner.Scan() {
			line = scanner.Text()
			switch {
			case strings.Contains(line, "Total Energy"):
				fields = strings.Fields(line)
				energy, _ = strconv.ParseFloat(fields[3], 64)
				if !GRAD {
					return
				}
			case strings.Contains(line, "Cartesian Gradient"):
				ingrad = true
			case ingrad && strings.Contains(line, "Nonadiabatic"):
				// g16 appears to start the gradient
				// with the x1y1 coordinate and end
				// with x1x1, so move that to the
				// front
				lg := len(gradient)
				gradient = append(gradient[lg-1:], gradient[:lg-1]...)
				return
			case ingrad:
				fields = strings.Fields(line)
				for _, f := range fields {
					v, _ := strconv.ParseFloat(f, 64)
					gradient = append(gradient, v)
				}
			}
		}
		fmt.Printf("can't find gradient in %s, retrying\n", filename)
		f.Seek(0, io.SeekStart)
		scanner = bufio.NewScanner(f)
		time.Sleep(1 * time.Second)
	}
}

// ReadOut reads a Gaussian output file and returns the resulting
// energy, the wall time taken in seconds, the gradient vector, and an
// error describing the status of the output
func (g *Gaussian) ReadOut(filename string) (energy, time float64,
	grad []float64, err error) {
	// TODO signal error on problem reading gradient
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		err = ErrGaussNotFound
		return
	}
	scanner := bufio.NewScanner(f)
	err = ErrEnergyNotFound
	var (
		i int
	)
	var line string
	for i = 0; scanner.Scan(); i++ {
		line = scanner.Text()
		switch {
		// kill switch
		case i == 0 && strings.Contains(strings.ToUpper(line), "PANIC"):
			panic("panic requested in output file")
		case i == 0 && strings.Contains(strings.ToUpper(line), "ERROR"):
			return energy, time, grad, ErrFileContainsError
		case strings.Contains(strings.ToLower(line), "error") &&
			GaussErrorLine.MatchString(line):
			return energy, time, grad, ErrFileContainsError
		case strings.Contains(line, "Normal termination of Gaussian"):
			basename := TrimExt(filename)
			energy, grad = readChk(basename + ".fchk")
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
		return energy, time, grad, ErrBlankOutput
	}
	return energy, time, grad, err
}

// HandleOutput extracts the optimized geometry in Cartesian (bohr)
// and Z-matrix (angstrom) form from a Gaussian output file. It also
// checks the output file for warnings and errors.
func (g *Gaussian) HandleOutput(filename string) (string, string, error) {
	f, err := os.Open(filename + ".out")
	defer f.Close()
	if err != nil {
		panic(err)
	}
	warn := regexp.MustCompile(`(?i)warning`)
	var (
		vars bool
		zmat strings.Builder
		skip int
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
		}
	}
	return cartChk(filename), zmat.String(), nil
}

// cartChk extracts a more precise Cartesian geometry from a formatted
// Gaussian checkpoint file
func cartChk(filename string) string {
	f, err := os.Open(filename + ".fchk")
	defer f.Close()
	for err != nil {
		fmt.Printf("trying to open %s again\n", filename)
		time.Sleep(1 * time.Second)
		f, err = os.Open(filename)
	}
	scanner := bufio.NewScanner(f)
	var (
		inatom = false
		incart = false
		atoms  []string
		coords []float64
	)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.Contains(line, "Atomic numbers"):
			inatom = true
		case inatom:
			fields := strings.Fields(line)
			for _, f := range fields {
				sym, ok := ATOMIC_NUMBERS[f]
				if !ok {
					panic("atomic number " + f +
						" not found in map")
				}
				atoms = append(atoms, sym)
			}
			inatom = false
		case strings.Contains(line, "Current cartesian coordinates"):
			incart = true
		case incart && strings.Contains(line, "Number of symbols"):
			incart = false
		case incart:
			fields := strings.Fields(line)
			for _, f := range fields {
				v, _ := strconv.ParseFloat(f, 64)
				coords = append(coords, v)
			}
		}
	}
	// TODO fix this race condition if end of output file reached
	// before formcheck finishes running - see outer loop in
	// readchk for ideas
	if atoms == nil || coords == nil || len(atoms) == 0 || len(coords) == 0 {
		panic("atoms or coords not found in fchk => race condition hit")
	}
	return ZipXYZ(atoms, coords)
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
		ret[i] *= ANGBOHR
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
			new[i] = fmt.Sprintf("%s %.10f %.10f %.10f",
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
	outfile := filepath.Join(dir, name+".out")
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
			Jobs:     []string{infile},
		})
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
