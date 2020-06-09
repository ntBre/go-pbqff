package main

import (
	"testing"
)

func TestWritePBS(t *testing.T) {
	p := Job{MakeName(Input[Geometry]), "opt.inp", 35}
	WritePBS("testfiles/opt/mp.pbs", &p)
}

func TestSubmit(t *testing.T) {
	got := Submit("opt/mp.pbs")
	if got != nil {
		t.Error("got an error, didn't want one")
	}
}
