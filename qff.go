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
		"H": "1", "HE": "4", "LI": "7",
		"BE": "9", "B": "11", "C": "12",
		"N": "14", "O": "16", "F": "19",
		"NE": "20", "NA": "23", "MG": "24",
		"AL": "27", "SI": "28", "P": "31",
		"S": "32", "CL": "35", "AR": "40",
	}
)

// RunProgram runs a program, redirecting STDIN from filename.in
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
		return fmt.Errorf("error RunProgram: failed with %v running %q on %q"+
			"\nstdout: %q",
			err, progName, infile, out)
	}
	return nil
}

// RunIntder takes a filename like pts/intder, runs intder
// on pts/intder.in and redirects the output into
// pts/intder.out
func RunIntder(filename string) {
	err := RunProgram(Input[IntderCmd], filename)
	if err != nil {
		log.Fatal(err)
	}
}

// RunAnpass takes a filename like freqs/anpass1, runs anpass
// on freqs/anpass1.in and redirects the output into
// freqs/anpass1.out
func RunAnpass(filename string) {
	err := RunProgram(Input[AnpassCmd], filename)
	if err != nil {
		panic(err)
	}
}

// RunSpectro takes a filename like freqs/spectro, runs spectro
// on freqs/spectro.in and redirects the output into
// freqs/spectro.out
func RunSpectro(filename string) {
	err := RunProgram(Input[SpectroCmd], filename)
	if err != nil {
		panic(err)
	}
}

// Calc holds the name of a job to be run and its result's index in
// the output array
type Calc struct {
	Name  string
	Index int
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
func (mp *Molpro) BuildPoints(filename string, atomNames []string, write bool) (jobs []Calc) {
	lines := ReadFile(filename)
	l := len(atomNames)
	i := 0
	var buf bytes.Buffer
	dir := path.Dir(filename)
	name := strings.Join(atomNames, "")
	geom := 0
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
					mp.WriteInput(fname, none)
					AddCommand(fname)
				}
				jobs = append(jobs, Calc{basename, geom})
				geom++
				buf.Reset()
			}
			fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
			i++
		}
	}
	if write {
		// TODO maple specific for now
		pbs = ptsMaple
		WritePBS("main.pbs", &Job{"pts", "", 35})
	}
	return
}
