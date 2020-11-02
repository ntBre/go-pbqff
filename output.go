package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Subtract(a, b []float64) []float64 {
	if len(a) != len(b) {
		panic("Subtract: dimension mismatch")
	}
	diff := make([]float64, 0, len(a))
	for i := range a {
		diff = append(diff, a[i]-b[i])
	}
	return diff
}

// FormatOutput parses existing output files and generates an
// ANPASS-style summary of their displacments and energies
func FormatOutput(dir string) {
	refCoords, refEnergy := ParseOutput(dir+"ref.out", true)
	for _ = range refCoords {
		fmt.Printf("%10.6f", 0.0)
	}
	fmt.Printf("%20.12f\n", 0.0)
	jobfiles, err := filepath.Glob(dir + "job.*.out")
	if err != nil {
		panic(err)
	}
	for _, file := range jobfiles {
		coords, energy := ParseOutput(file, false)
		diff := Subtract(coords, refCoords)
		for _, e := range diff {
			fmt.Printf("%10.6f", e)
		}
		fmt.Printf("%20.12f\n", energy-refEnergy)
	}
}

// ParseReference reads pts/inp/ref.out and returns the reference
// geometry and reference energy
func ParseOutput(file string, comment bool) (coords []float64, energy float64) {
	infile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(infile)
	var (
		ingeom bool
		skip   int
	)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case skip > 0:
			skip--
			continue
		case strings.Contains(line, "geometry={"):
			ingeom = true
			if comment {
				skip += 2
			}
			continue
		case ingeom && strings.Contains(line, "}"):
			ingeom = false
			continue
		case ingeom:
			fields := strings.Fields(line)
			if len(fields) == 4 {
				for _, f := range fields[1:] {
					v, _ := strconv.ParseFloat(f, 64)
					coords = append(coords, v)
				}
			}
		case strings.Contains(line, "energy="):
			fields := strings.Fields(line)
			energy, _ = strconv.ParseFloat(fields[len(fields)-1], 64)
		}
	}
	return
}
