package main

import (
	"reflect"
	"testing"
)

func TestLoadSpectro(t *testing.T) {
	LoadSpectro("testfiles/spectro.in")
}

func TestWriteSpectroInput(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in")
	spec.WriteInput("testfiles/freqs/spectro.in")
}

func TestReadSpectroOutput(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in")
	spec.ReadOutput("testfiles/spectro.out")
}

func TestCheckPolyad(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in")
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
	spec := LoadSpectro("testfiles/spectro.in")
	spec.Nfreqs = 6
	spec.ReadOutput("testfiles/spectro.out")
	gzpt, gharm, gfund, gcorr := spec.FreqReport("testfiles/spectro.out")
	wzpt := 4682.2527
	wharm := []float64{3811.360, 2337.700, 1267.577, 1086.351, 496.788, 437.756}
	wfund := []float64{3623.015, 2299.805, 1231.309, 1081.661, 513.228, 454.579}
	wcorr := []float64{3623.0149, 2299.8053, 1231.3094, 1081.6611, 513.2276, 454.5787}
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
}
