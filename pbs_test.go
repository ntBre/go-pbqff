package main

import (
	"testing"
)

func TestWritePBS(t *testing.T) {
	p := Job{MakeName(Input[Geometry]), "opt.inp", 35}
	write := "testfiles/write/mp.pbs"
	right := "testfiles/right/mp.pbs"
	WritePBS(write, &p, pbsSequoia)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", right, write)
	}
}

func TestSubmit(t *testing.T) {
	got := Submit("opt/mp.pbs")
	want := "775241"
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
