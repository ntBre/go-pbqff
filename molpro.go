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
	lines, err := ReadFile(filename)
	if err != nil {
		return brokenFloat, ErrFileNotFound
	}
	err = ErrEnergyNotFound
	result = brokenFloat
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
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	l := len(atomNames)
	i := 0
	var (
		buf   bytes.Buffer
		geom  int
		count *int
		pf    *int
	)
	count = new(int)
	pf = new(int)
	*count = 0
	*pf = 0
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
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
					end := li == len(lines)-1
					Push(dir+"/inp", pf, count, []string{fname}, []Calc{{Name: basename, Targets: []Target{{1, target, geom}}}}, ch, end)
				} else {
					ch <- Calc{Name: basename, Targets: []Target{{1, target, geom}}}
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

// Index3 returns the index in the third derivative array expected by SPECTRO
// corresponding to x, y, and z
func Index3(x, y, z int) int {
	return x + (y-1)*y/2 + (z-1)*z*(z+1)/6 - 1
}

// Index4 returns the index in the fourth derivative array expected by
// SPECTRO corresponding to x, y, z and w
func Index4(x, y, z, w int) int {
	return x + (y-1)*y/2 + (z-1)*z*(z+1)/6 + (w-1)*w*(w+1)*(w+2)/24 - 1
}

var jobNum int

// HashName returns a hashed filename
func HashName() string {
	// var h maphash.Hash
	// h.SetSeed(maphash.MakeSeed())
	// return "job" + strconv.FormatUint(h.Sum64(), 16)
	defer func() {
		jobNum++
	}()
	return fmt.Sprintf("job.%010d", jobNum)
}

type ProtoCalc struct {
	Coeff float64
	Name  string
	Steps []int
	Index []int
}

// Step adjusts coords by delta in the steps indices
func Step(coords []float64, steps ...int) []float64 {
	var c = make([]float64, len(coords))
	copy(c, coords)
	for _, v := range steps {
		if v < 0 {
			v = -1 * v
			c[v-1] = c[v-1] - delta
		} else {
			c[v-1] += delta
		}
	}
	return c
}

// type ProtoCalc struct {Coeff float64, Name string, Steps []int, Index []int}
// type Calc struct {Name string, Targets []Target}
// type Target struct {Coeff float64, Slice *[]float64, Index int}

// Derivative is a helper for calling Make(2|3|4)D in the same way
func Derivative(prog *Molpro, names []string, coords []float64, target *[]float64, dims ...int) (fnames []string, calcs []Calc) {
	var protos []ProtoCalc
	dir := "pts/inp/"
	switch len(dims) {
	case 2:
		protos = Make2D(dims[0], dims[1])
	case 3:
		protos = Make3D(dims[0], dims[1], dims[2])
	case 4:
		protos = Make4D(dims[0], dims[1], dims[2], dims[3])
	}
	for _, p := range protos {
		coords := Step(coords, p.Steps...)
		prog.Geometry = ZipXYZ(names, coords) + "}\n"
		fname := dir + p.Name + ".inp"
		fnames = append(fnames, fname)
		prog.WriteInput(fname, none)
		// TODO handle multiple targets - need to pass multiple targets in func call => []*[]float64?
		temp := Calc{Name: dir + p.Name}
		for _, v := range Index(len(coords), p.Index...) {
			temp.Targets = append(temp.Targets,
				Target{Coeff: p.Coeff, Slice: target, Index: v})
		}
		calcs = append(calcs, temp)
	}
	return
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
		fmt.Fprintf(&buf, "%s %.10f %.10f %.10f\n", names[i], coords[3*i], coords[3*i+1], coords[3*i+2])
	}
	return buf.String()
}

// Index returns the 1-dimensional array index of force constants in
// 2,3,4-D arrays
func Index(ncoords int, id ...int) []int {
	sort.Ints(id)
	switch len(id) {
	case 2:
		if id[0] == id[1] {
			return []int{ncoords*(id[0]-1) + id[1] - 1}
		} else {
			return []int{ncoords*(id[0]-1) + id[1] - 1, ncoords*(id[1]-1) + id[0] - 1}
		}
	case 3:
		return []int{Index3(id[0], id[1], id[2])}
	case 4:
		return []int{Index4(id[0], id[1], id[2], id[3])}
	}
	panic("wrong number of indices in call to Index")
}

// Push sends calculations to the queue
func Push(dir string, pf, count *int, files []string, calcs []Calc, ch chan Calc, end bool) {
	subfile := fmt.Sprintf("%s/main%d.pbs", dir, pf)
	cmdfile := fmt.Sprintf("%s/commands%d.txt", dir, pf)
	for f := range calcs {
		ch <- calcs[f]
		submitted++
		if calcs[f].Name != "E0" {
			AddCommand(cmdfile, files[f])
			if *count == chunkSize || (f == len(files)-1 && end) {
				WritePBS(subfile, &Job{"pts", cmdfile, 35}, ptsMaple)
				jobid := Submit(subfile)
				ptsJobs = append(ptsJobs, jobid)
				*count = 0
				*pf++
				subfile = fmt.Sprintf("%s/main%d.pbs", dir, pf)
				cmdfile = fmt.Sprintf("%s/commands%d.txt", dir, pf)
			}
			*count++
		}
	}
}

// BuildCartPoints constructs the calculations needed to run a
// Cartesian quartic force field
func (mp *Molpro) BuildCartPoints(names []string, coords []float64, fc2, fc3, fc4 *[]float64, ch chan Calc) {
	var (
		count *int
		pf    *int
		end   bool
	)
	count = new(int)
	pf = new(int)
	*count = 0
	*pf = 0
	dir := "pts/inp"
	ncoords := len(coords)
	for i := 1; i <= ncoords; i++ {
		for j := 1; j <= i; j++ {
			files, calcs := Derivative(mp, names, coords, fc2, i, j)
			end = i == ncoords && j == i && nDerivative == 2
			Push(dir, pf, count, files, calcs, ch, end)
			if nDerivative > 2 {
				for k := 1; k <= j; k++ {
					files, calcs := Derivative(mp, names, coords, fc3, i, j, k)
					end = i == ncoords && j == i && k == j && nDerivative == 3
					Push(dir, pf, count, files, calcs, ch, end)
					if nDerivative > 3 {
						for l := 1; l <= k; l++ {
							files, calcs := Derivative(mp, names, coords, fc4, i, j, k, l)
							end = i == ncoords && j == i && k == j && l == k && nDerivative == 4
							Push(dir, pf, count, files, calcs, ch, end)
						}
					}
				}
			}
		}
	}
	close(ch)
	return
}
