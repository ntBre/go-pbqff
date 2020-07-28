package main

import (
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
	spec, _ := LoadSpectro("testfiles/spectro.in", names, coords)
	spec.WriteInput("testfiles/freqs/spectro.in")
}

func TestReadSpectroOutput(t *testing.T) {
	t.Run("all resonances present", func(t *testing.T) {
		spec, _ := LoadSpectro("testfiles/spectro.in", names, coords)
		spec.ReadOutput("testfiles/spectro.out")
	})
	t.Run("no fermi 2 resonances present", func(t *testing.T) {
		spec, _ := LoadSpectro("testfiles/spectro.in", names, coords)
		spec.ReadOutput("testfiles/spectro.prob")
	})
}

func TestCheckPolyad(t *testing.T) {
	spec, _ := LoadSpectro("testfiles/spectro.in", names, coords)
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
