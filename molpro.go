package main

import (
	"bufio"
	"bytes"
	"fmt"
	"hash/maphash"
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
					ch <- Calc{Name: basename, Targets: []Target{{1, target, geom}}}
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

// HashName returns a hashed filename
func HashName() string {
	var h maphash.Hash
	h.SetSeed(maphash.MakeSeed())
	return "job" + strconv.FormatUint(h.Sum64(), 16)
}

type ProtoCalc struct {
	Coeff float64
	Name  string
	Steps []int
	Index []int
}

// Make2D makes the Job slices for finite differences second
// derivative force constants
func Make2D(i, j int) []ProtoCalc {
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, i}, []int{i, i}},
			{-2, "E0", []int{}, []int{i, i}},
			{1, HashName(), []int{-i, -i}, []int{i, i}},
		}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, j}, []int{i, j}},
			{-1, HashName(), []int{i, -j}, []int{i, j}},
			{-1, HashName(), []int{-i, j}, []int{i, j}},
			{1, HashName(), []int{-i, -j}, []int{i, j}},
		}
	default:
		panic("No cases matched")
	}
}

// Make3D makes the ProtoCalc slices for finite differences third derivative
// force constants
func Make3D(i, j, k int) []ProtoCalc {
	switch {
	case i == j && i == k:
		// E(+i+i+i) - 3*E(i) + 3*E(-i) -E(-i-i-i) / (2d)^3
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i}, []int{i, i, i}},
			{-3, HashName(), []int{i}, []int{i, i, i}},
			{3, HashName(), []int{-i}, []int{i, i, i}},
			{-1, HashName(), []int{-i, -i, -i}, []int{i, i, i}},
		}
	case i == j && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k}, []int{i, i, k}},
			{-2, HashName(), []int{k}, []int{i, i, k}},
			{1, HashName(), []int{-i, -i, k}, []int{i, i, k}},
			{-1, HashName(), []int{i, i, -k}, []int{i, i, k}},
			{2, HashName(), []int{-k}, []int{i, i, k}},
			{-1, HashName(), []int{-i, -i, -k}, []int{i, i, k}},
		}
	case i == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j}, []int{i, i, j}},
			{-2, HashName(), []int{j}, []int{i, i, j}},
			{1, HashName(), []int{-i, -i, j}, []int{i, i, j}},
			{-1, HashName(), []int{i, i, -j}, []int{i, i, j}},
			{2, HashName(), []int{-j}, []int{i, i, j}},
			{-1, HashName(), []int{-i, -i, -j}, []int{i, i, j}},
		}
	case j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i}, []int{j, j, i}},
			{-2, HashName(), []int{i}, []int{j, j, i}},
			{1, HashName(), []int{-j, -j, i}, []int{j, j, i}},
			{-1, HashName(), []int{j, j, -i}, []int{j, j, i}},
			{2, HashName(), []int{-i}, []int{j, j, i}},
			{-1, HashName(), []int{-j, -j, -i}, []int{j, j, i}},
		}
	case i != j && i != k && j != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k}, []int{i, j, k}},
			{-1, HashName(), []int{i, -j, k}, []int{i, j, k}},
			{-1, HashName(), []int{-i, j, k}, []int{i, j, k}},
			{1, HashName(), []int{-i, -j, k}, []int{i, j, k}},
			{-1, HashName(), []int{i, j, -k}, []int{i, j, k}},
			{1, HashName(), []int{i, -j, -k}, []int{i, j, k}},
			{1, HashName(), []int{-i, j, -k}, []int{i, j, k}},
			{-1, HashName(), []int{-i, -j, -k}, []int{i, j, k}},
		}
	default:
		panic("No cases matched")
	}
}

