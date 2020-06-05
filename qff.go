package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	intderCmd = "/home/brent/Projects/pbqff/intder"
	anpassCmd = "anpass"
)

var (
	ptable = map[string]string{
		"H": "1", "He": "4", "Li": "7",
		"Be": "9", "B": "11", "C": "12",
		"N": "14", "O": "16", "F": "19",
		"Ne": "20", "Na": "23", "Mg": "24",
		"Al": "27", "Si": "28", "P": "31",
		"S": "32", "Cl": "35", "Ar": "40",
	}
)

type Intder struct {
	Head     string
	Geometry string
	Tail     string
}

// Loads an intder input file with the geometry lines
// stripped out
func LoadIntder(filename string) *Intder {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		buf  bytes.Buffer
		line string
		i    Intder
	)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "DISP") {
			i.Head = buf.String()
			buf.Reset()
		}
		fmt.Fprintln(&buf, line)
	}
	i.Tail = buf.String()
	return &i
}

// Takes a cartesian geometry as a single string
// and formats it as needed by intder, saving the
// result in the passed in Intder
func (i *Intder) ConvertCart(cart string) {
	lines := strings.Split(cart, "\n")
	// slice off last newline
	lines = lines[:len(lines)-1]
	var buf bytes.Buffer
	for _, line := range lines {
		if len(line) > 3 {
			fields := strings.Fields(line)
			x, _ := strconv.ParseFloat(fields[1], 64)
			y, _ := strconv.ParseFloat(fields[2], 64)
			z, _ := strconv.ParseFloat(fields[3], 64)
			fmt.Fprintf(&buf, "%17.9f%19.9f%19.9f\n", x, y, z)
		}
	}
	// remove last newline
	buf.Truncate(buf.Len() - 1)
	i.Geometry = buf.String()
}

// Write an intder.in file for points to filename
func (i *Intder) WritePts(filename string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n" + i.Tail)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// Write an intder_geom.in file to filename, using
// longLine as the displacement
func (i *Intder) WriteGeom(filename, longLine string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n")
	fmt.Fprintf(&buf, "DISP%4d\n", 1)
	fields := strings.Fields(longLine)
	for i, val := range fields[:len(fields)-1] {
		val, _ := strconv.ParseFloat(val, 64)
		// skip values that are zero to the precision of the printing
		if math.Abs(val) > 1e-10 {
			fmt.Fprintf(&buf, "%5d%20.10f\n", i+1, val)
		}
	}
	fmt.Fprintf(&buf, "%5d\n", 0)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// Write an intder.in file for freqs to filename
// TODO might need updating for many atoms - multiline mass format?
func (i *Intder) WriteFreqs(filename string, names []string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n")
	for i, name := range names {
		num, ok := ptable[name]
		if !ok {
			fmt.Errorf("WriteFreqs: element %q not found in ptable\n", name)
		}
		switch i {
		case 0:
			fmt.Fprintf(&buf, "%11s", name+num)
		case 1:
			fmt.Fprintf(&buf, "%13s", name+num)
		default:
			fmt.Fprintf(&buf, "%12s", name+num)
		}
	}
	fmt.Fprint(&buf, "\n")
	buf.WriteString(i.Tail)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// Update i.Geometry with the results of intder_geom
func (i *Intder) ReadGeom(filename string) {
	const target = "NEW CARTESIAN GEOMETRY (BOHR)"
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var (
		line string
		geom bool
		buf  bytes.Buffer
	)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, target) {
			geom = true
			continue
		}
		if geom && line != "" {
			buf.WriteString(line + "\n")
		}
	}
	// skip last newline
	buf.Truncate(buf.Len() - 1)
	i.Geometry = buf.String()
}

// Read a freqs/intder.out and return the harmonic
// frequencies found therein
func (i *Intder) ReadOut(filename string) (freqs []float64) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	mode := regexp.MustCompile(`^\s+MODE`)
	var (
		line  string
		modes bool
	)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case mode.MatchString(line):
			modes = true
		case modes:
			if strings.Contains(line, "NORMAL") {
				return
			}
			if line != "" {
				fields := strings.Fields(line)
				val, _ := strconv.ParseFloat(fields[1], 64)
				freqs = append(freqs, val)
			}
		}
	}
	return
}

// Set i.Tail to ouput from anpass for freqs/intder
func (i *Intder) Read9903(filename string) {
	var buf bytes.Buffer
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		line          string
		third, fourth bool
	)
	for scanner.Scan() {
		line = scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 5 {
			if !third && fields[2] != "0" {
				fmt.Fprintf(&buf, "%5d\n", 0)
				third = true
			}
			if !fourth && fields[3] != "0" {
				fmt.Fprintf(&buf, "%5d\n", 0)
				fourth = true
			}
			if fields[0] != "0" && fields[1] != "0" {
				fmt.Fprintln(&buf, line)
			}
		}
	}
	fmt.Fprintf(&buf, "%5d\n", 0)
	i.Tail = buf.String()
}

// Run a program, redirecting STDIN from filename.in
// and STDOUT to filename.out
func RunProgram(progName, filename string) error {
	current, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fpath := path.Dir(filename)
	if err = os.Chdir(fpath); err != nil {
		panic(err)
	}
	file := path.Base(filename)
	infile := file + ".in"
	outfile := file + ".out"
	_, err = exec.Command("bash", "-c", progName+" < "+infile+" > "+outfile).Output()
	os.Chdir(current)
	return err
}

// Takes a filename like pts/intder, runs intder
// on pts/intder.in and redirects the output into
// pts/intder.out
func RunIntder(filename string) {
	err := RunProgram(intderCmd, filename)
	if err != nil {
		panic(err)
	}
}

// Takes a filename like freqs/anpass1, runs anpass
// on freqs/anpass1.in and redirects the output into
// freqs/anpass1.out
func RunAnpass(filename string) {
	err := RunProgram(anpassCmd, filename)
	if err != nil {
		panic(err)
	}
}

// Uses ./pts/file07 to construct the single-point
// energy calculations. Return an array of jobs to run
func BuildPoints(filename string, atomNames []string) (jobs []string) {
	lines := ReadFile(filename)[1:17]
	l := len(atomNames)
	i := 0
	var buf bytes.Buffer
	mp := Molpro{
		Basis:  Input[Basis],
		Charge: Input[Charge],
		Spin:   Input[Spin],
		Method: Input[Method],
	}
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
	geom := 0
	for _, line := range lines {
		if !strings.Contains(line, "#") {
			ind := i % l
			if ind == 0 && i > 0 {
				mp.Geometry = fmt.Sprint(buf.String(), "}\n")
				basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
				fname := basename + ".inp"
				pname := basename + ".pbs"
				geom++
				mp.WriteInput(fname, "templates/molpro.in")
				tmp := &Job{path.Base(fname), fname, 35}
				WritePBS(pname, "templates/pbs.in", tmp)
				jobs = append(jobs, basename)
				buf.Reset()
			}
			fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
			i++
		}
	}
	return
}
