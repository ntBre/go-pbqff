package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	Dir      string
	Head     string
	Geometry string
	Tail     string
	Opt      string
	Extra    string
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
	buf.WriteString(m.Geometry + "\n")
	buf.WriteString(m.Tail)
	switch p {
	case opt:
		buf.WriteString(m.Opt)
	case freq:
		buf.WriteString("{frequencies}\n")
	}
	buf.WriteString(m.Extra)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
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
	m.Geometry = strings.Join(out, "\n")
	return
}

// FormatCart formats a Cartesian geometry for use in Molpro input and
// places it in the Geometry field of m
func (m *Molpro) FormatCart(geom string) (err error) {
	m.Geometry = geom + "\n}\n"
	return
}

// UpdateZmat updates an old zmat with new parameters
func (m *Molpro) UpdateZmat(new string) {
	old := m.Geometry
	lines := strings.Split(old, "\n")
	for i, line := range lines {
		if strings.Contains(line, "}") {
			lines = lines[:i+1]
			break
		}
	}
	updated := strings.Join(lines, "\n")
	m.Geometry = updated + "\n" + new
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
		case Conf.RE(EnergyLine).MatchString(line) &&
			!strings.Contains(line, "gthresh") &&
			!strings.Contains(line, "hf"):
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
func (m Molpro) HandleOutput(filename string) (string, string, error) {
	outfile := filename + ".out"
	logfile := filename + ".log"
	lines, err := ReadFile(outfile)
	if err != nil {
		panic(err)
	}
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
		panic(err)
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

// AugmentHead augments the header of a molpro input file
// with a specification of the geometry type and units
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

// BuildPoints uses a file07 file from Intder to construct the
// single-point energy calculations and return an array of jobs to
// run. If write is set to true, write the necessary files. Otherwise
// just return the list of jobs.
func (m *Molpro) BuildPoints(filename string, atomNames []string, target *[]CountFloat, write bool) func() ([]Calc, bool) {
	// TODO I'd like a scanner here but not straightforward
	// because it's nice to know that we're on the last line
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
	m.AugmentHead()
	calcs := make([]Calc, 0)
	// read file07, assemble list of calcs, and write molpro files
	for li, line := range lines {
		if !strings.Contains(line, "#") {
			ind := i % l
			if (ind == 0 && i > 0) || li == len(lines)-1 {
				// last line needs to write first
				if li == len(lines)-1 {
					fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
				}
				m.Geometry = fmt.Sprint(buf.String(), "}\n")
				basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
				fname := basename + ".inp"
				if write {
					m.WriteInput(fname, none)
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

// HashName returns a hashed filename. Well it used to, but now it
// returns JobNum and increments it
func HashName() string {
	defer func() {
		Global.JobNum++
	}()
	return fmt.Sprintf("job.%010d", Global.JobNum)
}

// ProtoCalc is a precursor to a Calc with information for setting up
// the Calc itself
type ProtoCalc struct {
	Coeff float64
	Name  string
	Steps []int
	Index []int
	Scale float64
}

// Step adjusts coords by delta in the steps indices
func Step(coords []float64, steps ...int) []float64 {
	var c = make([]float64, len(coords))
	copy(c, coords)
	for _, v := range steps {
		if v < 0 {
			v = -1 * v
			c[v-1] = c[v-1] - Conf.FlSlice(Deltas)[v-1]
		} else {
			c[v-1] += Conf.FlSlice(Deltas)[v-1]
		}
	}
	return c
}

// type ProtoCalc struct {Coeff float64, Name string, Steps []int, Index []int}
// type Calc struct {Name string, Targets []Target}
// type Target struct {Coeff float64, Slice *[]float64, Index int}

// Derivative is a helper for calling Make(2|3|4)D in the same way
func (m *Molpro) Derivative(dir string, names []string,
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
		m.Geometry = ZipXYZ(names, coords) + "}\n"
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
				m.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}

// E2dIndex converts n to an index in E2d
// water example:
// options for steps:
// 1 2 3 4 5 6 7 8 9 -1 -2 -3 -4 -5 -6 -7 -8 -9
// indices:
// 0 1 2 3 4 5 6 7 8  9 10 11 12 13 14 15 16 17
// grid is then 18x18 = 2ncoords x 2ncoords
func E2dIndex(ncoords int, ns ...int) []int {
	out := make([]int, 0)
	for _, n := range ns {
		if n < 0 {
			out = append(out, IntAbs(n)+ncoords)
		} else {
			out = append(out, n)
		}
	}
	return Index(2*ncoords, false, out...)
}

// Index returns the 1-dimensional array index of force constants in
// 2,3,4-D arrays
func Index(ncoords int, nosort bool, id ...int) []int {
	if !nosort {
		sort.Ints(id)
	}
	switch len(id) {
	case 2:
		if id[0] == id[1] {
			return []int{ncoords*(id[0]-1) + id[1] - 1}
		}
		return []int{
			ncoords*(id[0]-1) + id[1] - 1,
			ncoords*(id[1]-1) + id[0] - 1,
		}
	case 3:
		return []int{
			id[0] + (id[1]-1)*id[1]/2 +
				(id[2]-1)*id[2]*(id[2]+1)/6 - 1,
		}
	case 4:
		return []int{
			id[0] + (id[1]-1)*id[1]/2 +
				(id[2]-1)*id[2]*(id[2]+1)/6 +
				(id[3]-1)*id[3]*(id[3]+1)*(id[3]+2)/24 - 1,
		}
	}
	panic("wrong number of indices in call to Index")
}

// IntAbs returns the absolute value of n
func IntAbs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

// ZipXYZ puts slices of atom names and Cartesian coordinates together
// into a single string
func ZipXYZ(names []string, coords []float64) string {
	var buf bytes.Buffer
	if len(names) != len(coords)/3 {
		panic("ZipXYZ: dimension mismatch on names and coords")
	} else if len(coords)%3 != 0 {
		panic("ZipXYZ: coords not divisible by 3")
	}
	for i := range names {
		fmt.Fprintf(&buf, "%s %.10f %.10f %.10f\n",
			names[i], coords[3*i], coords[3*i+1], coords[3*i+2])
	}
	return buf.String()
}

// SelectNode returns a node and queue from the Global node list
func SelectNode() (node, queue string) {
	queue = Conf.Str(Queue)
	// regenerate empty node list if empty
	if len(Global.Nodes) == 0 {
		Global.Nodes = PBSnodes()
	}
	if len(Global.Nodes) > 0 {
		tmp := strings.Split(Global.Nodes[0], ":")
		if queue == "" || tmp[0] == queue {
			node = tmp[1]
			queue = tmp[0]
			Global.Nodes = Global.Nodes[1:]
		}
	}
	return
}

// Push sends calculations to the queue
func Push(dir string, pf, count int, calcs []Calc) []Calc {
	subfile := fmt.Sprintf("%s/main%d.pbs", dir, pf)
	cmdfile := fmt.Sprintf("%s/commands%d", dir, pf)
	f, err := os.Create(cmdfile)
	defer f.Close()
	if err != nil {
		msg := fmt.Sprintf("Cannot open commands file: %s with %v\n", cmdfile, err)
		panic(msg)
	}
	for c := range calcs {
		calcs[c].CmdFile = cmdfile
		calcs[c].ChunkNum = pf
		if !calcs[c].noRun {
			submitted++
			fmt.Fprintf(f, "%s %s --no-xml-output\n",
				mapleCmd, calcs[c].Name+".inp")
		} else {
			count++
		}
	}
	node, queue := SelectNode()
	// This should be using the PBS from Config
	WritePBS(subfile,
		&Job{
			Name:     MakeName(Conf.Str(Geometry)) + "pts",
			Filename: cmdfile,
			Signal:   35,
			Host:     node,
			Queue:    queue,
			NumJobs:  Conf.Int(NumJobs),
		}, ptsMaple)
	jobid := Submit(subfile)
	if *debug {
		fmt.Println(subfile, jobid)
	}
	ptsJobs = append(ptsJobs, jobid)
	paraJobs = append(paraJobs, jobid)
	paraCount[jobid] = Conf.Int(ChunkSize)
	count = 1
	pf++
	// if end reached with no calcs, which can happen on continue
	// from checkpoints
	return calcs
}

// BuildCartPoints constructs the calculations needed to run a
// Cartesian quartic force field
func (m *Molpro) BuildCartPoints(dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(m.Dir, dir)
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
							m.Derivative(dir, names, coords, i, j, k, l)...,
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
func (m *Molpro) GradDerivative(dir string, names []string, coords []float64,
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
		m.Geometry = ZipXYZ(names, coords) + "}\n"
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
				m.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}

// BuildGradPoints constructs the calculations needed to run a
// Cartesian quartic force field using gradients
func (m *Molpro) BuildGradPoints(dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(m.Dir, dir)
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
						m.GradDerivative(dir, names, coords, i, j, k)...,
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
