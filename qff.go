package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const (
	intderCmd = "/home/brent/Projects/pbqff/intder"
	anpassCmd = "anpass"
)

type Intder struct {
	Geometry string
}

func NewIntder(cart string) *Intder {
	lines := strings.Split(cart, "\n")
	// slice off last newline
	lines = lines[:len(lines)-1]
	var buf bytes.Buffer
	for i, line := range lines {
		if len(line) > 3 {
			fields := strings.Fields(line)
			x, _ := strconv.ParseFloat(fields[1], 64)
			y, _ := strconv.ParseFloat(fields[2], 64)
			z, _ := strconv.ParseFloat(fields[3], 64)
			fmt.Fprintf(&buf, "%17.9f%19.9f%19.9f", x, y, z)
			if i < len(lines)-1 {
				fmt.Fprint(&buf, "\n")
			}
		}
	}
	return &Intder{buf.String()}
}

// Takes the target intder filename, cartesian geometry
// and an intder template file and writes an intder input file
// for use in pts
func (i *Intder) WritePtsIntder(filename, tfile string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	t := LoadTemplate(tfile)
	t.Execute(f, i)
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
