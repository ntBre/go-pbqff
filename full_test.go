package main

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
)

func TestSIC(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	Conf = NewConfig()
	defer func() {
		flags = 0
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
	}()
	prog, intder, anpass := initialize("tests/sic/sic.in")
	names := strings.Fields("H O H")
	intder.WritePts("tests/sic/pts/intder.in")
	RunIntder("tests/sic/pts/intder")
	var cenergies []CountFloat
	gen := BuildPoints(prog, "tests/sic/pts/file07", names, &cenergies, true)
	E0 := -76.369839620287
	min, _ := Drain(prog, 0, E0, gen)
	energies := FloatsFromCountFloats(cenergies)
	for i := range energies {
		energies[i] -= min
	}
	fmt.Println(filepath.Join(prog.GetDir(), "freqs"))
	longLine, _ := DoAnpass(anpass, filepath.Join(prog.GetDir(), "freqs"), energies, nil)
	coords, _ := DoIntder(intder, names, longLine, prog.GetDir(), false)
	spec, err := spectro.Load("tests/sic/spectro.in")
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, coords)
	spec.WriteInput("tests/sic/freqs/spectro.in")
	err = spec.DoSpectro("tests/sic/freqs/")
	if err != nil {
		errExit(err, "running spectro")
	}
	res := summarize.SpectroFile(
		filepath.Join("tests/sic/", "freqs", "spectro2.out"))
	want := []float64{3753.2, 3656.5, 1598.8}
	if !compfloat(res.Corr, want, 1e-1) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

func compfloat(a, b []float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func TestCart(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	Conf = NewConfig()
	defer func() {
		flags = 0
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
		fc2 = *new([]CountFloat)
		fc3 = *new([]CountFloat)
		fc4 = *new([]CountFloat)
	}()
	prog, _, _ := initialize("tests/cart/cart.in")
	prog.FormatCart(Conf.Str(Geometry))
	cart := prog.GetGeom()
	E0 := prog.Run(none)
	names, coords := XYZGeom(cart)
	natoms := len(names)
	ncoords := len(coords)
	gen := prog.BuildCartPoints("pts/inp", names, coords)
	Drain(prog, ncoords, E0, gen)
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	PrintFortFile(fc2, natoms, 6*natoms, filepath.Join(prog.GetDir(), "fort.15"))
	PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(), "fort.30"))
	PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(), "fort.40"))
	var buf bytes.Buffer
	for i := range coords {
		if i%3 == 0 && i > 0 {
			fmt.Fprint(&buf, "\n")
		}
		fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
	}
	specin := filepath.Join(prog.GetDir(), "spectro.in")
	spec, err := spectro.Load(specin)
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, buf.String())
	spec.WriteInput(specin)
	err = spec.DoSpectro(prog.GetDir())
	if err != nil {
		errExit(err, "running spectro")
	}
	res := summarize.SpectroFile(filepath.Join(prog.GetDir(), "spectro2.out"))
	want := []float64{3753.2, 3656.5, 1598.5}
	if !compfloat(res.Corr, want, 1e-1) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

func TestGrad(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	Conf = NewConfig()
	defer func() {
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
		flags = 0
	}()
	prog, _, _ := initialize("tests/grad/grad.in")
	prog.FormatCart(Conf.Str(Geometry))
	cart := prog.GetGeom()
	E0 := 0.0
	names, coords := XYZGeom(cart)
	natoms := len(names)
	ncoords := len(coords)
	gen := prog.BuildGradPoints("pts/inp", names, coords)
	Drain(prog, ncoords, E0, gen)
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	PrintFortFile(fc2, natoms, 6*natoms, filepath.Join(prog.GetDir(), "fort.15"))
	PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(), "fort.30"))
	PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(), "fort.40"))
	var buf bytes.Buffer
	for i := range coords {
		if i%3 == 0 && i > 0 {
			fmt.Fprint(&buf, "\n")
		}
		fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
	}
	specin := filepath.Join(prog.GetDir(), "spectro.in")
	spec, err := spectro.Load(specin)
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, buf.String())
	spec.WriteInput(specin)
	err = spec.DoSpectro(prog.GetDir())
	if err != nil {
		errExit(err, "running spectro in test")
	}
	res := summarize.SpectroFile(filepath.Join(prog.GetDir(), "spectro2.out"))
	want := []float64{3739.1, 3651.1, 1579.4}
	if !compfloat(res.Corr, want, 1e-1) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}
