package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
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
	out, err := exec.Command("bash", "-c", progName+" < "+infile+" > "+outfile).Output()
	os.Chdir(current)
	if err != nil {
		return fmt.Errorf("RunProgram: failed with %v running %q on %q\nstdout: %q\n",
			err, progName, infile, out)
	}
	return nil
}

// Takes a filename like pts/intder, runs intder
// on pts/intder.in and redirects the output into
// pts/intder.out
func RunIntder(filename string) {
	err := RunProgram(Input[IntderCmd], filename)
	if err != nil {
		log.Fatal(err)
	}
}

// Takes a filename like freqs/anpass1, runs anpass
// on freqs/anpass1.in and redirects the output into
// freqs/anpass1.out
func RunAnpass(filename string) {
	err := RunProgram(Input[AnpassCmd], filename)
	if err != nil {
		panic(err)
	}
}

// Takes a filename like freqs/spectro, runs spectro
// on freqs/spectro.in and redirects the output into
// freqs/spectro.out
func RunSpectro(filename string) {
	err := RunProgram(Input[SpectroCmd], filename)
	if err != nil {
		panic(err)
	}
}

type Calc struct {
	Name  string
	Index int
}

// Uses ./pts/file07 to construct the single-point
// energy calculations. Return an array of jobs to run
func (mp *Molpro) BuildPoints(filename string, atomNames []string) (jobs []Calc) {
	lines := ReadFile(filename)
	l := len(atomNames)
	i := 0
	var buf bytes.Buffer
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
	geom := 0
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
				pname := basename + ".pbs"
				mp.WriteInput(fname, none)
				tmp := &Job{path.Base(fname), fname, 35}
				WritePBS(pname, tmp)
				jobs = append(jobs, Calc{basename, geom})
				geom++
				buf.Reset()
			}
			fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
			i++
		}
	}
	return
}
