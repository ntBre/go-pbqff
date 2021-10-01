package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

func (g *Gaussian) GetGeometry() string {
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
		fmt.Fprint(w, "opt\n")
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
		if strings.Contains(line, "}") {
			lines = lines[:i+1]
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
	for scanner.Scan() {
		if line := scanner.Text(); strings.Contains(line, "SCF Energy") {
			fields := strings.Fields(line)
			v, _ := strconv.ParseFloat(fields[3], 64)
			return v
		}
	}
	panic("energy not found")
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
			fmt.Printf("%q\n", fields)
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

// BuildPoints uses a file07 file from Intder to construct the
// single-point energy calculations and return an array of jobs to
// run. If write is set to true, write the necessary files. Otherwise
// just return the list of jobs.
func (g *Gaussian) BuildPoints(filename string, atomNames []string,
	target *[]CountFloat, write bool) func() ([]Calc, bool) {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	l := len(atomNames)
	i := 0
	var (
		buf  strings.Builder
		geom int
	)
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
	calcs := make([]Calc, 0)
	for li, line := range lines {
		if !strings.Contains(line, "#") {
			ind := i % l
			if (ind == 0 && i > 0) || li == len(lines)-1 {
				// last line needs to write first
				if li == len(lines)-1 {
					fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
				}
				g.Geom = toAngstrom(buf.String())
				basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
				fname := basename + ".inp"
				if write {
					g.WriteInput(fname, none)
				}
				for len(*target) <= geom {
					*target = append(*target, CountFloat{Count: 1})
				}
				calcs = append(calcs, Calc{
					Name:  basename,
					Scale: 1.0,
					Targets: []Target{
						{
							Coeff: 1,
							Slice: target,
							Index: geom,
						},
					},
				})
				geom++
				buf.Reset()
			}
			fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
			i++
		}
	}
	var (
		start int
		pf    int
		count int
		end   int
	)
	cs := Conf.Int(ChunkSize)
	if start+cs > len(calcs) {
		end = len(calcs)
	} else {
		end = start + cs
	}
	// returns a list of calcs and whether or not it should be
	// called again
	return func() ([]Calc, bool) {
		defer func() {
			pf++
			count++
			start += cs
			if end+cs > len(calcs) {
				end = len(calcs)
			} else {
				end += cs
			}
		}()
		if end == len(calcs) {
			return Push(filepath.Join(dir, "inp"), pf, count, calcs[start:end]), false
		}
		return Push(filepath.Join(dir, "inp"), pf, count, calcs[start:end]), true
	}
}

// Derivative is a helper for calling Make(2|3|4)D in the same way
func (g *Gaussian) Derivative(dir string, names []string,
	coords []float64, i, j, k, l int) (calcs []Calc) {
	var (
		protos []ProtoCalc
		target *[]CountFloat
		ndims  int
	)
	ncoords := len(coords)
	switch {
	case k == 0 && l == 0:
		protos = Make2D(i, j)
		target = &fc2
		ndims = 2
	case l == 0:
		protos = Make3D(i, j, k)
		target = &fc3
		ndims = 3
	default:
		protos = Make4D(i, j, k, l)
		target = &fc4
		ndims = 4
	}
	for _, p := range protos {
		coords := Step(coords, p.Steps...)
		g.Geom = ZipXYZ(names, coords) + "}\n"
		temp := Calc{
			Name:  filepath.Join(dir, p.Name),
			Scale: p.Scale,
		}
		for _, v := range Index(ncoords, false, p.Index...) {
			for len(*target) <= v {
				*target = append(*target, CountFloat{Val: 0, Count: 0})
			}
			if !(*target)[v].Loaded {
				(*target)[v].Count = len(protos)
			}
			temp.Targets = append(temp.Targets,
				Target{Coeff: p.Coeff, Slice: target, Index: v})
		}
		if len(p.Steps) == 2 && ndims == 2 {
			for _, v := range E2dIndex(ncoords, p.Steps...) {
				// also have to append to e2d, but count is always 1 there
				for len(e2d) <= v {
					e2d = append(e2d, CountFloat{Val: 0, Count: 1})
				}
				// if it was loaded, the count is
				// already 0 from the checkpoint
				if !e2d[v].Loaded {
					e2d[v].Count = 1
				}
				temp.Targets = append(temp.Targets,
					Target{Coeff: 1, Slice: &e2d, Index: v})
			}
		} else if len(p.Steps) == 2 && ndims == 4 {
			// either take fourth derivative from finished
			// e2d point or promise a source for later
			if id := E2dIndex(ncoords, p.Steps...)[0]; len(e2d) > id &&
				e2d[id].Val != 0 {
				temp.Result = e2d[id].Val
			} else {
				temp.Src = &Source{&e2d, id}
			}
			temp.noRun = true
		}
		// if target was loaded, remove it from list of targets
		// then only submit if len(Targets) > 0
		for t := 0; t < len(temp.Targets); {
			targ := temp.Targets[t]
			if (*targ.Slice)[targ.Index].Loaded {
				temp.Targets = append(temp.Targets[:t], temp.Targets[t+1:]...)
			} else {
				t++
			}
		}
		if len(temp.Targets) > 0 {
			fname := filepath.Join(dir, p.Name+".inp")
			if strings.Contains(p.Name, "E0") {
				temp.noRun = true
			}
			if !temp.noRun {
				g.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}

// BuildCartPoints constructs the calculations needed to run a
// Cartesian quartic force field
func (g *Gaussian) BuildCartPoints(dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(g.Dir, dir)
	ncoords := len(coords)
	var (
		start int
		pf    int
		count int
	)
	cs := Conf.Int(ChunkSize)
	jnit, knit, lnit := 1, 0, 0
	i, j, k, l := 1, jnit, knit, lnit
	// returns a list of calcs and whether or not it should be
	// called again
	return func() ([]Calc, bool) {
		defer func() {
			pf++
			count++
			start += cs
		}()
		calcs := make([]Calc, 0)
		for ; i <= ncoords; i++ {
			for j = jnit; j <= i; j++ {
				for k = knit; k <= j; k++ {
					for l = lnit; l <= k; l++ {
						calcs = append(calcs,
							g.Derivative(dir, names, coords, i, j, k, l)...,
						)
						if len(calcs) >= Conf.Int(ChunkSize) {
							jnit, knit, lnit = j, k, l+1
							return Push(dir, pf, count, calcs), true
						}
					}
					lnit = 0
				}
				knit = 0
			}
			jnit = 1
		}
		return Push(dir, pf, count, calcs), false
	}
}

// GradDerivative is the Derivative analog for Gradients
func (g *Gaussian) GradDerivative(dir string, names []string, coords []float64,
	i, j, k int) (calcs []Calc) {
	ncoords := len(coords)
	var (
		protos []ProtoCalc
		dimmax int
		ndims  int
		target *[]CountFloat
	)
	switch {
	case j == 0 && k == 0:
		// gradient second derivatives are just first derivatives and so on
		protos = Make1D(i)
		dimmax = ncoords
		ndims = 1
		target = &fc2
	case k == 0:
		// except E0 needs to be G(ref geom) == 0, handled this in Drain
		protos = Make2D(i, j)
		dimmax = j
		ndims = 2
		target = &fc3
	default:
		protos = Make3D(i, j, k)
		dimmax = k
		ndims = 3
		target = &fc4
	}
	for _, p := range protos {
		coords := Step(coords, p.Steps...)
		g.Geom = ZipXYZ(names, coords) + "}\n"
		temp := Calc{Name: filepath.Join(dir, p.Name), Scale: p.Scale}
		var index int
		for g := 1; g <= dimmax; g++ {
			switch ndims {
			case 1:
				index = Index(ncoords, true, i, g)[0]
			case 2:
				index = Index(ncoords, false, i, j, g)[0]
			case 3:
				index = Index(ncoords, false, i, j, k, g)[0]
			}
			temp.Targets = append(temp.Targets, Target{
				Coeff: p.Coeff,
				Slice: target,
				Index: index,
			})
			for len(*target) <= index {
				*target = append(*target, CountFloat{})
			}
			// every time this index is added as a target,
			// increment its count
			if !(*target)[index].Loaded {
				(*target)[index].Count++
			}
		}
		// if target was loaded, remove it from list of targets
		// then only submit if len(Targets) > 0
		for t := 0; t < len(temp.Targets); {
			targ := temp.Targets[t]
			if (*targ.Slice)[targ.Index].Loaded {
				temp.Targets = append(temp.Targets[:t],
					temp.Targets[t+1:]...)
			} else {
				t++
			}
		}
		if len(temp.Targets) > 0 {
			fname := filepath.Join(dir, p.Name+".inp")
			if strings.Contains(p.Name, "E0") {
				temp.noRun = true
			}
			if !temp.noRun {
				g.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}

// BuildGradPoints constructs the calculations needed to run a
// Cartesian quartic force field using gradients
func (g *Gaussian) BuildGradPoints(dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(g.Dir, dir)
	ncoords := len(coords)
	var (
		start int
		pf    int
		count int
	)
	cs := Conf.Int(ChunkSize)
	jnit, knit := 0, 0
	i, j, k := 1, jnit, knit
	// returns a list of calcs and whether or not it should be
	// called again
	return func() ([]Calc, bool) {
		defer func() {
			pf++
			count++
			start += cs
		}()
		calcs := make([]Calc, 0)
		for ; i <= ncoords; i++ {
			for j = jnit; j <= i; j++ {
				for k = knit; k <= j; k++ {
					calcs = append(calcs,
						g.GradDerivative(dir, names, coords, i, j, k)...,
					)
					if len(calcs) >= Conf.Int(ChunkSize) {
						jnit, knit = j, k+1
						return Push(dir, pf, count, calcs), true
					}
				}
				knit = 0
			}
			jnit = 0
		}
		return Push(dir, pf, count, calcs), false
	}
}

// Run runs a single Gaussian calculation. The type of calculation is
// determined by proc. opt calls for a geometry optimization, freq
// calls for a harmonic frequency calculation, and none calls for a
// single point
func (g *Gaussian) Run(proc Procedure) (E0 float64) {
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
	WritePBS(pbsfile,
		&Job{
			Name: fmt.Sprintf("%s-%s",
				MakeName(Conf.Str(Geometry)), proc),
			Filename: infile,
			NumCPUs:  Conf.Int(NumCPUs),
			PBSMem:   Conf.Int(PBSMem),
		}, pbsMaple)
	jobid := Submit(pbsfile)
	jobMap := make(map[string]bool)
	jobMap[jobid] = false
	// only wait for opt and ref to run
	for proc != freq && err != nil {
		E0, _, _, err = g.ReadOut(outfile)
		Qstat(&jobMap)
		if err == ErrFileNotFound && !jobMap[jobid] {
			fmt.Fprintf(os.Stderr, "resubmitting %s for %v\n",
				pbsfile, err)
			jobid = Submit(pbsfile)
			jobMap[jobid] = false
		}
		time.Sleep(time.Duration(Conf.Int(SleepInt)) * time.Second)
	}
	return
}
