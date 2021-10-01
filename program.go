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
	Run(Procedure) float64
	HandleOutput(string) (string, string, error)
	UpdateZmat(string)
	FormatCart(string) error
	BuildCartPoints(string, []string, []float64) func() ([]Calc, bool)
	BuildGradPoints(string, []string, []float64) func() ([]Calc, bool)
	ReadOut(string) (float64, float64, []float64, error)
	ReadFreqs(string) []float64
}

// BuildPoints uses a file07 file from Intder to construct the
// single-point energy calculations and return an array of jobs to
// run. If write is set to true, write the necessary files. Otherwise
// just return the list of jobs.
func BuildPoints(p Program, filename string, atomNames []string,
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
			return Push(filepath.Join(dir, "inp"), pf, count, calcs[start:end]), false
		}
		return Push(filepath.Join(dir, "inp"), pf, count, calcs[start:end]), true
	}
}