// Make4D makes the ProtoCalc slices for finite differences fourth
// derivative force constants
func Make4D(i, j, k, l int) []ProtoCalc {
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, i}, []int{i, i, i, i}},
			{-4, HashName(), []int{i, i}, []int{i, i, i, i}},
			{6, "E0", []int{}, []int{i, i, i, i}},
			{-4, HashName(), []int{-i, -i}, []int{i, i, i, i}},
			{1, HashName(), []int{-i, -i, -i, -i}, []int{i, i, i, i}},
		}
	// 3 and 1
	case i == j && i == k && i != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, l}, []int{i, i, i, l}},
			{-3, HashName(), []int{i, l}, []int{i, i, i, l}},
			{3, HashName(), []int{-i, l}, []int{i, i, i, l}},
			{-1, HashName(), []int{-i, -i, -i, l}, []int{i, i, i, l}},
			{-1, HashName(), []int{i, i, i, -l}, []int{i, i, i, l}},
			{3, HashName(), []int{i, -l}, []int{i, i, i, l}},
			{-3, HashName(), []int{-i, -l}, []int{i, i, i, l}},
			{1, HashName(), []int{-i, -i, -i, -l}, []int{i, i, i, l}},
		}
	case i == j && i == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, k}, []int{i, i, i, k}},
			{-3, HashName(), []int{i, k}, []int{i, i, i, k}},
			{3, HashName(), []int{-i, k}, []int{i, i, i, k}},
			{-1, HashName(), []int{-i, -i, -i, k}, []int{i, i, i, k}},
			{-1, HashName(), []int{i, i, i, -k}, []int{i, i, i, k}},
			{3, HashName(), []int{i, -k}, []int{i, i, i, k}},
			{-3, HashName(), []int{-i, -k}, []int{i, i, i, k}},
			{1, HashName(), []int{-i, -i, -i, -k}, []int{i, i, i, k}},
		}
	case i == k && i == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, j}, []int{i, i, i, j}},
			{-3, HashName(), []int{i, j}, []int{i, i, i, j}},
			{3, HashName(), []int{-i, j}, []int{i, i, i, j}},
			{-1, HashName(), []int{-i, -i, -i, j}, []int{i, i, i, j}},
			{-1, HashName(), []int{i, i, i, -j}, []int{i, i, i, j}},
			{3, HashName(), []int{i, -j}, []int{i, i, i, j}},
			{-3, HashName(), []int{-i, -j}, []int{i, i, i, j}},
			{1, HashName(), []int{-i, -i, -i, -j}, []int{i, i, i, j}},
		}
	case j == k && j == l && j != i:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, j, i}, []int{j, j, j, i}},
			{-3, HashName(), []int{j, i}, []int{j, j, j, i}},
			{3, HashName(), []int{-j, i}, []int{j, j, j, i}},
			{-1, HashName(), []int{-j, -j, -j, i}, []int{j, j, j, i}},
			{-1, HashName(), []int{j, j, j, -i}, []int{j, j, j, i}},
			{3, HashName(), []int{j, -i}, []int{j, j, j, i}},
			{-3, HashName(), []int{-j, -i}, []int{j, j, j, i}},
			{1, HashName(), []int{-j, -j, -j, -i}, []int{j, j, j, i}},
		}
	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		// x -> i, y -> k, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, l}, []int{i, i, k, l}},
			{-2, HashName(), []int{k, l}, []int{i, i, k, l}},
			{1, HashName(), []int{-i, -i, k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}},
			{2, HashName(), []int{-k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{-i, -i, -k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{i, i, k, -l}, []int{i, i, k, l}},
			{2, HashName(), []int{k, -l}, []int{i, i, k, l}},
			{-1, HashName(), []int{-i, -i, k, -l}, []int{i, i, k, l}},
			{1, HashName(), []int{i, i, -k, -l}, []int{i, i, k, l}},
			{-2, HashName(), []int{-k, -l}, []int{i, i, k, l}},
			{1, HashName(), []int{-i, -i, -k, -l}, []int{i, i, k, l}},
		}
	case i == k && i != j && i != l && j != l:
		// x -> i, y -> j, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, l}, []int{i, i, j, l}},
			{-2, HashName(), []int{j, l}, []int{i, i, j, l}},
			{1, HashName(), []int{-i, -i, j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{i, i, -j, l}, []int{i, i, j, l}},
			{2, HashName(), []int{-j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{-i, -i, -j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{i, i, j, -l}, []int{i, i, j, l}},
			{2, HashName(), []int{j, -l}, []int{i, i, j, l}},
			{-1, HashName(), []int{-i, -i, j, -l}, []int{i, i, j, l}},
			{1, HashName(), []int{i, i, -j, -l}, []int{i, i, j, l}},
			{-2, HashName(), []int{-j, -l}, []int{i, i, j, l}},
			{1, HashName(), []int{-i, -i, -j, -l}, []int{i, i, j, l}},
		}
	case i == l && i != j && i != k && j != k:
		// x -> i, y -> k, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, j}, []int{i, i, k, j}},
			{-2, HashName(), []int{k, j}, []int{i, i, k, j}},
			{1, HashName(), []int{-i, -i, k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{i, i, -k, j}, []int{i, i, k, j}},
			{2, HashName(), []int{-k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{-i, -i, -k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{i, i, k, -j}, []int{i, i, k, j}},
			{2, HashName(), []int{k, -j}, []int{i, i, k, j}},
			{-1, HashName(), []int{-i, -i, k, -j}, []int{i, i, k, j}},
			{1, HashName(), []int{i, i, -k, -j}, []int{i, i, k, j}},
			{-2, HashName(), []int{-k, -j}, []int{i, i, k, j}},
			{1, HashName(), []int{-i, -i, -k, -j}, []int{i, i, k, j}},
		}
	case j == k && j != i && j != l && i != l:
		// x -> j, y -> i, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, l}, []int{j, j, i, l}},
			{-2, HashName(), []int{i, l}, []int{j, j, i, l}},
			{1, HashName(), []int{-j, -j, i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{j, j, -i, l}, []int{j, j, i, l}},
			{2, HashName(), []int{-i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{-j, -j, -i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{j, j, i, -l}, []int{j, j, i, l}},
			{2, HashName(), []int{i, -l}, []int{j, j, i, l}},
			{-1, HashName(), []int{-j, -j, i, -l}, []int{j, j, i, l}},
			{1, HashName(), []int{j, j, -i, -l}, []int{j, j, i, l}},
			{-2, HashName(), []int{-i, -l}, []int{j, j, i, l}},
			{1, HashName(), []int{-j, -j, -i, -l}, []int{j, j, i, l}},
		}
	case j == l && j != i && j != k && i != k:
		// x -> j, y -> i, z -> k
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, k}, []int{j, j, i, k}},
			{-2, HashName(), []int{i, k}, []int{j, j, i, k}},
			{1, HashName(), []int{-j, -j, i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{j, j, -i, k}, []int{j, j, i, k}},
			{2, HashName(), []int{-i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{-j, -j, -i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{j, j, i, -k}, []int{j, j, i, k}},
			{2, HashName(), []int{i, -k}, []int{j, j, i, k}},
			{-1, HashName(), []int{-j, -j, i, -k}, []int{j, j, i, k}},
			{1, HashName(), []int{j, j, -i, -k}, []int{j, j, i, k}},
			{-2, HashName(), []int{-i, -k}, []int{j, j, i, k}},
			{1, HashName(), []int{-j, -j, -i, -k}, []int{j, j, i, k}},
		}
	case k == l && k != i && k != j && i != j:
		// x -> k, y -> i, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{k, k, i, j}, []int{k, k, i, j}},
			{-2, HashName(), []int{i, j}, []int{k, k, i, j}},
			{1, HashName(), []int{-k, -k, i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{k, k, -i, j}, []int{k, k, i, j}},
			{2, HashName(), []int{-i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{-k, -k, -i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{k, k, i, -j}, []int{k, k, i, j}},
			{2, HashName(), []int{i, -j}, []int{k, k, i, j}},
			{-1, HashName(), []int{-k, -k, i, -j}, []int{k, k, i, j}},
			{1, HashName(), []int{k, k, -i, -j}, []int{k, k, i, j}},
			{-2, HashName(), []int{-i, -j}, []int{k, k, i, j}},
			{1, HashName(), []int{-k, -k, -i, -j}, []int{k, k, i, j}},
		}
	// 2 and 2
	case i == j && k == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, k}, []int{i, i, k, k}},
			{1, HashName(), []int{-i, -i, -k, -k}, []int{i, i, k, k}},
			{1, HashName(), []int{-i, -i, k, k}, []int{i, i, k, k}},
			{1, HashName(), []int{i, i, -k, -k}, []int{i, i, k, k}},
			{-2, HashName(), []int{i, i}, []int{i, i, k, k}},
			{-2, HashName(), []int{k, k}, []int{i, i, k, k}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, k, k}},
			{-2, HashName(), []int{-k, -k}, []int{i, i, k, k}},
			{4, "E0", []int{}, []int{i, i, k, k}},
		}
	case i == k && j == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}},
			{4, "E0", []int{}, []int{i, i, j, j}},
		}
	case i == l && j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}},
			{4, "E0", []int{}, []int{i, i, j, j}},
		}
	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, -j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, j, k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, -j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, j, -k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, -j, -k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, j, -k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, -j, -k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, -j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, j, k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, -j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, j, -k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, -j, -k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, j, -k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, -j, -k, -l}, []int{i, j, k, l}},
		}
	default:
		panic("No cases matched")
	}
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
		if p.Name != "E0" {
			coords := Step(coords, p.Steps...)
			prog.Geometry = ZipXYZ(names, coords) + "}\n"
			fname := dir + p.Name + ".inp"
			fnames = append(fnames, fname)
			prog.WriteInput(fname, none)
			// TODO handle multiple targets
			calcs = append(calcs, Calc{Name: dir + p.Name, Targets: []Target{{p.Coeff, target, Index(len(coords), p.Index...)}}})
		} else {
			// TODO E0 case
		}
	}
	return
}

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

