package main

import (
	"testing"
)

func TestWriteInputPBS(t *testing.T) {
	p := PBS{MakeName(Input[Geometry]), "opt.inp"}
	p.WriteInput("testfiles/opt/mp.pbs", "templates/pbs.in")
}
