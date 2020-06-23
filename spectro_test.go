package main

import (
	"reflect"
	"testing"
)

var (
	names  = []string{"Al", "O", "O", "Al"}
	coords = `0.000000000        2.391678166        0.000000000
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000       -2.391678166        0.000000000
`
)

func TestLoadSpectro(t *testing.T) {
	LoadSpectro("testfiles/spectro.in", names, coords)
}

func TestWriteSpectroInput(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in", names, coords)
	spec.WriteInput("testfiles/freqs/spectro.in")
}

func TestReadSpectroOutput(t *testing.T) {
	t.Run("all resonances present", func(t *testing.T) {
		spec := LoadSpectro("testfiles/spectro.in", names, coords)
		spec.ReadOutput("testfiles/spectro.out")
	})
	t.Run("no fermi 2 resonances present", func(t *testing.T) {
		spec := LoadSpectro("testfiles/spectro.in", names, coords)
		spec.ReadOutput("testfiles/spectro.prob")
	})
}

func TestCheckPolyad(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in", names, coords)
	spec.Nfreqs = 6
	spec.ReadOutput("testfiles/spectro.out")
	spec.WriteInput("testfiles/freqs/spectro2.in")
}

func TestMakeKey(t *testing.T) {
	got := MakeKey([]int{1, 2, 3})
	want := "1 2 3"
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestResinLine(t *testing.T) {
	t.Run("One frequency on lhs", func(t *testing.T) {
		got := ResinLine(6, 2, 2)
		want := "    0    2    0    0    0    0\n"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("two frequencies on lhs", func(t *testing.T) {
		got := ResinLine(6, 1, 2, 1)
		want := "    1    1    0    0    0    0\n"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}

func TestFreqReport(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in", names, coords)
	spec.Nfreqs = 6
	spec.ReadOutput("testfiles/spectro.out")
	gzpt, gharm, gfund, gcorr,
		rotABC, deltas, phis := spec.FreqReport("testfiles/spectro.out")
	wzpt := 4682.2527
	wharm := []float64{3811.360, 2337.700, 1267.577, 1086.351, 496.788, 437.756}
	wfund := []float64{3623.015, 2299.805, 1231.309, 1081.661, 513.228, 454.579}
	wcorr := []float64{3623.0149, 2299.8053, 1231.3094, 1081.6611, 513.2276, 454.5787}
	wrotABC := [][]float64{
		{0.3533242, 0.3473852, 22.5883184},
		{0.3531433, 0.3469946, 21.5758850},
		{0.3508629, 0.3449969, 22.5509605},
		{0.3536392, 0.3472748, 23.9984685},
		{0.3517191, 0.3456623, 22.5514979},
		{0.3538316, 0.3484810, 53.7297798},
		{0.3547570, 0.3480413, -8.6483579},
	}
	wdeltas := []float64{
		0.0041596072,
		276.2016104107,
		0.6722227103,
		0.0000596455,
		0.3035637199,
	}
	wphis := []float64{
		-0.4946484183E-03,
		0.2374310264E+06,
		0.3252182153E+01,
		-0.2896689993E+04,
		0.4401912504E-04,
		0.1940400502E+01,
		0.6140585605E+04,
	}
	if gzpt != wzpt {
		t.Errorf("got %f, wanted %f\n", gzpt, wzpt)
	}
	if !reflect.DeepEqual(gharm, wharm) {
		t.Errorf("got %v, wanted %v\n", gharm, wharm)
	}
	if !reflect.DeepEqual(gfund, wfund) {
		t.Errorf("got %v, wanted %v\n", gfund, wfund)
	}
	if !reflect.DeepEqual(gcorr, wcorr) {
		t.Errorf("got %v, wanted %v\n", gcorr, wcorr)
	}
	if !reflect.DeepEqual(rotABC, wrotABC) {
		t.Errorf("got %v, wanted %v\n", rotABC, wrotABC)
	}
	if !reflect.DeepEqual(deltas, wdeltas) {
		t.Errorf("got %v, wanted %v\n", deltas, wdeltas)
	}
	if !reflect.DeepEqual(phis, wphis) {
		t.Errorf("got %v, wanted %v\n", phis, wphis)
	}
}
