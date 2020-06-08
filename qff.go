package main

import (
	"bytes"
	"fmt"
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

// Takes a filename like freqs/spectro, runs spectro
// on freqs/spectro.in and redirects the output into
// freqs/spectro.out
func RunSpectro(filename string) {
	err := RunProgram(spectroCmd, filename)
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
