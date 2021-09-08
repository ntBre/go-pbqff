package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed embed/taylor.py
var taylor string

func Taylor(names []string, intder *Intder) {
	// Geometry has to match the coordinates in the template file
	fields := strings.Fields(intder.Geometry)
	coords := make([]float64, len(fields))
	for i, f := range fields {
		coords[i], _ = strconv.ParseFloat(f, 64)
	}
	// mol := symm.ReadXYZ(strings.NewReader(ZipXYZ(names, coords)))
	params := strings.Fields(strings.Split(intder.Head, "\n")[1])
	var (
		nsic    int
		nsicStr string
	)
	if params[2] == "0" {
		// accept number of simple internals if no SICs
		nsic, _ = strconv.Atoi(params[1])
		nsicStr = params[1]
	} else {
		nsic, _ = strconv.Atoi(params[2])
		nsicStr = params[2]
	}
	var str strings.Builder
	fmt.Fprintf(&str, "DISP%4d\n", nsic)
	for i := 0; i < nsic; i++ {
		fmt.Fprintf(&str, "%5d %18.10f\n%5d\n", i+1, 0.005, 0)
	}
	// These are the only fields needed by WritePts
	tmpder := &Intder{
		Head:     intder.Head,
		Geometry: intder.Geometry,
		Tail:     str.String(),
	}
	dir := os.TempDir()
	infile := filepath.Join(dir, "intder")
	tmpder.WritePts(infile + ".in")
	RunIntder(infile)
	// TODO parse file07 - extract the read file07 part from
	// Molpro.BuildPoints and reuse it here. see python and CL
	// implementations
	flags := ""
	// Example usage:
	// groups are b2, b1, a2, although b2 vs b1 shouldn't matter
	// python2 taylor.py 5 3 -m 2:[2-2,0-0,0-0] -q 1:[2-2,0-0,0-0]
	cmd := exec.Command("python2", "-c", taylor,
		// hard-code deriv=4 for now, giving 5
		"5", nsicStr, flags)
	cmd.Run()
	// symm.ReadXYZ(cartesian geometry) -> Molecule

	// actually need to take the geometry from the intder input
	// since it has to be in the right order relative to the
	// coordinates

	// then read intder.in, write one disp for each SIC, run
	// intder on it to get file07, call symm.Symmetry(Molecule) on
	// each of the resulting geometries to get the irreps for the
	// modes

	// sort the SICs if needed, probably skip this for now

	// run taylor.py with the input corresponding to the order -
	// should make a directory for this since it's going to be
	// messy

	// parse taylor.py output files to generate anpass and the
	// rest of the intder file. this also means I can actually use
	// delta and deltas keywords for SICs now
}
