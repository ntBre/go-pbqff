package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
)

func TestSIC(t *testing.T) {
	// TODO testing full QFF procedure using dummy interface
	// implementations
}

func TestCart(t *testing.T) {
	*test = true
	qsub = "/home/brent/Projects/go/src/github.com/ntBre/chemutils/qsub/qsub"
	defer func() {
		*test = false
		qsub = "qsub"
	}()
	prog, _, _ := initialize("tests/cart/cart.in")
	prog.FormatCart(Conf.Str(Geometry))
	cart := prog.Geometry
	E0 := prog.RefEnergy()
	ch := make(chan Calc, Conf.Int(JobLimit))
	names, coords := XYZGeom(cart)
	natoms := len(names)
	ncoords := len(coords)
	go func() {
		prog.BuildCartPoints("pts/inp", names, coords,
			&fc2, &fc3, &fc4, ch)
	}()
	Drain(prog, ncoords, ch, E0)
	N3N := natoms * 3 // from spectro manual pg 12
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	PrintFortFile(fc2, natoms, 6*natoms, "fort.15")
	PrintFortFile(fc3, natoms, other3, "fort.30")
	PrintFortFile(fc4, natoms, other4, "fort.40")
	var buf bytes.Buffer
	for i := range coords {
		if i%3 == 0 && i > 0 {
			fmt.Fprint(&buf, "\n")
		}
		fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
	}
	spec, err := spectro.Load("spectro.in")
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, buf.String())
	spec.WriteInput("spectro.in")
	err = spec.DoSpectro(".")
	if err != nil {
		errExit(err, "running spectro")
	}
	res := summarize.Spectro("spectro2.out")
	want := []float64{3753.1, 3656.8, 1599.9}
	if !reflect.DeepEqual(res.Corr, want) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

func TestGrad(t *testing.T) {
	// TODO
}