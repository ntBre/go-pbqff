package main

import (
	"reflect"
	"testing"
)

func TestFormatZmat(t *testing.T) {
	got := FormatZmat(Input[Geometry])
	want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
}
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestWriteInputMolpro(t *testing.T) {
	mp := Molpro{FormatZmat(Input[Geometry]), Input[Basis],
		Input[Charge], Input[Spin], Input[Method]}
	mp.WriteInput("testfiles/opt/opt.inp", "templates/molpro.in")
}
