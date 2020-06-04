package main

import (
	"reflect"
	"testing"
)

func TestNewIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	got := NewIntder(cart)
	want := &Intder{`      1.000000000        0.118481857       -2.183553663
      0.000000000       -1.563325812       -2.884671935
      0.000000000       -0.014536611        0.273763522
      0.000000000       -0.010373662        2.467030139`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestWritePtsIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := NewIntder(cart)
	i.WritePtsIntder("testfiles/pts/intder.in", "templates/intder.pts")
}

func TestRunIntder(t *testing.T) {
	RunIntder("testfiles/pts/intder")
}

func TestBuildPoints(t *testing.T) {
	prog := Molpro{
		Geometry: Input[Geometry],
		Basis:    Input[Basis],
		Charge:   Input[Charge],
		Spin:     Input[Spin],
		Method:   Input[Method],
	}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	names := GetNames(cart)
	got := BuildPoints("testfiles/pts/file07", names)
	want := []string{
		"testfiles/pts/inp/NHHH.00000",
		"testfiles/pts/inp/NHHH.00001",
		"testfiles/pts/inp/NHHH.00002",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