func Index(ncoords int, id ...int) int {
	sort.Ints(id)
	switch ncoords {
	case 2:
		return 3*id[0] + id[1] // TODO actually need to return two here
	case 3:
		return Index3(id[0], id[1], id[2])
	case 4:
		return Index4(id[0], id[1], id[2], id[3])
	}
	panic("wrong number of coords in call to Index")
	return -1
}

// BuildCartPoints constructs the calculations needed to run a
// Cartesian quartic force field
func (mp *Molpro) BuildCartPoints(names []string, coords []float64, fc2, fc3, fc4 *[]float64, ch chan Calc) {
	pbs = ptsMaple // Maple parallel only
	var (
		count int
		pf    int
	)
	dir := "pts/inp"
	subfile := fmt.Sprintf("%s/main%d.pbs", dir, pf)
	cmdfile := fmt.Sprintf("%s/commands%d.txt", dir, pf)
	for i := 1; i <= len(coords); i++ {
		for j := 1; j <= i; j++ {
			files, calcs := Derivative(mp, names, coords, fc2, i, j)
			for f := range files {
				AddCommand(cmdfile, files[f])
				ch <- calcs[f]
				submitted++
				if count == chunkSize {
					subfile = fmt.Sprintf("%s/main%d.pbs", dir, pf)
					cmdfile = fmt.Sprintf("%s/commands%d.txt", dir, pf)
					WritePBS(subfile, &Job{"pts", cmdfile, 35})
					Submit(subfile)
					count = 0
					pf++
				}
				count++
			}
			if nDerivative > 2 {
				for k := 1; k <= j; k++ {
					jobs := Derivative(mp, names, coords, fc3, i, j, k)
					if nDerivative > 3 {
						for l := 1; l <= k; l++ {
							jobs := Derivative(mp, names, coords, fc4, i, j, k, l)
						}
					}
				}
			}
		}
	}
	close(ch)
	return
}
