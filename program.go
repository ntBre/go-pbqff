package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// Program is an interface for using different quantum chemical
// programs in the place of Molpro. TODO this is a massive interface,
// how many of these are really necessary?
type Program interface {
	SetDir(string)
	GetDir() string
	SetGeom(string)
	GetGeom() string
	WriteInput(string, Procedure)
	FormatZmat(string) error
	FormatGeom(string) string
	AugmentHead()
	Run(Procedure, Queue) float64
	HandleOutput(string) (string, string, error)
	UpdateZmat(string)
	FormatCart(string) error
	ReadOut(string) (float64, float64, []float64, error)
	ReadFreqs(string) []float64
	Derivative(string, []string, []float64, int, int, int, int) []Calc
	GradDerivative(string, []string, []float64, int, int, int) []Calc
}

// BuildPoints uses a file07 file from Intder to construct the
// single-point energy calculations and return an array of jobs to
// run. If write is set to true, write the necessary files. Otherwise
// just return the list of jobs.
func BuildPoints(p Program, q Queue, filename string, atomNames []string,
	target *[]CountFloat, write bool) func() ([]Calc, bool) {
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
	p.AugmentHead()
	calcs := make([]Calc, 0)
	for li, line := range lines {
		if !strings.Contains(line, "#") {
			ind := i % l
			if (ind == 0 && i > 0) || li == len(lines)-1 {
				// last line needs to write first
				if li == len(lines)-1 {
					fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
				}
				p.SetGeom(p.FormatGeom(buf.String()))
				basename := fmt.Sprintf("%s/inp/%s.%05d", dir, name, geom)
				fname := basename + ".inp"
				if write {
					p.WriteInput(fname, none)
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
			return Push(q, filepath.Join(dir, "inp"), pf, count, calcs[start:end]), false
		}
		return Push(q, filepath.Join(dir, "inp"), pf, count, calcs[start:end]), true
	}
}

func BuildCartPoints(p Program, q Queue, dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(p.GetDir(), dir)
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
							p.Derivative(dir, names, coords, i, j, k, l)...,
						)
						if len(calcs) >= Conf.Int(ChunkSize) {
							jnit, knit, lnit = j, k, l+1
							return Push(q, dir, pf, count, calcs), true
						}
					}
					lnit = 0
				}
				knit = 0
			}
			jnit = 1
		}
		return Push(q, dir, pf, count, calcs), false
	}
}

// BuildGradPoints constructs the calculations needed to run a
// Cartesian quartic force field using gradients
func BuildGradPoints(p Program, q Queue, dir string, names []string,
	coords []float64) func() ([]Calc, bool) {
	dir = filepath.Join(p.GetDir(), dir)
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
						p.GradDerivative(dir, names, coords, i, j, k)...,
					)
					if len(calcs) >= Conf.Int(ChunkSize) {
						jnit, knit = j, k+1
						return Push(q, dir, pf, count, calcs), true
					}
				}
				knit = 0
			}
			jnit = 0
		}
		return Push(q, dir, pf, count, calcs), false
	}
}
