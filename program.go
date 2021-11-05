package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	symm "github.com/ntBre/chemutils/symmetry"
)

// Program is an interface for using different quantum chemical
// programs in the place of Molpro. TODO this is a massive interface,
// how many of these are really necessary?
type Program interface {
	GetDir() string
	SetDir(string)
	GetGeom() string
	SetGeom(string)
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
}

// BuildPoints uses a file07 file from Intder to construct the
// single-point energy calculations and return an array of jobs to
// run. If write is set to true, write the necessary files. Otherwise
// just return the list of jobs.
func BuildPoints(p Program, q Queue, filename string, atomNames []string,
	write bool) func() ([]Calc, bool) {
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
				var (
					norun    bool
					basename string
					targs    []Target
					res      float64
				)
				// last line needs to write first
				if li == len(lines)-1 {
					fmt.Fprintf(&buf, "%s %s\n", atomNames[ind], line)
				}
				for len(cenergies) <= geom {
					cenergies = append(cenergies, CountFloat{Count: 0})
				}
				if !cenergies[geom].Loaded {
					cenergies[geom].Count = 1
					p.SetGeom(p.FormatGeom(buf.String()))
					basename = fmt.Sprintf("%s/inp/%s.%05d",
						dir, name, geom)
					fname := basename + ".inp"
					if write {
						p.WriteInput(fname, none)
					}
					targs = []Target{
						{
							Coeff: 1,
							Slice: &cenergies,
							Index: geom,
						},
					}
				} else {
					norun = true
					res = cenergies[geom].Val
				}
				calcs = append(calcs, Calc{
					Name:    basename,
					Scale:   1.0,
					Targets: targs,
					noRun:   norun,
					Result:  res,
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
			return Push(q, filepath.Join(dir, "inp"), pf, count,
				calcs[start:end]), false
		}
		return Push(q, filepath.Join(dir, "inp"), pf, count,
			calcs[start:end]), true
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BuildCartPoints(p Program, q Queue, dir string, names []string,
	coords []float64, mol symm.Molecule) func() ([]Calc, bool) {
	dir = filepath.Join(p.GetDir(), dir)
	ncoords := len(coords)
	var (
		start int
		pf    int
		count int
	)
	kmax, lmax := ncoords, ncoords
	switch Conf.Int(Deriv) {
	case 4:
	case 3:
		lmax = 0
	case 2:
		lmax = 0
		kmax = 0
	default:
		panic("unrecognized derivative level")
	}
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
				// if we don't want these derivative
				// levels, {k,l}max will be 0 and
				// these will only be entered once
				for k = knit; k <= min(j, kmax); k++ {
					for l = lnit; l <= min(k, lmax); l++ {
						calcs = append(calcs,
							Derivative(p, dir, names,
								coords, i, j, k, l)...,
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
	coords []float64, mol symm.Molecule) func() ([]Calc, bool) {
	dir = filepath.Join(p.GetDir(), dir)
	ncoords := len(coords)
	var (
		start int
		pf    int
		count int
	)
	jmax, kmax := ncoords, ncoords
	switch Conf.Int(Deriv) {
	case 4:
	case 3:
		kmax = 0
	case 2:
		jmax = 0
		kmax = 0
	default:
		panic("unrecognized derivative level")
	}
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
			for j = jnit; j <= min(i, jmax); j++ {
				for k = knit; k <= min(j, kmax); k++ {
					calcs = append(calcs,
						GradDerivative(p, dir, names,
							coords, i, j, k)...,
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

// Derivative is a helper for calling Make(2|3|4)D in the same way
func Derivative(prog Program, dir string, names []string,
	coords []float64, i, j, k, l int) (calcs []Calc) {
	var (
		protos []ProtoCalc
		target *[]CountFloat
		ndims  int
	)
	ncoords := len(coords)
	switch {
	case k == 0 && l == 0:
		protos = Make2D(i, j)
		target = &fc2
		ndims = 2
	case l == 0:
		protos = Make3D(i, j, k)
		target = &fc3
		ndims = 3
	default:
		protos = Make4D(i, j, k, l)
		target = &fc4
		ndims = 4
	}
	for _, p := range protos {
		coords := Step(coords, p.Steps...)
		prog.FormatCart(ZipXYZ(names, coords))
		temp := Calc{
			Name:  filepath.Join(dir, p.Name),
			Scale: p.Scale,
		}
		for _, v := range Index(ncoords, false, p.Index...) {
			for len(*target) <= v {
				*target = append(*target, CountFloat{Val: 0, Count: 0})
			}
			if !(*target)[v].Loaded {
				(*target)[v].Count = len(protos)
			}
			temp.Targets = append(temp.Targets,
				Target{Coeff: p.Coeff, Slice: target, Index: v})
		}
		if len(p.Steps) == 2 && ndims == 2 {
			for _, v := range E2dIndex(ncoords, p.Steps...) {
				// also have to append to e2d, but count is always 1 there
				for len(e2d) <= v {
					e2d = append(e2d, CountFloat{Val: 0, Count: 1})
				}
				// if it was loaded, the count is
				// already 0 from the checkpoint
				if !e2d[v].Loaded {
					e2d[v].Count = 1
				}
				temp.Targets = append(temp.Targets,
					Target{Coeff: 1, Slice: &e2d, Index: v})
			}
		} else if len(p.Steps) == 2 && ndims == 4 {
			// either take fourth derivative from finished
			// e2d point or promise a source for later
			if id := E2dIndex(ncoords, p.Steps...)[0]; len(e2d) > id &&
				e2d[id].Val != 0 {
				temp.Result = e2d[id].Val
			} else {
				temp.Src = &Source{&e2d, id}
			}
			temp.noRun = true
		}
		// if target was loaded, remove it from list of targets
		// then only submit if len(Targets) > 0
		for t := 0; t < len(temp.Targets); {
			targ := temp.Targets[t]
			if (*targ.Slice)[targ.Index].Loaded {
				temp.Targets = append(temp.Targets[:t], temp.Targets[t+1:]...)
			} else {
				t++
			}
		}
		if len(temp.Targets) > 0 {
			fname := filepath.Join(dir, p.Name+".inp")
			if strings.Contains(p.Name, "E0") {
				temp.noRun = true
			}
			if !temp.noRun {
				prog.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}

// GradDerivative is the Derivative analog for Gradients
func GradDerivative(prog Program, dir string, names []string,
	coords []float64, i, j, k int) (calcs []Calc) {
	ncoords := len(coords)
	var (
		protos []ProtoCalc
		dimmax int
		ndims  int
		target *[]CountFloat
	)
	switch {
	case j == 0 && k == 0:
		// gradient second derivatives are just first derivatives and so on
		protos = Make1D(i)
		dimmax = ncoords
		ndims = 1
		target = &fc2
	case k == 0:
		// except E0 needs to be G(ref geom) == 0, handled this in Drain
		protos = Make2D(i, j)
		dimmax = j
		ndims = 2
		target = &fc3
	default:
		protos = Make3D(i, j, k)
		dimmax = k
		ndims = 3
		target = &fc4
	}
	for _, p := range protos {
		coords := Step(coords, p.Steps...)
		prog.FormatCart(ZipXYZ(names, coords))
		temp := Calc{Name: filepath.Join(dir, p.Name), Scale: p.Scale}
		var index int
		for g := 1; g <= dimmax; g++ {
			switch ndims {
			case 1:
				index = Index(ncoords, true, i, g)[0]
			case 2:
				index = Index(ncoords, false, i, j, g)[0]
			case 3:
				index = Index(ncoords, false, i, j, k, g)[0]
			}
			temp.Targets = append(temp.Targets, Target{
				Coeff: p.Coeff,
				Slice: target,
				Index: index,
			})
			for len(*target) <= index {
				*target = append(*target, CountFloat{})
			}
			// every time this index is added as a target,
			// increment its count
			if !(*target)[index].Loaded {
				(*target)[index].Count++
			}
		}
		// if target was loaded, remove it from list of targets
		// then only submit if len(Targets) > 0
		for t := 0; t < len(temp.Targets); {
			targ := temp.Targets[t]
			if (*targ.Slice)[targ.Index].Loaded {
				temp.Targets = append(temp.Targets[:t],
					temp.Targets[t+1:]...)
			} else {
				t++
			}
		}
		if len(temp.Targets) > 0 {
			fname := filepath.Join(dir, p.Name+".inp")
			if strings.Contains(p.Name, "E0") {
				temp.noRun = true
			}
			if !temp.noRun {
				prog.WriteInput(fname, none)
			}
			calcs = append(calcs, temp)
		}
	}
	return
}
