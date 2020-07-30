package main

import (
	"testing"
)

func TestWritePBS(t *testing.T) {
	temp := pbs
	pbs = pbsSequoia
	defer func() {
		pbs = temp
	}()
	p := Job{MakeName(Input[Geometry]), "opt.inp", 35}
	write := "testfiles/write/mp.pbs"
	right := "testfiles/right/mp.pbs"
	WritePBS(write, &p)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", right, write)
	}
}

func TestSubmit(t *testing.T) {
	got := Submit("opt/mp.pbs")
	if got != nil {
		t.Error("got an error, didn't want one")
	}
}
